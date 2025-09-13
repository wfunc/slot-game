package hardware

import "time"

// HardwareController 硬件控制器接口
type HardwareController interface {
	// 连接管理
	Connect() error
	Disconnect() error
	IsConnected() bool
	
	// 币机控制
	DispenseCoins(count uint16, speed byte) error
	RefundCoins(count uint16) error
	PrintTickets(count uint16) error
	
	// 推币控制
	PushControl(action byte, param byte) error
	StartPushing() error
	StopPushing() error
	SetPushSpeed(speed byte) error
	PushCoin(force int, duration time.Duration) error
	
	// 灯光控制
	LightControl(pattern byte, brightness byte, duration byte) error
	
	// 状态查询
	QueryStatus(statusType byte) error
	SendHeartbeat() error
	
	// 故障恢复
	FaultRecovery(faultCode byte, action byte, retryCount byte) error
	
	// 统计信息
	GetStatistics() *CoinStatistics

	// 回调设置
	SetCoinInsertedCallback(callback func(count byte))
	SetCoinReturnedCallback(callback func(data *CoinReturnData))
	SetButtonPressedCallback(callback func(event *ButtonEvent))
	SetFaultReportCallback(callback func(event *FaultEvent))
}