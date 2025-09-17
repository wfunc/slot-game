//go:build !noserialhw
// +build !noserialhw

package hardware

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tarm/serial"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)


// DefaultSTM32Config 默认配置
func DefaultSTM32Config() *STM32Config {
	return &STM32Config{
		Port:         "/dev/ttyS3",
		BaudRate:     115200,  // 标准波特率
		DataBits:     8,
		StopBits:     2,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
		RetryCount:   3,
		HeartbeatInterval: 30 * time.Second,
	}
}

// STM32Controller STM32控制器
type STM32Controller struct {
	config      *STM32Config
	port        SerialPort    // Changed to interface for testing
	sequence    uint32        // 序列号（原子操作）
	mu          sync.RWMutex
	connected   bool
	logger      *zap.Logger
	
	// 通道
	stopCh      chan struct{}
	eventCh     chan *Frame
	echoCh      chan *Frame  // v1.2: Echo确认通道
	
	// 待确认命令
	pendingCmds map[uint16]*PendingCommand
	cmdMu       sync.RWMutex
	
	// 回调函数
	onCoinInserted  func(count byte)
	onCoinReturned  func(data *CoinReturnData)
	onButtonPressed func(event *ButtonEvent)
	onFaultReport   func(event *FaultEvent)
	
	// 统计数据
	stats       CoinStatistics
	statsMu     sync.RWMutex
	
	// 游戏逻辑接口
	gameLogic   GameLogicInterface
	
	// 资源锁定
	resourceLock sync.Mutex    // 资源操作锁
	lockedResources map[byte]bool // 已锁定的资源ID映射
}



// NewSTM32Controller 创建STM32控制器
func NewSTM32Controller(config *STM32Config, gameLogic GameLogicInterface) *STM32Controller {
	if config == nil {
		config = DefaultSTM32Config()
	}
	
	return &STM32Controller{
		config:      config,
		logger:      logger.GetLogger(),
		stopCh:      make(chan struct{}),
		eventCh:     make(chan *Frame, 100),
		echoCh:      make(chan *Frame, 10),  // v1.2: Echo通道
		pendingCmds: make(map[uint16]*PendingCommand),
		gameLogic:   gameLogic,
		lockedResources: make(map[byte]bool),
	}
}

// Connect 连接串口
func (c *STM32Controller) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected {
		return nil
	}
	
	// 配置串口 - 8O2配置（8数据位，奇校验，2停止位）
	cfg := &serial.Config{
		Name:        c.config.Port,
		Baud:        c.config.BaudRate,
		Size:        byte(c.config.DataBits),
		StopBits:    serial.Stop2,     // 2个停止位
		Parity:      serial.ParityOdd,  // 奇校验
		ReadTimeout: c.config.ReadTimeout,
	}
	
	// 打开串口
	port, err := serial.OpenPort(cfg)
	if err != nil {
		c.logger.Error("Failed to open serial port", 
			zap.String("port", c.config.Port),
			zap.Error(err))
		return fmt.Errorf("open serial port failed: %w", err)
	}
	
	c.port = port
	c.connected = true
	
	// 加载历史统计数据
	if err := c.loadStatistics(); err != nil {
		c.logger.Warn("Failed to load statistics", zap.Error(err))
		// 不影响连接，继续执行
	}
	
	// 启动后台任务
	go c.readLoop()
	go c.eventLoop()
	go c.heartbeatLoop()
	
	c.logger.Info("STM32 controller connected",
		zap.String("port", c.config.Port),
		zap.Int("baudrate", c.config.BaudRate))
	
	return nil
}

// Disconnect 断开连接
func (c *STM32Controller) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected {
		return nil
	}
	
	// 停止后台任务
	close(c.stopCh)
	
	// 保存统计数据
	c.saveStatistics()
	
	// 关闭串口
	if c.port != nil {
		err := c.port.Close()
		if err != nil {
			c.logger.Error("Failed to close serial port", zap.Error(err))
			return err
		}
		c.port = nil
	}
	
	c.connected = false
	c.logger.Info("STM32 controller disconnected")
	
	return nil
}

// IsConnected 检查连接状态
func (c *STM32Controller) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// getNextSeq 获取下一个序列号（奇数）
func (c *STM32Controller) getNextSeq() uint16 {
	seq := atomic.AddUint32(&c.sequence, 2)
	// 确保是奇数
	if seq%2 == 0 {
		seq++
	}
	return uint16(seq)
}

// sendCommand 发送命令并等待Echo确认 (v1.2)
func (c *STM32Controller) sendCommand(cmd byte, data []byte) error {
	return c.sendCommandWithTimeout(cmd, data, 3*time.Second)
}

