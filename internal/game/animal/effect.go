package animal

import (
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
)

// EffectType 效果类型
type EffectType int

const (
	EffectTypeNone      EffectType = iota
	EffectTypeLightning             // 皮卡丘闪电链
	EffectTypeBomb                  // 炸弹人爆炸
	EffectTypeFreeze                // 冰冻效果
	EffectTypeGold                  // 金币爆炸
	EffectTypeRedBag                // 红包效果
)

// Effect 特殊效果
type Effect struct {
	ID           uint32        // 效果ID
	Type         EffectType    // 效果类型
	SourceID     uint32        // 触发源ID（动物或玩家）
	TargetIDs    []uint32      // 目标ID列表
	X, Y         float32       // 效果中心位置
	Radius       float32       // 影响半径
	Damage       int32         // 造成伤害
	Duration     time.Duration // 持续时间
	CreateTime   time.Time     // 创建时间
	EndTime      time.Time     // 结束时间
	IsActive     bool          // 是否激活
	ChainCount   int           // 连锁次数
	ChainDelay   time.Duration // 连锁延迟
}

// EffectManager 效果管理器
type EffectManager struct {
	effects    map[uint32]*Effect
	effectPool *sync.Pool
	idCounter  uint32
	mu         sync.RWMutex
}

// NewEffectManager 创建效果管理器
func NewEffectManager() *EffectManager {
	return &EffectManager{
		effects: make(map[uint32]*Effect),
		effectPool: &sync.Pool{
			New: func() interface{} {
				return &Effect{
					TargetIDs: make([]uint32, 0, 10),
				}
			},
		},
	}
}

// CreateLightningEffect 创建闪电链效果（皮卡丘）
func (em *EffectManager) CreateLightningEffect(sourceAnimal *Animal, targetAnimals []*Animal) *Effect {
	effect := em.getEffect()
	effect.Type = EffectTypeLightning
	effect.SourceID = sourceAnimal.ID
	effect.X, effect.Y = sourceAnimal.GetPosition()
	effect.Radius = 200 // 200像素范围
	effect.ChainCount = 3 // 最多连锁3次
	effect.ChainDelay = 100 * time.Millisecond
	effect.Damage = 50 // 每次连锁50点伤害

	// 查找连锁目标
	chainTargets := em.findChainTargets(sourceAnimal, targetAnimals, effect.Radius, effect.ChainCount)
	effect.TargetIDs = chainTargets

	em.addEffect(effect)
	return effect
}

// CreateBombEffect 创建爆炸效果（炸弹人）
func (em *EffectManager) CreateBombEffect(sourceAnimal *Animal, targetAnimals []*Animal) *Effect {
	effect := em.getEffect()
	effect.Type = EffectTypeBomb
	effect.SourceID = sourceAnimal.ID
	effect.X, effect.Y = sourceAnimal.GetPosition()
	effect.Radius = 300 // 全屏爆炸
	effect.Damage = 100 // 爆炸伤害
	effect.Duration = 500 * time.Millisecond
	effect.EndTime = time.Now().Add(effect.Duration)

	// 查找爆炸范围内的所有目标
	for _, animal := range targetAnimals {
		if animal.ID == sourceAnimal.ID || !animal.IsAliveAtomic() {
			continue
		}

		ax, ay := animal.GetPosition()
		if distance(effect.X, effect.Y, ax, ay) <= effect.Radius {
			effect.TargetIDs = append(effect.TargetIDs, animal.ID)
		}
	}

	em.addEffect(effect)
	return effect
}

// CreateFreezeEffect 创建冰冻效果
func (em *EffectManager) CreateFreezeEffect(centerX, centerY, radius float32, duration time.Duration, targetAnimals []*Animal) *Effect {
	effect := em.getEffect()
	effect.Type = EffectTypeFreeze
	effect.X = centerX
	effect.Y = centerY
	effect.Radius = radius
	effect.Duration = duration
	effect.EndTime = time.Now().Add(duration)

	// 查找冰冻范围内的目标
	for _, animal := range targetAnimals {
		if !animal.IsAliveAtomic() {
			continue
		}

		ax, ay := animal.GetPosition()
		if distance(centerX, centerY, ax, ay) <= radius {
			effect.TargetIDs = append(effect.TargetIDs, animal.ID)
			animal.Freeze(duration)
		}
	}

	em.addEffect(effect)
	return effect
}

// CreateGoldExplosion 创建金币爆炸效果
func (em *EffectManager) CreateGoldExplosion(x, y float32, amount uint32) *Effect {
	effect := em.getEffect()
	effect.Type = EffectTypeGold
	effect.X = x
	effect.Y = y
	effect.Duration = 1 * time.Second
	effect.EndTime = time.Now().Add(effect.Duration)
	effect.Damage = int32(amount) // 用Damage字段存储金币数量

	em.addEffect(effect)
	return effect
}

// CreateRedBagEffect 创建红包效果
func (em *EffectManager) CreateRedBagEffect(x, y float32, amount uint32) *Effect {
	effect := em.getEffect()
	effect.Type = EffectTypeRedBag
	effect.X = x
	effect.Y = y
	effect.Duration = 2 * time.Second
	effect.EndTime = time.Now().Add(effect.Duration)
	effect.Damage = int32(amount) // 用Damage字段存储红包金额

	em.addEffect(effect)
	return effect
}

