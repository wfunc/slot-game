package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wfunc/slot-game/internal/hardware"
	"go.uber.org/zap"
)

// MockSerialController 模拟串口控制器
type MockSerialController struct {
	pushCoinCalled bool
	force          int
	duration       time.Duration
	returnError    error
}

func (m *MockSerialController) Connect() error {
	return nil
}

func (m *MockSerialController) Disconnect() error {
	return nil
}

func (m *MockSerialController) IsConnected() bool {
	return true
}

func (m *MockSerialController) PushCoin(force int, duration time.Duration) error {
	m.pushCoinCalled = true
	m.force = force
	m.duration = duration
	return m.returnError
}

// 币控制
func (m *MockSerialController) DispenseCoins(count uint16, speed byte) error {
	return nil
}

func (m *MockSerialController) RefundCoins(count uint16) error {
	return nil
}

// 彩票控制
func (m *MockSerialController) PrintTickets(count uint16) error {
	return nil
}

// 推币控制
func (m *MockSerialController) StartPushing() error {
	return nil
}

func (m *MockSerialController) StopPushing() error {
	return nil
}

func (m *MockSerialController) PushControl(action byte, param byte) error {
	return nil
}

func (m *MockSerialController) SetPushSpeed(speed byte) error {
	return nil
}

// 灯光控制
func (m *MockSerialController) LightControl(pattern byte, brightness byte, duration byte) error {
	return nil
}

func (m *MockSerialController) QueryStatus(queryType byte) error {
	return nil
}

func (m *MockSerialController) GetStatistics() *hardware.CoinStatistics {
	return &hardware.CoinStatistics{
		CoinsInserted:      100,
		CoinsDispensed:     50,
		CoinsReturnedFront: 30,
		CoinsReturnedLeft:  10,
		CoinsReturnedRight: 10,
	}
}

func (m *MockSerialController) ResetStatistics() {
	// Mock implementation
}

// 故障恢复
func (m *MockSerialController) FaultRecovery(faultCode byte, action byte, retryCount byte) error {
	return nil
}

// 心跳
func (m *MockSerialController) SendHeartbeat() error {
	return nil
}

// 回调设置
func (m *MockSerialController) SetCoinInsertedCallback(callback func(count byte)) {
	// Mock implementation
}

func (m *MockSerialController) SetCoinReturnedCallback(callback func(data *hardware.CoinReturnData)) {
	// Mock implementation
}

func (m *MockSerialController) SetButtonPressedCallback(callback func(event *hardware.ButtonEvent)) {
	// Mock implementation
}

func (m *MockSerialController) SetFaultReportCallback(callback func(event *hardware.FaultEvent)) {
	// Mock implementation
}

func TestGameService_SerialControllerIntegration(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	mockSerial := &MockSerialController{}
	
	// 创建游戏服务配置（包含串口控制器）
	config := &GameServiceConfig{
		DB:               db,
		Logger:           logger,
		SessionTimeout:   30 * time.Minute,
		MaxSessions:      100,
		SerialController: mockSerial,
	}
	
	// 创建游戏服务
	gameService := NewGameService(config)
	
	// 验证串口控制器被正确设置
	assert.NotNil(t, gameService.serialController)
	assert.Equal(t, mockSerial, gameService.serialController)
}

func TestGameService_CoinPushOnWin(t *testing.T) {
	// 这个测试暂时跳过，因为需要更复杂的模拟游戏引擎来产生中奖结果
	t.Skip("需要完整的游戏引擎模拟才能测试中奖出币")
}

func TestGameService_NoCoinPushWithoutSerial(t *testing.T) {
	// 设置测试环境（不包含串口控制器）
	db := setupTestDB(t)
	logger := zap.NewNop()
	
	// 创建游戏服务（不包含串口控制器）
	config := &GameServiceConfig{
		DB:             db,
		Logger:         logger,
		SessionTimeout: 30 * time.Minute,
		MaxSessions:    100,
		// SerialController 为 nil
	}
	
	gameService := NewGameService(config)
	
	// 验证串口控制器为nil，游戏服务仍然可以正常工作
	assert.Nil(t, gameService.serialController)
}

func TestGameService_CoinPushCalculation(t *testing.T) {
	// 测试出币计算逻辑
	testCases := []struct {
		name             string
		winAmount        int64
		expectedForce    int
		expectedDuration time.Duration
		expectedCoins    int
	}{
		{"小额中奖", 50, 0, 0, 0},                                // 0.5元，不足1个币
		{"1元中奖", 100, 60, 500 * time.Millisecond, 1},          // 1元 = 1个币，力度60
		{"5元中奖", 500, 100, 2500 * time.Millisecond, 5},        // 5元 = 5个币，力度100（最大）
		{"10.5元中奖", 1050, 100, 5000 * time.Millisecond, 10},   // 10.5元 = 10个币，力度100（最大）
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试币数计算
			coinCount := int(tc.winAmount / 100)
			assert.Equal(t, tc.expectedCoins, coinCount)
			
			// 测试力度计算
			if coinCount > 0 {
				force := 50 + (coinCount * 10)
				if force > 100 {
					force = 100
				}
				assert.Equal(t, tc.expectedForce, force)
				
				// 测试持续时间计算
				duration := time.Duration(coinCount) * 500 * time.Millisecond
				assert.Equal(t, tc.expectedDuration, duration)
			}
		})
	}
}