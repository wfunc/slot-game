package hardware

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tarm/serial"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// SerialController 串口控制器接口
type SerialController interface {
	// 基础操作
	Connect() error
	Disconnect() error
	IsConnected() bool
	
	// 推币机控制
	PushCoin(force int, duration time.Duration) error
	StopPush() error
	
	// 电机控制
	StartMotor(speed int) error
	StopMotor() error
	SetMotorSpeed(speed int) error
	
	// 灯光控制
	SetLights(pattern LightPattern) error
	FlashLights(duration time.Duration) error
	
	// 传感器读取
	ReadCoinCount() (int, error)
	ReadSensorStatus() (*SensorStatus, error)
	
	// 状态监控
	GetStatus() (*DeviceStatus, error)
	SetStatusCallback(callback StatusCallback)
}

// StatusCallback 状态回调函数
type StatusCallback func(status *DeviceStatus)

// LightPattern 灯光模式
type LightPattern int

const (
	LightOff     LightPattern = 0
	LightStatic  LightPattern = 1
	LightBlink   LightPattern = 2
	LightBreath  LightPattern = 3
	LightRainbow LightPattern = 4
)

// SensorStatus 传感器状态
type SensorStatus struct {
	CoinDetected   bool      `json:"coin_detected"`
	MotorRunning   bool      `json:"motor_running"`
	DoorOpen       bool      `json:"door_open"`
	Temperature    float32   `json:"temperature"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// DeviceStatus 设备状态
type DeviceStatus struct {
	Connected      bool           `json:"connected"`
	MotorSpeed     int            `json:"motor_speed"`
	PushForce      int            `json:"push_force"`
	CoinCount      int            `json:"coin_count"`
	LightPattern   LightPattern   `json:"light_pattern"`
	SensorStatus   *SensorStatus  `json:"sensor_status"`
	LastCommand    string         `json:"last_command"`
	LastCommandTime time.Time     `json:"last_command_time"`
	ErrorCount     int            `json:"error_count"`
}

// Command 命令类型
type Command byte

const (
	CmdPushCoin     Command = 0x01
	CmdStopPush     Command = 0x02
	CmdStartMotor   Command = 0x03
	CmdStopMotor    Command = 0x04
	CmdSetSpeed     Command = 0x05
	CmdSetLight     Command = 0x06
	CmdReadSensor   Command = 0x10
	CmdReadCoin     Command = 0x11
	CmdGetStatus    Command = 0x20
	CmdReset        Command = 0x30
	CmdHeartbeat    Command = 0x40
)

// SerialControllerImpl 串口控制器实现
type SerialControllerImpl struct {
	config         *SerialConfig
	port           *serial.Port
	connected      bool
	mu             sync.RWMutex
	status         *DeviceStatus
	statusCallback StatusCallback
	stopChan       chan struct{}
	logger         *zap.Logger
}

// SerialConfig 串口配置
type SerialConfig struct {
	Port         string        `yaml:"port"`
	BaudRate     int           `yaml:"baud_rate"`
	DataBits     byte          `yaml:"data_bits"`
	StopBits     byte          `yaml:"stop_bits"`
	Parity       string        `yaml:"parity"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	RetryTimes   int           `yaml:"retry_times"`
	RetryInterval time.Duration `yaml:"retry_interval"`
}

// NewSerialController 创建串口控制器
func NewSerialController(config *SerialConfig) *SerialControllerImpl {
	return &SerialControllerImpl{
		config: config,
		status: &DeviceStatus{
			SensorStatus: &SensorStatus{},
		},
		stopChan: make(chan struct{}),
		logger:   logger.GetLogger(),
	}
}

