package hardware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSerialPort 模拟串口
type MockSerialPort struct {
	mock.Mock
	readData  []byte
	writeData []byte
}

func (m *MockSerialPort) Read(b []byte) (n int, err error) {
	args := m.Called(b)
	if m.readData != nil && len(m.readData) > 0 {
		n = copy(b, m.readData)
		m.readData = m.readData[n:]
		return n, args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

func (m *MockSerialPort) Write(b []byte) (n int, err error) {
	args := m.Called(b)
	m.writeData = append(m.writeData, b...)
	
	// 检查是否使用了 Return 函数
	if len(args) > 0 {
		// 如果第一个参数是函数
		if fn, ok := args.Get(0).(func([]byte) (int, error)); ok {
			return fn(b)
		}
		
		// 如果有两个参数 (int, error)
		if len(args) >= 2 {
			// 尝试获取 int 和 error
			n := args.Int(0)
			err := args.Error(1)
			return n, err
		}
		
		// 如果只有一个参数且是 nil 或 error
		if args.Get(0) == nil {
			return len(b), nil
		}
		if err, ok := args.Get(0).(error); ok {
			return len(b), err
		}
	}
	
	// 默认返回
	return len(b), nil
}

func (m *MockSerialPort) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSerialPort) Flush() error {
	args := m.Called()
	return args.Error(0)
}

// MockGameLogic 模拟游戏逻辑
type MockGameLogic struct {
	mock.Mock
}

func (m *MockGameLogic) GetCurrentMode() byte {
	args := m.Called()
	return args.Get(0).(byte)
}

func (m *MockGameLogic) HasCredits() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGameLogic) GetPendingCoins() uint16 {
	args := m.Called()
	return args.Get(0).(uint16)
}

func (m *MockGameLogic) AddCredits(count byte) {
	m.Called(count)
}

func (m *MockGameLogic) AddPlayerCoins(count byte) {
	m.Called(count)
}

func (m *MockGameLogic) UpdateReturnRate(rate float64) {
	m.Called(rate)
}

func (m *MockGameLogic) GetRefundableCoins() uint16 {
	args := m.Called()
	return args.Get(0).(uint16)
}

func (m *MockGameLogic) GetAvailableTickets() uint16 {
	args := m.Called()
	return args.Get(0).(uint16)
}

func (m *MockGameLogic) DeductCoins(count uint16) {
	m.Called(count)
}

func (m *MockGameLogic) RedeemTickets(count uint16) {
	m.Called(count)
}

func (m *MockGameLogic) StartGame(coinCount uint16) {
	m.Called(coinCount)
}

func (m *MockGameLogic) SetDifficulty(level byte) {
	m.Called(level)
}

func TestNewSTM32Controller(t *testing.T) {
	mockGame := &MockGameLogic{}
	
	// Test with default config
	controller := NewSTM32Controller(nil, mockGame)
	assert.NotNil(t, controller)
	assert.NotNil(t, controller.config)
	assert.Equal(t, "/dev/ttyS3", controller.config.Port)
	assert.Equal(t, 115200, controller.config.BaudRate)
	assert.Equal(t, mockGame, controller.gameLogic)
	
	// Test with custom config
	customConfig := &STM32Config{
		Port:     "/dev/ttyUSB0",
		BaudRate: 9600,
	}
	controller2 := NewSTM32Controller(customConfig, mockGame)
	assert.NotNil(t, controller2)
	assert.Equal(t, "/dev/ttyUSB0", controller2.config.Port)
	assert.Equal(t, 9600, controller2.config.BaudRate)
}

func TestSTM32Controller_IsConnected(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	// Initially not connected
	assert.False(t, controller.IsConnected())
	
	// Simulate connection
	controller.connected = true
	assert.True(t, controller.IsConnected())
}

func TestSTM32Controller_GetNextSeq(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	// Should return odd numbers
	seq1 := controller.getNextSeq()
	assert.True(t, seq1%2 == 1, "Sequence should be odd")
	
	seq2 := controller.getNextSeq()
	assert.True(t, seq2%2 == 1, "Sequence should be odd")
	assert.Greater(t, seq2, seq1)
}

func TestSTM32Controller_WriteFrame(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	mockPort := &MockSerialPort{}
	controller.port = mockPort
	controller.connected = true
	
	frame := NewFrame(CmdHeartbeat, 123, []byte{0x01, 0x02})
	
	mockPort.On("Write", mock.Anything).Return(len(frame.ToBytes()), nil)
	
	err := controller.writeFrame(frame)
	assert.NoError(t, err)
	mockPort.AssertExpectations(t)
}

func TestSTM32Controller_HandleCoinInserted(t *testing.T) {
	mockGame := &MockGameLogic{}
	controller := NewSTM32Controller(nil, mockGame)
	
	// Set up expectations
	mockGame.On("AddCredits", byte(5)).Return()
	
	// Create test frame
	frame := &Frame{
		Command:  EventCoinInserted,
		Sequence: 100,
		Data:     []byte{5}, // 5 coins inserted
	}
	
	// Set up callback
	callbackCalled := false
	controller.onCoinInserted = func(count byte) {
		callbackCalled = true
		assert.Equal(t, byte(5), count)
	}
	
	// Mock port for ACK response
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Handle event
	controller.handleCoinInserted(frame)
	
	// Verify
	assert.True(t, callbackCalled)
	assert.Equal(t, uint16(5), controller.stats.CoinsInserted)
	mockGame.AssertExpectations(t)
}

