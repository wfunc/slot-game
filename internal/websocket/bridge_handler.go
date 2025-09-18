package websocket

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BridgeSession 游戏桥接会话
type BridgeSession struct {
	Conn        *websocket.Conn
	PlayerID    uint32
	CurrentGame string  // 当前游戏："animal", "slot", etc.
	PreviousGame string // 之前的游戏
	BridgeData  *pb.PBridgeData // 桥接数据
	LastActive  time.Time
}

// BridgeHandler 游戏桥接处理器
type BridgeHandler struct {
	mu             sync.RWMutex
	sessions       map[uint32]*BridgeSession // playerID -> session
	gameHandlers   map[string]interface{}     // 游戏名称 -> handler实例
	logger         *zap.Logger
	db             *gorm.DB
	animalHandler  *AnimalHandler
	slotHandler    *SlotHandler
}

// NewBridgeHandler 创建桥接处理器
func NewBridgeHandler(logger *zap.Logger, db *gorm.DB) *BridgeHandler {
	h := &BridgeHandler{
		sessions:     make(map[uint32]*BridgeSession),
		gameHandlers: make(map[string]interface{}),
		logger:       logger,
		db:           db,
	}

	// 启动会话清理协程
	go h.cleanupSessions()

	return h
}

// RegisterGameHandler 注册游戏处理器
func (h *BridgeHandler) RegisterGameHandler(gameName string, handler interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.gameHandlers[gameName] = handler

	// 特殊处理常用的handler
	switch gameName {
	case "animal":
		if ah, ok := handler.(*AnimalHandler); ok {
			h.animalHandler = ah
		}
	case "slot":
		if sh, ok := handler.(*SlotHandler); ok {
			h.slotHandler = sh
		}
	}

	h.logger.Info("[BridgeHandler] 注册游戏处理器",
		zap.String("game", gameName))
}

// HandleConnection 处理WebSocket连接
func (h *BridgeHandler) HandleConnection(conn *websocket.Conn, playerID uint32) {
	h.mu.Lock()
	session := &BridgeSession{
		Conn:        conn,
		PlayerID:    playerID,
		CurrentGame: "",
		LastActive:  time.Now(),
	}
	h.sessions[playerID] = session
	h.mu.Unlock()

	h.logger.Info("[BridgeHandler] 玩家连接到桥接服务",
		zap.Uint32("player_id", playerID))
}

// DisconnectPlayer 断开玩家连接
func (h *BridgeHandler) DisconnectPlayer(playerID uint32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if session, ok := h.sessions[playerID]; ok {
		delete(h.sessions, playerID)
		h.logger.Info("[BridgeHandler] 玩家断开桥接连接",
			zap.Uint32("player_id", playerID),
			zap.String("current_game", session.CurrentGame))
	}
}

// TriggerGameSwitch 触发游戏切换
func (h *BridgeHandler) TriggerGameSwitch(playerID uint32, fromGame, toGame string,
	switchType pb.ESwitchType, bridgeData *pb.PBridgeData) error {

	h.mu.RLock()
	session, exists := h.sessions[playerID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player session not found: %d", playerID)
	}

	// 发送游戏切换通知
	// TODO: 当定义了具体的protobuf消息类型后，这里应该编码并发送消息
	// 目前仅记录日志
	h.logger.Info("[BridgeHandler] 准备发送游戏切换通知",
		zap.Uint32("player_id", playerID),
		zap.String("from", fromGame),
		zap.String("to", toGame),
		zap.String("switch_type", switchType.String()))

	// 更新会话状态
	h.mu.Lock()
	session.PreviousGame = session.CurrentGame
	session.CurrentGame = toGame
	session.BridgeData = bridgeData
	session.LastActive = time.Now()
	h.mu.Unlock()

	h.logger.Info("[BridgeHandler] 游戏切换通知已发送",
		zap.Uint32("player_id", playerID),
		zap.String("from", fromGame),
		zap.String("to", toGame),
		zap.String("type", switchType.String()))

	// 如果是立即切换，自动处理游戏切换逻辑
	if switchType == pb.ESwitchType_switch_immediate {
		return h.performGameSwitch(playerID, fromGame, toGame, bridgeData)
	}

	return nil
}

