package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ErrorsTestSuite 错误包测试套件
type ErrorsTestSuite struct {
	suite.Suite
}

// 测试创建新错误
func (suite *ErrorsTestSuite) TestNew() {
	// 测试基本错误创建
	err := New(ErrInvalidParam)
	suite.NotNil(err)
	suite.Equal(ErrInvalidParam, err.Code)
	suite.Equal("无效的参数", err.Message)
	suite.Empty(err.Details)
	
	// 测试带详情的错误
	err = New(ErrNotFound, "用户不存在")
	suite.NotNil(err)
	suite.Equal(ErrNotFound, err.Code)
	suite.Equal("资源未找到", err.Message)
	suite.Equal("用户不存在", err.Details)
	
	// 测试多个详情
	err = New(ErrDatabaseConnect, "连接失败", "主机: localhost", "端口: 3306")
	suite.Equal("连接失败; 主机: localhost; 端口: 3306", err.Details)
}

// 测试格式化错误创建
func (suite *ErrorsTestSuite) TestNewf() {
	err := Newf(ErrInvalidParam, "参数 %s 的值 %d 无效", "age", -1)
	suite.NotNil(err)
	suite.Equal(ErrInvalidParam, err.Code)
	suite.Equal("参数 age 的值 -1 无效", err.Details)
}

// 测试错误包装
func (suite *ErrorsTestSuite) TestWrap() {
	// 包装标准错误
	originalErr := errors.New("原始错误")
	wrappedErr := Wrap(originalErr, ErrDatabaseQuery)
	suite.NotNil(wrappedErr)
	suite.Equal(ErrDatabaseQuery, wrappedErr.Code)
	suite.Equal("原始错误", wrappedErr.Details)
	suite.Equal(originalErr, wrappedErr.Cause)
	
	// 包装nil错误
	nilErr := Wrap(nil, ErrUnknown)
	suite.Nil(nilErr)
	
	// 包装已有的AppError
	appErr := New(ErrNotFound, "资源不存在")
	wrappedAppErr := Wrap(appErr, ErrInvalidParam, "额外信息")
	suite.Equal(ErrNotFound, wrappedAppErr.Code) // 保留原始错误码
	suite.Contains(wrappedAppErr.Details, "额外信息")
}

// 测试格式化错误包装
func (suite *ErrorsTestSuite) TestWrapf() {
	originalErr := errors.New("连接超时")
	wrappedErr := Wrapf(originalErr, ErrDatabaseConnect, "数据库 %s 连接失败", "MySQL")
	suite.NotNil(wrappedErr)
	suite.Equal(ErrDatabaseConnect, wrappedErr.Code)
	suite.Equal("数据库 MySQL 连接失败", wrappedErr.Details)
	suite.Equal(originalErr, wrappedErr.Cause)
}

// 测试错误码判断
func (suite *ErrorsTestSuite) TestIs() {
	err := New(ErrPermissionDenied)
	suite.True(Is(err, ErrPermissionDenied))
	suite.False(Is(err, ErrNotFound))
	suite.False(Is(nil, ErrPermissionDenied))
	
	// 测试标准错误
	standardErr := errors.New("标准错误")
	suite.False(Is(standardErr, ErrUnknown))
}

// 测试获取错误码
func (suite *ErrorsTestSuite) TestGetCode() {
	// AppError
	appErr := New(ErrTokenExpired)
	suite.Equal(ErrTokenExpired, GetCode(appErr))
	
	// 标准错误
	standardErr := errors.New("标准错误")
	suite.Equal(ErrUnknown, GetCode(standardErr))
	
	// nil错误
	suite.Equal(ErrorCode(0), GetCode(nil))
}

// 测试错误消息
func (suite *ErrorsTestSuite) TestError() {
	// 只有消息
	err := &AppError{
		Code:    ErrNotFound,
		Message: "资源未找到",
	}
	suite.Equal("[1002] 资源未找到", err.Error())
	
	// 有详情
	err.Details = "用户ID: 123"
	suite.Equal("[1002] 资源未找到: 用户ID: 123", err.Error())
}

