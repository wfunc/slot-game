package websocket

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(client *ProtocolClient, msg *ClientMessage) (*ServerMessage, error)
}

// ProtocolClient WebSocket协议客户端
type ProtocolClient struct {
	ID       string
	UserID   uint
	conn     *websocket.Conn
	protocol *Protocol
	handler  MessageHandler
	logger   *zap.Logger

	// 发送通道
	send     chan []byte

	// 关闭信号
	done     chan struct{}
	once     sync.Once

	// 统计信息
	lastPing time.Time
	msgCount int64
}

// NewProtocolClient 创建协议客户端
func NewProtocolClient(id string, conn *websocket.Conn, handler MessageHandler, logger *zap.Logger) *ProtocolClient {
	return &ProtocolClient{
		ID:       id,
		conn:     conn,
		protocol: NewProtocol(),
		handler:  handler,
		logger:   logger,
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}
}

// Start 启动客户端处理
func (c *ProtocolClient) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump 读取消息循环
func (c *ProtocolClient) readPump() {
	defer c.Close()

	// 设置读取参数
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		// 读取二进制消息
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket读取错误", zap.String("client_id", c.ID), zap.Error(err))
			}
			return
		}

		// 只处理二进制消息
		if messageType != websocket.BinaryMessage {
			c.logger.Warn("收到非二进制消息", zap.String("client_id", c.ID), zap.Int("type", messageType))
			continue
		}

		// 解码客户端消息
		clientMsg, err := c.protocol.DecodeClientMessage(data)
		if err != nil {
			c.logger.Error("解码客户端消息失败",
				zap.String("client_id", c.ID),
				zap.Error(err),
				zap.Int("data_len", len(data)))

			// 发送错误响应
			errResp := c.protocol.CreateErrorResponse(0, 0, 1001, "消息解码失败")
			c.SendMessage(errResp)
			continue
		}

		// 更新统计
		c.msgCount++
		c.lastPing = time.Now()

		// 记录日志
		c.logger.Info("[1] 收到客户端消息",
			zap.String("client_id", c.ID),
			zap.Uint16("cmd", clientMsg.Cmd),
			zap.Uint32("flag", clientMsg.Flag),
			zap.Int("data_size", len(clientMsg.Data)))

		if len(clientMsg.Data) > 0 {
			c.logger.Info("[1.1] 客户端消息Data内容",
				zap.String("data_hex", fmt.Sprintf("%x", clientMsg.Data)),
				zap.String("data_string", string(clientMsg.Data)))
		}

		// 处理消息
		if c.handler != nil {
			c.logger.Info("[2] 调用消息处理器",
				zap.String("handler_type", fmt.Sprintf("%T", c.handler)))

			response, err := c.handler.HandleMessage(c, clientMsg)
			if err != nil {
				c.logger.Error("[2.1] 处理消息失败",
					zap.String("client_id", c.ID),
					zap.Uint16("cmd", clientMsg.Cmd),
					zap.Error(err))

				// 发送错误响应
				errResp := c.protocol.CreateErrorResponse(clientMsg.Cmd, clientMsg.Flag, 1002, err.Error())
				c.SendMessage(errResp)
			} else if response != nil {
				// 发送响应
				c.logger.Info("[3] 准备发送响应",
					zap.Uint16("resp_cmd", response.Cmd),
					zap.Uint32("resp_flag", response.Flag),
					zap.Uint16("error_id", response.ErrorID),
					zap.Int("data_len", len(response.Data)))

				if len(response.Data) > 0 {
					c.logger.Info("[3.1] 响应Data内容（发送前）",
						zap.String("data_hex", fmt.Sprintf("%x", response.Data)),
						zap.String("data_string", string(response.Data)))
				}

				c.SendMessage(response)
			}
		}
	}
}