func TestSTM32Controller_HandleCoinReturned(t *testing.T) {
	mockGame := &MockGameLogic{}
	controller := NewSTM32Controller(nil, mockGame)
	
	// Set up expectations
	mockGame.On("AddPlayerCoins", byte(10)).Return()
	mockGame.On("UpdateReturnRate", mock.AnythingOfType("float64")).Return()
	
	// Create test frame (new optimized format)
	frame := &Frame{
		Command:  EventCoinReturned,
		Sequence: 200,
		Data:     []byte{10, 3, 2}, // Front: 10, Left: 3, Right: 2
	}
	
	// Set up callback
	callbackCalled := false
	controller.onCoinReturned = func(data *CoinReturnData) {
		callbackCalled = true
		assert.Equal(t, byte(10), data.FrontCount)
		assert.Equal(t, byte(3), data.LeftCount)
		assert.Equal(t, byte(2), data.RightCount)
	}
	
	// Mock port for ACK response
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Handle event
	controller.handleCoinReturned(frame)
	
	// Verify statistics
	assert.True(t, callbackCalled)
	assert.Equal(t, uint16(10), controller.stats.CoinsReturnedFront)
	assert.Equal(t, uint16(3), controller.stats.CoinsReturnedLeft)
	assert.Equal(t, uint16(2), controller.stats.CoinsReturnedRight)
	
	// Verify return rate calculation
	expectedRate := float64(10) / float64(10+3+2) * 100
	assert.InDelta(t, expectedRate, controller.stats.ReturnRate, 0.01)
	
	mockGame.AssertExpectations(t)
}

func TestSTM32Controller_HandleButtonPressed(t *testing.T) {
	mockGame := &MockGameLogic{}
	controller := NewSTM32Controller(nil, mockGame)
	
	// Test Start button
	t.Run("StartButton", func(t *testing.T) {
		mockGame.On("HasCredits").Return(true)
		mockGame.On("GetPendingCoins").Return(uint16(5))
		mockGame.On("StartGame", uint16(5)).Return()
		
		frame := &Frame{
			Command:  EventButtonPressed,
			Sequence: 300,
			Data:     []byte{KeyTypeGame, KeyStart, KeyActionDown},
		}
		
		mockPort := &MockSerialPort{}
		mockPort.On("Write", mock.Anything).Return(0, nil)
		controller.port = mockPort
		
		controller.handleButtonPressed(frame)
		mockGame.AssertExpectations(t)
	})
	
	// Test Refund button (refund mode)
	t.Run("RefundButton_RefundMode", func(t *testing.T) {
		mockGame := &MockGameLogic{}
		controller := NewSTM32Controller(nil, mockGame)
		
		mockGame.On("GetCurrentMode").Return(byte(0x01)) // Refund mode
		mockGame.On("GetRefundableCoins").Return(uint16(10))
		mockGame.On("DeductCoins", uint16(10)).Return()
		
		frame := &Frame{
			Command:  EventButtonPressed,
			Sequence: 400,
			Data:     []byte{KeyTypeGame, KeyRefundTicket, KeyActionDown},
		}
		
		mockPort := &MockSerialPort{}
		mockPort.On("Write", mock.Anything).Return(0, nil)
		controller.port = mockPort
		
		controller.handleButtonPressed(frame)
		mockGame.AssertExpectations(t)
	})
	
	// Test Refund button (ticket mode)
	t.Run("RefundButton_TicketMode", func(t *testing.T) {
		mockGame := &MockGameLogic{}
		controller := NewSTM32Controller(nil, mockGame)
		
		mockGame.On("GetCurrentMode").Return(byte(0x02)) // Ticket mode
		mockGame.On("GetAvailableTickets").Return(uint16(5))
		mockGame.On("RedeemTickets", uint16(5)).Return()
		
		frame := &Frame{
			Command:  EventButtonPressed,
			Sequence: 500,
			Data:     []byte{KeyTypeGame, KeyRefundTicket, KeyActionDown},
		}
		
		mockPort := &MockSerialPort{}
		mockPort.On("Write", mock.Anything).Return(0, nil)
		controller.port = mockPort
		
		controller.handleButtonPressed(frame)
		mockGame.AssertExpectations(t)
	})
}

func TestSTM32Controller_HandleFaultReport(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	// Set up callback
	callbackCalled := false
	var receivedEvent *FaultEvent
	controller.onFaultReport = func(event *FaultEvent) {
		callbackCalled = true
		receivedEvent = event
	}
	
	// Create test frame
	frame := &Frame{
		Command:  EventFaultReport,
		Sequence: 600,
		Data:     []byte{FaultCoinMotorStuck, FaultLevelError, 0xFF},
	}
	
	// Mock port for ACK response
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Handle event
	controller.handleFaultReport(frame)
	
	// Verify
	assert.True(t, callbackCalled)
	assert.NotNil(t, receivedEvent)
	assert.Equal(t, FaultCoinMotorStuck, receivedEvent.FaultCode)
	assert.Equal(t, FaultLevelError, receivedEvent.Level)
	assert.Equal(t, []byte{0xFF}, receivedEvent.ExtraInfo)
	assert.Equal(t, uint8(1), controller.stats.FaultCount)
}

