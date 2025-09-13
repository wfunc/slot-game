package hardware

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// MockController 模拟STM32控制器（用于测试）
type MockController struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	connected       bool
	gameLogic       GameLogicInterface
	
	// 模拟状态
	coinCount       int
	ticketCount     int
	pushing         bool
	lightOn         bool
	faults          []byte
	
	// 事件回调
	onCoinInserted  func(count byte)
	onCoinReturned  func(data *CoinReturnData)
	onButtonPressed func(event *ButtonEvent)
	onFaultReport   func(event *FaultEvent)
	
	// 统计
	stats           *CoinStatistics
	
	// 停止通道
	stopCh          chan struct{}
}

// NewMockController 创建模拟控制器
func NewMockController(gameLogic GameLogicInterface) *MockController {
	return &MockController{
		logger:    logger.GetLogger(),
		gameLogic: gameLogic,
		stats: &CoinStatistics{
			Timestamp: time.Now(),
		},
		stopCh: make(chan struct{}),
	}
}

// Connect 模拟连接
func (m *MockController) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.connected {
		return fmt.Errorf("already connected")
	}
	
	m.connected = true
	m.logger.Info("模拟控制器已连接")
	
	// 启动模拟事件生成器
	go m.eventSimulator()
	
	return nil
}

// Disconnect 模拟断开连接
func (m *MockController) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return nil
	}
	
	m.connected = false
	close(m.stopCh)
	
	m.logger.Info("模拟控制器已断开")
	return nil
}

// IsConnected 是否连接
func (m *MockController) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// DispenseCoins 模拟上币
func (m *MockController) DispenseCoins(count uint16, speed byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.coinCount += int(count)
	m.stats.CoinsDispensed += uint16(count)
	
	m.logger.Info("模拟上币", 
		zap.Uint16("count", count),
		zap.Uint8("speed", speed))
	
	// 模拟延迟
	go func() {
		time.Sleep(time.Duration(count) * 100 * time.Millisecond)
		m.logger.Info("上币完成")
	}()
	
	return nil
}

// RefundCoins 模拟退币
func (m *MockController) RefundCoins(count uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	if m.coinCount < int(count) {
		return fmt.Errorf("insufficient coins")
	}
	
	m.coinCount -= int(count)
	m.stats.CoinsRefunded += uint16(count)
	
	m.logger.Info("模拟退币", zap.Uint16("count", count))
	
	// 触发回币事件
	if m.onCoinReturned != nil {
		go m.onCoinReturned(&CoinReturnData{
			FrontCount: byte(count / 3),
			LeftCount:  byte(count / 3),
			RightCount: byte(count - count/3*2),
		})
	}
	
	return nil
}

// PrintTickets 模拟打印彩票
func (m *MockController) PrintTickets(count uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.ticketCount += int(count)
	
	m.logger.Info("模拟打印彩票", zap.Uint16("count", count))
	
	// 模拟打印延迟
	go func() {
		time.Sleep(time.Duration(count) * 200 * time.Millisecond)
		m.logger.Info("彩票打印完成")
	}()
	
	return nil
}

// DispenseTickets 兼容方法
func (m *MockController) DispenseTickets(count uint16) error {
	return m.PrintTickets(count)
}

// PushControl 模拟推币控制
func (m *MockController) PushControl(action byte, param byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	switch action {
	case PushActionContinuous:
		m.pushing = true
		m.logger.Info("开始推币")
	case PushActionStop:
		m.pushing = false
		m.logger.Info("停止推币")
	case PushActionSingle:
		m.logger.Info("脉冲推币", zap.Uint8("count", param))
	}
	
	return nil
}

// StartPushing 开始推币
func (m *MockController) StartPushing() error {
	return m.PushControl(PushActionContinuous, 0)
}

// StopPushing 停止推币
func (m *MockController) StopPushing() error {
	return m.PushControl(PushActionStop, 0)
}

// SetPushSpeed 设置推币速度
func (m *MockController) SetPushSpeed(speed byte) error {
	m.logger.Info("设置推币速度", zap.Uint8("speed", speed))
	return nil
}

// LightControl 模拟灯光控制
func (m *MockController) LightControl(pattern byte, brightness byte, duration byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	switch pattern {
	case LightOff:
		m.lightOn = false
		m.logger.Info("关闭灯光")
	case LightOn:
		m.lightOn = true
		m.logger.Info("开启灯光", 
			zap.Uint8("brightness", brightness),
			zap.Uint8("duration", duration))
	case LightFlash:
		m.logger.Info("闪烁灯光",
			zap.Uint8("brightness", brightness),
			zap.Uint8("duration", duration))
	}
	
	return nil
}

