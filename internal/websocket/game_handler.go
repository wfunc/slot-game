package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wfunc/slot-game/internal/game"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GameMessageHandler WebSocket游戏消息处理器
type GameMessageHandler struct {
	hub         *Hub
	db          *gorm.DB
	gameService *game.GameService
	walletRepo  repository.WalletRepository
	logger      *zap.Logger
	
	// 用户会话映射
	userSessions map[uint]string // userID -> sessionID
}

// NewGameMessageHandler 创建游戏消息处理器
func NewGameMessageHandler(hub *Hub, db *gorm.DB, gameService *game.GameService, logger *zap.Logger) *GameMessageHandler {
	return &GameMessageHandler{
		hub:         hub,
		db:          db,
		gameService: gameService,
		walletRepo:  repository.NewWalletRepository(db),
		logger:      logger,
		userSessions: make(map[uint]string),
	}
}

// HandleClientMessage 处理客户端消息
func (h *GameMessageHandler) HandleClientMessage(client *Client, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Error("解析消息失败",
			zap.String("client_id", client.ID),
			zap.Error(err))
		h.sendError(client, "消息格式错误")
		client.Close()
		return
	}
	
	// 验证消息类型不为空
	if msg.Type == "" {
		h.logger.Warn("收到空消息类型",
			zap.String("client_id", client.ID))
		h.sendError(client, "消息类型不能为空")
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
		
	case MessageTypeGameStart:
		h.handleGameStart(client, &msg)
		
	case MessageTypeGameSpin:
		h.handleGameSpin(client, &msg)
		
	case MessageTypeGameSettle:
		h.handleGameSettle(client, &msg)

	default:
		h.logger.Warn("未知消息类型",
			zap.String("client_id", client.ID),
			zap.String("type", msg.Type))
		h.sendError(client, "不支持的消息类型: "+msg.Type)
		client.Close()
	}
}