// Connect 连接串口
func (s *SerialControllerImpl) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.connected {
		return nil
	}
	
	// 解析校验位
	parity := serial.ParityNone
	switch s.config.Parity {
	case "O", "odd":
		parity = serial.ParityOdd
	case "E", "even":
		parity = serial.ParityEven
	}
	
	// 配置串口
	config := &serial.Config{
		Name:        s.config.Port,
		Baud:        s.config.BaudRate,
		Size:        s.config.DataBits,
		Parity:      parity,
		StopBits:    serial.StopBits(s.config.StopBits),
		ReadTimeout: s.config.ReadTimeout,
	}
	
	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		s.logger.Error("打开串口失败", 
			zap.String("port", s.config.Port),
			zap.Error(err))
		return fmt.Errorf("open serial port: %w", err)
	}
	
	s.port = port
	s.connected = true
	s.status.Connected = true
	
	// 启动心跳和状态监控
	go s.heartbeatLoop()
	go s.statusMonitor()
	
	s.logger.Info("串口连接成功",
		zap.String("port", s.config.Port),
		zap.Int("baud_rate", s.config.BaudRate))
	
	return nil
}

// Disconnect 断开连接
func (s *SerialControllerImpl) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.connected {
		return nil
	}
	
	// 停止监控
	close(s.stopChan)
	
	// 关闭串口
	if s.port != nil {
		if err := s.port.Close(); err != nil {
			s.logger.Error("关闭串口失败", zap.Error(err))
			return err
		}
	}
	
	s.connected = false
	s.status.Connected = false
	s.port = nil
	
	s.logger.Info("串口已断开")
	
	return nil
}

// IsConnected 检查连接状态
func (s *SerialControllerImpl) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// PushCoin 推币
func (s *SerialControllerImpl) PushCoin(force int, duration time.Duration) error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	// 限制推币力度范围
	if force < 0 {
		force = 0
	} else if force > 100 {
		force = 100
	}
	
	// 构建命令
	cmd := s.buildCommand(CmdPushCoin, []byte{
		byte(force),
		byte(duration.Milliseconds() / 10), // 10ms为单位
	})
	
	// 发送命令
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send push coin command: %w", err)
	}
	
	// 更新状态
	s.mu.Lock()
	s.status.PushForce = force
	s.status.LastCommand = "PushCoin"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	s.logger.Debug("推币命令已发送",
		zap.Int("force", force),
		zap.Duration("duration", duration))
	
	return nil
}

// StopPush 停止推币
func (s *SerialControllerImpl) StopPush() error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	cmd := s.buildCommand(CmdStopPush, nil)
	
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send stop push command: %w", err)
	}
	
	s.mu.Lock()
	s.status.PushForce = 0
	s.status.LastCommand = "StopPush"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	return nil
}

// StartMotor 启动电机
func (s *SerialControllerImpl) StartMotor(speed int) error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	// 限制速度范围
	if speed < 0 {
		speed = 0
	} else if speed > 100 {
		speed = 100
	}
	
	cmd := s.buildCommand(CmdStartMotor, []byte{byte(speed)})
	
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send start motor command: %w", err)
	}
	
	s.mu.Lock()
	s.status.MotorSpeed = speed
	s.status.LastCommand = "StartMotor"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	return nil
}

// StopMotor 停止电机
func (s *SerialControllerImpl) StopMotor() error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	cmd := s.buildCommand(CmdStopMotor, nil)
	
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send stop motor command: %w", err)
	}
	
	s.mu.Lock()
	s.status.MotorSpeed = 0
	s.status.LastCommand = "StopMotor"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	return nil
}

// SetMotorSpeed 设置电机速度
func (s *SerialControllerImpl) SetMotorSpeed(speed int) error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	// 限制速度范围
	if speed < 0 {
		speed = 0
	} else if speed > 100 {
		speed = 100
	}
	
	cmd := s.buildCommand(CmdSetSpeed, []byte{byte(speed)})
	
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send set speed command: %w", err)
	}
	
	s.mu.Lock()
	s.status.MotorSpeed = speed
	s.status.LastCommand = "SetMotorSpeed"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	return nil
}

