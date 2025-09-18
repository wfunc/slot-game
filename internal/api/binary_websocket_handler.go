package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/wfunc/slot-game/internal/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BinaryWebSocketHandler 处理二进制格式的WebSocket连接（前端协议）
type BinaryWebSocketHandler struct {
	db       *gorm.DB
	router   *ws.BinaryProtocolRouter
	upgrader websocket.Upgrader
	logger   *zap.Logger
}

// NewBinaryWebSocketHandler 创建二进制 WebSocket处理器
func NewBinaryWebSocketHandler(db *gorm.DB, logger *zap.Logger) *BinaryWebSocketHandler {
	return &BinaryWebSocketHandler{
		db:     db,
		router: ws.NewBinaryProtocolRouter(db, logger),
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

// HandleBinaryConnection 处理二进制 WebSocket连接
func (h *BinaryWebSocketHandler) HandleBinaryConnection(c *gin.Context) {
	// 获取客户端信息
	clientIP := c.ClientIP()
	token := c.Query("token")     // 可选的JWT token
	game := c.DefaultQuery("game", "slot")

	h.logger.Info("新的二进制协议 WebSocket连接请求",
		zap.String("ip", clientIP),
		zap.Bool("has_token", token != ""),
		zap.String("game", game))

	// 升级HTTP连接为WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket升级失败",
			zap.String("ip", clientIP),
			zap.Error(err))
		return
	}
	defer conn.Close()

	// 创建协议处理器
	protocol := ws.NewProtocol()
	protocol.Debug = true // 开启调试日志

	// 使用路由器作为消息处理器
	// router 实现了 MessageHandler 接口

	// 创建协议客户端
	clientID := c.ClientIP() + "_" + c.Request.Header.Get("User-Agent")
	protocolClient := ws.NewProtocolClient(clientID, conn, h.router, h.logger)

	// 启动客户端处理
	protocolClient.Start()

	// 等待客户端断开
	<-protocolClient.GetDoneChannel()

	// 清理客户端
	if clientManager := h.router.GetClientManager(); clientManager != nil {
		clientManager.RemoveClient(clientID)
	}

	h.logger.Info("二进制协议客户端断开连接",
		zap.String("client_id", clientID))
}