// handleGameStart 处理游戏开始
func (h *GameMessageHandler) handleGameStart(client *Client, msg *Message) {
	// 解析投注金额
	var data struct {
		BetAmount int64 `json:"bet_amount"`
	}
	
	if msg.Data != nil {
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			h.sendError(client, "投注参数错误")
			return
		}
	}
	
	// 默认投注金额
	if data.BetAmount <= 0 {
		data.BetAmount = 100 // 默认100
	}
	
	// 访客模式模拟游戏
	if client.UserID == 999999 || client.UserID == 0 {
		sessionID := uuid.New().String()
		client.SessionID = sessionID
		h.userSessions[client.UserID] = sessionID
		
		response := &Message{
			Type:      MessageTypeGameStart,
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(fmt.Sprintf(`{"session_id":"%s","bet_amount":%d,"state":"started","message":"游戏已开始（访客模式）","demo":true}`, sessionID, data.BetAmount)),
		}
		h.hub.SendToClient(client.ID, response)
		
		h.logger.Info("访客模式游戏开始",
			zap.String("session_id", sessionID),
			zap.Int64("bet_amount", data.BetAmount))
		return
	}
	
	// 真实用户游戏
	ctx := context.Background()
	
	// 生成会话ID
	sessionID := uuid.New().String()
	
	// 调用游戏服务开始游戏
	if err := h.gameService.StartGame(ctx, client.UserID, sessionID, data.BetAmount); err != nil {
		h.logger.Error("开始游戏失败",
			zap.Uint("user_id", client.UserID),
			zap.Error(err))
		h.sendError(client, fmt.Sprintf("开始游戏失败: %v", err))
		return
	}
	
	// 保存会话信息
	client.SessionID = sessionID
	h.userSessions[client.UserID] = sessionID
	
	// 获取更新后的余额
	wallet, _ := h.walletRepo.GetByUserID(ctx, client.UserID)
	
	// 发送游戏开始响应
	response := &Message{
		Type:      MessageTypeGameStart,
		UserID:    client.UserID,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"session_id":"%s","bet_amount":%d,"balance":%d,"state":"started","message":"游戏已开始"}`, sessionID, data.BetAmount, wallet.Balance)),
	}
	h.hub.SendToClient(client.ID, response)
	
	// 广播余额更新
	h.broadcastBalanceUpdate(client.UserID, wallet.Balance)
}

// handleGameSpin 处理游戏转动
func (h *GameMessageHandler) handleGameSpin(client *Client, msg *Message) {
	// 检查是否有活跃会话
	sessionID := client.SessionID
	if sessionID == "" {
		// 尝试从用户会话映射中获取
		if sid, exists := h.userSessions[client.UserID]; exists {
			sessionID = sid
			client.SessionID = sid
		} else {
			h.sendError(client, "请先开始游戏")
			return
		}
	}
	
	// 访客模式模拟转动
	if client.UserID == 999999 || client.UserID == 0 {
		// 生成随机结果
		symbols := [][]int{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		}
		
		// 随机决定是否中奖
		isWin := time.Now().UnixNano()%3 == 0
		var winAmount int64 = 0
		var winLines []map[string]interface{}
		
		if isWin {
			winAmount = 200 + (time.Now().UnixNano()%10)*100
			winLines = []map[string]interface{}{
				{
					"line": 1,
					"symbol": 5,
					"count": 3,
					"payout": winAmount,
				},
			}
		}
		
		result := map[string]interface{}{
			"symbols":      symbols,
			"win_lines":    winLines,
			"total_payout": winAmount,
			"is_win":       isWin,
			"demo":         true,
		}
		
		resultData, _ := json.Marshal(result)
		response := &Message{
			Type:      MessageTypeGameResult,
			SessionID: sessionID,
			Timestamp: time.Now().Unix(),
			Data:      resultData,
		}
		h.hub.SendToClient(client.ID, response)
		
		h.logger.Info("访客模式转动完成",
			zap.String("session_id", sessionID),
			zap.Bool("is_win", isWin),
			zap.Int64("win_amount", winAmount))
		return
	}
	
	// 真实用户转动
	ctx := context.Background()
	
	// 调用游戏服务执行转动
	spinResp, err := h.gameService.Spin(ctx, sessionID)
	if err != nil {
		h.logger.Error("转动失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
		h.sendError(client, fmt.Sprintf("转动失败: %v", err))
		return
	}
	
	// 获取更新后的余额
	wallet, _ := h.walletRepo.GetByUserID(ctx, client.UserID)
	
	// 构建响应数据
	resultData := map[string]interface{}{
		"result":    spinResp.Result,
		"state":     spinResp.State,
		"total_bet": spinResp.TotalBet,
		"total_win": spinResp.TotalWin,
		"balance":   wallet.Balance,
	}
	
	data, _ := json.Marshal(resultData)
	response := &Message{
		Type:      MessageTypeGameResult,
		UserID:    client.UserID,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
	
	// 如果有中奖，广播余额更新
	if spinResp.Result.TotalPayout > 0 {
		h.broadcastBalanceUpdate(client.UserID, wallet.Balance)
	}
}

// handleGameSettle 处理游戏结算
func (h *GameMessageHandler) handleGameSettle(client *Client, msg *Message) {
	sessionID := client.SessionID
	if sessionID == "" {
		if sid, exists := h.userSessions[client.UserID]; exists {
			sessionID = sid
		} else {
			h.sendError(client, "没有活跃的游戏会话")
			return
		}
	}
	
	// 访客模式
	if client.UserID == 999999 || client.UserID == 0 {
		// 清除会话
		delete(h.userSessions, client.UserID)
		client.SessionID = ""
		
		response := &Message{
			Type:      MessageTypeGameSettle,
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(`{"message":"游戏已结算（访客模式）","demo":true}`),
		}
		h.hub.SendToClient(client.ID, response)
		return
	}
	
	// 真实用户结算
	ctx := context.Background()
	
	// 调用游戏服务结算
	if err := h.gameService.Settle(ctx, sessionID); err != nil {
		h.logger.Error("结算失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
		h.sendError(client, fmt.Sprintf("结算失败: %v", err))
		return
	}
	
	// 获取会话信息
	sessionInfo, _ := h.gameService.GetSessionInfo(ctx, sessionID)
	
	// 获取最终余额
	wallet, _ := h.walletRepo.GetByUserID(ctx, client.UserID)
	
	// 清除会话
	delete(h.userSessions, client.UserID)
	client.SessionID = ""
	
	// 构建结算响应
	settleData := map[string]interface{}{
		"total_bet": sessionInfo.TotalBet,
		"total_win": sessionInfo.TotalWin,
		"rtp":       sessionInfo.RTP,
		"balance":   wallet.Balance,
		"message":   "游戏已结算",
	}
	
	data, _ := json.Marshal(settleData)
	response := &Message{
		Type:      MessageTypeGameSettle,
		UserID:    client.UserID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
	
	// 结束会话
	h.gameService.EndSession(ctx, sessionID)
}

// handlePing 处理ping消息
func (h *GameMessageHandler) handlePing(client *Client) {
	response := &Message{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"message":"pong"}`),
	}
	h.hub.SendToClient(client.ID, response)
}

