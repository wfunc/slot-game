package animal

import (
	"math/rand"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
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

// Room 表示一个动物房间实例

type Room struct {
	ID            uint32      // 房间唯一ID
	Type          pb.EZooType
	BetValues     []uint32
	MaxPlayer     uint32
	MinVIP        uint32
	CurrentPlayers uint32      // 当前玩家数

	animals      map[uint32]*AnimalRoute
	nextAnimalID uint32

	players map[uint32]*PlayerSession

	jackpot uint64
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
	SkillGain    []*pb.PAnimalSkill
	FreeGold     uint64
	KilledRoutes []*AnimalRoute
}
