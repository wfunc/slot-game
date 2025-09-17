package animal

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
)

// Animal 动物实体
type Animal struct {
	// 基础属性
	ID        uint32       // 唯一ID
	Type      pb.EAnimal   // 动物类型
	State     pb.EAnimalState // 状态（正常/冰冻）

	// 位置和移动
	PathID    uint32       // 路径ID
	Progress  float32      // 路径进度 (0.0 - 1.0)
	Speed     float32      // 移动速度
	X         float32      // 当前X坐标
	Y         float32      // 当前Y坐标
	Direction float32      // 朝向角度

	// 游戏属性
	HP        int32        // 生命值
	MaxHP     int32        // 最大生命值
	Odds      float32      // 赔率
	BaseOdds  float32      // 基础赔率
	HasRedBag bool         // 是否携带红包
	RedBagAmount uint32    // 红包金额

	// 状态控制
	IsFrozen     bool         // 是否被冰冻
	FreezeEndTime time.Time   // 冰冻结束时间
	IsLocked     bool         // 是否被锁定
	LockedBy     uint32       // 锁定玩家ID

	// 特殊效果
	SpecialEffect SpecialEffectType // 特殊效果类型

	// 生命周期
	SpawnTime    time.Time    // 生成时间
	IsAlive      int32        // 是否存活 (原子操作)

	// 并发控制
	mu          sync.RWMutex  // 读写锁
}

// SpecialEffectType 特殊效果类型
type SpecialEffectType int

const (
	EffectNone      SpecialEffectType = iota
	EffectLightning                   // 皮卡丘闪电链
	EffectBomb                        // 炸弹人全屏爆炸
)

// NewAnimal 创建新动物
func NewAnimal(id uint32, animalType pb.EAnimal, pathID uint32) *Animal {
	a := &Animal{
		ID:        id,
		Type:      animalType,
		State:     pb.EAnimalState_state_normal,
		PathID:    pathID,
		Progress:  0,
		Speed:     GetAnimalSpeed(animalType),
		HP:        GetAnimalHP(animalType),
		MaxHP:     GetAnimalHP(animalType),
		BaseOdds:  GetAnimalBaseOdds(animalType),
		Odds:      GetAnimalBaseOdds(animalType),
		SpawnTime: time.Now(),
		IsAlive:   1,
	}

	// 设置特殊效果
	switch animalType {
	case pb.EAnimal_pikachu:
		a.SpecialEffect = EffectLightning
	case pb.EAnimal_bomber:
		a.SpecialEffect = EffectBomb
	}

	return a
}

// Update 更新动物状态
func (a *Animal) Update(deltaTime float32, path *Path) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检查冰冻状态
	if a.IsFrozen {
		if time.Now().After(a.FreezeEndTime) {
			a.IsFrozen = false
			a.State = pb.EAnimalState_state_normal
		} else {
			return // 冰冻状态不更新位置
		}
	}

	// 更新路径进度
	a.Progress += a.Speed * deltaTime

	// 更新位置
	if path != nil {
		a.X, a.Y = path.GetPosition(a.Progress)
		a.Direction = path.GetDirection(a.Progress)
	}

	// 检查是否到达终点
	if a.Progress >= 1.0 {
		atomic.StoreInt32(&a.IsAlive, 0)
	}
}

// TakeDamage 受到伤害
func (a *Animal) TakeDamage(damage int32) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.HP -= damage
	if a.HP <= 0 {
		atomic.StoreInt32(&a.IsAlive, 0)
		return true // 动物死亡
	}
	return false
}

// Freeze 冰冻动物
func (a *Animal) Freeze(duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.IsFrozen = true
	a.State = pb.EAnimalState_state_ice
	a.FreezeEndTime = time.Now().Add(duration)
}

// Lock 锁定动物
func (a *Animal) Lock(playerID uint32) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.IsLocked = true
	a.LockedBy = playerID
}

// Unlock 解锁动物
func (a *Animal) Unlock() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.IsLocked = false
	a.LockedBy = 0
}

// ApplyOddsBoost 应用赔率提升
func (a *Animal) ApplyOddsBoost(multiplier float32) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Odds = a.BaseOdds * multiplier
}

// GetPosition 获取当前位置（线程安全）
func (a *Animal) GetPosition() (float32, float32) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.X, a.Y
}

// GetState 获取当前状态（线程安全）
func (a *Animal) GetState() pb.EAnimalState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.State
}

// IsAliveAtomic 原子检查是否存活
func (a *Animal) IsAliveAtomic() bool {
	return atomic.LoadInt32(&a.IsAlive) == 1
}

// GetBoundingBox 获取碰撞边界
func (a *Animal) GetBoundingBox() (x1, y1, x2, y2 float32) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 根据动物类型返回不同大小的碰撞箱
	size := GetAnimalSize(a.Type)
	halfSize := size / 2

	return a.X - halfSize, a.Y - halfSize,
	       a.X + halfSize, a.Y + halfSize
}

// ToProto 转换为Protobuf消息
func (a *Animal) ToProto() *pb.PRoute {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &pb.PRoute{
		Id:       &a.ID,
		Bet:      a.Type.Enum(),
		LineId:   &a.PathID,
		Point:    proto.Uint32(uint32(a.Progress * 100)),
		RedState: &a.HasRedBag,
		Status:   a.State.Enum(),
	}
}

