package hardware

import (
	"time"
)

// STM32Config STM32控制器配置
type STM32Config struct {
	Port         string        // 串口端口
	BaudRate     int           // 波特率
	DataBits     int           // 数据位
	StopBits     int           // 停止位
	Parity       string        // 校验位
	ReadTimeout  time.Duration // 读取超时
	WriteTimeout time.Duration // 写入超时
	RetryCount   int           // 重试次数
	RetryDelay   time.Duration // 重试延迟
}

// Command 命令结构
type Command struct {
	Type     string      // 命令类型
	Data     interface{} // 命令数据
	Response chan error  // 响应通道
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

// GameLogicInterface 游戏逻辑接口别名（保持兼容）
type GameLogicInterface = GameLogic

// GameStatus 游戏状态
type GameStatus struct {
	IsRunning bool
	Credits   int
	BetAmount int
	WinAmount int
}

// CoinStatistics 硬币统计
type CoinStatistics struct {
	TotalInserted   uint64    // 总投入
	TotalReturned   uint64    // 总返回
	CurrentCredits  uint64    // 当前积分
	LastInsertTime  time.Time // 最后投币时间
	LastReturnTime  time.Time // 最后回币时间
	InsertRate      float64   // 投币速率（个/分钟）
	ReturnRate      float64   // 回币速率（个/分钟）
	ReturnRatio     float64   // 回报率
}

// ACMConfig ACM控制器配置
type ACMConfig struct {
	DevicePattern string        // 设备匹配模式
	BaudRate      int           // 波特率
	DataBits      int           // 数据位
	StopBits      int           // 停止位
	Parity        string        // 校验位
	ReadTimeout   time.Duration // 读取超时
	WriteTimeout  time.Duration // 写入超时
	RetryCount    int           // 重试次数
	RetryDelay    time.Duration // 重试延迟
}