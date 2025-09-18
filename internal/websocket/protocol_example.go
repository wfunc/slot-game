package websocket

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ExampleMessageHandler 示例消息处理器
type ExampleMessageHandler struct {
	logger *zap.Logger
}

// NewExampleMessageHandler 创建示例消息处理器
func NewExampleMessageHandler(logger *zap.Logger) *ExampleMessageHandler {
	return &ExampleMessageHandler{
		logger: logger,
	}
}

// HandleMessage 处理客户端消息
func (h *ExampleMessageHandler) HandleMessage(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	h.logger.Info("处理消息",
		zap.String("client_id", client.ID),
		zap.Uint16("cmd", msg.Cmd),
		zap.Uint32("flag", msg.Flag),
		zap.Int("data_len", len(msg.Data)))

	// 根据命令处理不同的消息
	switch msg.Cmd {
	case 1001: // 登录请求
		return h.handleLogin(client, msg)
	case 1002: // 心跳
		return h.handleHeartbeat(client, msg)
	case 1003: // 游戏请求
		return h.handleGameRequest(client, msg)
	case 2099: // 特殊命令（前端测试）
		return h.handleCommand2099(client, msg)
	default:
		// 未知命令也返回响应，避免客户端断开
		h.logger.Warn("收到未知命令",
			zap.Uint16("cmd", msg.Cmd),
			zap.Uint32("flag", msg.Flag))
		return client.GetProtocol().CreateErrorResponse(msg.Cmd, msg.Flag, 1000, fmt.Sprintf("未知命令: %d", msg.Cmd)), nil
	}
}

// handleLogin 处理登录请求
func (h *ExampleMessageHandler) handleLogin(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	// 解析登录数据（这里假设是JSON格式）
	var loginData map[string]interface{}
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &loginData); err != nil {
			return client.GetProtocol().CreateErrorResponse(msg.Cmd, msg.Flag, 2001, "登录数据格式错误"), nil
		}
	}

	// 模拟登录成功
	responseData := map[string]interface{}{
		"user_id":  12345,
		"username": "test_user",
		"token":    "mock_token_123456",
		"status":   "success",
	}

	respBytes, _ := json.Marshal(responseData)
	return client.GetProtocol().CreateSuccessResponse(msg.Cmd, msg.Flag, respBytes), nil
}

// handleHeartbeat 处理心跳
func (h *ExampleMessageHandler) handleHeartbeat(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	// 心跳响应
	return client.GetProtocol().CreateSuccessResponse(msg.Cmd, msg.Flag, []byte("pong")), nil
}

// handleGameRequest 处理游戏请求
func (h *ExampleMessageHandler) handleGameRequest(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	// 解析游戏请求数据
	var gameData map[string]interface{}
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &gameData); err != nil {
			return client.GetProtocol().CreateErrorResponse(msg.Cmd, msg.Flag, 3001, "游戏数据格式错误"), nil
		}
	}

	// 模拟游戏响应
	responseData := map[string]interface{}{
		"game_id":   "game_123",
		"result":    "win",
		"score":     1000,
		"timestamp": 1234567890,
	}

	respBytes, _ := json.Marshal(responseData)
	return client.GetProtocol().CreateSuccessResponse(msg.Cmd, msg.Flag, respBytes), nil
}

// handleCommand2099 处理2099命令（可能是获取配置或状态查询）
func (h *ExampleMessageHandler) handleCommand2099(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error) {
	h.logger.Info("处理2099命令",
		zap.String("client_id", client.ID),
		zap.Uint32("flag", msg.Flag),
		zap.Int("data_len", len(msg.Data)))

	// 返回成功响应，包含一些配置信息
	responseData := map[string]interface{}{
		"status": "success",
		"code":   0,
		"data": map[string]interface{}{
			"serverVersion": "1.0.0",
			"protocolVersion": 1,
			"supportedCmds": []uint16{1001, 1002, 1003, 2099},
			"timestamp": time.Now().Unix(),
		},
		"message": "配置获取成功",
	}

	respBytes, _ := json.Marshal(responseData)

	// 返回成功响应（ErrorID = 0 表示成功）
	return client.GetProtocol().CreateSuccessResponse(msg.Cmd, msg.Flag, respBytes), nil
}

// CommandDefinitions 命令定义（与前端对应）
const (
	CmdLogin     uint16 = 1001 // 登录
	CmdHeartbeat uint16 = 1002 // 心跳
	CmdGame      uint16 = 1003 // 游戏

	// 游戏相关命令
	CmdSlotStart     uint16 = 2001 // 老虎机开始
	CmdSlotSpin      uint16 = 2002 // 老虎机转动
	CmdSlotSettle    uint16 = 2003 // 老虎机结算
	CmdSlotBatchSpin uint16 = 2004 // 批量转动

	// 钱包相关命令
	CmdWalletBalance  uint16 = 3001 // 查询余额
	CmdWalletDeposit  uint16 = 3002 // 充值
	CmdWalletWithdraw uint16 = 3003 // 提现
)

// ErrorCodes 错误码定义
const (
	ErrSuccess       uint16 = 0    // 成功
	ErrUnknown       uint16 = 1000 // 未知错误
	ErrDecode        uint16 = 1001 // 解码失败
	ErrProcess       uint16 = 1002 // 处理失败
	ErrAuth          uint16 = 2000 // 认证失败
	ErrInvalidData   uint16 = 2001 // 数据无效
	ErrGameError     uint16 = 3000 // 游戏错误
	ErrInvalidBet    uint16 = 3001 // 投注无效
	ErrInsufficientBalance uint16 = 3002 // 余额不足
)