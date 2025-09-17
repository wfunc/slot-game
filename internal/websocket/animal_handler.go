package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/game/animal"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/pb"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// AnimalInitOptions 初始化参数
type AnimalInitOptions struct {
	UserID   uint
	PlayerID uint32
	Name     string
	Icon     string
	VIP      uint32
	ZooType  pb.EZooType
}

// AnimalSession 会话结构
type AnimalSession struct {
	ID       string
	UserID   uint
	PlayerID uint32
	Name     string
	Icon     string
	VIP      uint32
	ZooType  pb.EZooType

	Conn  *websocket.Conn
	Codec *ProtobufCodec
}

// AnimalHandler 动物游戏处理器
type AnimalHandler struct {
	mu             sync.RWMutex
	sessions       map[string]*AnimalSession
	playerSessions map[uint32]map[string]*AnimalSession

	db         *gorm.DB
	walletRepo repository.WalletRepository
	manager    *animal.Manager
	logger     *zap.Logger
}

// NewAnimalHandler 创建处理器
func NewAnimalHandler(db *gorm.DB, logger *zap.Logger) *AnimalHandler {
	return &AnimalHandler{
		sessions:       make(map[string]*AnimalSession),
		playerSessions: make(map[uint32]map[string]*AnimalSession),
		db:             db,
		walletRepo:     repository.NewWalletRepository(db),
		manager:        animal.NewManager(),
		logger:         logger,
	}
}

// HandleConnection 处理连接
func (h *AnimalHandler) HandleConnection(conn *websocket.Conn, opts *AnimalInitOptions) {
	sessionID := uuid.New().String()

	userID := uint(0)
	playerID := uint32(0)
	name := ""
	icon := ""
	vip := uint32(0)

	if opts != nil {
		userID = opts.UserID
		playerID = opts.PlayerID
		name = opts.Name
		icon = opts.Icon
		vip = opts.VIP
	}

	if userID == 0 {
		userID = h.getOrCreateTestUser()
	}

	if playerID == 0 {
		playerID = uint32(userID)
	}

	if vip == 0 {
		vip = 1
	}

	zooType := pb.EZooType_civilian
	if opts != nil && opts.ZooType != 0 {
		zooType = opts.ZooType
	}

	session := &AnimalSession{
		ID:       sessionID,
		UserID:   userID,
		PlayerID: playerID,
		Name:     name,
		Icon:     icon,
		VIP:      vip,
		ZooType:  zooType,
		Conn:     conn,
		Codec:    NewProtobufCodec(),
	}

	h.mu.Lock()
	h.sessions[sessionID] = session
	if _, ok := h.playerSessions[playerID]; !ok {
		h.playerSessions[playerID] = make(map[string]*AnimalSession)
	}
	h.playerSessions[playerID][sessionID] = session
	h.mu.Unlock()

	h.logger.Info("[AnimalHandler] 新连接",
		zap.String("session_id", sessionID),
		zap.Uint("user_id", userID),
		zap.Uint32("player_id", playerID))

	h.listen(session)

	h.cleanupSession(session)
}

func (h *AnimalHandler) listen(session *AnimalSession) {
	defer session.Conn.Close()

	for {
		msgType, data, err := session.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Warn("[AnimalHandler] 读取消息失败", zap.Error(err))
			}
			break
		}

		if msgType != websocket.BinaryMessage {
			h.logger.Warn("[AnimalHandler] 收到非二进制消息")
			continue
		}

		msgID, payload, err := session.Codec.Decode(data)
		if err != nil {
			h.logger.Error("[AnimalHandler] 解码失败", zap.Error(err))
			continue
		}

		switch msgID {
		case 1801:
			h.handleEnterRoom(session, payload)
		case 1802:
			h.handleLeaveRoom(session, payload)
		case 1803:
			h.handleBet(session, payload)
		case 1804:
			h.handleGetRecord(session, payload)
		case 1805:
			h.handleGetReward(session, payload)
		case 1806:
			h.handleUseSkill(session, payload)
		case 1807:
			h.handleGetZooInfo(session, payload)
		case 1808:
			h.handleBuyTool(session, payload)
		case 1809:
			h.handleGetToolPrice(session, payload)
		default:
			h.logger.Warn("[AnimalHandler] 未知消息", zap.Uint16("msg_id", msgID))
		}
	}
}

func (h *AnimalHandler) cleanupSession(session *AnimalSession) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sessions, session.ID)
	if sessions, ok := h.playerSessions[session.PlayerID]; ok {
		delete(sessions, session.ID)
		if len(sessions) == 0 {
			delete(h.playerSessions, session.PlayerID)
		}
	}

	h.logger.Info("[AnimalHandler] 连接关闭",
		zap.String("session_id", session.ID),
		zap.Uint32("player_id", session.PlayerID))
}

func (h *AnimalHandler) handleEnterRoom(session *AnimalSession, payload []byte) {
	req := &pb.M_1801Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析进入房间失败", zap.Error(err))
		return
	}

	resp, pushes, err := h.manager.EnterRoom(session.PlayerID, session.Name, session.Icon, session.VIP, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 进入房间失败", zap.Error(err))
		return
	}

	session.ZooType = req.GetType()

	h.sendMessage(session, 1801, resp)
	h.dispatchPushes(session, pushes)
}