// sendCommandWithTimeout 发送命令并等待Echo（带超时） (v1.2)
func (c *STM32Controller) sendCommandWithTimeout(cmd byte, data []byte, timeout time.Duration) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// 检查资源锁定状态
	resourceID := c.getResourceIDForCommand(cmd)
	if resourceID != 0 {
		c.resourceLock.Lock()
		if c.lockedResources[resourceID] {
			c.resourceLock.Unlock()
			return fmt.Errorf("resource %d is locked for cmd 0x%02X", resourceID, cmd)
		}
		c.lockedResources[resourceID] = true
		c.resourceLock.Unlock()
		
		// 确保资源解锁
		defer func() {
			c.resourceLock.Lock()
			delete(c.lockedResources, resourceID)
			c.resourceLock.Unlock()
		}()
	}
	
	seq := c.getNextSeq()
	frame := NewFrame(cmd, seq, data)
	
	// 创建待确认命令
	respCh := make(chan error, 1)
	pending := &PendingCommand{
		Cmd:      cmd,
		Seq:      seq,
		Time:     time.Now(),
		Response: respCh,
	}
	
	// 记录待确认命令
	c.cmdMu.Lock()
	c.pendingCmds[seq] = pending
	c.cmdMu.Unlock()
	
	// 清理函数
	defer func() {
		c.cmdMu.Lock()
		delete(c.pendingCmds, seq)
		c.cmdMu.Unlock()
	}()
	
	// 发送帧
	if err := c.writeFrame(frame); err != nil {
		return fmt.Errorf("write frame failed: %w", err)
	}
	
	// 等待Echo确认 (v1.2)
	select {
	case err := <-respCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("wait Echo timeout for cmd 0x%02X seq %d", cmd, seq)
	}
}

// getResourceIDForCommand 根据命令获取资源ID
func (c *STM32Controller) getResourceIDForCommand(cmd byte) byte {
	// 资源映射：不同命令对应的资源ID
	// 只对真正的物理冲突资源进行锁定
	switch cmd {
	case CmdCoinDispense:
		return 1 // 上币电机资源
	case CmdCoinRefund:
		return 2 // 退币电机资源（独立的电机）
	case CmdTicketPrint:
		return 3 // 打印机资源
	case CmdPushControl:
		return 4 // 推币机构资源
	case CmdLightControl:
		return 0 // 灯光控制不需要锁定（可并发）
	default:
		return 0 // 无需锁定资源
	}
}

// writeFrame 写入数据帧
func (c *STM32Controller) writeFrame(frame *Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.port == nil {
		return fmt.Errorf("port not open")
	}
	
	data := frame.ToBytes()
	
	// 打印发送数据（十六进制）
	c.logger.Info("STM32发送数据",
		zap.String("hex", fmt.Sprintf("% X", data)),
		zap.Int("bytes", len(data)),
		zap.Uint8("cmd", frame.Command),
		zap.Uint16("seq", frame.Sequence))
	
	n, err := c.port.Write(data)
	if err != nil {
		return err
	}
	
	if n != len(data) {
		return fmt.Errorf("incomplete write: %d/%d", n, len(data))
	}
	
	c.logger.Debug("STM32发送成功", zap.Int("bytes_written", n))
	
	return nil
}

// readLoop 读取循环
func (c *STM32Controller) readLoop() {
	buf := make([]byte, 4096)
	frameBuf := make([]byte, 0, 4096)
	
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		
		// 读取数据
		n, err := c.port.Read(buf)
		if err != nil {
			if err.Error() != "EOF" && !strings.Contains(err.Error(), "timeout") {
				c.logger.Error("Read error", zap.Error(err))
			}
			continue
		}
		
		if n > 0 {
			// 打印接收到的原始数据（十六进制）
			c.logger.Debug("STM32接收原始数据",
				zap.String("hex", fmt.Sprintf("% X", buf[:n])),
				zap.Int("bytes", n))
			
			frameBuf = append(frameBuf, buf[:n]...)
			
			// 尝试解析帧
			for len(frameBuf) >= int(MinFrameLen) {
				// 查找帧头
				idx := -1
				for i := 0; i < len(frameBuf); i++ {
					if frameBuf[i] == FrameHeader {
						idx = i
						break
					}
				}
				
				if idx < 0 {
					// 没有找到帧头，清空缓冲区
					frameBuf = frameBuf[:0]
					break
				}
				
				if idx > 0 {
					// 丢弃帧头之前的数据
					frameBuf = frameBuf[idx:]
				}
				
				// 检查长度字段
				if len(frameBuf) < 3 {
					break
				}
				
				frameLen := binary.BigEndian.Uint16(frameBuf[1:3])
				if len(frameBuf) < int(frameLen) {
					// 数据不完整，等待更多数据
					break
				}
				
				// 解析帧
				frame := &Frame{}
				if err := frame.FromBytes(frameBuf[:frameLen]); err != nil {
					c.logger.Error("Parse frame failed", zap.Error(err))
					// 丢弃这个错误的帧
					frameBuf = frameBuf[1:]
					continue
				}
				
				// 打印接收到的完整帧
				c.logger.Info("STM32接收完整帧",
					zap.String("hex", fmt.Sprintf("% X", frameBuf[:frameLen])),
					zap.Uint8("cmd", frame.Command),
					zap.Uint16("seq", frame.Sequence),
					zap.Int("dataLen", len(frame.Data)))
				
				// 处理帧
				c.handleFrame(frame)
				
				// 移除已处理的帧
				frameBuf = frameBuf[frameLen:]
			}
		}
	}
}