// performGameSwitch 执行游戏切换
func (h *BridgeHandler) performGameSwitch(playerID uint32, fromGame, toGame string,
	bridgeData *pb.PBridgeData) error {

	h.logger.Info("[BridgeHandler] 执行游戏切换",
		zap.Uint32("player_id", playerID),
		zap.String("from", fromGame),
		zap.String("to", toGame))

	// 从源游戏退出
	if fromGame != "" {
		if err := h.exitGame(playerID, fromGame); err != nil {
			h.logger.Error("[BridgeHandler] 退出源游戏失败",
				zap.Error(err),
				zap.String("game", fromGame))
		}
	}

	// 进入目标游戏
	if toGame != "" {
		if err := h.enterGame(playerID, toGame, bridgeData); err != nil {
			h.logger.Error("[BridgeHandler] 进入目标游戏失败",
				zap.Error(err),
				zap.String("game", toGame))
			return err
		}
	}

	return nil
}

// exitGame 退出游戏
func (h *BridgeHandler) exitGame(playerID uint32, gameName string) error {
	h.mu.RLock()
	handler, ok := h.gameHandlers[gameName]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("game handler not found: %s", gameName)
	}

	// 根据游戏类型调用相应的退出方法
	switch gameName {
	case "animal":
		if animalHandler, ok := handler.(*AnimalHandler); ok {
			// 通知Animal游戏玩家退出
			animalHandler.DisconnectPlayer(playerID)
		}
	case "slot":
		if slotHandler, ok := handler.(*SlotHandler); ok {
			// 通知Slot游戏玩家退出
			slotHandler.DisconnectPlayer(playerID)
		}
	}

	return nil
}

// enterGame 进入游戏
func (h *BridgeHandler) enterGame(playerID uint32, gameName string, bridgeData *pb.PBridgeData) error {
	h.mu.RLock()
	handler, ok := h.gameHandlers[gameName]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("game handler not found: %s", gameName)
	}

	// 根据游戏类型调用相应的进入方法
	switch gameName {
	case "animal":
		if _, ok := handler.(*AnimalHandler); ok {
			// 创建进入房间的请求
			h.logger.Info("[BridgeHandler] 引导玩家进入Animal游戏",
				zap.Uint32("player_id", playerID))
			// 这里需要调用AnimalHandler的进入房间逻辑
		}
	case "slot":
		if _, ok := handler.(*SlotHandler); ok {
			// 创建进入Slot游戏的请求
			h.logger.Info("[BridgeHandler] 引导玩家进入Slot游戏",
				zap.Uint32("player_id", playerID))
			// 这里需要调用SlotHandler的进入房间逻辑
		}
	}

	return nil
}

// NotifyGameReturn 通知游戏返回
func (h *BridgeHandler) NotifyGameReturn(playerID uint32, fromGame string,
	bonusWin uint64, result *pb.PBridgeResult) error {

	h.mu.RLock()
	session, exists := h.sessions[playerID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player session not found: %d", playerID)
	}

	// 发送游戏返回通知
	// TODO: 当定义了具体的protobuf消息类型后，这里应该编码并发送消息
	// 目前仅记录日志
	h.logger.Info("[BridgeHandler] 准备发送游戏返回通知",
		zap.Uint32("player_id", playerID),
		zap.String("from", fromGame),
		zap.Uint64("bonus_win", bonusWin))

	h.logger.Info("[BridgeHandler] 游戏返回通知已发送",
		zap.Uint32("player_id", playerID),
		zap.String("from", fromGame),
		zap.Uint64("bonus_win", bonusWin))

	// 更新会话状态
	h.mu.Lock()
	session.CurrentGame = session.PreviousGame
	session.PreviousGame = fromGame
	session.LastActive = time.Now()
	h.mu.Unlock()

	return nil
}

// NotifyAnimalTrigger 通知Animal游戏触发
func (h *BridgeHandler) NotifyAnimalTrigger(playerID uint32, triggered bool,
	triggerType string, bridgeData *pb.PBridgeData) error {

	h.mu.RLock()
	session, exists := h.sessions[playerID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player session not found: %d", playerID)
	}

	// 发送Animal触发通知
	// TODO: 当定义了具体的protobuf消息类型后，这里应该编码并发送消息
	// 目前仅记录日志
	h.logger.Info("[BridgeHandler] 准备发送Animal触发通知",
		zap.Uint32("player_id", playerID),
		zap.Bool("triggered", triggered),
		zap.String("trigger_type", triggerType))

	h.logger.Info("[BridgeHandler] Animal触发通知已发送",
		zap.Uint32("player_id", playerID),
		zap.Bool("triggered", triggered),
		zap.String("trigger_type", triggerType))

	// 如果触发了，可能需要自动切换到Animal游戏
	if triggered && h.animalHandler != nil {
		currentGame := session.CurrentGame
		if currentGame != "animal" {
			// 触发游戏切换到Animal
			return h.TriggerGameSwitch(playerID, currentGame, "animal",
				pb.ESwitchType_switch_immediate, bridgeData)
		}
	}

	return nil
}

