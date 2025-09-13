package hardware

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// HardwareManager 硬件管理器
// 负责管理STM32控制器、ACM控制器和游戏逻辑的交互
type HardwareManager struct {
	mu     sync.RWMutex
	logger *zap.Logger

	// 核心组件
	controller    HardwareController // 硬件控制器接口（STM32）
	acmController *ACMController     // ACM算法控制器
	gameLogic     *GameLogicAdapter  // 游戏逻辑适配器

	// 运行状态
	ctx     context.Context
	cancel  context.CancelFunc
	running bool

	// 配置
	config *HardwareConfig

	// 统计
	stats *HardwareStats

	// 事件处理器
	eventHandlers map[string]EventHandler

	// 桥接模式
	bridgeEnabled bool
}

// HardwareConfig 硬件配置
type HardwareConfig struct {
	// STM32串口配置
	SerialPort string `json:"serial_port"`
	BaudRate   int    `json:"baud_rate"`

	// ACM设备配置
	ACMEnabled    bool   `json:"acm_enabled"`
	ACMPort       string `json:"acm_port"`
	ACMBaudRate   int    `json:"acm_baud_rate"`
	ACMAutoDetect bool   `json:"acm_auto_detect"`

	// 游戏配置
	InitialCredits int64    `json:"initial_credits"`
	GameMode       GameMode `json:"game_mode"`

	// 运行配置
	AutoStart         bool          `json:"auto_start"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	BridgeMode        bool          `json:"bridge_mode"` // ACM<->STM32桥接模式

	// 开发配置
	DebugMode bool `json:"debug_mode"`
	MockMode  bool `json:"mock_mode"` // 模拟模式（不连接真实硬件）
}

// DefaultHardwareConfig 默认配置
func DefaultHardwareConfig() *HardwareConfig {
	return &HardwareConfig{
		SerialPort:        "/dev/ttyS3",
		BaudRate:          9600, // STM32实际使用9600
		ACMEnabled:        true,
		ACMPort:           "auto",
		ACMBaudRate:       115200,
		ACMAutoDetect:     true,
		GameMode:          ModeCoinRefund,
		HeartbeatInterval: 30 * time.Second,
		BridgeMode:        true,
		AutoStart:         false,
		DebugMode:         false,
		MockMode:          false,
	}
}

// HardwareStats 硬件统计
type HardwareStats struct {
	mu sync.RWMutex

	// 通信统计
	PacketsSent     uint64
	PacketsReceived uint64
	PacketsLost     uint64
	LastHeartbeat   time.Time

	// 硬件事件统计
	CoinInserted  uint64
	CoinReturned  uint64
	ButtonPressed uint64
	FaultCount    uint32

	// 运行时间
	StartTime time.Time
	Uptime    time.Duration
}

// EventHandler 事件处理器
type EventHandler func(event interface{})

// NewHardwareManager 创建硬件管理器
func NewHardwareManager(config *HardwareConfig) *HardwareManager {
	if config == nil {
		config = DefaultHardwareConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HardwareManager{
		logger:        logger.GetLogger(),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		stats:         &HardwareStats{},
		eventHandlers: make(map[string]EventHandler),
	}
}

// Initialize 初始化硬件管理器
func (m *HardwareManager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("hardware manager already running")
	}

	// 创建游戏逻辑适配器
	m.gameLogic = NewGameLogicAdapter()
	m.gameLogic.SetCurrentMode(m.config.GameMode)

	// 设置游戏逻辑回调
	m.gameLogic.SetCallbacks(
		m.onGameStart,
		m.onGameEnd,
		m.onBalanceChange,
	)

	// 根据配置选择控制器
	if m.config.MockMode {
		m.logger.Info("使用模拟控制器")
		// 这里可以使用MockController进行测试
		return m.initializeMockController()
	}

	// 创建STM32控制器配置
	stm32Config := &STM32Config{
		Port:              m.config.SerialPort,
		BaudRate:          m.config.BaudRate,
		DataBits:          8,
		StopBits:          2,
		ReadTimeout:       100 * time.Millisecond,
		WriteTimeout:      100 * time.Millisecond,
		RetryCount:        3,
		HeartbeatInterval: m.config.HeartbeatInterval,
	}

	// 创建STM32控制器
	m.controller = NewSTM32Controller(stm32Config, m.gameLogic)

	// 初始化ACM控制器（如果启用）
	if m.config.ACMEnabled {
		acmConfig := &ACMConfig{
			Port:         m.config.ACMPort,
			BaudRate:     m.config.ACMBaudRate,
			AutoDetect:   m.config.ACMAutoDetect,
			ReadTimeout:  100 * time.Millisecond,
			WriteTimeout: 100 * time.Millisecond,
		}

		m.acmController = NewACMController(acmConfig)

		// 设置STM32控制器引用（用于桥接）
		if stm32, ok := m.controller.(*STM32Controller); ok {
			m.acmController.SetSTM32Controller(stm32)
		}

		// 如果启用桥接模式，设置桥接
		if m.config.BridgeMode {
			m.setupBridge()
			m.bridgeEnabled = true
		}

		m.logger.Info("ACM控制器已配置",
			zap.String("port", m.config.ACMPort),
			zap.Int("baudRate", m.config.ACMBaudRate),
			zap.Bool("bridgeMode", m.config.BridgeMode))
	}

	// 设置硬件事件回调
	m.setupEventCallbacks()

	m.logger.Info("硬件管理器初始化完成",
		zap.String("stm32Port", m.config.SerialPort),
		zap.Int("stm32BaudRate", m.config.BaudRate),
		zap.Bool("acmEnabled", m.config.ACMEnabled),
		zap.Bool("mockMode", m.config.MockMode))

	return nil
}

// Start 启动硬件管理器
func (m *HardwareManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("hardware manager already running")
	}

	if m.controller == nil {
		return fmt.Errorf("hardware not initialized")
	}

	// 连接STM32硬件
	if err := m.controller.Connect(); err != nil {
		return fmt.Errorf("connect STM32 hardware failed: %w", err)
	}

	// 连接ACM设备（如果启用）
	if m.acmController != nil {
		if err := m.acmController.Connect(); err != nil {
			m.logger.Warn("连接ACM设备失败，继续运行但不使用ACM功能", zap.Error(err))
			// 不返回错误，允许系统在没有ACM的情况下运行
			m.acmController = nil
		} else {
			m.logger.Info("ACM设备连接成功")
		}
	}

	m.running = true
	m.stats.StartTime = time.Now()

	// 启动监控协程
	go m.monitorLoop()

	// 发送初始状态查询
	go func() {
		time.Sleep(1 * time.Second)
		if err := m.controller.QueryStatus(0x01); err != nil {
			m.logger.Error("查询初始状态失败", zap.Error(err))
		}
	}()

	m.logger.Info("硬件管理器启动成功")

	return nil
}

// Stop 停止硬件管理器
func (m *HardwareManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	m.cancel()

	// 断开ACM连接
	if m.acmController != nil {
		if err := m.acmController.Disconnect(); err != nil {
			m.logger.Error("断开ACM连接失败", zap.Error(err))
		}
	}

	// 断开STM32硬件连接
	if m.controller != nil {
		if err := m.controller.Disconnect(); err != nil {
			m.logger.Error("断开STM32硬件连接失败", zap.Error(err))
		}
	}

	m.logger.Info("硬件管理器已停止")

	return nil
}

// setupEventCallbacks 设置硬件事件回调
func (m *HardwareManager) setupEventCallbacks() {
	// 根据控制器类型设置回调
	switch ctrl := m.controller.(type) {
	case *STM32Controller:
		ctrl.onCoinInserted = func(count byte) {
			m.stats.mu.Lock()
			m.stats.CoinInserted += uint64(count)
			m.stats.mu.Unlock()

			m.logger.Info("检测到投币", zap.Uint8("count", count))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["coin_inserted"]; ok {
				handler(count)
			}
		}

		ctrl.onCoinReturned = func(data *CoinReturnData) {
			m.stats.mu.Lock()
			m.stats.CoinReturned += uint64(data.FrontCount + data.LeftCount + data.RightCount)
			m.stats.mu.Unlock()

			m.logger.Info("检测到回币",
				zap.Uint8("front", data.FrontCount),
				zap.Uint8("left", data.LeftCount),
				zap.Uint8("right", data.RightCount))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["coin_returned"]; ok {
				handler(data)
			}
		}

		ctrl.onButtonPressed = func(event *ButtonEvent) {
			m.stats.mu.Lock()
			m.stats.ButtonPressed++
			m.stats.mu.Unlock()

			m.logger.Info("检测到按键",
				zap.Uint8("type", event.KeyType),
				zap.Uint8("code", event.KeyCode),
				zap.Uint8("action", event.Action))

			// 处理游戏按键
			if event.KeyType == KeyTypeGame && event.Action == KeyActionDown {
				m.handleGameButton(event.KeyCode)
			}

			// 触发事件处理器
			if handler, ok := m.eventHandlers["button_pressed"]; ok {
				handler(event)
			}
		}

		ctrl.onFaultReport = func(event *FaultEvent) {
			m.stats.mu.Lock()
			m.stats.FaultCount++
			m.stats.mu.Unlock()

			m.logger.Error("硬件故障",
				zap.Uint8("code", event.FaultCode),
				zap.Uint8("level", event.Level))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["fault"]; ok {
				handler(event)
			}

			// 尝试自动恢复
			if event.Level <= 2 { // 提示或警告级别
				go m.attemptAutoRecovery(event.FaultCode)
			}
		}

	case *MockController:
		ctrl.onCoinInserted = func(count byte) {
			m.stats.mu.Lock()
			m.stats.CoinInserted += uint64(count)
			m.stats.mu.Unlock()

			m.logger.Info("检测到投币", zap.Uint8("count", count))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["coin_inserted"]; ok {
				handler(count)
			}
		}

		ctrl.onCoinReturned = func(data *CoinReturnData) {
			m.stats.mu.Lock()
			m.stats.CoinReturned += uint64(data.FrontCount + data.LeftCount + data.RightCount)
			m.stats.mu.Unlock()

			m.logger.Info("检测到回币",
				zap.Uint8("front", data.FrontCount),
				zap.Uint8("left", data.LeftCount),
				zap.Uint8("right", data.RightCount))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["coin_returned"]; ok {
				handler(data)
			}
		}

		ctrl.onButtonPressed = func(event *ButtonEvent) {
			m.stats.mu.Lock()
			m.stats.ButtonPressed++
			m.stats.mu.Unlock()

			m.logger.Info("检测到按键",
				zap.Uint8("type", event.KeyType),
				zap.Uint8("code", event.KeyCode),
				zap.Uint8("action", event.Action))

			// 处理游戏按键
			if event.KeyType == KeyTypeGame && event.Action == KeyActionDown {
				m.handleGameButton(event.KeyCode)
			}

			// 触发事件处理器
			if handler, ok := m.eventHandlers["button_pressed"]; ok {
				handler(event)
			}
		}

		ctrl.onFaultReport = func(event *FaultEvent) {
			m.stats.mu.Lock()
			m.stats.FaultCount++
			m.stats.mu.Unlock()

			m.logger.Error("硬件故障",
				zap.Uint8("code", event.FaultCode),
				zap.Uint8("level", event.Level))

			// 触发事件处理器
			if handler, ok := m.eventHandlers["fault"]; ok {
				handler(event)
			}

			// 尝试自动恢复
			if event.Level <= 2 { // 提示或警告级别
				go m.attemptAutoRecovery(event.FaultCode)
			}
		}
	}
}

// handleGameButton 处理游戏按键
func (m *HardwareManager) handleGameButton(keyCode byte) {
	switch keyCode {
	case KeyStart:
		// 开始按键 - 执行上币
		if m.gameLogic.HasCredits() {
			coins := m.gameLogic.GetPendingCoins()
			if coins > 0 {
				if err := m.controller.DispenseCoins(coins, 5); err != nil {
					m.logger.Error("上币失败", zap.Error(err))
				} else {
					m.gameLogic.StartGame(coins)
					// 启动推币
					m.controller.PushControl(PushActionContinuous, 0)
				}
			}
		}

	case KeyRefundTicket:
		// 退币/彩票按键 - 根据模式处理
		mode := m.gameLogic.GetCurrentMode()
		if mode == byte(ModeCoinRefund) {
			// 退币模式
			coins := m.gameLogic.GetRefundableCoins()
			if coins > 0 {
				if err := m.controller.RefundCoins(coins); err != nil {
					m.logger.Error("退币失败", zap.Error(err))
				} else {
					m.gameLogic.DeductCoins(coins)
				}
			}
		} else {
			// 彩票模式
			tickets := m.gameLogic.GetAvailableTickets()
			if tickets > 0 {
				if err := m.controller.PrintTickets(tickets); err != nil {
					m.logger.Error("打印彩票失败", zap.Error(err))
				} else {
					m.gameLogic.RedeemTickets(tickets)
				}
			}
		}
	}
}

// attemptAutoRecovery 尝试自动恢复
func (m *HardwareManager) attemptAutoRecovery(faultCode byte) {
	m.logger.Info("尝试自动恢复", zap.Uint8("faultCode", faultCode))

	// 根据故障类型选择恢复策略
	var action byte
	switch faultCode {
	case 0x01, 0x02, 0x05: // 电机故障
		action = 0x04 // 重试操作
	case 0x03, 0x04: // 彩票机故障
		action = 0x02 // 清除故障标志
	default:
		action = 0x02 // 默认清除故障
	}

	if err := m.controller.FaultRecovery(faultCode, action, 3); err != nil {
		m.logger.Error("自动恢复失败", zap.Error(err))
	} else {
		m.logger.Info("自动恢复成功")
	}
}

// monitorLoop 监控循环
func (m *HardwareManager) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			// 更新运行时间
			m.stats.mu.Lock()
			m.stats.Uptime = time.Since(m.stats.StartTime)
			m.stats.mu.Unlock()

			// 检查心跳
			if m.controller != nil {
				stats := m.controller.GetStatistics()
				m.stats.mu.Lock()
				m.stats.LastHeartbeat = stats.Timestamp
				m.stats.mu.Unlock()
			}
		}
	}
}

// 游戏逻辑回调函数
func (m *HardwareManager) onGameStart(coins uint16) {
	m.logger.Info("游戏开始回调", zap.Uint16("coins", coins))
}

func (m *HardwareManager) onGameEnd(result *GameResult) {
	m.logger.Info("游戏结束回调",
		zap.Bool("won", result.Won),
		zap.Int64("winAmount", result.WinAmount))
}

func (m *HardwareManager) onBalanceChange(credits int64) {
	m.logger.Debug("余额变化", zap.Int64("credits", credits))
}

// initializeMockController 初始化模拟控制器
func (m *HardwareManager) initializeMockController() error {
	// 创建模拟控制器
	m.controller = NewMockController(m.gameLogic)

	// 设置硬件事件回调
	m.setupEventCallbacks()

	m.logger.Info("模拟控制器初始化完成")
	return nil
}

// RegisterEventHandler 注册事件处理器
func (m *HardwareManager) RegisterEventHandler(event string, handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandlers[event] = handler
}

// GetStatistics 获取统计信息
func (m *HardwareManager) GetStatistics() map[string]interface{} {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()

	gameStats := m.gameLogic.GetStatistics()

	return map[string]interface{}{
		"hardware": map[string]interface{}{
			"packetsSent":     m.stats.PacketsSent,
			"packetsReceived": m.stats.PacketsReceived,
			"packetsLost":     m.stats.PacketsLost,
			"lastHeartbeat":   m.stats.LastHeartbeat,
			"coinInserted":    m.stats.CoinInserted,
			"coinReturned":    m.stats.CoinReturned,
			"buttonPressed":   m.stats.ButtonPressed,
			"faultCount":      m.stats.FaultCount,
			"uptime":          m.stats.Uptime.String(),
		},
		"game": gameStats,
	}
}

// IsRunning 是否正在运行
func (m *HardwareManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetController 获取硬件控制器（用于直接操作）
func (m *HardwareManager) GetController() HardwareController {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.controller
}

// GetGameLogic 获取游戏逻辑
func (m *HardwareManager) GetGameLogic() *GameLogicAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gameLogic
}

// GetACMController 获取ACM控制器
func (m *HardwareManager) GetACMController() *ACMController {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acmController
}

// setupBridge 设置ACM和STM32之间的桥接
func (m *HardwareManager) setupBridge() {
	if m.acmController == nil || m.controller == nil {
		m.logger.Warn("无法设置桥接，控制器未初始化")
		return
	}

	// 设置ACM命令处理器
	m.acmController.SetCommandHandler(func(cmd string) (interface{}, error) {
		// 处理algo命令
		if strings.HasPrefix(cmd, "algo") {
			// algo命令由ACM控制器内部处理
			// 这里可以添加与STM32的交互逻辑
			return nil, nil
		}

		// 其他命令可以转发到STM32
		return m.forwardACMToSTM32(cmd)
	})

	m.logger.Info("ACM<->STM32桥接已设置")
}

// forwardACMToSTM32 将ACM命令转发到STM32
func (m *HardwareManager) forwardACMToSTM32(cmd string) (interface{}, error) {
	// 根据ACM命令构建STM32协议帧
	switch cmd {
	case "status":
		// 查询状态
		if err := m.controller.QueryStatus(0x01); err != nil {
			return nil, err
		}
		// 返回状态信息
		return map[string]interface{}{
			"status":  "ok",
			"running": m.running,
		}, nil

	case "coin_insert":
		// 模拟投币
		if err := m.controller.DispenseCoins(1, 0); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"result": "coin_inserted",
		}, nil

	default:
		return nil, fmt.Errorf("未知命令: %s", cmd)
	}
}

// SendAlgoCommand 发送algo命令到ACM设备
func (m *HardwareManager) SendAlgoCommand(bet int, prize int) (map[string]interface{}, error) {
	if m.acmController == nil {
		return nil, fmt.Errorf("ACM控制器未初始化")
	}

	return m.acmController.SendAlgoCommand(bet, prize)
}