// writePump 写入消息循环
func (c *ProtocolClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 发送二进制消息
			c.logger.Info("[8] 通过WebSocket发送二进制数据",
				zap.String("client_id", c.ID),
				zap.Int("message_len", len(message)))

			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				c.logger.Error("[8.1] WebSocket发送失败",
					zap.String("client_id", c.ID),
					zap.Error(err))
				return
			}

			c.logger.Info("[9] 消息发送成功",
				zap.String("client_id", c.ID))

		case <-ticker.C:
			// 发送心跳
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// SendMessage 发送服务端消息
func (c *ProtocolClient) SendMessage(msg *ServerMessage) error {
	// 检查连接状态
	if !c.IsConnected() {
		c.logger.Warn("[4] 客户端已断开，无法发送消息",
			zap.String("client_id", c.ID),
			zap.Uint16("cmd", msg.Cmd))
		return fmt.Errorf("客户端已断开")
	}

	c.logger.Info("[4] 开始编码服务端消息",
		zap.String("client_id", c.ID),
		zap.Uint16("cmd", msg.Cmd),
		zap.Uint32("flag", msg.Flag),
		zap.Uint16("error_id", msg.ErrorID),
		zap.Uint8("data_status", msg.DataStatus),
		zap.Int("data_len", len(msg.Data)))

	// 编码消息
	data, err := c.protocol.EncodeServerMessage(msg)
	if err != nil {
		c.logger.Error("[4.1] 编码服务端消息失败",
			zap.String("client_id", c.ID),
			zap.Error(err))
		return err
	}

	c.logger.Info("[5] 编码后的消息",
		zap.String("client_id", c.ID),
		zap.Int("encoded_len", len(data)),
		zap.String("encoded_hex", fmt.Sprintf("%x", data)))

	// 打印前30字节的详细内容
	if len(data) > 0 {
		maxBytes := len(data)
		if maxBytes > 30 {
			maxBytes = 30
		}
		var bytes []string
		for i := 0; i < maxBytes; i++ {
			bytes = append(bytes, fmt.Sprintf("%d", data[i]))
		}
		c.logger.Info("[5.1] 编码后的字节序列（前30字节）",
			zap.String("bytes", strings.Join(bytes, ",")))
	}

	// 发送到通道
	c.logger.Info("[6] 将消息加入发送队列",
		zap.Int("byte_count", len(data)))

	// 使用defer捕获panic，防止向已关闭的channel发送
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("[6.1] 发送消息时发生panic",
				zap.String("client_id", c.ID),
				zap.Any("panic", r))
		}
	}()

	select {
	case c.send <- data:
		c.logger.Info("[7] 消息已加入发送队列")
		return nil
	case <-time.After(time.Second * 5):
		return fmt.Errorf("发送超时")
	case <-c.done:
		return fmt.Errorf("客户端已关闭")
	}
}

// SendRawMessage 发送原始二进制数据
func (c *ProtocolClient) SendRawMessage(data []byte) error {
	select {
	case c.send <- data:
		return nil
	case <-time.After(time.Second * 5):
		return fmt.Errorf("发送超时")
	case <-c.done:
		return fmt.Errorf("客户端已关闭")
	}
}

// Close 关闭客户端连接
func (c *ProtocolClient) Close() {
	c.once.Do(func() {
		close(c.done)
		close(c.send)
		c.conn.Close()

		c.logger.Info("客户端连接关闭",
			zap.String("client_id", c.ID),
			zap.Int64("msg_count", c.msgCount),
			zap.Duration("duration", time.Since(c.lastPing)))
	})
}

// IsConnected 检查连接状态
func (c *ProtocolClient) IsConnected() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

// GetConn 获取底层WebSocket连接
func (c *ProtocolClient) GetConn() *websocket.Conn {
	return c.conn
}

// SetUserID 设置用户ID
func (c *ProtocolClient) SetUserID(userID uint) {
	c.UserID = userID
}

// GetProtocol 获取协议处理器
func (c *ProtocolClient) GetProtocol() *Protocol {
	return c.protocol
}

// GetDoneChannel 获取done channel
func (c *ProtocolClient) GetDoneChannel() <-chan struct{} {
	return c.done
}