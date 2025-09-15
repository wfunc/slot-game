package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/wfunc/slot-game/internal/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProtobufWebSocketHandler 处理protobuf格式的WebSocket连接
type ProtobufWebSocketHandler struct {
	slotHandler *ws.SlotHandler
	upgrader    websocket.Upgrader
	logger      *zap.Logger
}

// NewProtobufWebSocketHandler 创建protobuf WebSocket处理器
func NewProtobufWebSocketHandler(db *gorm.DB, logger *zap.Logger) *ProtobufWebSocketHandler {
	return &ProtobufWebSocketHandler{
		slotHandler: ws.NewSlotHandler(db),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该检查Origin
				// TODO: 实现更严格的CORS检查
				return true
			},
		},
		logger: logger,
	}
}

// HandleProtobufConnection 处理protobuf WebSocket连接
func (h *ProtobufWebSocketHandler) HandleProtobufConnection(c *gin.Context) {
	// 获取客户端信息
	clientIP := c.ClientIP()
	token := c.Query("token") // 可选的JWT token

	h.logger.Info("新的Protobuf WebSocket连接请求",
		zap.String("ip", clientIP),
		zap.Bool("has_token", token != ""))

	// 升级HTTP连接为WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket升级失败",
			zap.String("ip", clientIP),
			zap.Error(err))
		return
	}

	// 交给slot handler处理
	h.slotHandler.HandleConnection(conn)
}