package websocket

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wfunc/slot-game/internal/pb"
	"github.com/wfunc/slot-game/internal/repository"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// AnimalHandlerSimple 简化的Animal游戏处理器（用于触发测试）
type AnimalHandlerSimple struct {
	sessions   map[string]*AnimalSessionSimple
	db         *gorm.DB
	walletRepo repository.WalletRepository
	mu         sync.RWMutex
}

// AnimalSessionSimple 简化的Animal游戏会话
type AnimalSessionSimple struct {
	ID         string
	UserID     uint
	Conn       *websocket.Conn
	Codec      *ProtobufCodec
	FreeRounds int
	Multiplier float64
	BonusPool  int64
	TotalWin   int64
	StartTime  time.Time
	mu         sync.RWMutex
}

// NewAnimalHandlerSimple 创建简化的Animal游戏处理器
func NewAnimalHandlerSimple(db *gorm.DB) *AnimalHandlerSimple {
	return &AnimalHandlerSimple{
		sessions:   make(map[string]*AnimalSessionSimple),
		db:         db,
		walletRepo: repository.NewWalletRepository(db),
	}
}

// HandleConnection 处理新连接（从Slot游戏触发）
func (h *AnimalHandlerSimple) HandleConnection(conn *websocket.Conn, bridgeData *pb.PBridgeData) {
	sessionID := uuid.New().String()

	// 创建会话
	session := &AnimalSessionSimple{
		ID:        sessionID,
		UserID:    1, // TODO: 从认证系统获取
		Conn:      conn,
		Codec:     NewProtobufCodec(),
		StartTime: time.Now(),
	}

	// 如果有桥接数据，设置游戏参数
	if bridgeData != nil {
		session.FreeRounds = int(bridgeData.GetFreeRounds())
		session.Multiplier = float64(bridgeData.GetMultiplier())
		session.BonusPool = int64(bridgeData.GetBonusPool())

		log.Printf("[AnimalHandlerSimple] 从Slot触发: FreeRounds=%d, Multiplier=%.2f, BonusPool=%d",
			session.FreeRounds, session.Multiplier, session.BonusPool)
	}

	// 保存会话
	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	// 处理消息循环
	h.handleMessages(session)

	// 清理会话
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
}

// handleMessages 处理消息循环
func (h *AnimalHandlerSimple) handleMessages(session *AnimalSessionSimple) {
	defer session.Conn.Close()

	// 模拟Animal游戏
	log.Printf("[AnimalHandlerSimple] 开始Animal游戏，免费轮数: %d", session.FreeRounds)

	// 模拟游戏进行
	for i := 0; i < session.FreeRounds; i++ {
		// 模拟每轮游戏
		time.Sleep(2 * time.Second)

		// 随机赢分
		winAmount := int64(100 + i*50)
		winAmount = int64(float64(winAmount) * session.Multiplier)

		session.mu.Lock()
		session.TotalWin += winAmount
		session.mu.Unlock()

		log.Printf("[AnimalHandlerSimple] 第%d轮，赢分: %d，总赢分: %d",
			i+1, winAmount, session.TotalWin)
	}

	// 加上奖池
	session.mu.Lock()
	session.TotalWin += session.BonusPool
	finalWin := session.TotalWin
	session.mu.Unlock()

	log.Printf("[AnimalHandlerSimple] Animal游戏结束，总赢分: %d", finalWin)

	// 发送游戏返回通知（返回Slot）
	returnNotify := &pb.M_9902Toc{
		FromGame: proto.String("animal"),
		BonusWin: proto.Uint64(uint64(finalWin)),
		BridgeResult: &pb.PBridgeResult{
			TotalWin:     proto.Uint64(uint64(finalWin)),
			Achievements: proto.Uint32(0),
			ExtraData:    proto.String("{}"),
		},
	}

	if err := h.sendMessage(session, 9902, returnNotify); err != nil {
		log.Printf("[AnimalHandlerSimple] 发送游戏返回通知失败: %v", err)
	}
}

// sendMessage 发送消息
func (h *AnimalHandlerSimple) sendMessage(session *AnimalSessionSimple, msgID uint16, msg proto.Message) error {
	// 序列化消息
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 编码消息（包含消息头）
	var encoded []byte
	encoded = append(encoded, byte(msgID>>8), byte(msgID&0xFF))
	encoded = append(encoded, byte(len(data)>>8), byte(len(data)&0xFF))
	encoded = append(encoded, data...)

	// 发送消息
	session.mu.Lock()
	err = session.Conn.WriteMessage(websocket.BinaryMessage, encoded)
	session.mu.Unlock()

	return err
}