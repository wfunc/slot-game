package hardware

import (
	"time"
)

// STM32Config STM32控制器配置
type STM32Config struct {
	Port              string        // 串口端口
	BaudRate          int           // 波特率
	DataBits          int           // 数据位
	StopBits          int           // 停止位
	Parity            string        // 校验位
	ReadTimeout       time.Duration // 读取超时
	WriteTimeout      time.Duration // 写入超时
	RetryCount        int           // 重试次数
	RetryDelay        time.Duration // 重试延迟
	HeartbeatInterval time.Duration // 心跳间隔
}

// Command 命令结构
type Command struct {
	Type     string      // 命令类型
	Data     interface{} // 命令数据
	Response chan error  // 响应通道
}

// PendingCommand 待确认的命令
type PendingCommand struct {
	Cmd      byte
	Seq      uint16
	Time     time.Time
	Response chan error
}

// STM32Stats 统计信息
type STM32Stats struct {
	PacketsSent     uint64
	PacketsReceived uint64
	ErrorCount      uint64
	LastError       string
	LastErrorTime   time.Time
	CoinsInserted   uint16
	CoinsReturned   uint16
	ConnectedTime   time.Time
	Uptime          time.Duration
}

// JackpotData 中奖数据
type JackpotData struct {
	Type      string    // 中奖类型
	Amount    uint32    // 金额
	Timestamp time.Time // 时间戳
}

// GameLogic 游戏逻辑接口
type GameLogic interface {
	ProcessCoinInserted(count uint8)
	ProcessCoinReturned(position string, count uint8)
	ProcessJackpot(data JackpotData)
	GetStatus() GameStatus
}

// GameLogicInterface STM32专用的游戏逻辑接口
type GameLogicInterface interface {
	GameLogic  // 嵌入基础接口

	// STM32特有方法
	GetCurrentMode() byte           // 获取当前模式（退币/彩票）
	HasCredits() bool               // 是否有余额
	GetPendingCoins() uint16        // 获取待上币数量
	AddCredits(count byte)          // 增加余额
	AddPlayerCoins(count byte)      // 增加玩家币数
	UpdateReturnRate(rate float64)  // 更新回币率
	GetRefundableCoins() uint16     // 获取可退币数
	GetAvailableTickets() uint16    // 获取可用彩票数
	DeductCoins(count uint16)       // 扣除币数
	RedeemTickets(count uint16)     // 兑换彩票
	StartGame(coinCount uint16)     // 开始游戏
	SetDifficulty(level byte)       // 设置难度
}

// GameStatus 游戏状态
type GameStatus struct {
	IsRunning bool
	Credits   int
	BetAmount int
	WinAmount int
}

// CoinStatistics 硬币统计
type CoinStatistics struct {
	// Common fields used by ACM
	TotalInserted   uint64    // 总投入
	TotalReturned   uint64    // 总返回
	CurrentCredits  uint64    // 当前积分
	LastInsertTime  time.Time // 最后投币时间
	LastReturnTime  time.Time // 最后回币时间
	InsertRate      float64   // 投币速率（个/分钟）
	ReturnRate      float64   // 回币速率（个/分钟）
	ReturnRatio     float64   // 回报率

	// STM32 specific fields
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
}

// ACMConfig ACM控制器配置
type ACMConfig struct {
	Port              string        // 串口端口（"auto"表示自动检测）
	DevicePattern     string        // 设备匹配模式
	BaudRate          int           // 波特率
	DataBits          int           // 数据位
	StopBits          int           // 停止位
	Parity            string        // 校验位
	ReadTimeout       time.Duration // 读取超时
	WriteTimeout      time.Duration // 写入超时
	RetryCount        int           // 重试次数
	RetryDelay        time.Duration // 重试延迍
	AutoDetect        bool          // 是否自动检测设备

	// Algo命令定时器配置
	AlgoTimerEnabled  bool          // 是否启用algo定时器
	AlgoTimerInterval time.Duration // algo命令发送间隔
	AlgoBet           int           // algo命令的bet参数
	AlgoPrize         int           // algo命令的prize参数
}