func TestSTM32Controller_DispenseCoins(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	// Test not connected
	err := controller.DispenseCoins(10, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
	
	// Test connected
	controller.connected = true
	mockPort := &MockSerialPort{}
	controller.port = mockPort
	
	// Setup mock to return success and handle the ACK
	mockPort.On("Write", mock.Anything).Return(func(b []byte) (int, error) {
		// When command is sent, immediately send ACK response
		go func() {
			time.Sleep(10 * time.Millisecond)
			controller.cmdMu.Lock()
			for _, cmd := range controller.pendingCmds {
				if cmd.Response != nil {
					cmd.Response <- nil
				}
			}
			controller.cmdMu.Unlock()
		}()
		return len(b), nil
	})
	
	err = controller.DispenseCoins(10, 5)
	assert.NoError(t, err)
	assert.Equal(t, uint16(10), controller.stats.CoinsDispensed)
}

func TestSTM32Controller_RefundCoins(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	controller.connected = true
	
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Simulate successful command
	go func() {
		time.Sleep(10 * time.Millisecond)
		controller.cmdMu.Lock()
		for seq, cmd := range controller.pendingCmds {
			if cmd.Response != nil {
				cmd.Response <- nil
				delete(controller.pendingCmds, seq)
				break
			}
		}
		controller.cmdMu.Unlock()
	}()
	
	err := controller.RefundCoins(20)
	assert.NoError(t, err)
	assert.Equal(t, uint16(20), controller.stats.CoinsRefunded)
}

func TestSTM32Controller_DispenseTickets(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	controller.connected = true
	
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Simulate successful command
	go func() {
		time.Sleep(10 * time.Millisecond)
		controller.cmdMu.Lock()
		for seq, cmd := range controller.pendingCmds {
			if cmd.Response != nil {
				cmd.Response <- nil
				delete(controller.pendingCmds, seq)
				break
			}
		}
		controller.cmdMu.Unlock()
	}()
	
	err := controller.DispenseTickets(15)
	assert.NoError(t, err)
	assert.Equal(t, uint16(15), controller.stats.TicketsPrinted)
}

func TestSTM32Controller_PushControl(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	controller.connected = true
	
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	tests := []struct {
		name   string
		action byte
		param  byte
	}{
		{"Continuous", PushActionContinuous, 0},
		{"Stop", PushActionStop, 0},
		{"Single", PushActionSingle, 3},
		{"SetSpeed", PushActionSetSpeed, 7},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate successful command
			go func() {
				time.Sleep(10 * time.Millisecond)
				controller.cmdMu.Lock()
				for seq, cmd := range controller.pendingCmds {
					if cmd.Response != nil {
						cmd.Response <- nil
						delete(controller.pendingCmds, seq)
						break
					}
				}
				controller.cmdMu.Unlock()
			}()
			
			err := controller.PushControl(tt.action, tt.param)
			assert.NoError(t, err)
		})
	}
}

func TestSTM32Controller_SetLights(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	controller.connected = true
	
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	// Simulate successful command
	go func() {
		time.Sleep(10 * time.Millisecond)
		controller.cmdMu.Lock()
		for seq, cmd := range controller.pendingCmds {
			if cmd.Response != nil {
				cmd.Response <- nil
				delete(controller.pendingCmds, seq)
				break
			}
		}
		controller.cmdMu.Unlock()
	}()
	
	err := controller.SetLights(LightAll)
	assert.NoError(t, err)
}

func TestSTM32Controller_Statistics(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	// Add some stats
	controller.stats.CoinsInserted = 100
	controller.stats.CoinsDispensed = 50
	controller.stats.ReturnRate = 75.5
	
	// Get statistics
	stats := controller.GetStatistics()
	assert.Equal(t, uint16(100), stats.CoinsInserted)
	assert.Equal(t, uint16(50), stats.CoinsDispensed)
	assert.Equal(t, 75.5, stats.ReturnRate)
	
	// Reset statistics
	controller.ResetStatistics()
	assert.Equal(t, uint16(0), controller.stats.CoinsInserted)
	assert.Equal(t, uint16(0), controller.stats.CoinsDispensed)
	assert.Equal(t, float64(0), controller.stats.ReturnRate)
}

func TestSTM32Controller_SendACKResponse(t *testing.T) {
	controller := NewSTM32Controller(nil, nil)
	
	mockPort := &MockSerialPort{}
	mockPort.On("Write", mock.Anything).Return(nil) // 只返回 error
	controller.port = mockPort
	
	controller.sendACKResponse(100, CmdCoinDispense, StatusSuccess)
	
	mockPort.AssertExpectations(t)
	
	// Verify the written data
	assert.NotEmpty(t, mockPort.writeData)
}