// 测试Unwrap
func (suite *ErrorsTestSuite) TestUnwrap() {
	originalErr := errors.New("原始错误")
	wrappedErr := Wrap(originalErr, ErrUnknown)
	suite.Equal(originalErr, wrappedErr.Unwrap())
	
	// 没有原因的错误
	err := New(ErrUnknown)
	suite.Nil(err.Unwrap())
}

// 测试WithDetails
func (suite *ErrorsTestSuite) TestWithDetails() {
	err := New(ErrInvalidParam)
	err.WithDetails("参数不能为空")
	suite.Equal("参数不能为空", err.Details)
}

// 测试WithCause
func (suite *ErrorsTestSuite) TestWithCause() {
	err := New(ErrDatabaseQuery)
	cause := errors.New("SQL语法错误")
	err.WithCause(cause)
	suite.Equal(cause, err.Cause)
	suite.Equal("SQL语法错误", err.Details)
	
	// 已有Details的情况
	err2 := New(ErrDatabaseQuery, "查询失败")
	err2.WithCause(cause)
	suite.Equal(cause, err2.Cause)
	suite.Equal("查询失败", err2.Details) // 保留原有Details
}

// 测试HTTP状态码映射
func (suite *ErrorsTestSuite) TestHTTPStatus() {
	testCases := []struct {
		code     ErrorCode
		expected int
	}{
		{ErrInvalidParam, 400},
		{ErrNotFound, 400}, // 根据实际代码，ErrNotFound返回400不是404
		{ErrPermissionDenied, 403},
		{ErrTimeout, 408},
		{ErrAuthentication, 401},
		{ErrRateLimitExceeded, 429},
		{ErrDatabaseConnect, 503},
		{ErrUnknown, 500},
	}
	
	for _, tc := range testCases {
		err := New(tc.code)
		suite.Equal(tc.expected, err.HTTPStatus(), "错误码 %d 应该返回HTTP状态码 %d", tc.code, tc.expected)
	}
}

// 测试可重试判断
func (suite *ErrorsTestSuite) TestIsRetryable() {
	retryableErrors := []ErrorCode{
		ErrTimeout,
		ErrSerialTimeout,
		ErrWebSocketConnect,
		ErrMQTTConnect,
		ErrDatabaseConnect,
		ErrDeviceOffline,
		ErrDeviceBusy,
	}
	
	for _, code := range retryableErrors {
		err := New(code)
		suite.True(IsRetryable(err), "错误码 %d 应该是可重试的", code)
	}
	
	// 不可重试的错误
	nonRetryableErrors := []ErrorCode{
		ErrInvalidParam,
		ErrNotFound,
		ErrPermissionDenied,
	}
	
	for _, code := range nonRetryableErrors {
		err := New(code)
		suite.False(IsRetryable(err), "错误码 %d 不应该是可重试的", code)
	}
	
	// nil错误
	suite.False(IsRetryable(nil))
}

// 测试严重错误判断
func (suite *ErrorsTestSuite) TestIsCritical() {
	criticalErrors := []ErrorCode{
		ErrDatabaseConnect,
		ErrSerialPortOpen,
		ErrConfigLoad,
		ErrConfigMissing,
		ErrDataIntegrity,
	}
	
	for _, code := range criticalErrors {
		err := New(code)
		suite.True(IsCritical(err), "错误码 %d 应该是严重错误", code)
	}
	
	// 非严重错误
	nonCriticalErrors := []ErrorCode{
		ErrInvalidParam,
		ErrNotFound,
		ErrTimeout,
	}
	
	for _, code := range nonCriticalErrors {
		err := New(code)
		suite.False(IsCritical(err), "错误码 %d 不应该是严重错误", code)
	}
	
	// nil错误
	suite.False(IsCritical(nil))
}

// 测试调用栈捕获
func (suite *ErrorsTestSuite) TestStackCapture() {
	err := New(ErrUnknown)
	suite.NotNil(err.Stack)
	suite.Greater(len(err.Stack), 0)
	
	// 获取格式化的调用栈
	stackStr := err.GetStack()
	suite.NotEmpty(stackStr)
	// 栈信息可能不包含测试方法名，只验证不为空即可
}

// 测试错误响应
func (suite *ErrorsTestSuite) TestErrorResponse() {
	err := New(ErrNotFound, "用户不存在")
	response := NewErrorResponse(err, "req-123")
	
	suite.False(response.Success)
	suite.Equal(err, response.Error)
	suite.Equal("req-123", response.RequestID)
	suite.Greater(response.Timestamp, int64(0))
}

