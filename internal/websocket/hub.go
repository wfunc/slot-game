package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Hub WebSocket连接管理中心
type Hub struct {
	// 客户端连接池
	clients    map[string]*Client
	clientsMu  sync.RWMutex

	// 用户ID到客户端的映射
	userClients map[uint][]*Client
	userMu      sync.RWMutex

	// 消息广播通道
	broadcast  chan *Message

	// 注册/注销通道
	register   chan *Client
	unregister chan *Client

	// 日志
	logger *zap.Logger
}

// Client WebSocket客户端
type Client struct {
	ID        string          // 客户端ID
	UserID    uint            // 用户ID
	Hub       *Hub            // Hub引用
	Conn      *websocket.Conn // WebSocket连接
	Send      chan []byte     // 发送通道
	SessionID string          // 游戏会话ID
}

// Message WebSocket消息
type Message struct {
	Type      string          `json:"type"`      // 消息类型
	UserID    uint            `json:"user_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Data      json.RawMessage `json:"data"`      // 消息数据
	Timestamp int64           `json:"timestamp"` // 时间戳
}

// MessageType 消息类型
const (
	// 系统消息
	MessageTypeConnected    = "connected"
	MessageTypeDisconnected = "disconnected"
	MessageTypePing         = "ping"
	MessageTypePong         = "pong"
	MessageTypeError        = "error"

	// 游戏消息
	MessageTypeGameStart    = "game_start"
	MessageTypeGameSpin     = "game_spin"
	MessageTypeGameResult   = "game_result"
	MessageTypeGameSettle   = "game_settle"
	MessageTypeGameState    = "game_state"

	// 钱包消息
	MessageTypeBalanceUpdate = "balance_update"
	MessageTypeTransaction   = "transaction"
)

// NewHub 创建Hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		userClients: make(map[uint][]*Client),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		logger:      logger,
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	// 启动心跳检测
	go h.runHeartbeat()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.clientsMu.Lock()
	h.clients[client.ID] = client
	h.clientsMu.Unlock()

	// 添加到用户客户端映射
	if client.UserID > 0 {
		h.userMu.Lock()
		h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
		h.userMu.Unlock()
	}

	h.logger.Info("WebSocket客户端连接",
		zap.String("client_id", client.ID),
		zap.Uint("user_id", client.UserID))

	// 发送连接成功消息
	msg := &Message{
		Type:      MessageTypeConnected,
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"message":"连接成功"}`),
	}
	h.SendToClient(client.ID, msg)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMu.Lock()
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
	}
	h.clientsMu.Unlock()

	// 从用户客户端映射中移除
	if client.UserID > 0 {
		h.userMu.Lock()
		clients := h.userClients[client.UserID]
		for i, c := range clients {
			if c.ID == client.ID {
				h.userClients[client.UserID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		if len(h.userClients[client.UserID]) == 0 {
			delete(h.userClients, client.UserID)
		}
		h.userMu.Unlock()
	}

	h.logger.Info("WebSocket客户端断开",
		zap.String("client_id", client.ID),
		zap.Uint("user_id", client.UserID))
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	h.clientsMu.RLock()
	for _, client := range h.clients {
		select {
		case client.Send <- data:
		default:
			// 发送缓冲区满，关闭连接
			h.logger.Warn("客户端发送缓冲区满",
				zap.String("client_id", client.ID))
		}
	}
	h.clientsMu.RUnlock()
}

// SendToClient 发送消息给指定客户端
func (h *Hub) SendToClient(clientID string, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.clientsMu.RLock()
	client, ok := h.clients[clientID]
	h.clientsMu.RUnlock()

	if !ok {
		return ErrClientNotFound
	}

	select {
	case client.Send <- data:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// SendToUser 发送消息给指定用户的所有客户端
func (h *Hub) SendToUser(userID uint, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.userMu.RLock()
	clients := h.userClients[userID]
	h.userMu.RUnlock()

	if len(clients) == 0 {
		return ErrUserNotConnected
	}

	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			h.logger.Warn("用户客户端发送缓冲区满",
				zap.String("client_id", client.ID),
				zap.Uint("user_id", userID))
		}
	}

	return nil
}

// SendToSession 发送消息给指定游戏会话的所有客户端
func (h *Hub) SendToSession(sessionID string, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	sent := false
	for _, client := range h.clients {
		if client.SessionID == sessionID {
			select {
			case client.Send <- data:
				sent = true
			default:
				h.logger.Warn("会话客户端发送缓冲区满",
					zap.String("client_id", client.ID),
					zap.String("session_id", sessionID))
			}
		}
	}

	if !sent {
		return ErrSessionNotFound
	}
	return nil
}

// GetOnlineUsers 获取在线用户列表
func (h *Hub) GetOnlineUsers() []uint {
	h.userMu.RLock()
	defer h.userMu.RUnlock()

	users := make([]uint, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// GetOnlineCount 获取在线人数
func (h *Hub) GetOnlineCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// runHeartbeat 运行心跳检测
func (h *Hub) runHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		ping := &Message{
			Type:      MessageTypePing,
			Timestamp: time.Now().Unix(),
		}
		h.broadcast <- ping
	}
}

// Broadcast 广播消息（公开方法）
func (h *Hub) Broadcast(message *Message) {
	h.broadcast <- message
}

// Register 注册客户端（公开方法）
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端（公开方法）
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}