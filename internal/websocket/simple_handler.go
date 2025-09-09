package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SimpleMessageHandler 简化的WebSocket消息处理器
type SimpleMessageHandler struct {
	hub        *Hub
	db         *gorm.DB
	walletRepo repository.WalletRepository
	logger     *zap.Logger
}

// NewSimpleMessageHandler 创建简化的消息处理器
func NewSimpleMessageHandler(hub *Hub, db *gorm.DB, logger *zap.Logger) *SimpleMessageHandler {
	return &SimpleMessageHandler{
		hub:        hub,
		db:         db,
		walletRepo: repository.NewWalletRepository(db),
		logger:     logger,
	}
}

// HandleClientMessage 处理客户端消息
func (h *SimpleMessageHandler) HandleClientMessage(client *Client, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Error("解析消息失败",
			zap.String("client_id", client.ID),
			zap.Error(err))
		h.sendError(client, "消息格式错误")
		// 断开恶意或格式错误的连接
		client.Close()
		return
	}
	
	// 验证消息类型不为空
	if msg.Type == "" {
		h.logger.Warn("收到空消息类型",
			zap.String("client_id", client.ID))
		h.sendError(client, "消息类型不能为空")
		// 断开连接
		client.Close()
		return
	}

	// 设置消息元数据
	msg.UserID = client.UserID
	msg.Timestamp = time.Now().Unix()

	h.logger.Info("收到WebSocket消息",
		zap.String("client_id", client.ID),
		zap.String("type", msg.Type),
		zap.Uint("user_id", client.UserID))

	// 根据消息类型处理
	switch msg.Type {
	case MessageTypePing:
		h.handlePing(client)

	case MessageTypePong:
		h.handlePong(client)

	case MessageTypeHeartbeat:
		h.handleHeartbeat(client)

	case MessageTypeSubscribe:
		h.handleSubscribe(client, &msg)

	case MessageTypeGetBalance:
		h.handleGetBalance(client)

	case MessageTypeGetStatus:
		h.handleGetStatus(client)

	case MessageTypeGameState:
		h.handleGetGameState(client)

	default:
		h.logger.Warn("未知消息类型",
			zap.String("client_id", client.ID),
			zap.String("type", msg.Type))
		h.sendError(client, "不支持的消息类型: "+msg.Type)
		// 断开发送无效消息类型的连接
		client.Close()
	}
}

// handlePing 处理ping消息
func (h *SimpleMessageHandler) handlePing(client *Client) {
	response := &Message{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"message":"pong"}`),
	}
	h.hub.SendToClient(client.ID, response)
}

// handlePong 处理pong消息
func (h *SimpleMessageHandler) handlePong(client *Client) {
	h.logger.Debug("收到pong",
		zap.String("client_id", client.ID))
}

// handleHeartbeat 处理心跳消息
func (h *SimpleMessageHandler) handleHeartbeat(client *Client) {
	response := &Message{
		Type:      "heartbeat",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"status":"alive","server_time":%d}`, time.Now().Unix())),
	}
	h.hub.SendToClient(client.ID, response)
}

// handleSubscribe 处理订阅消息
func (h *SimpleMessageHandler) handleSubscribe(client *Client, msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		// 如果没有数据，默认订阅game频道
		data = map[string]interface{}{"channel": "game"}
	}

	channel, _ := data["channel"].(string)
	if channel == "" {
		channel = "game"
	}

	h.logger.Info("客户端订阅频道",
		zap.String("client_id", client.ID),
		zap.String("channel", channel))

	response := &Message{
		Type:      "subscribe",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"channel":"` + channel + `","status":"subscribed","message":"订阅成功"}`),
	}
	h.hub.SendToClient(client.ID, response)
}

// handleGetBalance 处理获取余额请求
func (h *SimpleMessageHandler) handleGetBalance(client *Client) {
	// 如果是访客用户，返回模拟数据
	if client.UserID == 999999 || client.UserID == 0 {
		response := &Message{
			Type:      "balance",
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(`{"balance":10000,"currency":"CNY","demo":true,"message":"访客模式余额"}`),
		}
		h.hub.SendToClient(client.ID, response)
		return
	}

	// 获取真实用户余额
	ctx := context.Background()
	wallet, err := h.walletRepo.GetByUserID(ctx, client.UserID)
	if err != nil {
		h.logger.Error("获取钱包失败",
			zap.Uint("user_id", client.UserID),
			zap.Error(err))
		// 返回默认余额
		response := &Message{
			Type:      "balance",
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(`{"balance":0,"currency":"CNY","error":"获取余额失败"}`),
		}
		h.hub.SendToClient(client.ID, response)
		return
	}

	balanceData := map[string]interface{}{
		"balance":  wallet.Balance,
		"currency": "CNY",
		"demo":     false,
		"message":  "余额查询成功",
	}

	data, _ := json.Marshal(balanceData)
	response := &Message{
		Type:      "balance",
		UserID:    client.UserID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
}

// handleGetStatus 处理获取状态请求
func (h *SimpleMessageHandler) handleGetStatus(client *Client) {
	// 获取服务器状态
	statusData := map[string]interface{}{
		"server_status": "online",
		"connected":     true,
		"user_id":       client.UserID,
		"client_id":     client.ID,
		"session_id":    client.SessionID,
		"timestamp":     time.Now().Unix(),
		"message":       "服务器运行正常",
	}

	data, _ := json.Marshal(statusData)
	response := &Message{
		Type:      "status",
		UserID:    client.UserID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
}

// handleGetGameState 处理获取游戏状态请求
func (h *SimpleMessageHandler) handleGetGameState(client *Client) {
	// 简化版：返回基本游戏状态
	gameState := map[string]interface{}{
		"session_id": client.SessionID,
		"state":      "idle",
		"message":    "当前没有进行中的游戏",
	}

	if client.SessionID != "" {
		gameState["state"] = "active"
		gameState["message"] = "游戏会话活跃"
	}

	data, _ := json.Marshal(gameState)
	response := &Message{
		Type:      "game_state",
		UserID:    client.UserID,
		SessionID: client.SessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
}

// handleEcho 处理echo消息（用于测试）
func (h *SimpleMessageHandler) handleEcho(client *Client, msg *Message) {
	// 返回原消息，添加echo标记
	echoData := map[string]interface{}{
		"echo":         true,
		"original_type": msg.Type,
		"message":      "Echo: " + msg.Type,
		"timestamp":    time.Now().Unix(),
	}

	if msg.Data != nil {
		var originalData interface{}
		if err := json.Unmarshal(msg.Data, &originalData); err == nil {
			echoData["original_data"] = originalData
		}
	}

	data, _ := json.Marshal(echoData)
	response := &Message{
		Type:      "echo",
		UserID:    client.UserID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
}

// sendError 发送错误消息
func (h *SimpleMessageHandler) sendError(client *Client, message string) {
	errorMsg := &Message{
		Type:      "error",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"error":"%s","timestamp":%d}`, message, time.Now().Unix())),
	}
	h.hub.SendToClient(client.ID, errorMsg)
}