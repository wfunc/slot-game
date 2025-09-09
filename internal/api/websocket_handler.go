package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/middleware"
	ws "github.com/wfunc/slot-game/internal/websocket"
	"go.uber.org/zap"
)

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	hub      *ws.Hub
	upgrader websocket.Upgrader
	logger   *zap.Logger
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(hub *ws.Hub, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该检查Origin
				return true
			},
		},
		logger: logger,
	}
}

// GameWebSocket 游戏WebSocket连接
func (h *WebSocketHandler) GameWebSocket(c *gin.Context) {
	// 获取用户ID（可选）
	userID, exists := middleware.GetUserID(c)
	
	// 如果没有认证，使用访客模式（用于测试）
	if !exists || userID == 0 {
		// 访客用户ID，使用一个特殊的标识
		userID = 999999 // 访客用户ID
		h.logger.Info("WebSocket访客连接", zap.String("ip", c.ClientIP()))
	}

	// 升级为WebSocket连接
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket升级失败",
			zap.Uint("user_id", userID),
			zap.Error(err))
		return
	}

	// 创建客户端
	client := ws.NewClient(h.hub, conn, userID)

	// 获取会话ID（如果有）
	sessionID := c.Query("session_id")
	if sessionID != "" {
		client.SessionID = sessionID
	}

	// 注册客户端
	h.hub.Register(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()

	h.logger.Info("WebSocket连接建立",
		zap.String("client_id", client.ID),
		zap.Uint("user_id", userID),
		zap.String("session_id", sessionID))
}

// SendGameStart 发送游戏开始消息
func (h *WebSocketHandler) SendGameStart(userID uint, sessionID string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	msg := &ws.Message{
		Type:      ws.MessageTypeGameStart,
		UserID:    userID,
		SessionID: sessionID,
		Data:      jsonData,
	}
	h.hub.SendToUser(userID, msg)
}

// SendGameResult 发送游戏结果消息
func (h *WebSocketHandler) SendGameResult(userID uint, sessionID string, result interface{}) {
	jsonData, _ := json.Marshal(result)
	msg := &ws.Message{
		Type:      ws.MessageTypeGameResult,
		UserID:    userID,
		SessionID: sessionID,
		Data:      jsonData,
	}
	h.hub.SendToUser(userID, msg)
}

// SendBalanceUpdate 发送余额更新消息
func (h *WebSocketHandler) SendBalanceUpdate(userID uint, balance int64) {
	data := map[string]int64{"balance": balance}
	jsonData, _ := json.Marshal(data)
	msg := &ws.Message{
		Type:   ws.MessageTypeBalanceUpdate,
		UserID: userID,
		Data:   jsonData,
	}
	h.hub.SendToUser(userID, msg)
}

// SendGameState 发送游戏状态消息
func (h *WebSocketHandler) SendGameState(sessionID string, state interface{}) {
	jsonData, _ := json.Marshal(state)
	msg := &ws.Message{
		Type:      ws.MessageTypeGameState,
		SessionID: sessionID,
		Data:      jsonData,
	}
	h.hub.SendToSession(sessionID, msg)
}

// BroadcastMessage 广播消息
func (h *WebSocketHandler) BroadcastMessage(msgType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	msg := &ws.Message{
		Type: msgType,
		Data: jsonData,
	}
	h.hub.Broadcast(msg)
}

// GetOnlineCount 获取在线人数
func (h *WebSocketHandler) GetOnlineCount(c *gin.Context) {
	count := h.hub.GetOnlineCount()
	users := h.hub.GetOnlineUsers()
	
	c.JSON(http.StatusOK, gin.H{
		"online_count": count,
		"online_users": users,
	})
}