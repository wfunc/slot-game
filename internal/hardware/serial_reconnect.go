package hardware

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	
	"github.com/tarm/serial"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// SerialPortExists 检查串口设备是否存在
func SerialPortExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SerialReconnectManager 串口重连管理器
type SerialReconnectManager struct {
	deviceType      string           // 设备类型（ACM或STM32）
	devicePattern   string           // 设备名称模式（如 "ttyACM" 或 "ttyS"）
	config          *serial.Config   // 串口配置
	port            *serial.Port     // 当前串口连接
	logger          *zap.Logger      // 日志记录器
	
	connected       bool             // 连接状态
	reconnecting    bool             // 是否正在重连
	lastDevicePath  string           // 最后成功连接的设备路径
	
	onConnect       func(*serial.Port) error  // 连接成功回调
	onDisconnect    func()                    // 断开连接回调
	onReconnect     func(*serial.Port) error  // 重连成功回调
	
	stopCh          chan struct{}    // 停止通道
	reconnectCh     chan struct{}    // 触发重连通道
	mu              sync.RWMutex     // 保护并发访问
}

// NewSerialReconnectManager 创建串口重连管理器
func NewSerialReconnectManager(deviceType string, pattern string, config *serial.Config) *SerialReconnectManager {
	return &SerialReconnectManager{
		deviceType:    deviceType,
		devicePattern: pattern,
		config:        config,
		logger:        logger.GetLogger(),
		reconnectCh:   make(chan struct{}, 1),
	}
}

// SetCallbacks 设置回调函数
func (m *SerialReconnectManager) SetCallbacks(
	onConnect func(*serial.Port) error,
	onDisconnect func(),
	onReconnect func(*serial.Port) error,
) {
	m.onConnect = onConnect
	m.onDisconnect = onDisconnect
	m.onReconnect = onReconnect
	
	m.logger.Info("回调函数已设置",
		zap.String("device_type", m.deviceType),
		zap.Bool("onConnect_set", onConnect != nil),
		zap.Bool("onDisconnect_set", onDisconnect != nil),
		zap.Bool("onReconnect_set", onReconnect != nil))
}

// Start 启动管理器
func (m *SerialReconnectManager) Start() error {
	m.mu.Lock()
	
	if m.stopCh != nil {
		m.mu.Unlock()
		return fmt.Errorf("重连管理器已启动")
	}
	
	m.stopCh = make(chan struct{})
	
	// 记录回调状态
	m.logger.Info("Start()开始执行，检查回调状态",
		zap.String("device_type", m.deviceType),
		zap.Bool("onConnect_exists", m.onConnect != nil),
		zap.Bool("onDisconnect_exists", m.onDisconnect != nil),
		zap.Bool("onReconnect_exists", m.onReconnect != nil))
	
	// 在调用connect()之前释放锁，避免死锁
	m.mu.Unlock()
	
	// 尝试初始连接（不持有锁）
	if err := m.connect(); err != nil {
		m.logger.Warn("初始连接失败，将在后台重试",
			zap.String("device_type", m.deviceType),
			zap.Error(err))
		// 触发重连
		m.triggerReconnect()
	}
	
	// 启动重连监控
	go m.reconnectLoop()
	
	return nil
}

// Stop 停止管理器
func (m *SerialReconnectManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.stopCh != nil {
		close(m.stopCh)
		m.stopCh = nil
	}
	
	if m.port != nil {
		m.port.Close()
		m.port = nil
	}
	
	m.connected = false
}

// TriggerReconnect 手动触发重连（用于错误处理）
func (m *SerialReconnectManager) TriggerReconnect() {
	m.triggerReconnect()
}

// triggerReconnect 内部触发重连
func (m *SerialReconnectManager) triggerReconnect() {
	select {
	case m.reconnectCh <- struct{}{}:
	default:
		// 已经有重连请求在队列中
	}
}

// IsConnected 检查连接状态
func (m *SerialReconnectManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetPort 获取当前串口
func (m *SerialReconnectManager) GetPort() *serial.Port {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.port
}

// GetCurrentDevice 获取当前设备路径
func (m *SerialReconnectManager) GetCurrentDevice() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastDevicePath
}

