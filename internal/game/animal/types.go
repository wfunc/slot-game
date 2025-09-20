package animal

import (
	"math/rand"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// PushMessage 表示需要发送给客户端的主动推送
// Targets 为空表示同房间广播，否则仅发送给指定玩家ID
// ZooType 用于区分房间或活动场景
// Message 需要发送的protobuf消息
// MsgID 对应animal.proto中定义的消息编号

type PushMessage struct {
	MsgID   uint16
	ZooType pb.EZooType
	RoomID  uint32      // 房间ID，用于动态房间管理
	Targets []uint32
	Message proto.Message
}

// Manager 负责管理所有动物房间和活动场

const (
	// MAX_PLAYERS_PER_ROOM 每个房间最大玩家数
	MAX_PLAYERS_PER_ROOM = 4
)

type Manager struct {
	mu sync.RWMutex

	// 动态房间管理
	rooms        map[uint32]*Room              // roomID -> Room
	roomsByType  map[pb.EZooType][]uint32      // zooType -> roomID list
	nextRoomID   uint32                        // 下一个房间ID
	
	players      map[uint32]*Player
	rewards      []*pb.PAnimalReward
	rewardCursor uint32
	rand         *rand.Rand
}

// RoomConfig 房间类型配置
type RoomConfig struct {
	BetValues   []uint32 // 下注档位
	MinVIP      uint32   // 最低VIP等级
	MaxPlayer   uint32   // 最大玩家数
	UseFreeGold bool     // 是否使用体验币
	OddsType    string   // 赔率类型: "normal" 或 "free"
}

// RoomProfitControl 房间盈亏控制
type RoomProfitControl struct {
	mu       sync.RWMutex
	TotalBet uint64 // P值：总投注
	TotalWin uint64 // W值：总赢取
}

// Room 表示一个动物房间实例
type Room struct {
	ID            uint32      // 房间唯一ID
	Type          pb.EZooType
	Config        *RoomConfig
	BetValues     []uint32
	MaxPlayer     uint32
	MinVIP        uint32
	CurrentPlayers uint32      // 当前玩家数

	animals      map[uint32]*AnimalRoute
	nextAnimalID uint32

	players map[uint32]*PlayerSession

	// 房间盈亏控制
	profitControl *RoomProfitControl

	// 彩金池
	jackpot *JackpotPool

	// 任务管理器
	taskManager *TaskManager

	// 一击必杀管理器
	oneBlowManager *OneBlowManager

	// 技能状态
	iceTime     time.Time // 全场冰冻结束时间
	skillStates map[pb.EAnimalSkillType]time.Time

	redBag  bool
}

// AnimalRoute 房间中动物当前状态

type AnimalRoute struct {
	ID       uint32
	Animal   pb.EAnimal
	LineID   uint32
	Point    uint32
	Red      bool
	State    pb.EAnimalState
	SpawnAt  time.Time
}

// PlayerSession 玩家在房间内的实时状态

type PlayerSession struct {
	Player     *Player
	ZooType    pb.EZooType
	EnteredAt  time.Time
	Skills     map[pb.EAnimalSkillType]*PlayerSkill
	SkillEnds  map[pb.EAnimalSkillType]time.Time
	CurrentBet uint32
	TotalWin   uint64
	Seat       uint32  // 座位号 1-4
	ClientID   string  // WebSocket客户端ID
}

// Player 玩家信息（跨房间共享）

type Player struct {
	ID       uint32
	Name     string
	Icon     string
	VIP      uint32
	Balance  uint64
	FreeGold uint64

	History []*pb.PPlayerAnimal
	Skills  map[pb.EAnimalSkillType]*PlayerSkill
}

// PlayerSkill 玩家技能库存及状态

type PlayerSkill struct {
	Type  pb.EAnimalSkillType
	Value uint32
	Count uint32
	Time  uint32
}

// BetOutcome 投注结果
type BetOutcome struct {
	WinAmount    uint32
	RedBag       uint32
	GoldAmount   uint32              // 实际获得的金豆（包含红包转换）
	SkillGain    []*pb.PAnimalSkill
	FreeGold     uint64
	KilledRoutes []*AnimalRoute
	EffectType   pb.EAnimalType     // 击杀效果类型
	ChainKills   []uint32           // 连锁击杀的动物ID
	JackpotWin   uint64             // 彩金中奖金额
}

// JackpotPool 彩金池系统
type JackpotPool struct {
	mu            sync.RWMutex
	CurrentAmount uint64                  // 当前彩金池金额
	Contributors  map[uint32]uint64       // playerID -> 贡献金额
	TriggerRatio  float32                 // 触发概率
	MinAmount     uint64                  // 最小触发金额
	AccumulateRate float32                // 积累比例
	LastTriggerTime time.Time             // 上次触发时间
	TriggerHistory []JackpotTriggerRecord // 触发历史
}

// JackpotTriggerRecord 彩金触发记录
type JackpotTriggerRecord struct {
	PlayerID  uint32
	Amount    uint64
	Timestamp time.Time
}

// NewJackpotPool 创建彩金池
func NewJackpotPool() *JackpotPool {
	return &JackpotPool{
		CurrentAmount:   5000000,  // 初始金额500万
		Contributors:    make(map[uint32]uint64),
		TriggerRatio:    0.001,    // 0.1% 触发概率
		MinAmount:       10000000, // 最小触发金额1000万
		AccumulateRate:  0.02,     // 2% 积累比例
		LastTriggerTime: time.Now(),
		TriggerHistory:  make([]JackpotTriggerRecord, 0),
	}
}

// Accumulate 积累彩金
func (j *JackpotPool) Accumulate(betAmount uint64, playerID uint32) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// 按比例积累到彩金池
	accumulate := uint64(float32(betAmount) * j.AccumulateRate)
	j.CurrentAmount += accumulate

	// 记录贡献者
	if _, exists := j.Contributors[playerID]; !exists {
		j.Contributors[playerID] = 0
	}
	j.Contributors[playerID] += accumulate
}