// CheckAnimalTrigger 检查Animal游戏触发条件
func (h *BridgeHandler) CheckAnimalTrigger(playerID uint32, gameResult interface{}) (*pb.PBridgeData, bool) {
	// 这里实现触发Animal游戏的条件检查逻辑
	// 例如：在Slot游戏中获得特定组合时触发Animal游戏

	// 示例逻辑：
	// 1. 检查游戏结果是否满足触发条件
	// 2. 如果满足，创建桥接数据
	// 3. 返回桥接数据和触发状态

	// 暂时返回不触发
	return nil, false
}

// BroadcastToGame 向特定游戏的所有玩家广播消息
func (h *BridgeHandler) BroadcastToGame(gameName string, msgID uint16, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for playerID, session := range h.sessions {
		if session.CurrentGame == gameName {
			// TODO: 需要使用codec编码消息
			if session.Conn != nil {
				if err := session.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					h.logger.Error("[BridgeHandler] 广播消息失败",
						zap.Error(err),
						zap.Uint32("player_id", playerID))
				}
			}
		}
	}
}

// GetPlayerSession 获取玩家会话
func (h *BridgeHandler) GetPlayerSession(playerID uint32) (*BridgeSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	session, ok := h.sessions[playerID]
	return session, ok
}

// GetPlayersInGame 获取在特定游戏中的玩家列表
func (h *BridgeHandler) GetPlayersInGame(gameName string) []uint32 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var players []uint32
	for playerID, session := range h.sessions {
		if session.CurrentGame == gameName {
			players = append(players, playerID)
		}
	}
	return players
}

// UpdateBridgeData 更新玩家的桥接数据
func (h *BridgeHandler) UpdateBridgeData(playerID uint32, bridgeData *pb.PBridgeData) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, ok := h.sessions[playerID]
	if !ok {
		return fmt.Errorf("player session not found: %d", playerID)
	}

	session.BridgeData = bridgeData
	session.LastActive = time.Now()

	return nil
}

// cleanupSessions 定期清理不活跃的会话
func (h *BridgeHandler) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for playerID, session := range h.sessions {
			// 清理超过30分钟不活跃的会话
			if now.Sub(session.LastActive) > 30*time.Minute {
				delete(h.sessions, playerID)
				h.logger.Info("[BridgeHandler] 清理不活跃会话",
					zap.Uint32("player_id", playerID))
			}
		}
		h.mu.Unlock()
	}
}

// GetStatistics 获取桥接统计信息
func (h *BridgeHandler) GetStatistics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	gameCount := make(map[string]int)
	for _, session := range h.sessions {
		if session.CurrentGame != "" {
			gameCount[session.CurrentGame]++
		}
	}

	return map[string]interface{}{
		"total_sessions": len(h.sessions),
		"games":         gameCount,
		"handlers":      len(h.gameHandlers),
	}
}

// HandleMessage 处理客户端消息（如果需要客户端主动请求切换）
func (h *BridgeHandler) HandleMessage(conn *websocket.Conn, msgID uint16, data []byte, userID uint) error {
	playerID := uint32(userID)

	h.logger.Debug("[BridgeHandler] 收到消息",
		zap.Uint16("msg_id", msgID),
		zap.Uint32("player_id", playerID))

	// 目前bridge主要是服务端推送，客户端请求较少
	// 如果后续需要添加客户端主动切换游戏的功能，可以在这里实现

	switch msgID {
	default:
		h.logger.Warn("[BridgeHandler] 未处理的消息ID",
			zap.Uint16("msg_id", msgID))
		return fmt.Errorf("unhandled message ID: %d", msgID)
	}
}

// CreateBridgeData 创建桥接数据
func CreateBridgeData(freeRounds uint32, multiplier float32, bonusPool uint64,
	triggerType string, triggerPos []uint32) *pb.PBridgeData {

	return &pb.PBridgeData{
		FreeRounds:  &freeRounds,
		Multiplier:  &multiplier,
		BonusPool:   &bonusPool,
		TriggerType: &triggerType,
		TriggerPos:  triggerPos,
	}
}

// CreateBridgeResult 创建桥接结果
func CreateBridgeResult(totalWin uint64, achievements uint32, extraData string) *pb.PBridgeResult {
	return &pb.PBridgeResult{
		TotalWin:     &totalWin,
		Achievements: &achievements,
		ExtraData:    &extraData,
	}
}