package websocket

import (
	"context"
	"fmt"
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
	RoomID   uint32      // 玩家所在房间ID

	Conn  *websocket.Conn
	Codec *ProtobufCodec
}

// AnimalHandler 动物游戏处理器
type AnimalHandler struct {
	mu             sync.RWMutex
	sessions       map[string]*AnimalSession
	playerSessions map[uint32]map[string]*AnimalSession

	db           *gorm.DB
	walletRepo   repository.WalletRepository
	manager      *animal.Manager
	logger       *zap.Logger
	configHandler *ConfigHandler // 添加配置处理器

	// 动态房间管理系统
	animalRooms    map[uint32]*animal.AnimalRoom        // roomID -> AnimalRoom
	roomsByType    map[pb.EZooType][]uint32             // zooType -> roomID list
	nextRoomID     uint32                              // 下一个房间ID
}

// NewAnimalHandler 创建处理器
func NewAnimalHandler(db *gorm.DB, logger *zap.Logger) *AnimalHandler {
	h := &AnimalHandler{
		sessions:       make(map[string]*AnimalSession),
		playerSessions: make(map[uint32]map[string]*AnimalSession),
		db:             db,
		walletRepo:     repository.NewWalletRepository(db),
		manager:        animal.NewManager(),
		logger:         logger,
		configHandler:  NewConfigHandler(db, logger), // 初始化配置处理器
		animalRooms:    make(map[uint32]*animal.AnimalRoom),
		roomsByType:    make(map[pb.EZooType][]uint32),
		nextRoomID:     1,
	}

	// 初始化动物房间系统
	h.initializeAnimalRooms()

	return h
}

// initializeAnimalRooms 初始化动物房间
func (h *AnimalHandler) initializeAnimalRooms() {
	// 只创建1个默认房间（civilian类型）
	defaultType := pb.EZooType_civilian
	roomID := h.nextRoomID
	h.nextRoomID++
	
	// 创建推送回调函数
	pushCallback := func(msg *animal.PushMessage) {
		// 设置房间ID
		msg.RoomID = roomID
		h.broadcastToRoom(msg)
	}
	
	room := animal.NewAnimalRoom(roomID, defaultType, h.logger, pushCallback)
	room.Start()
	
	h.animalRooms[roomID] = room
	h.roomsByType[defaultType] = []uint32{roomID}

	h.logger.Info("[AnimalHandler] 初始房间创建完成",
		zap.Uint32("room_id", roomID),
		zap.String("room_type", defaultType.String()))
}

// findOrCreateRoom 查找或创建动态房间
func (h *AnimalHandler) findOrCreateRoom(zooType pb.EZooType) (*animal.AnimalRoom, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	const MAX_PLAYERS_PER_ROOM = 4
	
	// 查找该类型的所有房间
	roomIDs, exists := h.roomsByType[zooType]
	if !exists || len(roomIDs) == 0 {
		// 该类型还没有房间，创建第一个
		return h.createRoom(zooType), nil
	}
	
	// 查找第一个未满员的房间
	for _, roomID := range roomIDs {
		room, ok := h.animalRooms[roomID]
		if !ok {
			continue
		}
		// 获取房间当前玩家数
		currentPlayers := room.GetPlayerCount()
		if currentPlayers < MAX_PLAYERS_PER_ROOM {
			h.logger.Info("[AnimalHandler] 找到空闲房间",
				zap.Uint32("room_id", roomID),
				zap.Uint32("current_players", currentPlayers),
				zap.String("room_type", zooType.String()))
			return room, nil
		}
	}
	
	// 所有房间都满员，创建新房间
	h.logger.Info("[AnimalHandler] 所有房间已满，创建新房间",
		zap.String("room_type", zooType.String()))
	return h.createRoom(zooType), nil
}

// createRoom 创建新的动物房间
func (h *AnimalHandler) createRoom(zooType pb.EZooType) *animal.AnimalRoom {
	roomID := h.nextRoomID
	h.nextRoomID++
	
	// 推送回调函数
	pushCallback := func(msg *animal.PushMessage) {
		// 设置房间ID
		msg.RoomID = roomID
		h.broadcastToRoom(msg)
	}
	
	room := animal.NewAnimalRoom(roomID, zooType, h.logger, pushCallback)
	room.Start()
	
	// 添加到管理器
	h.animalRooms[roomID] = room
	h.roomsByType[zooType] = append(h.roomsByType[zooType], roomID)
	
	h.logger.Info("[AnimalHandler] 创建新房间",
		zap.Uint32("room_id", roomID),
		zap.String("room_type", zooType.String()))
	
	return room
}