// TryTrigger 尝试触发彩金
func (j *JackpotPool) TryTrigger(playerID uint32, betAmount uint64) (bool, uint64) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// 检查是否满足最小触发金额
	if j.CurrentAmount < j.MinAmount {
		return false, 0
	}

	// 检查触发概率
	if rand.Float32() > j.TriggerRatio {
		return false, 0
	}

	// 触发彩金！
	winAmount := j.CurrentAmount

	// 记录触发历史
	record := JackpotTriggerRecord{
		PlayerID:  playerID,
		Amount:    winAmount,
		Timestamp: time.Now(),
	}
	j.TriggerHistory = append(j.TriggerHistory, record)

	// 重置彩金池（保留10%作为种子）
	j.CurrentAmount = winAmount / 10
	j.LastTriggerTime = time.Now()

	// 清空贡献者记录
	j.Contributors = make(map[uint32]uint64)

	return true, winAmount
}

// GetCurrentAmount 获取当前彩金池金额
func (j *JackpotPool) GetCurrentAmount() uint64 {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.CurrentAmount
}

// GetRecentWinners 获取最近的中奖记录
func (j *JackpotPool) GetRecentWinners(limit int) []JackpotTriggerRecord {
	j.mu.RLock()
	defer j.mu.RUnlock()

	start := len(j.TriggerHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]JackpotTriggerRecord, 0, limit)
	for i := start; i < len(j.TriggerHistory); i++ {
		result = append(result, j.TriggerHistory[i])
	}
	return result
}

// ZooTask 动物园任务
type ZooTask struct {
	ID          uint32
	Type        string    // "daily", "weekly", "achievement", "free"
	Target      uint32    // 目标数量
	Progress    uint32    // 当前进度
	Reward      uint64    // 奖励金额
	Status      string    // "active", "completed", "claimed"
	Description string    // 任务描述
	Animal      pb.EAnimal // 指定动物类型（如果有）
	BetLevel    uint32    // 指定下注档位（如果有）
}

