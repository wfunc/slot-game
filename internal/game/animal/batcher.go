package animal

import (
	"fmt"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
)

// SkillType 技能类型（临时定义，解决编译问题）
type SkillType int

// MessageBatcher 消息批量处理器
type MessageBatcher struct {
	// 批次缓存
	batches      map[uint32]*MessageBatch // playerID -> batch
	batchSize    int                       // 批次大小
	batchTimeout time.Duration             // 批次超时时间

	// 发送函数
	sendFunc SendFunction

	// 定时器
	ticker *time.Ticker

	// 并发控制
	mu     sync.RWMutex
	closed chan struct{}
}

// MessageBatch 消息批次
type MessageBatch struct {
	PlayerID  uint32
	Messages  []proto.Message
	StartTime time.Time
	mu        sync.Mutex
}

// SendFunction 发送函数类型
type SendFunction func(playerID uint32, messages []proto.Message)

// NewMessageBatcher 创建消息批量处理器
func NewMessageBatcher(batchSize int, batchTimeout time.Duration, sendFunc SendFunction) *MessageBatcher {
	mb := &MessageBatcher{
		batches:      make(map[uint32]*MessageBatch),
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		sendFunc:     sendFunc,
		ticker:       time.NewTicker(batchTimeout / 2),
		closed:       make(chan struct{}),
	}

	// 启动定时刷新
	go mb.flushLoop()

	return mb
}

// AddMessage 添加消息到批次
func (mb *MessageBatcher) AddMessage(playerID uint32, msg proto.Message) {
	mb.mu.Lock()
	batch, exists := mb.batches[playerID]
	if !exists {
		batch = &MessageBatch{
			PlayerID:  playerID,
			Messages:  make([]proto.Message, 0, mb.batchSize),
			StartTime: time.Now(),
		}
		mb.batches[playerID] = batch
	}
	mb.mu.Unlock()

	batch.mu.Lock()
	batch.Messages = append(batch.Messages, msg)
	shouldFlush := len(batch.Messages) >= mb.batchSize
	batch.mu.Unlock()

	// 达到批次大小，立即发送
	if shouldFlush {
		mb.flushBatch(playerID)
	}
}

// BroadcastMessage 广播消息给多个玩家
func (mb *MessageBatcher) BroadcastMessage(playerIDs []uint32, msg proto.Message) {
	for _, playerID := range playerIDs {
		mb.AddMessage(playerID, msg)
	}
}

// flushBatch 刷新指定玩家的批次
func (mb *MessageBatcher) flushBatch(playerID uint32) {
	mb.mu.Lock()
	batch, exists := mb.batches[playerID]
	if !exists {
		mb.mu.Unlock()
		return
	}
	delete(mb.batches, playerID)
	mb.mu.Unlock()

	batch.mu.Lock()
	messages := batch.Messages
	batch.Messages = nil
	batch.mu.Unlock()

	if len(messages) > 0 {
		mb.sendFunc(playerID, messages)
	}
}

// flushLoop 定时刷新循环
func (mb *MessageBatcher) flushLoop() {
	for {
		select {
		case <-mb.ticker.C:
			mb.flushTimeoutBatches()
		case <-mb.closed:
			mb.ticker.Stop()
			mb.flushAll()
			return
		}
	}
}

// flushTimeoutBatches 刷新超时的批次
func (mb *MessageBatcher) flushTimeoutBatches() {
	now := time.Now()

	mb.mu.RLock()
	playerIDs := make([]uint32, 0, len(mb.batches))
	for playerID, batch := range mb.batches {
		if now.Sub(batch.StartTime) >= mb.batchTimeout {
			playerIDs = append(playerIDs, playerID)
		}
	}
	mb.mu.RUnlock()

	for _, playerID := range playerIDs {
		mb.flushBatch(playerID)
	}
}

// flushAll 刷新所有批次
func (mb *MessageBatcher) flushAll() {
	mb.mu.Lock()
	batches := mb.batches
	mb.batches = make(map[uint32]*MessageBatch)
	mb.mu.Unlock()

	for playerID, batch := range batches {
		if len(batch.Messages) > 0 {
			mb.sendFunc(playerID, batch.Messages)
		}
	}
}

// Close 关闭批量处理器
func (mb *MessageBatcher) Close() {
	close(mb.closed)
}

// AnimalUpdateBatcher 动物更新批量处理器
type AnimalUpdateBatcher struct {
	batcher    *MessageBatcher
	updatePool *sync.Pool
}

// NewAnimalUpdateBatcher 创建动物更新批量处理器
func NewAnimalUpdateBatcher(sendFunc SendFunction) *AnimalUpdateBatcher {
	return &AnimalUpdateBatcher{
		batcher: NewMessageBatcher(10, 50*time.Millisecond, sendFunc),
		updatePool: &sync.Pool{
			New: func() interface{} {
				return &pb.M_1887Toc{
					Animal: make([]*pb.PRoute, 0, 10),
				}
			},
		},
	}
}

// AddAnimalEnter 添加动物进入消息
func (aub *AnimalUpdateBatcher) AddAnimalEnter(playerID uint32, animals []*Animal) {
	msg := aub.updatePool.Get().(*pb.M_1887Toc)
	msg.Animal = msg.Animal[:0]

	for _, animal := range animals {
		msg.Animal = append(msg.Animal, animal.ToProto())
	}

	aub.batcher.AddMessage(playerID, msg)
}

