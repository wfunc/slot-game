package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorCode 错误码类型
type ErrorCode int

// 错误码定义（按模块分组）
const (
	// 通用错误 (1000-1999)
	ErrUnknown         ErrorCode = 1000
	ErrInvalidParam    ErrorCode = 1001
	ErrNotFound        ErrorCode = 1002
	ErrAlreadyExists   ErrorCode = 1003
	ErrPermissionDenied ErrorCode = 1004
	ErrTimeout         ErrorCode = 1005
	ErrCanceled        ErrorCode = 1006
	ErrNotImplemented  ErrorCode = 1007
	
	// 游戏错误 (2000-2999)
	ErrGameNotStarted   ErrorCode = 2000
	ErrGameAlreadyStarted ErrorCode = 2001
	ErrInsufficientCoins ErrorCode = 2002
	ErrInvalidBet       ErrorCode = 2003
	ErrGameStateError   ErrorCode = 2004
	ErrSpinInProgress   ErrorCode = 2005
	ErrInvalidWinPattern ErrorCode = 2006
	
	// 硬件错误 (3000-3999)
	ErrSerialPortOpen   ErrorCode = 3000
	ErrSerialPortWrite  ErrorCode = 3001
	ErrSerialPortRead   ErrorCode = 3002
	ErrSerialTimeout    ErrorCode = 3003
	ErrDeviceOffline    ErrorCode = 3004
	ErrDeviceBusy       ErrorCode = 3005
	ErrCommandFailed    ErrorCode = 3006
	ErrInvalidResponse  ErrorCode = 3007
	
	// 通信错误 (4000-4999)
	ErrWebSocketConnect  ErrorCode = 4000
	ErrWebSocketSend     ErrorCode = 4001
	ErrWebSocketReceive  ErrorCode = 4002
	ErrWebSocketClosed   ErrorCode = 4003
	ErrMQTTConnect       ErrorCode = 4004
	ErrMQTTPublish       ErrorCode = 4005
	ErrMQTTSubscribe     ErrorCode = 4006
	ErrMessageFormat     ErrorCode = 4007
	
	// 数据库错误 (5000-5999)
	ErrDatabaseConnect  ErrorCode = 5000
	ErrDatabaseQuery    ErrorCode = 5001
	ErrDatabaseInsert   ErrorCode = 5002
	ErrDatabaseUpdate   ErrorCode = 5003
	ErrDatabaseDelete   ErrorCode = 5004
	ErrTransaction      ErrorCode = 5005
	ErrDataIntegrity    ErrorCode = 5006
	
	// 配置错误 (6000-6999)
	ErrConfigLoad       ErrorCode = 6000
	ErrConfigParse      ErrorCode = 6001
	ErrConfigValidate   ErrorCode = 6002
	ErrConfigMissing    ErrorCode = 6003
	
	// 安全错误 (7000-7999)
	ErrAuthentication   ErrorCode = 7000
	ErrAuthorization    ErrorCode = 7001
	ErrTokenExpired     ErrorCode = 7002
	ErrTokenInvalid     ErrorCode = 7003
	ErrRateLimitExceeded ErrorCode = 7004
	ErrEncryption       ErrorCode = 7005
	ErrDecryption       ErrorCode = 7006
)

// 错误码消息映射
var errorMessages = map[ErrorCode]string{
	// 通用错误
	ErrUnknown:          "未知错误",
	ErrInvalidParam:     "无效的参数",
	ErrNotFound:         "资源未找到",
	ErrAlreadyExists:    "资源已存在",
	ErrPermissionDenied: "权限不足",
	ErrTimeout:          "操作超时",
	ErrCanceled:         "操作已取消",
	ErrNotImplemented:   "功能未实现",
	
	// 游戏错误
	ErrGameNotStarted:     "游戏未开始",
	ErrGameAlreadyStarted: "游戏已经开始",
	ErrInsufficientCoins:  "币数不足",
	ErrInvalidBet:         "无效的投注金额",
	ErrGameStateError:     "游戏状态错误",
	ErrSpinInProgress:     "转轮正在进行中",
	ErrInvalidWinPattern:  "无效的中奖模式",
	
	// 硬件错误
	ErrSerialPortOpen:  "串口打开失败",
	ErrSerialPortWrite: "串口写入失败",
	ErrSerialPortRead:  "串口读取失败",
	ErrSerialTimeout:   "串口通信超时",
	ErrDeviceOffline:   "设备离线",
	ErrDeviceBusy:      "设备忙",
	ErrCommandFailed:   "命令执行失败",
	ErrInvalidResponse: "无效的设备响应",
	
	// 通信错误
	ErrWebSocketConnect: "WebSocket连接失败",
	ErrWebSocketSend:    "WebSocket发送失败",
	ErrWebSocketReceive: "WebSocket接收失败",
	ErrWebSocketClosed:  "WebSocket连接已关闭",
	ErrMQTTConnect:      "MQTT连接失败",
	ErrMQTTPublish:      "MQTT发布失败",
	ErrMQTTSubscribe:    "MQTT订阅失败",
	ErrMessageFormat:    "消息格式错误",
	
	// 数据库错误
	ErrDatabaseConnect: "数据库连接失败",
	ErrDatabaseQuery:   "数据库查询失败",
	ErrDatabaseInsert:  "数据库插入失败",
	ErrDatabaseUpdate:  "数据库更新失败",
	ErrDatabaseDelete:  "数据库删除失败",
	ErrTransaction:     "事务处理失败",
	ErrDataIntegrity:   "数据完整性错误",
	
	// 配置错误
	ErrConfigLoad:     "配置加载失败",
	ErrConfigParse:    "配置解析失败",
	ErrConfigValidate: "配置验证失败",
	ErrConfigMissing:  "配置项缺失",
	
	// 安全错误
	ErrAuthentication:    "认证失败",
	ErrAuthorization:     "授权失败",
	ErrTokenExpired:      "令牌已过期",
	ErrTokenInvalid:      "无效的令牌",
	ErrRateLimitExceeded: "请求频率超限",
	ErrEncryption:        "加密失败",
	ErrDecryption:        "解密失败",
}