// TaskManager 任务管理器
type TaskManager struct {
	mu           sync.RWMutex
	dailyTasks   map[uint32]*ZooTask
	weeklyTasks  map[uint32]*ZooTask
	freeTasks    map[uint32]*ZooTask // 体验场任务
	achievements map[uint32]*ZooTask
	lastReset    time.Time
}

// Activity 活动系统
type Activity struct {
	ID          uint32
	PreTime     uint32         // 准备时间（秒）
	WorkTime    uint32         // 活动时间（秒）
	SendGold    uint32         // 活动赠送金豆
	MaxPlayers  uint32         // 最大参与人数
	BetValues   []uint32       // 下注档位
	Percent     uint32         // 打动物的概率 1000分比
	Status      string         // "prepare", "active", "settlement", "idle"
	Rankings    []*PlayerRank  // 排行榜
	Rewards     []string       // 奖励配置
	StartTime   time.Time
	EndTime     time.Time
	AgentID     uint32         // 渠道ID
}

// PlayerRank 玩家排行
type PlayerRank struct {
	Rank     uint32
	PlayerID uint32
	Name     string
	Icon     string
	Value    uint64 // 排行值（下注总额或赢取总额）
	VIP      uint32
}

// ActivityManager 活动管理器
type ActivityManager struct {
	mu              sync.RWMutex
	currentActivity *Activity
	history         []*Activity
	scheduler       *time.Timer
	logger          *zap.Logger
}

// OneBlow 一击必杀设置
type OneBlow struct {
	PlayerID  uint32
	ExpiresAt time.Time
	Active    bool
	Remaining int // 剩余次数
}

// OneBlowManager 一击必杀管理器
type OneBlowManager struct {
	mu       sync.RWMutex
	settings map[uint32]*OneBlow // playerID -> OneBlow
}

// NewOneBlowManager 创建一击必杀管理器
func NewOneBlowManager() *OneBlowManager {
	return &OneBlowManager{
		settings: make(map[uint32]*OneBlow),
	}
}

// EnableOneBlow 为玩家启用一击必杀
func (m *OneBlowManager) EnableOneBlow(playerID uint32, duration time.Duration, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settings[playerID] = &OneBlow{
		PlayerID:  playerID,
		ExpiresAt: time.Now().Add(duration),
		Active:    true,
		Remaining: count,
	}
}

// DisableOneBlow 禁用玩家的一击必杀
func (m *OneBlowManager) DisableOneBlow(playerID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.settings, playerID)
}

// CheckOneBlow 检查玩家是否有一击必杀权限
func (m *OneBlowManager) CheckOneBlow(playerID uint32) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	setting, exists := m.settings[playerID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(setting.ExpiresAt) {
		return false
	}

	// 检查是否还有次数
	return setting.Active && setting.Remaining > 0
}

// ConsumeOneBlow 消耗一次一击必杀
func (m *OneBlowManager) ConsumeOneBlow(playerID uint32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	setting, exists := m.settings[playerID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(setting.ExpiresAt) {
		delete(m.settings, playerID)
		return false
	}

	// 检查是否还有次数
	if !setting.Active || setting.Remaining <= 0 {
		return false
	}

	// 消耗一次
	setting.Remaining--
	if setting.Remaining <= 0 {
		delete(m.settings, playerID)
	}

	return true
}

// GetOneBlowStatus 获取一击必杀状态
func (m *OneBlowManager) GetOneBlowStatus(playerID uint32) (active bool, remaining int, expiresIn time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	setting, exists := m.settings[playerID]
	if !exists {
		return false, 0, 0
	}

	if time.Now().After(setting.ExpiresAt) {
		return false, 0, 0
	}

	expiresIn = time.Until(setting.ExpiresAt)
	return setting.Active, setting.Remaining, expiresIn
}