// SetLights 设置灯光
func (s *SerialControllerImpl) SetLights(pattern LightPattern) error {
	if !s.IsConnected() {
		return errors.New("serial port not connected")
	}
	
	cmd := s.buildCommand(CmdSetLight, []byte{byte(pattern)})
	
	if err := s.sendCommand(cmd); err != nil {
		return fmt.Errorf("send set lights command: %w", err)
	}
	
	s.mu.Lock()
	s.status.LightPattern = pattern
	s.status.LastCommand = "SetLights"
	s.status.LastCommandTime = time.Now()
	s.mu.Unlock()
	
	return nil
}

// FlashLights 闪烁灯光
func (s *SerialControllerImpl) FlashLights(duration time.Duration) error {
	// 先设置闪烁模式
	if err := s.SetLights(LightBlink); err != nil {
		return err
	}
	
	// 等待指定时间
	time.Sleep(duration)
	
	// 恢复静态模式
	return s.SetLights(LightStatic)
}

// ReadCoinCount 读取币数
func (s *SerialControllerImpl) ReadCoinCount() (int, error) {
	if !s.IsConnected() {
		return 0, errors.New("serial port not connected")
	}
	
	cmd := s.buildCommand(CmdReadCoin, nil)
	
	// 发送命令并等待响应
	resp, err := s.sendCommandWithResponse(cmd, 100*time.Millisecond)
	if err != nil {
		return 0, fmt.Errorf("read coin count: %w", err)
	}
	
	if len(resp) < 2 {
		return 0, errors.New("invalid response length")
	}
	
	count := int(binary.BigEndian.Uint16(resp))
	
	s.mu.Lock()
	s.status.CoinCount = count
	s.mu.Unlock()
	
	return count, nil
}

// ReadSensorStatus 读取传感器状态
func (s *SerialControllerImpl) ReadSensorStatus() (*SensorStatus, error) {
	if !s.IsConnected() {
		return nil, errors.New("serial port not connected")
	}
	
	cmd := s.buildCommand(CmdReadSensor, nil)
	
	// 发送命令并等待响应
	resp, err := s.sendCommandWithResponse(cmd, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("read sensor status: %w", err)
	}
	
	if len(resp) < 6 {
		return nil, errors.New("invalid response length")
	}
	
	status := &SensorStatus{
		CoinDetected:   resp[0] == 1,
		MotorRunning:   resp[1] == 1,
		DoorOpen:       resp[2] == 1,
		Temperature:    float32(resp[3]) + float32(resp[4])/100,
		LastUpdateTime: time.Now(),
	}
	
	s.mu.Lock()
	s.status.SensorStatus = status
	s.mu.Unlock()
	
	return status, nil
}

// GetStatus 获取设备状态
func (s *SerialControllerImpl) GetStatus() (*DeviceStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回状态副本
	status := *s.status
	if s.status.SensorStatus != nil {
		sensorCopy := *s.status.SensorStatus
		status.SensorStatus = &sensorCopy
	}
	
	return &status, nil
}

// SetStatusCallback 设置状态回调
func (s *SerialControllerImpl) SetStatusCallback(callback StatusCallback) {
	s.statusCallback = callback
}

// buildCommand 构建命令包
func (s *SerialControllerImpl) buildCommand(cmd Command, data []byte) []byte {
	// 协议格式: [起始符(0xAA)] [命令] [数据长度] [数据...] [校验和] [结束符(0x55)]
	packet := []byte{0xAA, byte(cmd), byte(len(data))}
	packet = append(packet, data...)
	
	// 计算校验和
	checksum := byte(0)
	for _, b := range packet[1:] {
		checksum ^= b
	}
	
	packet = append(packet, checksum, 0x55)
	
	return packet
}

// sendCommand 发送命令
func (s *SerialControllerImpl) sendCommand(cmd []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.port == nil {
		return errors.New("port not open")
	}
	
	// 重试机制
	for i := 0; i < s.config.RetryTimes; i++ {
		_, err := s.port.Write(cmd)
		if err == nil {
			return nil
		}
		
		if i < s.config.RetryTimes-1 {
			time.Sleep(s.config.RetryInterval)
		}
	}
	
	s.status.ErrorCount++
	return errors.New("send command failed after retries")
}

