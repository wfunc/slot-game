package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/pb"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// ConfigHandler 处理配置相关的WebSocket消息
type ConfigHandler struct {
	mu              sync.RWMutex
	db              *gorm.DB
	logger          *zap.Logger
	walletRepo      repository.WalletRepository
	heartbeatTimers map[*websocket.Conn]*time.Timer
	heartbeatMu     sync.RWMutex
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(db *gorm.DB, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{
		db:              db,
		logger:          logger,
		walletRepo:      repository.NewWalletRepository(db),
		heartbeatTimers: make(map[*websocket.Conn]*time.Timer),
	}
}

// HandleMessage 处理配置相关的消息
func (h *ConfigHandler) HandleMessage(conn *websocket.Conn, msgID uint16, data []byte, userID uint) error {
	switch msgID {
	case 2001: // 获取信息
		return h.handleGetInfo(conn, data, userID)
	case 2002: // 捕捉错误
		return h.handleCatchError(conn, data, userID)
	case 2099: // 心跳
		return h.handleHeartbeat(conn, data)
	default:
		h.logger.Warn("[ConfigHandler] 未知消息", zap.Uint16("msg_id", msgID))
		return nil
	}
}

// handleGetInfo 处理获取信息请求 (2001)
func (h *ConfigHandler) handleGetInfo(conn *websocket.Conn, data []byte, userID uint) error {
	// 解析请求 - 支持JSON格式（用于测试）和Protobuf格式
	req := &pb.M_2001Tos{}
	if err := proto.Unmarshal(data, req); err != nil {
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[ConfigHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于2001请求，通常是空的，所以直接继续处理
	}

	// 获取用户钱包信息
	wallet, err := h.walletRepo.GetByUserID(context.Background(), userID)
	if err != nil {
		h.logger.Error("[ConfigHandler] 获取钱包信息失败",
			zap.Uint("user_id", userID),
			zap.Error(err))

		// 如果钱包不存在，创建一个新的
		if err == gorm.ErrRecordNotFound {
			wallet = &models.Wallet{
				UserID:  userID,
				Balance: 0,
				Coins:   10000, // 默认给10000金币
			}
			if err := h.walletRepo.Create(context.Background(), wallet); err != nil {
				h.logger.Error("[ConfigHandler] 创建钱包失败", zap.Error(err))
				return err
			}
		} else {
			return err
		}
	}

	// 构造响应
	resp := &pb.M_2001Toc{
		RoleId:  proto.Uint32(uint32(userID)),
		Balance: proto.Uint64(uint64(wallet.Coins)),
	}

	// 发送响应
	if err := h.sendMessage(conn, 2001, resp); err != nil {
		h.logger.Error("[ConfigHandler] 发送获取信息响应失败", zap.Error(err))
		return err
	}

	h.logger.Info("[ConfigHandler] 发送用户信息",
		zap.Uint("user_id", userID),
		zap.Int64("balance", wallet.Coins))

	return nil
}

// handleCatchError 处理错误捕获 (2002)
func (h *ConfigHandler) handleCatchError(conn *websocket.Conn, data []byte, userID uint) error {
	// 解析请求
	req := &pb.M_2002Tos{}
	if err := proto.Unmarshal(data, req); err != nil {
		h.logger.Error("[ConfigHandler] 解析错误捕获请求失败", zap.Error(err))
		return err
	}

	// 记录客户端错误
	h.logger.Error("[ConfigHandler] 客户端错误报告",
		zap.Uint("user_id", userID),
		zap.String("error", req.GetError()),
		zap.Time("time", time.Now()))

	// TODO: 可以将错误信息存入数据库进行统计分析
	// 这里可以创建一个error_logs表来记录客户端错误

	// 构造响应
	resp := &pb.M_2002Toc{}

	// 发送响应
	if err := h.sendMessage(conn, 2002, resp); err != nil {
		h.logger.Error("[ConfigHandler] 发送错误捕获响应失败", zap.Error(err))
		return err
	}

	return nil
}

// handleHeartbeat 处理心跳 (2099)
func (h *ConfigHandler) handleHeartbeat(conn *websocket.Conn, data []byte) error {
	// 解析请求 - 支持JSON格式（用于测试）和Protobuf格式
	req := &pb.M_2099Tos{}
	if err := proto.Unmarshal(data, req); err != nil {
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[ConfigHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于心跳请求，通常是空的，所以直接继续处理
	}

	// 更新心跳计时器
	h.updateHeartbeatTimer(conn)

	// 构造响应
	resp := &pb.M_2099Toc{}

	// 发送响应
	if err := h.sendMessage(conn, 2099, resp); err != nil {
		h.logger.Error("[ConfigHandler] 发送心跳响应失败", zap.Error(err))
		return err
	}

	h.logger.Debug("[ConfigHandler] 心跳响应已发送")

	return nil
}

// updateHeartbeatTimer 更新心跳计时器
func (h *ConfigHandler) updateHeartbeatTimer(conn *websocket.Conn) {
	h.heartbeatMu.Lock()
	defer h.heartbeatMu.Unlock()

	// 取消旧的计时器
	if timer, exists := h.heartbeatTimers[conn]; exists {
		timer.Stop()
	}

	// 创建新的计时器，30秒没有心跳就断开连接
	h.heartbeatTimers[conn] = time.AfterFunc(30*time.Second, func() {
		h.logger.Warn("[ConfigHandler] 心跳超时，关闭连接")
		conn.Close()
		h.removeHeartbeatTimer(conn)
	})
}

// removeHeartbeatTimer 移除心跳计时器
func (h *ConfigHandler) removeHeartbeatTimer(conn *websocket.Conn) {
	h.heartbeatMu.Lock()
	defer h.heartbeatMu.Unlock()

	if timer, exists := h.heartbeatTimers[conn]; exists {
		timer.Stop()
		delete(h.heartbeatTimers, conn)
	}
}

// OnConnectionClose 连接关闭时的处理
func (h *ConfigHandler) OnConnectionClose(conn *websocket.Conn) {
	h.removeHeartbeatTimer(conn)
}

// sendMessage 发送消息
func (h *ConfigHandler) sendMessage(conn *websocket.Conn, msgID uint16, msg proto.Message) error {
	// 创建编解码器
	codec := NewProtobufCodec()

	// 编码消息
	data, err := codec.Encode(msgID, msg)
	if err != nil {
		return err
	}

	// 设置写超时
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// 发送消息
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return err
	}

	return nil
}

// StartHeartbeatCheck 启动心跳检查（在连接建立时调用）
func (h *ConfigHandler) StartHeartbeatCheck(conn *websocket.Conn) {
	h.updateHeartbeatTimer(conn)
}