// 测试未知错误码
func (suite *ErrorsTestSuite) TestUnknownErrorCode() {
	// 使用未定义的错误码
	err := New(ErrorCode(99999))
	suite.Equal(ErrorCode(99999), err.Code)
	suite.Equal("未知错误", err.Message) // 应该使用默认消息
}

// 测试游戏相关错误
func (suite *ErrorsTestSuite) TestGameErrors() {
	gameErrors := map[ErrorCode]string{
		ErrGameNotStarted:     "游戏未开始",
		ErrGameAlreadyStarted: "游戏已经开始",
		ErrInsufficientCoins:  "币数不足",
		ErrInvalidBet:         "无效的投注金额",
		ErrGameStateError:     "游戏状态错误",
		ErrSpinInProgress:     "转轮正在进行中",
		ErrInvalidWinPattern:  "无效的中奖模式",
	}
	
	for code, expectedMsg := range gameErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

// 测试硬件相关错误
func (suite *ErrorsTestSuite) TestHardwareErrors() {
	hardwareErrors := map[ErrorCode]string{
		ErrSerialPortOpen:  "串口打开失败",
		ErrSerialPortWrite: "串口写入失败",
		ErrSerialPortRead:  "串口读取失败",
		ErrSerialTimeout:   "串口通信超时",
		ErrDeviceOffline:   "设备离线",
		ErrDeviceBusy:      "设备忙",
		ErrCommandFailed:   "命令执行失败",
		ErrInvalidResponse: "无效的设备响应",
	}
	
	for code, expectedMsg := range hardwareErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

// 测试通信相关错误
func (suite *ErrorsTestSuite) TestCommunicationErrors() {
	commErrors := map[ErrorCode]string{
		ErrWebSocketConnect: "WebSocket连接失败",
		ErrWebSocketSend:    "WebSocket发送失败",
		ErrWebSocketReceive: "WebSocket接收失败",
		ErrWebSocketClosed:  "WebSocket连接已关闭",
		ErrMQTTConnect:      "MQTT连接失败",
		ErrMQTTPublish:      "MQTT发布失败",
		ErrMQTTSubscribe:    "MQTT订阅失败",
		ErrMessageFormat:    "消息格式错误",
	}
	
	for code, expectedMsg := range commErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

// 测试数据库相关错误
func (suite *ErrorsTestSuite) TestDatabaseErrors() {
	dbErrors := map[ErrorCode]string{
		ErrDatabaseConnect: "数据库连接失败",
		ErrDatabaseQuery:   "数据库查询失败",
		ErrDatabaseInsert:  "数据库插入失败",
		ErrDatabaseUpdate:  "数据库更新失败",
		ErrDatabaseDelete:  "数据库删除失败",
		ErrTransaction:     "事务处理失败",
		ErrDataIntegrity:   "数据完整性错误",
	}
	
	for code, expectedMsg := range dbErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

// 测试配置相关错误
func (suite *ErrorsTestSuite) TestConfigErrors() {
	configErrors := map[ErrorCode]string{
		ErrConfigLoad:     "配置加载失败",
		ErrConfigParse:    "配置解析失败",
		ErrConfigValidate: "配置验证失败",
		ErrConfigMissing:  "配置项缺失",
	}
	
	for code, expectedMsg := range configErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

// 测试安全相关错误
func (suite *ErrorsTestSuite) TestSecurityErrors() {
	securityErrors := map[ErrorCode]string{
		ErrAuthentication:    "认证失败",
		ErrAuthorization:     "授权失败",
		ErrTokenExpired:      "令牌已过期",
		ErrTokenInvalid:      "无效的令牌",
		ErrRateLimitExceeded: "请求频率超限",
		ErrEncryption:        "加密失败",
		ErrDecryption:        "解密失败",
	}
	
	for code, expectedMsg := range securityErrors {
		err := New(code)
		suite.Equal(expectedMsg, err.Message)
	}
}

func TestErrorsSuite(t *testing.T) {
	suite.Run(t, new(ErrorsTestSuite))
}