// GetAnimalSpeed 根据动物类型获取速度
func GetAnimalSpeed(animalType pb.EAnimal) float32 {
	speeds := map[pb.EAnimal]float32{
		pb.EAnimal_turtle:   0.02,  // 乌龟最慢
		pb.EAnimal_elephant: 0.03,  // 大象较慢
		pb.EAnimal_panda:    0.035, // 熊猫慢
		pb.EAnimal_hippo:    0.04,  // 河马
		pb.EAnimal_ox:       0.045, // 奶牛
		pb.EAnimal_bear:     0.05,  // 熊
		pb.EAnimal_zhu:      0.055, // 猪
		pb.EAnimal_sheep:    0.06,  // 羊驼
		pb.EAnimal_horse:    0.065, // 矮马
		pb.EAnimal_dog:      0.07,  // 斗牛犬
		pb.EAnimal_cock:     0.075, // 公鸡
		pb.EAnimal_monkey:   0.08,  // 金丝猴
		pb.EAnimal_tuzi:     0.085, // 兔子
		pb.EAnimal_lv:       0.09,  // 驴
		pb.EAnimal_lion:     0.095, // 狮子
		pb.EAnimal_tiger:    0.10,  // 老虎
		pb.EAnimal_baozi:    0.105, // 豹子
		pb.EAnimal_pikachu:  0.11,  // 皮卡丘
		pb.EAnimal_bomber:   0.08,  // 炸弹人
	}

	if speed, ok := speeds[animalType]; ok {
		// 添加10%的随机变化
		variation := speed * 0.1
		return speed + (variation * (2*randFloat() - 1))
	}
	return 0.05
}

// GetAnimalHP 根据动物类型获取生命值
func GetAnimalHP(animalType pb.EAnimal) int32 {
	// 高赔率动物有更高生命值
	hpMap := map[pb.EAnimal]int32{
		pb.EAnimal_turtle:   1,
		pb.EAnimal_cock:     1,
		pb.EAnimal_dog:      1,
		pb.EAnimal_sheep:    1,
		pb.EAnimal_tuzi:     1,
		pb.EAnimal_monkey:   2,
		pb.EAnimal_horse:    2,
		pb.EAnimal_ox:       2,
		pb.EAnimal_lv:       2,
		pb.EAnimal_zhu:      2,
		pb.EAnimal_bear:     3,
		pb.EAnimal_hippo:    3,
		pb.EAnimal_panda:    4,
		pb.EAnimal_lion:     4,
		pb.EAnimal_tiger:    4,
		pb.EAnimal_baozi:    5,
		pb.EAnimal_elephant: 5,
		pb.EAnimal_pikachu:  6,
		pb.EAnimal_bomber:   8,
	}

	if hp, ok := hpMap[animalType]; ok {
		return hp
	}
	return 1
}

// GetAnimalBaseOdds 根据动物类型获取基础赔率
func GetAnimalBaseOdds(animalType pb.EAnimal) float32 {
	odds := map[pb.EAnimal]float32{
		pb.EAnimal_turtle:   1.5,
		pb.EAnimal_cock:     2.0,
		pb.EAnimal_dog:      2.5,
		pb.EAnimal_sheep:    2.5,
		pb.EAnimal_tuzi:     2.0,
		pb.EAnimal_monkey:   3.0,
		pb.EAnimal_horse:    3.5,
		pb.EAnimal_ox:       4.0,
		pb.EAnimal_lv:       2.5,
		pb.EAnimal_zhu:      3.5,
		pb.EAnimal_bear:     5.0,
		pb.EAnimal_hippo:    5.0,
		pb.EAnimal_panda:    6.0,
		pb.EAnimal_lion:     5.5,
		pb.EAnimal_tiger:    6.0,
		pb.EAnimal_baozi:    7.0,
		pb.EAnimal_elephant: 8.0,
		pb.EAnimal_pikachu:  10.0,
		pb.EAnimal_bomber:   15.0,
	}

	if odd, ok := odds[animalType]; ok {
		return odd
	}
	return 1.0
}

// GetAnimalSize 根据动物类型获取碰撞体积
func GetAnimalSize(animalType pb.EAnimal) float32 {
	sizes := map[pb.EAnimal]float32{
		pb.EAnimal_turtle:   20,
		pb.EAnimal_cock:     22,
		pb.EAnimal_dog:      25,
		pb.EAnimal_sheep:    26,
		pb.EAnimal_tuzi:     20,
		pb.EAnimal_monkey:   24,
		pb.EAnimal_horse:    30,
		pb.EAnimal_ox:       32,
		pb.EAnimal_lv:       28,
		pb.EAnimal_zhu:      30,
		pb.EAnimal_bear:     35,
		pb.EAnimal_hippo:    38,
		pb.EAnimal_panda:    35,
		pb.EAnimal_lion:     32,
		pb.EAnimal_tiger:    34,
		pb.EAnimal_baozi:    30,
		pb.EAnimal_elephant: 45,
		pb.EAnimal_pikachu:  25,
		pb.EAnimal_bomber:   28,
	}

	if size, ok := sizes[animalType]; ok {
		return size
	}
	return 25
}

// 简单的随机数生成器（后续可以优化为使用统一的随机源）
func randFloat() float32 {
	return float32(time.Now().UnixNano()%1000) / 1000
}