// connect 执行连接
func (m *SerialReconnectManager) connect() error {
	// 查找设备
	device := m.findDevice()
	if device == "" {
		return fmt.Errorf("未找到%s设备", m.deviceType)
	}
	
	// 更新配置中的设备路径
	m.config.Name = device
	
	// 打开串口
	port, err := serial.OpenPort(m.config)
	if err != nil {
		return fmt.Errorf("打开串口失败: %w", err)
	}
	
	m.port = port
	m.lastDevicePath = device
	m.connected = true
	
	m.logger.Info("串口连接成功",
		zap.String("device_type", m.deviceType),
		zap.String("device", device))
	
	// 调用连接回调
	if m.onConnect != nil {
		m.logger.Info("准备调用连接回调",
			zap.String("device_type", m.deviceType))
		m.logger.Info("回调前状态检查",
			zap.Bool("port_valid", port != nil),
			zap.Bool("connected", m.connected))
		
		err := m.onConnect(port)
		
		m.logger.Info("回调函数返回",
			zap.Bool("has_error", err != nil))
		
		if err != nil {
			m.logger.Error("连接回调失败",
				zap.String("device_type", m.deviceType),
				zap.Error(err))
			port.Close()
			m.port = nil
			m.connected = false
			return err
		}
		m.logger.Info("连接回调执行成功",
			zap.String("device_type", m.deviceType))
	} else {
		m.logger.Warn("连接回调未设置",
			zap.String("device_type", m.deviceType))
	}
	
	return nil
}

// findDevice 查找设备
func (m *SerialReconnectManager) findDevice() string {
	// 优先尝试最后成功的设备
	if m.lastDevicePath != "" {
		if SerialPortExists(m.lastDevicePath) {
			return m.lastDevicePath
		}
	}
	
	// 搜索所有可能的设备
	for i := 0; i < 10; i++ {
		device := fmt.Sprintf("/dev/%s%d", m.devicePattern, i)
		if SerialPortExists(device) {
			m.logger.Info("找到设备",
				zap.String("device_type", m.deviceType),
				zap.String("device", device))
			return device
		}
	}
	
	return ""
}

// reconnectLoop 重连循环
func (m *SerialReconnectManager) reconnectLoop() {
	reconnectInterval := 5 * time.Second
	maxInterval := 30 * time.Second
	
	for {
		select {
		case <-m.stopCh:
			m.logger.Info("停止重连循环",
				zap.String("device_type", m.deviceType))
			return
			
		case <-m.reconnectCh:
			m.mu.Lock()
			if m.reconnecting {
				m.mu.Unlock()
				continue
			}
			m.reconnecting = true
			m.mu.Unlock()
			
			m.logger.Info("开始重连",
				zap.String("device_type", m.deviceType))
			
			// 断开现有连接
			m.disconnect()
			
			// 尝试重连
			retryCount := 0
			for {
				select {
				case <-m.stopCh:
					m.mu.Lock()
					m.reconnecting = false
					m.mu.Unlock()
					return
				default:
				}
				
				retryCount++
				m.logger.Info("尝试重连",
					zap.String("device_type", m.deviceType),
					zap.Int("retry", retryCount))
				
				m.mu.Lock()
				err := m.connect()
				m.mu.Unlock()
				
				if err == nil {
					// 重连成功
					m.logger.Info("重连成功",
						zap.String("device_type", m.deviceType),
						zap.String("device", m.lastDevicePath),
						zap.Int("retry_count", retryCount))
					
					// 调用重连回调
					if m.onReconnect != nil {
						m.mu.RLock()
						port := m.port
						m.mu.RUnlock()
						
						if err := m.onReconnect(port); err != nil {
							m.logger.Error("重连回调失败",
								zap.String("device_type", m.deviceType),
								zap.Error(err))
							// 继续重试
							time.Sleep(reconnectInterval)
							continue
						}
					}
					
					m.mu.Lock()
					m.reconnecting = false
					m.mu.Unlock()
					break
				}
				
				// 重连失败，等待后重试
				m.logger.Warn("重连失败，等待重试",
					zap.String("device_type", m.deviceType),
					zap.Error(err),
					zap.Duration("interval", reconnectInterval))
				
				time.Sleep(reconnectInterval)
				
				// 逐渐增加重连间隔
				if reconnectInterval < maxInterval {
					reconnectInterval = reconnectInterval * 2
					if reconnectInterval > maxInterval {
						reconnectInterval = maxInterval
					}
				}
			}
			
			// 重置重连间隔
			reconnectInterval = 5 * time.Second
		}
	}
}

// disconnect 断开连接
func (m *SerialReconnectManager) disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.port != nil {
		m.logger.Info("断开串口连接",
			zap.String("device_type", m.deviceType),
			zap.String("device", m.lastDevicePath))
		
		// 调用断开回调
		if m.onDisconnect != nil {
			m.onDisconnect()
		}
		
		// 关闭串口
		m.port.Close()
		m.port = nil
	}
	
	m.connected = false
}

// HandleError 处理串口错误（检测断线）
func (m *SerialReconnectManager) HandleError(err error) {
	if err == nil {
		return
	}
	
	// 检查是否是断线错误
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "input/output error") ||
		strings.Contains(errStr, "device not configured") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "permission denied") {
		
		m.logger.Error("检测到串口断线",
			zap.String("device_type", m.deviceType),
			zap.String("device", m.lastDevicePath),
			zap.Error(err))
		
		// 触发重连
		m.triggerReconnect()
	}
}