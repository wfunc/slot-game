//go:build noserialhw
// +build noserialhw

package hardware

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// STM32Controller 模拟STM32控制器 (无硬件版本)
// 这个版本用于在没有串口硬件的环境中编译和运行
type STM32Controller struct {
	config      STM32Config
	logger      *zap.Logger
	gameLogic   GameLogic
	isConnected atomic.Value

	// 通道
	stopCh       chan struct{}
	commandQueue chan Command
	echoCh       chan *Frame

	// 统计
	stats      STM32Stats
	statsMu    sync.RWMutex

	// 回调函数
	onCoinInserted func(count uint8)
	onCoinReturned func(position string, count uint8)
	onJackpot      func(data JackpotData)
	onDisconnect   func()

	// 内部状态
	currentSeq  uint16
	seqMu       sync.Mutex
	commandMap  map[uint16]chan *Frame
	commandMu   sync.RWMutex
	lastHeartbeatTime time.Time
	reconnectDelay    time.Duration
	maxReconnectDelay time.Duration
}

// NewSTM32Controller 创建新的STM32控制器（无硬件版本）
func NewSTM32Controller(config STM32Config, gameLogic GameLogic) (*STM32Controller, error) {
	logger := logger.NewLogger("stm32_nohw", "info", "")

	c := &STM32Controller{
		config:            config,
		logger:            logger,
		gameLogic:         gameLogic,
		stopCh:            make(chan struct{}),
		commandQueue:      make(chan Command, 100),
		echoCh:            make(chan *Frame, 10),
		commandMap:        make(map[uint16]chan *Frame),
		reconnectDelay:    time.Second,
		maxReconnectDelay: time.Minute,
	}

	c.isConnected.Store(false)

	c.logger.Info("STM32 Controller initialized (no hardware version)",
		zap.String("mode", "simulation"))

	return c, nil
}

// Connect 模拟连接
func (c *STM32Controller) Connect() error {
	c.logger.Info("Simulating STM32 connection")
	c.isConnected.Store(true)
	c.lastHeartbeatTime = time.Now()

	// 启动模拟的读写协程
	go c.simulateReader()
	go c.simulateWriter()
	go c.heartbeatMonitor()

	return nil
}

// Disconnect 模拟断开连接
func (c *STM32Controller) Disconnect() error {
	c.logger.Info("Disconnecting simulated STM32")

	if !c.isConnected.Load().(bool) {
		return nil
	}

	c.isConnected.Store(false)
	close(c.stopCh)

	if c.onDisconnect != nil {
		c.onDisconnect()
	}

	return nil
}

// IsConnected 检查连接状态
func (c *STM32Controller) IsConnected() bool {
	return c.isConnected.Load().(bool)
}

// SendCommand 发送命令（模拟）
func (c *STM32Controller) SendCommand(cmd Command) error {
	if !c.IsConnected() {
		return fmt.Errorf("STM32 not connected (simulation)")
	}

	select {
	case c.commandQueue <- cmd:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("command queue timeout (simulation)")
	}
}

// simulateReader 模拟读取数据
func (c *STM32Controller) simulateReader() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// 模拟心跳响应
			c.lastHeartbeatTime = time.Now()
			c.logger.Debug("Simulated heartbeat received")
		}
	}
}

// simulateWriter 模拟写入数据
func (c *STM32Controller) simulateWriter() {
	for {
		select {
		case <-c.stopCh:
			return
		case cmd := <-c.commandQueue:
			// 模拟命令处理
			c.logger.Debug("Processing simulated command",
				zap.String("type", cmd.Type),
				zap.Any("data", cmd.Data))

			// 模拟响应延迟
			time.Sleep(10 * time.Millisecond)

			// 对某些命令生成模拟响应
			switch cmd.Type {
			case "coin_dispense":
				// 模拟投币成功
				if c.onCoinInserted != nil {
					if count, ok := cmd.Data.(uint8); ok {
						c.onCoinInserted(count)
					}
				}
			}
		}
	}
}

// heartbeatMonitor 心跳监控
func (c *STM32Controller) heartbeatMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			if time.Since(c.lastHeartbeatTime) > 30*time.Second {
				c.logger.Warn("Simulated heartbeat timeout")
				c.isConnected.Store(false)
			}
		}
	}
}

// GetStats 获取统计信息
func (c *STM32Controller) GetStats() STM32Stats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

// SetOnCoinInserted 设置投币回调
func (c *STM32Controller) SetOnCoinInserted(callback func(count uint8)) {
	c.onCoinInserted = callback
}

// SetOnCoinReturned 设置回币回调
func (c *STM32Controller) SetOnCoinReturned(callback func(position string, count uint8)) {
	c.onCoinReturned = callback
}

// SetOnJackpot 设置中奖回调
func (c *STM32Controller) SetOnJackpot(callback func(data JackpotData)) {
	c.onJackpot = callback
}

// SetOnDisconnect 设置断开连接回调
func (c *STM32Controller) SetOnDisconnect(callback func()) {
	c.onDisconnect = callback
}

// writeFrame 模拟写入帧
func (c *STM32Controller) writeFrame(frame *Frame) error {
	c.logger.Debug("Simulated frame write",
		zap.Uint8("command", frame.Command),
		zap.Uint16("sequence", frame.Sequence))
	return nil
}

// sendCommandWithTimeout 发送命令并等待响应（模拟）
func (c *STM32Controller) sendCommandWithTimeout(cmd byte, data []byte, timeout time.Duration) error {
	c.logger.Debug("Simulated command with timeout",
		zap.Uint8("command", cmd),
		zap.Duration("timeout", timeout))

	// 模拟成功响应
	time.Sleep(10 * time.Millisecond)
	return nil
}

// 实现其他必要的公共方法...
// 这些方法在模拟模式下可能只是返回默认值或模拟数据