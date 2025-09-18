package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/pb"
	ws "github.com/wfunc/slot-game/internal/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProtobufWebSocketHandler 处理protobuf格式的WebSocket连接
type ProtobufWebSocketHandler struct {
	slotHandler     *ws.SlotHandler
	animalHandler   *ws.AnimalHandler
	bridgeHandler   *ws.BridgeHandler
	// unifiedHandler  *ws.UnifiedHandler // 统一处理器 (暂时注释，等待完善)
	upgrader        websocket.Upgrader
	logger          *zap.Logger
}

// NewProtobufWebSocketHandler 创建protobuf WebSocket处理器
func NewProtobufWebSocketHandler(db *gorm.DB, logger *zap.Logger) *ProtobufWebSocketHandler {
	// 创建handlers
	slotHandler := ws.NewSlotHandler(db)
	animalHandler := ws.NewAnimalHandler(db, logger)
	bridgeHandler := ws.NewBridgeHandler(logger, db)

	// 注册游戏处理器到桥接处理器
	bridgeHandler.RegisterGameHandler("slot", slotHandler)
	bridgeHandler.RegisterGameHandler("animal", animalHandler)

	return &ProtobufWebSocketHandler{
		slotHandler:     slotHandler,
		animalHandler:   animalHandler,
		bridgeHandler:   bridgeHandler,
		// unifiedHandler:  ws.NewUnifiedHandler(db, logger), // 创建统一处理器 (暂时注释)
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
	game := c.DefaultQuery("game", "slot")

	h.logger.Info("新的Protobuf WebSocket连接请求",
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

	// 使用统一处理器处理所有游戏 (暂时注释，等待完善)
	// if game == "unified" || game == "all" {
	// 	// 解析初始化参数
	// 	userID := uint(0)
	// 	playerID := uint32(0)
	// 	name := c.Query("name")
	// 	icon := c.Query("icon")
	// 	vip := uint32(0)

	// 	if userIDStr := c.Query("user_id"); userIDStr != "" {
	// 		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
	// 			userID = uint(id)
	// 		}
	// 	}

	// 	if playerIDStr := c.Query("player_id"); playerIDStr != "" {
	// 		if id, err := strconv.ParseUint(playerIDStr, 10, 32); err == nil {
	// 			playerID = uint32(id)
	// 		}
	// 	}

	// 	if vipStr := c.Query("vip"); vipStr != "" {
	// 		if v, err := strconv.ParseUint(vipStr, 10, 32); err == nil {
	// 			vip = uint32(v)
	// 		}
	// 	}

	// 	// 使用统一处理器
	// 	h.unifiedHandler.HandleConnection(conn, userID, playerID, name, icon, vip)
	// 	return
	// }

	// 兼容旧的单独游戏模式
	if game == "animal" {
		opts := parseAnimalInitOptions(c)
		h.animalHandler.HandleConnection(conn, opts)
		return
	}

	// 默认交给slot handler处理
	h.slotHandler.HandleConnection(conn)
}

func parseAnimalInitOptions(c *gin.Context) *ws.AnimalInitOptions {
	opts := &ws.AnimalInitOptions{}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			opts.UserID = uint(id)
		}
	}

	if playerIDStr := c.Query("player_id"); playerIDStr != "" {
		if id, err := strconv.ParseUint(playerIDStr, 10, 32); err == nil {
			opts.PlayerID = uint32(id)
		}
	}

	opts.Name = c.Query("name")
	opts.Icon = c.Query("icon")

	if vipStr := c.Query("vip"); vipStr != "" {
		if v, err := strconv.ParseUint(vipStr, 10, 32); err == nil {
			opts.VIP = uint32(v)
		}
	}

	if zooStr := c.Query("zoo_type"); zooStr != "" {
		if v, err := strconv.ParseUint(zooStr, 10, 32); err == nil {
			opts.ZooType = pb.EZooType(v)
		}
	}

	return opts
}
