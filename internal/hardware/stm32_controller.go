package hardware

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tarm/serial"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// STM32Config 配置
type STM32Config struct {
	Port         string        // 串口端口
	BaudRate     int           // 波特率
	DataBits     int           // 数据位
	StopBits     int           // 停止位
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时
	RetryCount   int           // 重试次数
	HeartbeatInterval time.Duration // 心跳间隔
}

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
	ackCh       chan *Frame
	
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

// PendingCommand 待确认的命令
type PendingCommand struct {
	Cmd      byte
	Seq      uint16
	Time     time.Time
	Response chan error
}

// CoinStatistics 币统计数据
type CoinStatistics struct {
	CoinsInserted      uint16    // 投入的币数
	CoinsDispensed     uint16    // 上币数量
	CoinsReturnedFront uint16    // 前方回币
	CoinsReturnedLeft  uint16    // 左侧回币
	CoinsReturnedRight uint16    // 右侧回币
	CoinsRefunded      uint16    // 退币数量
	TicketsPrinted     uint16    // 彩票打印数量
	FaultCount         uint8     // 故障次数
	RecoveryCount      uint8     // 恢复次数
	GameDuration       uint32    // 游戏时长（秒）
	Timestamp          time.Time // 统计时间戳
	ReturnRate         float64   // 回币率
}

// GameLogicInterface 游戏逻辑接口
type GameLogicInterface interface {
	GetCurrentMode() byte      // 获取当前模式（退币/彩票）
	HasCredits() bool          // 是否有余额
	GetPendingCoins() uint16  // 获取待上币数量
	AddCredits(count byte)     // 增加余额
	AddPlayerCoins(count byte) // 增加玩家币数
	UpdateReturnRate(rate float64) // 更新回币率
	GetRefundableCoins() uint16   // 获取可退币数
	GetAvailableTickets() uint16  // 获取可用彩票数
	DeductCoins(count uint16)     // 扣除币数
	RedeemTickets(count uint16)   // 兑换彩票
	StartGame(coinCount uint16)   // 开始游戏
	SetDifficulty(level byte)     // 设置难度
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
		ackCh:       make(chan *Frame, 10),
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

// sendCommand 发送命令并等待ACK
func (c *STM32Controller) sendCommand(cmd byte, data []byte) error {
	return c.sendCommandWithTimeout(cmd, data, 3*time.Second)
}

// sendCommandWithTimeout 发送命令并等待ACK（带超时）
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
	
	// 等待响应
	select {
	case err := <-respCh:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("wait ACK timeout for cmd 0x%02X seq %d", cmd, seq)
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
	n, err := c.port.Write(data)
	if err != nil {
		return err
	}
	
	if n != len(data) {
		return fmt.Errorf("incomplete write: %d/%d", n, len(data))
	}
	
	c.logger.Debug("Frame sent",
		zap.Uint8("cmd", frame.Command),
		zap.Uint16("seq", frame.Sequence),
		zap.Int("len", len(data)))
	
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
			if err.Error() != "EOF" {
				c.logger.Error("Read error", zap.Error(err))
			}
			continue
		}
		
		if n > 0 {
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
	
	switch frame.Command {
	case CmdACK:
		c.handleACK(frame)
	case CmdNACK:
		c.handleNACK(frame)
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

// handleACK 处理ACK响应
func (c *STM32Controller) handleACK(frame *Frame) {
	if len(frame.Data) < 2 {
		c.logger.Error("Invalid ACK data length")
		return
	}
	
	origSeq := binary.BigEndian.Uint16(frame.Data[0:2])
	// origCmd := frame.Data[2]
	// status := frame.Data[3]
	
	// 查找待确认命令
	c.cmdMu.Lock()
	pending, ok := c.pendingCmds[origSeq]
	c.cmdMu.Unlock()
	
	if ok && pending.Response != nil {
		pending.Response <- nil // 成功
	}
}

// handleNACK 处理NACK响应
func (c *STM32Controller) handleNACK(frame *Frame) {
	if len(frame.Data) < 3 {
		c.logger.Error("Invalid NACK data length")
		return
	}
	
	origSeq := binary.BigEndian.Uint16(frame.Data[0:2])
	origCmd := frame.Data[2]
	errorCode := frame.Data[3]
	
	// 查找待确认命令
	c.cmdMu.Lock()
	pending, ok := c.pendingCmds[origSeq]
	c.cmdMu.Unlock()
	
	if ok && pending.Response != nil {
		pending.Response <- fmt.Errorf("NACK: cmd=0x%02X, error=0x%02X", origCmd, errorCode)
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