// AddAnimalLeave 添加动物离开消息
func (aub *AnimalUpdateBatcher) AddAnimalLeave(playerID uint32, animalID uint32) {
	msg := &pb.M_1888Toc{
		Id: proto.Uint32(animalID),
	}
	aub.batcher.AddMessage(playerID, msg)
}

// AddAnimalDie 添加动物死亡消息
func (aub *AnimalUpdateBatcher) AddAnimalDie(playerID uint32, roleID uint32, effectType pb.EAnimalType, animals []*pb.PAnimalOne) {
	msg := &pb.M_1884Toc{
		RoleId: proto.Uint32(roleID),
		Type:   effectType.Enum(),
		Ids:    animals,
	}
	aub.batcher.AddMessage(playerID, msg)
}

// Close 关闭批量处理器
func (aub *AnimalUpdateBatcher) Close() {
	aub.batcher.Close()
}

// CombinedMessage 组合消息（用于批量发送）
type CombinedMessage struct {
	Messages []proto.Message
	mu       sync.RWMutex
}

// NewCombinedMessage 创建组合消息
func NewCombinedMessage() *CombinedMessage {
	return &CombinedMessage{
		Messages: make([]proto.Message, 0, 20),
	}
}

// Add 添加消息
func (cm *CombinedMessage) Add(msg proto.Message) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.Messages = append(cm.Messages, msg)
}

// GetAndClear 获取并清空消息
func (cm *CombinedMessage) GetAndClear() []proto.Message {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.Messages) == 0 {
		return nil
	}

	messages := cm.Messages
	cm.Messages = make([]proto.Message, 0, 20)
	return messages
}

// EventBatcher 事件批量处理器
type EventBatcher struct {
	events     []*GameEvent
	maxSize    int
	flushFunc  func([]*GameEvent)
	mu         sync.Mutex
}

// GameEvent 游戏事件
type GameEvent struct {
	Type      string
	PlayerID  uint32
	AnimalID  uint32
	Damage    int32
	Timestamp time.Time
	Data      map[string]interface{}
}

// NewEventBatcher 创建事件批量处理器
func NewEventBatcher(maxSize int, flushFunc func([]*GameEvent)) *EventBatcher {
	return &EventBatcher{
		events:    make([]*GameEvent, 0, maxSize),
		maxSize:   maxSize,
		flushFunc: flushFunc,
	}
}

// AddEvent 添加事件
func (eb *EventBatcher) AddEvent(event *GameEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.events = append(eb.events, event)

	if len(eb.events) >= eb.maxSize {
		eb.flushLocked()
	}
}

// Flush 手动刷新
func (eb *EventBatcher) Flush() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.flushLocked()
}

// flushLocked 刷新事件（需要持有锁）
func (eb *EventBatcher) flushLocked() {
	if len(eb.events) == 0 {
		return
	}

	events := eb.events
	eb.events = make([]*GameEvent, 0, eb.maxSize)

	// 异步处理事件
	go eb.flushFunc(events)
}

// StatsBatcher 统计信息批量处理器
type StatsBatcher struct {
	stats      map[string]*GameStats
	interval   time.Duration
	flushFunc  func(map[string]*GameStats)
	ticker     *time.Ticker
	mu         sync.RWMutex
	closed     chan struct{}
}

// GameStats 游戏统计
type GameStats struct {
	PlayerID     uint32
	BulletsShot  uint32
	AnimalsHit   uint32
	TotalDamage  int64
	TotalWin     int64
	SkillsUsed   map[SkillType]uint32
	LastUpdate   time.Time
}

// NewStatsBatcher 创建统计批量处理器
func NewStatsBatcher(interval time.Duration, flushFunc func(map[string]*GameStats)) *StatsBatcher {
	sb := &StatsBatcher{
		stats:     make(map[string]*GameStats),
		interval:  interval,
		flushFunc: flushFunc,
		ticker:    time.NewTicker(interval),
		closed:    make(chan struct{}),
	}

	go sb.flushLoop()

	return sb
}

// UpdateStats 更新统计
func (sb *StatsBatcher) UpdateStats(playerID uint32, update func(*GameStats)) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	key := fmt.Sprintf("%d", playerID)
	stats, exists := sb.stats[key]
	if !exists {
		stats = &GameStats{
			PlayerID:   playerID,
			SkillsUsed: make(map[SkillType]uint32),
			LastUpdate: time.Now(),
		}
		sb.stats[key] = stats
	}

	update(stats)
	stats.LastUpdate = time.Now()
}

// flushLoop 定时刷新循环
func (sb *StatsBatcher) flushLoop() {
	for {
		select {
		case <-sb.ticker.C:
			sb.flush()
		case <-sb.closed:
			sb.ticker.Stop()
			sb.flush()
			return
		}
	}
}

// flush 刷新统计
func (sb *StatsBatcher) flush() {
	sb.mu.Lock()
	stats := sb.stats
	sb.stats = make(map[string]*GameStats)
	sb.mu.Unlock()

	if len(stats) > 0 {
		sb.flushFunc(stats)
	}
}

// Close 关闭统计批量处理器
func (sb *StatsBatcher) Close() {
	close(sb.closed)
}