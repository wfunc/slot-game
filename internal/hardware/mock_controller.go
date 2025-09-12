package hardware

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MockController 模拟硬件控制器
type MockController struct {
	mu        sync.RWMutex
	connected bool
	stats     CoinStatistics
	logger    *zap.Logger
	
	// 回调函数
	coinInsertedCallback  func(count byte)
	coinReturnedCallback  func(data *CoinReturnData)
	buttonPressedCallback func(event *ButtonEvent)
	faultReportCallback   func(event *FaultEvent)
}

// NewMockController 创建模拟控制器
func NewMockController() *MockController {
	return &MockController{
		connected: false,
		logger:    zap.NewNop(), // 使用空日志器
	}
}

// Connect 模拟连接
func (m *MockController) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.connected = true
	m.logger.Info("模拟控制器已连接")
	return nil
}

// Disconnect 模拟断开连接
func (m *MockController) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.connected = false
	m.logger.Info("模拟控制器已断开")
	return nil
}

// IsConnected 返回连接状态
func (m *MockController) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// DispenseCoins 模拟出币
func (m *MockController) DispenseCoins(count uint16, speed byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟出币", 
		zap.Uint16("count", count),
		zap.Uint8("speed", speed))
	
	m.mu.Lock()
	// 更新统计
	m.mu.Unlock()
	
	// 模拟执行时间
	time.Sleep(time.Duration(count*100) * time.Millisecond)
	
	return nil
}

// RefundCoins 模拟退币
func (m *MockController) RefundCoins(count uint16) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟退币", zap.Uint16("count", count))
	
	// 模拟执行时间
	time.Sleep(time.Duration(count*50) * time.Millisecond)
	
	return nil
}

// DispenseTickets 模拟出票
func (m *MockController) DispenseTickets(count uint16) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟出票", zap.Uint16("count", count))
	
	// 模拟执行时间
	time.Sleep(time.Duration(count*200) * time.Millisecond)
	
	return nil
}

// StartPushing 模拟开始推币
func (m *MockController) StartPushing() error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟开始推币")
	return nil
}

// StopPushing 模拟停止推币
func (m *MockController) StopPushing() error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟停止推币")
	return nil
}

// SinglePush 模拟单次推币
func (m *MockController) SinglePush(times byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟单次推币", zap.Uint8("times", times))
	
	// 模拟执行时间
	time.Sleep(time.Duration(times) * 500 * time.Millisecond)
	
	return nil
}

// SetPushSpeed 模拟设置推币速度
func (m *MockController) SetPushSpeed(speed byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟设置推币速度", zap.Uint8("speed", speed))
	return nil
}

// PushCoin 模拟推币动作
func (m *MockController) PushCoin(force int, duration time.Duration) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟推币", 
		zap.Int("force", force),
		zap.Duration("duration", duration))
	
	// 模拟推币时间
	time.Sleep(duration)
	
	// 模拟回币事件
	go func() {
		time.Sleep(duration / 2)
		if m.coinReturnedCallback != nil {
			m.coinReturnedCallback(&CoinReturnData{
				FrontCount: 5,
				LeftCount:  2,
				RightCount: 1,
			})
		}
	}()
	
	return nil
}

// SetLights 模拟设置灯光
func (m *MockController) SetLights(lightBits byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟设置灯光", zap.Uint8("bits", lightBits))
	return nil
}

// TurnOnLight 模拟打开灯光
func (m *MockController) TurnOnLight(light byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟打开灯光", zap.Uint8("light", light))
	return nil
}

// TurnOffAllLights 模拟关闭所有灯光
func (m *MockController) TurnOffAllLights() error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟关闭所有灯光")
	return nil
}

// TurnOnAllLights 模拟打开所有灯光
func (m *MockController) TurnOnAllLights() error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟打开所有灯光")
	return nil
}

// QueryStatus 模拟查询状态
func (m *MockController) QueryStatus(queryType byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟查询状态", zap.Uint8("type", queryType))
	return nil
}

// GetStatistics 获取统计信息
func (m *MockController) GetStatistics() *CoinStatistics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := m.stats
	return &stats
}

// ResetStatistics 重置统计信息
func (m *MockController) ResetStatistics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stats = CoinStatistics{}
	m.logger.Debug("统计信息已重置")
}

// RecoverFault 模拟故障恢复
func (m *MockController) RecoverFault(faultCode byte, action byte, param byte) error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	m.logger.Debug("模拟故障恢复", 
		zap.Uint8("fault", faultCode),
		zap.Uint8("action", action),
		zap.Uint8("param", param))
	
	return nil
}

// SendHeartbeat 模拟发送心跳
func (m *MockController) SendHeartbeat() error {
	if !m.IsConnected() {
		return fmt.Errorf("未连接")
	}
	
	return nil
}

// SetCoinInsertedCallback 设置投币回调
func (m *MockController) SetCoinInsertedCallback(callback func(count byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coinInsertedCallback = callback
}

// SetCoinReturnedCallback 设置退币回调
func (m *MockController) SetCoinReturnedCallback(callback func(data *CoinReturnData)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.coinReturnedCallback = callback
}

// SetButtonPressedCallback 设置按钮回调
func (m *MockController) SetButtonPressedCallback(callback func(event *ButtonEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buttonPressedCallback = callback
}

// SetFaultReportCallback 设置故障回调
func (m *MockController) SetFaultReportCallback(callback func(event *FaultEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.faultReportCallback = callback
}

// SimulateCoinInsert 模拟投币事件
func (m *MockController) SimulateCoinInsert(count byte) {
	m.mu.Lock()
	// 更新统计
	callback := m.coinInsertedCallback
	m.mu.Unlock()
	
	if callback != nil {
		callback(count)
	}
}

// SimulateButtonPress 模拟按键事件
func (m *MockController) SimulateButtonPress(keyType, keyCode, action byte) {
	m.mu.RLock()
	callback := m.buttonPressedCallback
	m.mu.RUnlock()
	
	if callback != nil {
		callback(&ButtonEvent{
			KeyType: keyType,
			KeyCode: keyCode,
			Action:  action,
		})
	}
}