package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// SerialLogType 串口日志类型
type SerialLogType string

const (
	SerialLogTypeACMSend     SerialLogType = "ACM_SEND"     // ACM发送
	SerialLogTypeACMReceive  SerialLogType = "ACM_RECEIVE"  // ACM接收
	SerialLogTypeSTM32Send   SerialLogType = "STM32_SEND"   // STM32发送
	SerialLogTypeSTM32Receive SerialLogType = "STM32_RECEIVE" // STM32接收
)

// SerialLogLevel 日志级别
type SerialLogLevel string

const (
	SerialLogLevelInfo  SerialLogLevel = "INFO"
	SerialLogLevelDebug SerialLogLevel = "DEBUG"
	SerialLogLevelWarn  SerialLogLevel = "WARN"
	SerialLogLevelError SerialLogLevel = "ERROR"
)

// JSONData 用于存储JSON格式的数据
type JSONData map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSONData) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONData) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		strVal, ok := value.(string)
		if !ok {
			return nil
		}
		bytes = []byte(strVal)
	}
	return json.Unmarshal(bytes, j)
}

// SerialLog 串口通信日志
type SerialLog struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"index;not null" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 基础信息
	DeviceType SerialLogType  `gorm:"type:varchar(20);index;not null" json:"device_type"` // 设备类型 (ACM/STM32)
	Direction  string         `gorm:"type:varchar(10);index;not null" json:"direction"`   // 方向 (SEND/RECEIVE)
	Level      SerialLogLevel `gorm:"type:varchar(10);default:INFO" json:"level"`         // 日志级别

	// 命令相关
	Command  string `gorm:"type:varchar(255);index" json:"command,omitempty"`  // 命令内容 (如 "algo -b 1 -p 100")
	Function string `gorm:"type:varchar(100);index" json:"function,omitempty"` // 功能名称 (如 "algo", "status")
	MsgType  string `gorm:"type:varchar(50)" json:"msg_type,omitempty"`         // 消息类型 (如 "M1", "M2", "M4")

	// 数据内容
	RawData    string   `gorm:"type:text" json:"raw_data,omitempty"`      // 原始数据 (ASCII)
	HexData    string   `gorm:"type:text" json:"hex_data,omitempty"`      // 十六进制数据
	JSONData   JSONData `gorm:"type:json" json:"json_data,omitempty"`    // JSON格式的数据（用于ACM响应）
	BytesCount int      `gorm:"default:0" json:"bytes_count"`             // 字节数

	// 游戏相关数据（用于 algo 命令）
	Bet   float64 `gorm:"type:decimal(10,4);index" json:"bet,omitempty"`   // 下注金额
	Prize float64 `gorm:"type:decimal(10,4);index" json:"prize,omitempty"` // 奖池金额
	Win   float64 `gorm:"type:decimal(10,4);index" json:"win,omitempty"`   // 中奖金额

	// 响应相关
	ResponseCode int    `gorm:"index" json:"response_code,omitempty"`       // 响应代码
	ResponseMsg  string `gorm:"type:varchar(255)" json:"response_msg,omitempty"` // 响应消息
	ErrorMsg     string `gorm:"type:text" json:"error_msg,omitempty"`       // 错误信息

	// 关联信息
	RequestID  string `gorm:"type:varchar(100);index" json:"request_id,omitempty"`  // 请求ID（用于关联请求和响应）
	SessionID  string `gorm:"type:varchar(100);index" json:"session_id,omitempty"`  // 会话ID
	Ident      int    `gorm:"index" json:"ident,omitempty"`                         // 标识符（从ACM响应中获取）

	// 性能指标
	Duration  int64 `gorm:"default:0" json:"duration,omitempty"`  // 处理时长（毫秒）
	Timestamp int64 `gorm:"index" json:"timestamp"`               // Unix时间戳（毫秒）

	// 额外信息
	Caller  string `gorm:"type:varchar(255)" json:"caller,omitempty"`  // 调用位置（如 hardware/acm_controller.go:526）
	Message string `gorm:"type:text" json:"message,omitempty"`         // 日志消息
	Extra   string `gorm:"type:text" json:"extra,omitempty"`           // 额外信息
}

// TableName 指定表名
func (SerialLog) TableName() string {
	return "serial_logs"
}

// BeforeCreate 创建前的钩子
func (s *SerialLog) BeforeCreate(tx *gorm.DB) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.Timestamp == 0 {
		s.Timestamp = time.Now().UnixMilli()
	}
	return nil
}

// SerialLogQuery 查询参数
type SerialLogQuery struct {
	DeviceType SerialLogType  `json:"device_type,omitempty"`
	Direction  string         `json:"direction,omitempty"`
	Level      SerialLogLevel `json:"level,omitempty"`
	Command    string         `json:"command,omitempty"`
	Function   string         `json:"function,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	StartTime  *time.Time     `json:"start_time,omitempty"`
	EndTime    *time.Time     `json:"end_time,omitempty"`
	MinBet     *float64       `json:"min_bet,omitempty"`
	MaxBet     *float64       `json:"max_bet,omitempty"`
	MinWin     *float64       `json:"min_win,omitempty"`
	MaxWin     *float64       `json:"max_win,omitempty"`
	HasError   *bool          `json:"has_error,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	Offset     int            `json:"offset,omitempty"`
	OrderBy    string         `json:"order_by,omitempty"`
}

// SerialLogStats 统计信息
type SerialLogStats struct {
	TotalCount      int64   `json:"total_count"`
	TotalSend       int64   `json:"total_send"`
	TotalReceive    int64   `json:"total_receive"`
	TotalACM        int64   `json:"total_acm"`
	TotalSTM32      int64   `json:"total_stm32"`
	TotalErrors     int64   `json:"total_errors"`
	TotalBet        float64 `json:"total_bet"`
	TotalWin        float64 `json:"total_win"`
	AvgDuration     float64 `json:"avg_duration"`
	MaxDuration     int64   `json:"max_duration"`
	MinDuration     int64   `json:"min_duration"`
}