// AppError 应用错误结构
type AppError struct {
	Code      ErrorCode   `json:"code"`      // 错误码
	Message   string      `json:"message"`   // 错误消息
	Details   string      `json:"details"`   // 详细信息
	Cause     error       `json:"-"`         // 原始错误
	Stack     []StackFrame `json:"stack,omitempty"` // 调用栈
}

// StackFrame 调用栈帧
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithCause 添加原因错误
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	if cause != nil && e.Details == "" {
		e.Details = cause.Error()
	}
	return e
}

// New 创建新的应用错误
func New(code ErrorCode, details ...string) *AppError {
	message, ok := errorMessages[code]
	if !ok {
		message = errorMessages[ErrUnknown]
	}
	
	err := &AppError{
		Code:    code,
		Message: message,
	}
	
	if len(details) > 0 {
		err.Details = strings.Join(details, "; ")
	}
	
	// 捕获调用栈
	err.captureStack(2)
	
	return err
}

// Newf 创建格式化的应用错误
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	details := fmt.Sprintf(format, args...)
	return New(code, details)
}

// Wrap 包装错误
func Wrap(err error, code ErrorCode, details ...string) *AppError {
	if err == nil {
		return nil
	}
	
	// 如果已经是AppError，保留原始错误码
	if appErr, ok := err.(*AppError); ok {
		if len(details) > 0 {
			appErr.Details = strings.Join(details, "; ") + "; " + appErr.Details
		}
		return appErr
	}
	
	appErr := New(code, details...)
	appErr.Cause = err
	if appErr.Details == "" {
		appErr.Details = err.Error()
	}
	
	return appErr
}

// Wrapf 包装格式化错误
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	details := fmt.Sprintf(format, args...)
	return Wrap(err, code, details)
}

// Is 判断错误是否为指定错误码
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	
	appErr, ok := err.(*AppError)
	return ok && appErr.Code == code
}

// GetCode 获取错误码
func GetCode(err error) ErrorCode {
	if err == nil {
		return 0
	}
	
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	
	return ErrUnknown
}

// captureStack 捕获调用栈
func (e *AppError) captureStack(skip int) {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip+1, pcs)
	
	if n > 0 {
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			
			// 跳过runtime和本包的调用
			if strings.Contains(frame.Function, "runtime.") ||
				strings.Contains(frame.Function, "github.com/wfunc/slot-game/internal/errors") {
				if !more {
					break
				}
				continue
			}
			
			e.Stack = append(e.Stack, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})
			
			if !more {
				break
			}
			
			// 只保留前10个栈帧
			if len(e.Stack) >= 10 {
				break
			}
		}
	}
}

// GetStack 获取格式化的调用栈
func (e *AppError) GetStack() string {
	if len(e.Stack) == 0 {
		return ""
	}
	
	var builder strings.Builder
	for i, frame := range e.Stack {
		builder.WriteString(fmt.Sprintf("%d. %s\n   %s:%d\n",
			i+1, frame.Function, frame.File, frame.Line))
	}
	
	return builder.String()
}

// HTTPStatus 返回对应的HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch {
	case e.Code >= 1001 && e.Code <= 1003:
		return 400 // Bad Request
	case e.Code == ErrNotFound:
		return 404 // Not Found
	case e.Code == ErrPermissionDenied:
		return 403 // Forbidden
	case e.Code == ErrTimeout:
		return 408 // Request Timeout
	case e.Code >= 7000 && e.Code <= 7003:
		return 401 // Unauthorized
	case e.Code == ErrRateLimitExceeded:
		return 429 // Too Many Requests
	case e.Code >= 5000 && e.Code <= 5999:
		return 503 // Service Unavailable
	default:
		return 500 // Internal Server Error
	}
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	code := GetCode(err)
	switch code {
	case ErrTimeout,
		ErrSerialTimeout,
		ErrWebSocketConnect,
		ErrMQTTConnect,
		ErrDatabaseConnect,
		ErrDeviceOffline,
		ErrDeviceBusy:
		return true
	default:
		return false
	}
}

// IsCritical 判断是否为严重错误
func IsCritical(err error) bool {
	if err == nil {
		return false
	}
	
	code := GetCode(err)
	switch code {
	case ErrDatabaseConnect,
		ErrSerialPortOpen,
		ErrConfigLoad,
		ErrConfigMissing,
		ErrDataIntegrity:
		return true
	default:
		return false
	}
}

// ErrorResponse API错误响应结构
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     *AppError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err *AppError, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     err,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
}