// QueryStatus 模拟查询状态
func (m *MockController) QueryStatus(statusType byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.logger.Info("查询状态", zap.Uint8("type", statusType))
	
	// 返回模拟状态
	status := &DeviceStatus{
		DeviceType:    0x01,
		FirmwareVer:   0x0110,
		StatusFlags:   0x00,
		ErrorCode:     0x00,
		CoinCount:     uint16(m.coinCount),
		TicketCount:   uint16(m.ticketCount),
		Temperature:   25,
		Voltage:       12000,
		RunTime:       uint32(time.Since(m.stats.Timestamp).Seconds()),
	}
	
	m.logger.Info("设备状态",
		zap.Uint16("coins", status.CoinCount),
		zap.Uint16("tickets", status.TicketCount),
		zap.Uint32("runtime", status.RunTime))
	
	return nil
}

// SendHeartbeat 模拟心跳
func (m *MockController) SendHeartbeat() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	// 心跳计数不需要记录到统计信息中
	return nil
}

// FaultRecovery 模拟故障恢复
func (m *MockController) FaultRecovery(faultCode byte, action byte, retryCount byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.logger.Info("故障恢复",
		zap.Uint8("faultCode", faultCode),
		zap.Uint8("action", action),
		zap.Uint8("retryCount", retryCount))
	
	// 从故障列表中移除
	for i, f := range m.faults {
		if f == faultCode {
			m.faults = append(m.faults[:i], m.faults[i+1:]...)
			break
		}
	}
	
	return nil
}

// GetStatistics 获取统计信息
func (m *MockController) GetStatistics() *CoinStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := *m.stats
	stats.Timestamp = time.Now()
	return &stats
}

// eventSimulator 模拟事件生成器
func (m *MockController) eventSimulator() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.simulateRandomEvent()
		}
	}
}

// simulateRandomEvent 生成随机事件
func (m *MockController) simulateRandomEvent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return
	}
	
	// 随机生成事件
	event := rand.Intn(4)
	switch event {
	case 0:
		// 模拟投币
		count := byte(rand.Intn(5) + 1)
		m.stats.CoinsInserted += uint16(count)
		if m.onCoinInserted != nil {
			m.onCoinInserted(count)
		}
		if m.gameLogic != nil {
			m.gameLogic.AddCredits(count)
		}
		m.logger.Info("模拟投币事件", zap.Uint8("count", count))
		
	case 1:
		// 模拟回币
		if m.coinCount > 0 {
			count := rand.Intn(3) + 1
			if count > m.coinCount {
				count = m.coinCount
			}
			data := &CoinReturnData{
				FrontCount: byte(count),
				LeftCount:  0,
				RightCount: 0,
			}
			if m.onCoinReturned != nil {
				m.onCoinReturned(data)
			}
			if m.gameLogic != nil {
				m.gameLogic.AddPlayerCoins(byte(count))
			}
			m.logger.Info("模拟回币事件", zap.Int("count", count))
		}
		
	case 2:
		// 模拟按键
		keyCode := []byte{KeyStart, KeyRefundTicket}[rand.Intn(2)]
		event := &ButtonEvent{
			KeyType: KeyTypeGame,
			KeyCode: keyCode,
			Action:  KeyActionDown,
		}
		if m.onButtonPressed != nil {
			m.onButtonPressed(event)
		}
		m.logger.Info("模拟按键事件", zap.Uint8("key", keyCode))
		
	case 3:
		// 偶尔生成故障（10%概率）
		if rand.Float32() < 0.1 {
			faultCode := byte(rand.Intn(8) + 1)
			m.faults = append(m.faults, faultCode)
			if m.onFaultReport != nil {
				m.onFaultReport(&FaultEvent{
					FaultCode: faultCode,
					Level:     1,
					Timestamp: uint32(time.Now().Unix()),
				})
			}
			m.logger.Warn("模拟故障事件", zap.Uint8("fault", faultCode))
		}
	}
}

// PushCoin 模拟推币
func (m *MockController) PushCoin(force int, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	
	m.logger.Info("模拟推币动作",
		zap.Int("force", force),
		zap.Duration("duration", duration))
	
	// 模拟推币效果
	go func() {
		m.pushing = true
		time.Sleep(duration)
		m.pushing = false
		
		// 随机产生回币
		if rand.Float32() < 0.3 {
			coins := byte(rand.Intn(3) + 1)
			if m.onCoinReturned != nil {
				m.onCoinReturned(&CoinReturnData{
					FrontCount: coins,
					LeftCount:  0,
					RightCount: 0,
				})
			}
		}
	}()
	
	return nil
}
// SetCoinInsertedCallback 设置投币回调
func (m *MockController) SetCoinInsertedCallback(callback func(count byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onCoinInserted = callback
}

// SetCoinReturnedCallback 设置回币回调
func (m *MockController) SetCoinReturnedCallback(callback func(data *CoinReturnData)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onCoinReturned = callback
}

// SetButtonPressedCallback 设置按键回调
func (m *MockController) SetButtonPressedCallback(callback func(event *ButtonEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onButtonPressed = callback
}

// SetFaultReportCallback 设置故障回调
func (m *MockController) SetFaultReportCallback(callback func(event *FaultEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onFaultReport = callback
}