func (h *AnimalHandler) handleLeaveRoom(session *AnimalSession, payload []byte) {
	req := &pb.M_1802Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析离开房间失败", zap.Error(err))
		return
	}

	resp, pushes, err := h.manager.LeaveRoom(session.PlayerID, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 离开房间失败", zap.Error(err))
		return
	}

	session.ZooType = 0

	h.sendMessage(session, 1802, resp)
	h.dispatchPushes(session, pushes)
}

func (h *AnimalHandler) handleBet(session *AnimalSession, payload []byte) {
	req := &pb.M_1803Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析下注失败", zap.Error(err))
		return
	}

	resp, pushes, err := h.manager.Bet(session.PlayerID, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 下注失败", zap.Error(err))
		return
	}

	h.sendMessage(session, 1803, resp)
	h.dispatchPushes(session, pushes)
}

func (h *AnimalHandler) handleGetRecord(session *AnimalSession, payload []byte) {
	req := &pb.M_1804Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析记录失败", zap.Error(err))
		return
	}

	resp, err := h.manager.GetRecord(session.PlayerID, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 获取记录失败", zap.Error(err))
		return
	}

	h.sendMessage(session, 1804, resp)
}

func (h *AnimalHandler) handleGetReward(session *AnimalSession, payload []byte) {
	req := &pb.M_1805Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析大奖失败", zap.Error(err))
		return
	}

	resp := h.manager.GetRewards()
	h.sendMessage(session, 1805, resp)
}

func (h *AnimalHandler) handleUseSkill(session *AnimalSession, payload []byte) {
	req := &pb.M_1806Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析技能失败", zap.Error(err))
		return
	}

	resp, pushes, err := h.manager.UseSkill(session.PlayerID, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 使用技能失败", zap.Error(err))
		return
	}

	h.sendMessage(session, 1806, resp)
	h.dispatchPushes(session, pushes)
}

func (h *AnimalHandler) handleGetZooInfo(session *AnimalSession, payload []byte) {
	req := &pb.M_1807Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析获取场信息失败", zap.Error(err))
		return
	}

	resp := h.manager.GetZooTypes()
	h.sendMessage(session, 1807, resp)
}

func (h *AnimalHandler) handleBuyTool(session *AnimalSession, payload []byte) {
	req := &pb.M_1808Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析购买失败", zap.Error(err))
		return
	}

	resp, err := h.manager.BuyTool(session.PlayerID, req)
	if err != nil {
		h.logger.Error("[AnimalHandler] 购买道具失败", zap.Error(err))
		return
	}

	h.sendMessage(session, 1808, resp)
}

func (h *AnimalHandler) handleGetToolPrice(session *AnimalSession, payload []byte) {
	req := &pb.M_1809Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析获取价格失败", zap.Error(err))
		return
	}

	resp := h.manager.GetToolPrice()
	h.sendMessage(session, 1809, resp)
}

func (h *AnimalHandler) sendMessage(session *AnimalSession, msgID uint16, msg proto.Message) {
	data, err := session.Codec.Encode(msgID, msg)
	if err != nil {
		h.logger.Error("[AnimalHandler] 编码失败", zap.Error(err))
		return
	}

	session.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := session.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		h.logger.Error("[AnimalHandler] 发送失败", zap.Error(err))
	}
}

func (h *AnimalHandler) dispatchPushes(origin *AnimalSession, pushes []animal.PushMessage) {
	if len(pushes) == 0 {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, push := range pushes {
		targets := h.resolveTargets(push, origin)
		if len(targets) == 0 {
			continue
		}

		data, err := origin.Codec.Encode(push.MsgID, push.Message)
		if err != nil {
			h.logger.Error("[AnimalHandler] 推送编码失败", zap.Error(err))
			continue
		}

		for _, sess := range targets {
			sess.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := sess.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				h.logger.Warn("[AnimalHandler] 推送失败", zap.Error(err))
			}
		}
	}
}

func (h *AnimalHandler) resolveTargets(push animal.PushMessage, origin *AnimalSession) []*AnimalSession {
	recipients := make([]*AnimalSession, 0)

	if len(push.Targets) == 0 {
		for _, session := range h.sessions {
			if push.ZooType != 0 && session.ZooType != push.ZooType {
				continue
			}
			recipients = append(recipients, session)
		}
		return recipients
	}

	for _, playerID := range push.Targets {
		if sessions, ok := h.playerSessions[playerID]; ok {
			for _, session := range sessions {
				if push.ZooType != 0 && session.ZooType != push.ZooType {
					continue
				}
				recipients = append(recipients, session)
			}
		}
	}

	return recipients
}

func (h *AnimalHandler) getOrCreateTestUser() uint {
	var user models.User
	var wallet models.Wallet

	if err := h.db.Where("username = ?", "test_animal_user").First(&user).Error; err == gorm.ErrRecordNotFound {
		user = models.User{
			Username: "test_animal_user",
			Nickname: "动物玩家",
			Status:   "active",
			Level:    1,
		}
		if err := h.db.Create(&user).Error; err != nil {
			log.Printf("[AnimalHandler] 创建测试用户失败: %v", err)
			return 1
		}

		wallet = models.Wallet{
			UserID:  user.ID,
			Balance: 0,
			Coins:   1000000,
		}
		h.db.Create(&wallet)
	} else {
		if err := h.db.Where("user_id = ?", user.ID).First(&wallet).Error; err == gorm.ErrRecordNotFound {
			wallet = models.Wallet{
				UserID:  user.ID,
				Balance: 0,
				Coins:   1000000,
			}
			h.db.Create(&wallet)
		}
	}

	return user.ID
}
