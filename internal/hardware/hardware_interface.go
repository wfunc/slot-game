package hardware

import "time"

// HardwareController 硬件控制器接口
type HardwareController interface {
	// 连接管理
	Connect() error
	Disconnect() error
	IsConnected() bool
	
	// 币控制
	DispenseCoins(count uint16, speed byte) error
	RefundCoins(count uint16) error
	
	// 彩票控制
	DispenseTickets(count uint16) error
	
	// 推币控制
	StartPushing() error
	StopPushing() error
	SinglePush(times byte) error
	SetPushSpeed(speed byte) error
	PushCoin(force int, duration time.Duration) error
	
	// 灯光控制
	SetLights(lightBits byte) error
	TurnOnLight(light byte) error
	TurnOffAllLights() error
	TurnOnAllLights() error
	
	// 状态查询
	QueryStatus(queryType byte) error
	GetStatistics() *CoinStatistics
	ResetStatistics()
	
	// 故障恢复
	RecoverFault(faultCode byte, action byte, param byte) error
	
	// 心跳
	SendHeartbeat() error
	
	// 回调设置
	SetCoinInsertedCallback(callback func(count byte))
	SetCoinReturnedCallback(callback func(data *CoinReturnData))
	SetButtonPressedCallback(callback func(event *ButtonEvent))
	SetFaultReportCallback(callback func(event *FaultEvent))
}

// 为了向后兼容，定义一个别名
type SerialController = HardwareController