// ProcessLightningChain 处理闪电链
func (em *EffectManager) ProcessLightningChain(effect *Effect, animals []*Animal) []CollisionResult {
	var results []CollisionResult

	for i, targetID := range effect.TargetIDs {
		// 延迟处理，制造连锁效果
		delay := time.Duration(i) * effect.ChainDelay
		if time.Since(effect.CreateTime) < delay {
			continue
		}

		// 查找目标动物
		for _, animal := range animals {
			if animal.ID == targetID && animal.IsAliveAtomic() {
				isKill := animal.TakeDamage(effect.Damage)
				results = append(results, CollisionResult{
					AnimalID: targetID,
					Hit:      true,
					Damage:   effect.Damage,
					IsKill:   isKill,
				})
				break
			}
		}
	}

	return results
}

// ProcessBombExplosion 处理炸弹爆炸
func (em *EffectManager) ProcessBombExplosion(effect *Effect, animals []*Animal) []CollisionResult {
	var results []CollisionResult

	for _, targetID := range effect.TargetIDs {
		for _, animal := range animals {
			if animal.ID == targetID && animal.IsAliveAtomic() {
				// 根据距离计算伤害衰减
				ax, ay := animal.GetPosition()
				dist := distance(effect.X, effect.Y, ax, ay)
				damageFactor := 1.0 - (dist / effect.Radius) * 0.3
				actualDamage := int32(float32(effect.Damage) * damageFactor)

				isKill := animal.TakeDamage(actualDamage)
				results = append(results, CollisionResult{
					AnimalID: targetID,
					Hit:      true,
					Damage:   actualDamage,
					IsKill:   isKill,
				})
			}
		}
	}

	return results
}

// UpdateEffects 更新所有效果
func (em *EffectManager) UpdateEffects(animals []*Animal) {
	em.mu.Lock()
	defer em.mu.Unlock()

	now := time.Now()
	expiredEffects := []uint32{}

	for id, effect := range em.effects {
		if !effect.IsActive {
			continue
		}

		// 检查是否过期
		if effect.Duration > 0 && now.After(effect.EndTime) {
			effect.IsActive = false
			expiredEffects = append(expiredEffects, id)
			continue
		}

		// 处理不同类型的效果
		switch effect.Type {
		case EffectTypeLightning:
			// 闪电链逐步触发
			if len(effect.TargetIDs) > 0 {
				em.ProcessLightningChain(effect, animals)
			}

		case EffectTypeBomb:
			// 爆炸一次性触发
			if time.Since(effect.CreateTime) < 100*time.Millisecond {
				em.ProcessBombExplosion(effect, animals)
			}
		}
	}

	// 清理过期效果
	for _, id := range expiredEffects {
		em.removeEffect(id)
	}
}

// GetActiveEffects 获取活跃效果
func (em *EffectManager) GetActiveEffects() []*Effect {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var activeEffects []*Effect
	for _, effect := range em.effects {
		if effect.IsActive {
			activeEffects = append(activeEffects, effect)
		}
	}
	return activeEffects
}

// findChainTargets 查找连锁目标
func (em *EffectManager) findChainTargets(source *Animal, animals []*Animal, radius float32, maxCount int) []uint32 {
	var targets []uint32
	sx, sy := source.GetPosition()

	// 按距离排序
	type targetDist struct {
		id   uint32
		dist float32
	}
	var candidates []targetDist

	for _, animal := range animals {
		if animal.ID == source.ID || !animal.IsAliveAtomic() {
			continue
		}

		ax, ay := animal.GetPosition()
		dist := distance(sx, sy, ax, ay)

		if dist <= radius {
			candidates = append(candidates, targetDist{
				id:   animal.ID,
				dist: dist,
			})
		}
	}

	// 按距离排序，选择最近的目标
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].dist < candidates[i].dist {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// 选择前N个目标
	for i := 0; i < len(candidates) && i < maxCount; i++ {
		targets = append(targets, candidates[i].id)
	}

	return targets
}

// 辅助函数

func (em *EffectManager) getEffect() *Effect {
	effect := em.effectPool.Get().(*Effect)
	effect.reset()
	effect.ID = em.generateID()
	effect.CreateTime = time.Now()
	effect.IsActive = true
	return effect
}

func (em *EffectManager) addEffect(effect *Effect) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.effects[effect.ID] = effect
}

func (em *EffectManager) removeEffect(id uint32) {
	if effect, exists := em.effects[id]; exists {
		delete(em.effects, id)
		effect.reset()
		em.effectPool.Put(effect)
	}
}

func (em *EffectManager) generateID() uint32 {
	em.idCounter++
	return em.idCounter
}

func (effect *Effect) reset() {
	effect.ID = 0
	effect.Type = EffectTypeNone
	effect.SourceID = 0
	effect.TargetIDs = effect.TargetIDs[:0]
	effect.X = 0
	effect.Y = 0
	effect.Radius = 0
	effect.Damage = 0
	effect.Duration = 0
	effect.IsActive = false
	effect.ChainCount = 0
	effect.ChainDelay = 0
}

// GetEffectProto 转换为Protobuf消息
func (effect *Effect) GetEffectProto() *pb.PAnimalOne {
	return &pb.PAnimalOne{
		Id:     proto.Uint32(effect.SourceID),
		Win:    proto.Uint32(uint32(effect.Damage)),
		RedBag: proto.Uint32(0),
	}
}

// EffectTypeToProto 转换效果类型到Protobuf
func EffectTypeToProto(effectType EffectType) pb.EAnimalType {
	switch effectType {
	case EffectTypeLightning:
		return pb.EAnimalType_lightning
	case EffectTypeBomb:
		return pb.EAnimalType_boom
	default:
		return pb.EAnimalType_type_normal
	}
}