// broadcastToRoom 向房间广播消息
func (h *AnimalHandler) broadcastToRoom(msg *animal.PushMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, session := range h.sessions {
		// 检查是否是同一个房间的玩家
		if session.RoomID == msg.RoomID {
			// 如果有指定目标玩家，只发送给指定玩家
			if len(msg.Targets) > 0 {
				found := false
				for _, targetID := range msg.Targets {
					if session.PlayerID == targetID {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// 编码并发送消息
			data, err := session.Codec.Encode(msg.MsgID, msg.Message)
			if err != nil {
				h.logger.Error("[AnimalHandler] 编码广播消息失败",
					zap.Error(err),
					zap.Uint16("msg_id", msg.MsgID))
				continue
			}

			if err := session.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				h.logger.Error("[AnimalHandler] 发送广播消息失败",
					zap.Error(err),
					zap.String("session_id", session.ID))
			}
		}
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

	// 启动心跳检测
	h.configHandler.StartHeartbeatCheck(conn)

	h.listen(session)

	// 清理心跳计时器
	h.configHandler.OnConnectionClose(conn)
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
		case 1812:
			h.handleGetJackpotHistory(session, payload)
		case 1815:
			h.handleFireBullet(session, payload)
		// Config相关协议
		case 2001, 2002, 2099:
			h.configHandler.HandleMessage(session.Conn, msgID, payload, session.UserID)
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
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[AnimalHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于1801请求，通常是空的，所以直接继续处理
	}

	// 确定房间类型
	roomType := req.GetType()
	if roomType == 0 {
		roomType = pb.EZooType_free // 默认体验场
	}

	// 使用动态房间管理系统
	room, err := h.findOrCreateRoom(roomType)
	if err != nil {
		h.logger.Error("[AnimalHandler] 查找/创建房间失败", 
			zap.String("room_type", roomType.String()),
			zap.Error(err))
		return
	}

	// 调用新的动物房间进入逻辑
	resp, err := room.EnterRoom(session.PlayerID, session.Name, session.Icon)
	if err != nil {
		h.logger.Error("[AnimalHandler] 进入动物房间失败", zap.Error(err))
		return
	}

	// 更新会话房间信息
	session.ZooType = roomType
	session.RoomID = room.GetRoomID()

	h.sendMessage(session, 1801, resp)

	h.logger.Info("[AnimalHandler] 玩家进入动物房间",
		zap.Uint32("player_id", session.PlayerID),
		zap.String("room_type", roomType.String()),
		zap.Int("animals_count", len(resp.Animals)))
}

func (h *AnimalHandler) handleLeaveRoom(session *AnimalSession, payload []byte) {
	req := &pb.M_1802Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Error("[AnimalHandler] 解析离开房间失败", zap.Error(err))
		return
	}

	// 使用动态房间管理系统
	if session.RoomID != 0 {
		h.mu.RLock()
		room, exists := h.animalRooms[session.RoomID]
		h.mu.RUnlock()
		if exists {
			room.LeaveRoom(session.PlayerID)
			h.logger.Info("[AnimalHandler] 玩家离开房间",
				zap.Uint32("player_id", session.PlayerID),
				zap.Uint32("room_id", session.RoomID),
				zap.Uint32("current_players", room.GetPlayerCount()))
		}
	}

	// 重置会话房间信息
	session.ZooType = 0
	session.RoomID = 0

	// 返回简单的确认响应
	resp := &pb.M_1802Toc{}
	h.sendMessage(session, 1802, resp)

	h.logger.Info("[AnimalHandler] 玩家离开动物房间",
		zap.Uint32("player_id", session.PlayerID))
}

func (h *AnimalHandler) handleBet(session *AnimalSession, payload []byte) {
	req := &pb.M_1803Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[AnimalHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于1803请求，通常是空的或包含简单下注信息，继续处理
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

func (h *AnimalHandler) handleGetJackpotHistory(session *AnimalSession, payload []byte) {
	req := &pb.M_1812Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[AnimalHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于1812请求，通常是空的，所以直接继续处理
	}

	// 获取彩金历史（这里暂时返回空列表，需要后续实现完整的彩金系统）
	resp := &pb.M_1812Toc{
		List: []*pb.PCjLog{},
	}

	h.sendMessage(session, 1812, resp)

	h.logger.Info("[AnimalHandler] 返回彩金历史",
		zap.String("session_id", session.ID))
}

func (h *AnimalHandler) handleFireBullet(session *AnimalSession, payload []byte) {
	req := &pb.M_1815Tos{}
	if err := proto.Unmarshal(payload, req); err != nil {
		// 尝试作为JSON处理（测试客户端使用）
		h.logger.Debug("[AnimalHandler] Protobuf解析失败，尝试JSON格式", zap.Error(err))
		// 对于1815请求，默认使用最小下注值
		req.BetVal = proto.Uint32(1)
	}

	// 获取下注金额
	betVal := req.GetBetVal()

	// 扣除金币
	wallet, err := h.walletRepo.GetByUserID(context.Background(), session.UserID)
	if err != nil {
		h.logger.Error("[AnimalHandler] 获取钱包失败", zap.Error(err))
		return
	}

	if wallet.Coins < int64(betVal) {
		h.logger.Warn("[AnimalHandler] 余额不足",
			zap.Uint("user_id", session.UserID),
			zap.Uint32("bet_val", betVal),
			zap.Int64("balance", wallet.Coins))
		return
	}

	// 扣除金币
	if err := h.walletRepo.DeductBalance(context.Background(), session.UserID, int64(betVal)); err != nil {
		h.logger.Error("[AnimalHandler] 扣除金币失败", zap.Error(err))
		return
	}
	// 更新本地钱包余额以便返回正确的余额
	wallet.Coins -= int64(betVal)

	// 生成子弹ID
	bulletID := uuid.New().String()

	// 构造响应
	resp := &pb.M_1815Toc{
		BulletId: proto.String(bulletID),
		Balance:  proto.Uint64(uint64(wallet.Coins)),
	}

	h.sendMessage(session, 1815, resp)

	h.logger.Info("[AnimalHandler] 发射子弹",
		zap.String("bullet_id", bulletID),
		zap.Uint32("bet_val", betVal),
		zap.Int64("balance", wallet.Coins))
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
		// 生成唯一的phone和email (基于当前时间戳)
		timestamp := time.Now().Unix()
		phone := fmt.Sprintf("test_%d", timestamp)
		email := fmt.Sprintf("test_%d@example.com", timestamp)

		user = models.User{
			Username: "test_animal_user",
			Nickname: "动物玩家",
			Phone:    phone,
			Email:    email,
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