// handleFrame 处理接收到的帧
func (c *STM32Controller) handleFrame(frame *Frame) {
	c.logger.Debug("Frame received",
		zap.Uint8("cmd", frame.Command),
		zap.Uint16("seq", frame.Sequence))
	
	// v1.2: 检查是否是Echo响应
	if c.isEchoResponse(frame) {
		c.handleEcho(frame)
		return
	}

	switch frame.Command {
	case EventCoinInserted:
		c.handleCoinInserted(frame)
	case EventCoinReturned:
		c.handleCoinReturned(frame)
	case EventButtonPressed:
		c.handleButtonPressed(frame)
	case EventSensorTriggered:
		c.handleSensorEvent(frame)
	case EventStatusReport:
		c.handleStatusReport(frame)
	case EventFaultReport:
		c.handleFaultReport(frame)
	case EventProgress:
		c.handleProgress(frame)
	case CmdHeartbeat:
		c.handleHeartbeat(frame)
	default:
		c.logger.Warn("Unknown command", zap.Uint8("cmd", frame.Command))
	}
}

// isEchoResponse 检查是否是Echo响应 (v1.2)
func (c *STM32Controller) isEchoResponse(frame *Frame) bool {
	c.cmdMu.RLock()
	_, ok := c.pendingCmds[frame.Sequence]
	c.cmdMu.RUnlock()
	return ok
}

// handleEcho 处理Echo确认 (v1.2)
func (c *STM32Controller) handleEcho(frame *Frame) {
	c.logger.Debug("Echo received",
		zap.Uint8("cmd", frame.Command),
		zap.Uint16("seq", frame.Sequence))

	// 查找待确认命令
	c.cmdMu.Lock()
	pending, ok := c.pendingCmds[frame.Sequence]
	c.cmdMu.Unlock()

	if !ok {
		c.logger.Warn("No pending command for echo", zap.Uint16("seq", frame.Sequence))
		return
	}

	// 验证命令码是否匹配
	if pending.Cmd == frame.Command {
		// Echo确认成功
		if pending.Response != nil {
			pending.Response <- nil
		}
	} else {
		// Echo命令码不匹配
		if pending.Response != nil {
			pending.Response <- fmt.Errorf("echo command mismatch: expected 0x%02X, got 0x%02X",
				pending.Cmd, frame.Command)
		}
	}
}

// eventLoop 事件处理循环
func (c *STM32Controller) eventLoop() {
	for {
		select {
		case <-c.stopCh:
			return
		case frame := <-c.eventCh:
			// 异步处理事件
			go c.processEvent(frame)
		}
	}
}

// processEvent 处理事件
func (c *STM32Controller) processEvent(frame *Frame) {
	// 根据事件类型处理
	switch frame.Command {
	case EventCoinInserted:
		// 投币事件已在handleCoinInserted中处理
	case EventCoinReturned:
		// 回币事件已在handleCoinReturned中处理
	case EventButtonPressed:
		// 按键事件已在handleButtonPressed中处理
	default:
		c.logger.Debug("Event processed", zap.Uint8("cmd", frame.Command))
	}
}

// heartbeatLoop 心跳循环
func (c *STM32Controller) heartbeatLoop() {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			if err := c.SendHeartbeat(); err != nil {
				c.logger.Error("Heartbeat failed", zap.Error(err))
			}
		}
	}
}

// PushCoin 推币动作（根据力度和持续时间）
func (c *STM32Controller) PushCoin(force int, duration time.Duration) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// 将力度转换为速度参数 (0-100 -> 0-255)
	speed := byte(force * 255 / 100)
	
	// 先设置推币速度
	if err := c.SetPushSpeed(speed); err != nil {
		return err
	}
	
	// 开始推币
	if err := c.StartPushing(); err != nil {
		return err
	}
	
	// 等待指定时间
	time.Sleep(duration)
	
	// 停止推币
	return c.StopPushing()
}
// PrintTickets 打印彩票
func (c *STM32Controller) PrintTickets(count uint16) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// 构造数据
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, count)
	
	// 发送命令
	return c.sendCommand(CmdTicketPrint, data)
}

// FaultRecovery 故障恢复
func (c *STM32Controller) FaultRecovery(faultCode byte, action byte, retryCount byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// 构造数据
	data := []byte{
		faultCode,  // 故障代码
		action,     // 恢复动作
		retryCount, // 重试次数
	}
	
	// 发送命令
	return c.sendCommand(CmdFaultRecovery, data)
}

// LightControl 灯光控制
func (c *STM32Controller) LightControl(pattern byte, brightness byte, duration byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	// 构造数据
	data := []byte{
		pattern,    // 灯光模式
		brightness, // 亮度 (0-100)
		duration,   // 持续时间（秒，0表示持续）
	}
	
	// 发送命令
	return c.sendCommand(CmdLightControl, data)
}