// handlePong 处理pong消息
func (h *GameMessageHandler) handlePong(client *Client) {
	h.logger.Debug("收到pong",
		zap.String("client_id", client.ID))
}

// handleHeartbeat 处理心跳消息
func (h *GameMessageHandler) handleHeartbeat(client *Client) {
	response := &Message{
		Type:      "heartbeat",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"status":"alive","server_time":%d}`, time.Now().Unix())),
	}
	h.hub.SendToClient(client.ID, response)
}

// handleSubscribe 处理订阅消息
func (h *GameMessageHandler) handleSubscribe(client *Client, msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
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
func (h *GameMessageHandler) handleGetBalance(client *Client) {
	// 访客用户返回模拟数据
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
func (h *GameMessageHandler) handleGetStatus(client *Client) {
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
func (h *GameMessageHandler) handleGetGameState(client *Client) {
	sessionID := client.SessionID
	if sessionID == "" {
		sessionID, _ = h.userSessions[client.UserID]
	}
	
	// 没有活跃会话
	if sessionID == "" {
		gameState := map[string]interface{}{
			"state":   "idle",
			"message": "当前没有进行中的游戏",
		}
		
		data, _ := json.Marshal(gameState)
		response := &Message{
			Type:      MessageTypeGameState,
			UserID:    client.UserID,
			Timestamp: time.Now().Unix(),
			Data:      data,
		}
		h.hub.SendToClient(client.ID, response)
		return
	}
	
	// 访客模式
	if client.UserID == 999999 || client.UserID == 0 {
		gameState := map[string]interface{}{
			"session_id": sessionID,
			"state":      "active",
			"message":    "游戏会话活跃（访客模式）",
			"demo":       true,
		}
		
		data, _ := json.Marshal(gameState)
		response := &Message{
			Type:      MessageTypeGameState,
			UserID:    client.UserID,
			SessionID: sessionID,
			Timestamp: time.Now().Unix(),
			Data:      data,
		}
		h.hub.SendToClient(client.ID, response)
		return
	}
	
	// 获取真实会话信息
	ctx := context.Background()
	sessionInfo, err := h.gameService.GetSessionInfo(ctx, sessionID)
	if err != nil {
		// 会话不存在，清理映射
		delete(h.userSessions, client.UserID)
		client.SessionID = ""
		
		gameState := map[string]interface{}{
			"state":   "idle",
			"message": "游戏会话已过期",
		}
		
		data, _ := json.Marshal(gameState)
		response := &Message{
			Type:      MessageTypeGameState,
			UserID:    client.UserID,
			Timestamp: time.Now().Unix(),
			Data:      data,
		}
		h.hub.SendToClient(client.ID, response)
		return
	}
	
	// 返回详细的游戏状态
	gameState := map[string]interface{}{
		"session_id":  sessionInfo.SessionID,
		"state":       sessionInfo.State,
		"spin_count":  sessionInfo.SpinCount,
		"total_bet":   sessionInfo.TotalBet,
		"total_win":   sessionInfo.TotalWin,
		"rtp":         sessionInfo.RTP,
		"last_result": sessionInfo.LastResult,
		"message":     "游戏进行中",
	}
	
	data, _ := json.Marshal(gameState)
	response := &Message{
		Type:      MessageTypeGameState,
		UserID:    client.UserID,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	h.hub.SendToClient(client.ID, response)
}

// sendError 发送错误消息
func (h *GameMessageHandler) sendError(client *Client, message string) {
	errorMsg := &Message{
		Type:      "error",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"error":"%s","timestamp":%d}`, message, time.Now().Unix())),
	}
	h.hub.SendToClient(client.ID, errorMsg)
}

// broadcastBalanceUpdate 广播余额更新
func (h *GameMessageHandler) broadcastBalanceUpdate(userID uint, balance int64) {
	updateMsg := &Message{
		Type:      MessageTypeBalanceUpdate,
		UserID:    userID,
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(fmt.Sprintf(`{"balance":%d,"currency":"CNY"}`, balance)),
	}
	h.hub.SendToUser(userID, updateMsg)
}