// sendCommandWithResponse 发送命令并等待响应
func (s *SerialControllerImpl) sendCommandWithResponse(cmd []byte, timeout time.Duration) ([]byte, error) {
	// 发送命令
	if err := s.sendCommand(cmd); err != nil {
		return nil, err
	}
	
	// 等待响应
	s.mu.Lock()
	defer s.mu.Unlock()
	
	buffer := make([]byte, 256)
	// 设置读取超时（使用 goroutine 和 channel 实现超时）
	readChan := make(chan struct {
		n   int
		err error
	}, 1)
	
	go func() {
		n, err := s.port.Read(buffer)
		readChan <- struct {
			n   int
			err error
		}{n, err}
	}()
	
	var n int
	var err error
	select {
	case result := <-readChan:
		n, err = result.n, result.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("read timeout after %v", timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	// 解析响应
	if n < 4 {
		return nil, errors.New("response too short")
	}
	
	// 验证响应格式
	if buffer[0] != 0xAA || buffer[n-1] != 0x55 {
		return nil, errors.New("invalid response format")
	}
	
	// 提取数据
	dataLen := int(buffer[2])
	if n < dataLen+5 {
		return nil, errors.New("incomplete response")
	}
	
	return buffer[3 : 3+dataLen], nil
}

// heartbeatLoop 心跳循环
func (s *SerialControllerImpl) heartbeatLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.IsConnected() {
				cmd := s.buildCommand(CmdHeartbeat, nil)
				if err := s.sendCommand(cmd); err != nil {
					s.logger.Warn("心跳发送失败", zap.Error(err))
				}
			}
		}
	}
}

// statusMonitor 状态监控
func (s *SerialControllerImpl) statusMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.IsConnected() {
				// 定期读取传感器状态
				if _, err := s.ReadSensorStatus(); err != nil {
					s.logger.Debug("读取传感器状态失败", zap.Error(err))
				}
				
				// 触发回调
				if s.statusCallback != nil {
					status, _ := s.GetStatus()
					s.statusCallback(status)
				}
			}
		}
	}
}

// MockSerialController 模拟串口控制器（用于测试）
type MockSerialController struct {
	connected bool
	status    *DeviceStatus
	mu        sync.RWMutex
}

// NewMockSerialController 创建模拟控制器
func NewMockSerialController() *MockSerialController {
	return &MockSerialController{
		status: &DeviceStatus{
			SensorStatus: &SensorStatus{
				Temperature: 25.0,
			},
			CoinCount: 100,
		},
	}
}

func (m *MockSerialController) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	m.status.Connected = true
	return nil
}

func (m *MockSerialController) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	m.status.Connected = false
	return nil
}

func (m *MockSerialController) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *MockSerialController) PushCoin(force int, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.PushForce = force
	m.status.LastCommand = "PushCoin"
	m.status.LastCommandTime = time.Now()
	// 模拟币数减少
	if m.status.CoinCount > 0 {
		m.status.CoinCount--
	}
	return nil
}

func (m *MockSerialController) StopPush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.PushForce = 0
	return nil
}

func (m *MockSerialController) StartMotor(speed int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.MotorSpeed = speed
	m.status.SensorStatus.MotorRunning = true
	return nil
}

func (m *MockSerialController) StopMotor() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.MotorSpeed = 0
	m.status.SensorStatus.MotorRunning = false
	return nil
}

func (m *MockSerialController) SetMotorSpeed(speed int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.MotorSpeed = speed
	return nil
}

func (m *MockSerialController) SetLights(pattern LightPattern) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.LightPattern = pattern
	return nil
}

func (m *MockSerialController) FlashLights(duration time.Duration) error {
	return m.SetLights(LightBlink)
}

func (m *MockSerialController) ReadCoinCount() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.CoinCount, nil
}

func (m *MockSerialController) ReadSensorStatus() (*SensorStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.SensorStatus, nil
}

func (m *MockSerialController) GetStatus() (*DeviceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status, nil
}

func (m *MockSerialController) SetStatusCallback(callback StatusCallback) {
	// 模拟实现不需要回调
}