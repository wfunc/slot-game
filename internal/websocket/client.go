package websocket

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// 错误定义
var (
	ErrClientNotFound   = errors.New("客户端未找到")
	ErrUserNotConnected = errors.New("用户未连接")
	ErrSessionNotFound  = errors.New("会话未找到")
	ErrSendBufferFull   = errors.New("发送缓冲区已满")
	ErrInvalidMessage   = errors.New("无效的消息格式")
)

// WebSocket配置
const (
	// 写超时
	writeWait = 10 * time.Second

	// 读取pong超时
	pongWait = 60 * time.Second

	// ping发送周期（必须小于pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

// NewClient 创建新客户端
func NewClient(hub *Hub, conn *websocket.Conn, userID uint) *Client {
	return &Client{
		ID:     uuid.New().String(),
		UserID: userID,
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
}

// ReadPump 读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket读取错误",
					zap.String("client_id", c.ID),
					zap.Error(err))
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// WritePump 写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub关闭了通道
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(data []byte) {
	// 如果Hub有消息处理器，使用处理器
	if c.Hub.messageHandler != nil {
		c.Hub.messageHandler.HandleClientMessage(c, data)
		return
	}

	// 否则使用默认处理
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.Hub.logger.Error("解析WebSocket消息失败",
			zap.String("client_id", c.ID),
			zap.Error(err))
		c.sendError("消息格式错误")
		// 断开发送无效JSON的连接
		c.Close()
		return
	}
	
	// 验证消息类型不为空
	if msg.Type == "" {
		c.Hub.logger.Warn("收到空消息类型",
			zap.String("client_id", c.ID))
		c.sendError("消息类型不能为空")
		// 断开连接
		c.Close()
		return
	}

	// 设置消息元数据
	msg.UserID = c.UserID
	msg.Timestamp = time.Now().Unix()

	// 根据消息类型处理
	switch msg.Type {
	case MessageTypePong:
		// 客户端响应ping
		c.Hub.logger.Debug("收到pong",
			zap.String("client_id", c.ID))

	case MessageTypeGameState:
		// 请求游戏状态
		if msg.SessionID != "" {
			c.SessionID = msg.SessionID
		}

	default:
		// 不支持的消息类型
		c.Hub.logger.Warn("收到不支持的消息类型",
			zap.String("client_id", c.ID),
			zap.String("type", msg.Type))
		c.sendError("不支持的消息类型: " + msg.Type)
		// 断开发送无效消息类型的连接
		c.Close()
	}
}

// sendError 发送错误消息
func (c *Client) sendError(message string) {
	errorMsg := &Message{
		Type:      MessageTypeError,
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"error":"` + message + `"}`),
	}
	c.Hub.SendToClient(c.ID, errorMsg)
}

// SendMessage 发送消息给客户端
func (c *Client) SendMessage(msgType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := &Message{
		Type:      msgType,
		UserID:    c.UserID,
		SessionID: c.SessionID,
		Data:      jsonData,
		Timestamp: time.Now().Unix(),
	}

	return c.Hub.SendToClient(c.ID, msg)
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.Hub.unregister <- c
}