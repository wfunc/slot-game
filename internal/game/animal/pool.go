package animal

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/wfunc/slot-game/internal/pb"
)

// ObjectPool 通用对象池
type ObjectPool struct {
	// 对象池
	animalPool  *sync.Pool
	bulletPool  *sync.Pool
	effectPool  *sync.Pool
	messagePool *sync.Pool

	// 统计信息
	animalCreated  uint64
	animalReused   uint64
	bulletCreated  uint64
	bulletReused   uint64
	effectCreated  uint64
	effectReused   uint64
	messageCreated uint64
	messageReused  uint64

	// 配置
	maxAnimalCap  int
	maxBulletCap  int
	maxEffectCap  int
	maxMessageCap int
}

// NewObjectPool 创建对象池
func NewObjectPool() *ObjectPool {
	op := &ObjectPool{
		maxAnimalCap:  1000,
		maxBulletCap:  5000,
		maxEffectCap:  500,
		maxMessageCap: 10000,
	}

	// 初始化动物对象池
	op.animalPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&op.animalCreated, 1)
			return &Animal{
				IsAlive: 1,
			}
		},
	}

	// 初始化子弹对象池
	op.bulletPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&op.bulletCreated, 1)
			return &Bullet{
				HitAnimals: make([]uint32, 0, 10),
				IsAlive:    1,
			}
		},
	}

	// 初始化效果对象池
	op.effectPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&op.effectCreated, 1)
			return &Effect{
				TargetIDs: make([]uint32, 0, 10),
			}
		},
	}

	// 初始化消息对象池
	op.messagePool = &sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&op.messageCreated, 1)
			return &GameMessage{
				Data: make(map[string]interface{}),
			}
		},
	}

	return op
}

// GetAnimal 从池中获取动物对象
func (op *ObjectPool) GetAnimal() *Animal {
	atomic.AddUint64(&op.animalReused, 1)
	animal := op.animalPool.Get().(*Animal)
	animal.Reset()
	return animal
}

// PutAnimal 归还动物对象到池
func (op *ObjectPool) PutAnimal(animal *Animal) {
	if animal == nil {
		return
	}
	animal.Reset()
	op.animalPool.Put(animal)
}

// GetBullet 从池中获取子弹对象
func (op *ObjectPool) GetBullet() *Bullet {
	atomic.AddUint64(&op.bulletReused, 1)
	bullet := op.bulletPool.Get().(*Bullet)
	bullet.Reset()
	return bullet
}

// PutBullet 归还子弹对象到池
func (op *ObjectPool) PutBullet(bullet *Bullet) {
	if bullet == nil {
		return
	}
	bullet.Reset()
	op.bulletPool.Put(bullet)
}

// GetEffect 从池中获取效果对象
func (op *ObjectPool) GetEffect() *Effect {
	atomic.AddUint64(&op.effectReused, 1)
	effect := op.effectPool.Get().(*Effect)
	effect.reset()
	return effect
}

// PutEffect 归还效果对象到池
func (op *ObjectPool) PutEffect(effect *Effect) {
	if effect == nil {
		return
	}
	effect.reset()
	op.effectPool.Put(effect)
}

// GetMessage 从池中获取消息对象
func (op *ObjectPool) GetMessage() *GameMessage {
	atomic.AddUint64(&op.messageReused, 1)
	msg := op.messagePool.Get().(*GameMessage)
	msg.Reset()
	return msg
}

// PutMessage 归还消息对象到池
func (op *ObjectPool) PutMessage(msg *GameMessage) {
	if msg == nil {
		return
	}
	msg.Reset()
	op.messagePool.Put(msg)
}

// GetStats 获取对象池统计信息
func (op *ObjectPool) GetStats() map[string]uint64 {
	return map[string]uint64{
		"animal_created":  atomic.LoadUint64(&op.animalCreated),
		"animal_reused":   atomic.LoadUint64(&op.animalReused),
		"bullet_created":  atomic.LoadUint64(&op.bulletCreated),
		"bullet_reused":   atomic.LoadUint64(&op.bulletReused),
		"effect_created":  atomic.LoadUint64(&op.effectCreated),
		"effect_reused":   atomic.LoadUint64(&op.effectReused),
		"message_created": atomic.LoadUint64(&op.messageCreated),
		"message_reused":  atomic.LoadUint64(&op.messageReused),
	}
}

// Reset 重置动物对象
func (a *Animal) Reset() {
	a.ID = 0
	a.Type = pb.EAnimal_balance
	a.State = pb.EAnimalState_state_normal
	a.PathID = 0
	a.Progress = 0
	a.Speed = 0
	a.X = 0
	a.Y = 0
	a.Direction = 0
	a.HP = 0
	a.MaxHP = 0
	a.Odds = 0
	a.BaseOdds = 0
	a.HasRedBag = false
	a.RedBagAmount = 0
	a.IsFrozen = false
	a.FreezeEndTime = time.Time{}
	a.IsLocked = false
	a.LockedBy = 0
	a.SpecialEffect = EffectNone
	a.SpawnTime = time.Time{}
	atomic.StoreInt32(&a.IsAlive, 0)
}

// GameMessage 游戏消息
type GameMessage struct {
	Type      string
	PlayerID  uint32
	Timestamp time.Time
	Data      map[string]interface{}
}

// Reset 重置消息
func (m *GameMessage) Reset() {
	m.Type = ""
	m.PlayerID = 0
	m.Timestamp = time.Time{}
	for k := range m.Data {
		delete(m.Data, k)
	}
}

// PooledAnimalManager 使用对象池的动物管理器
type PooledAnimalManager struct {
	pool        *ObjectPool
	animals     map[uint32]*Animal
	idGenerator uint32
	mu          sync.RWMutex
}

// NewPooledAnimalManager 创建使用对象池的动物管理器
func NewPooledAnimalManager(pool *ObjectPool) *PooledAnimalManager {
	return &PooledAnimalManager{
		pool:    pool,
		animals: make(map[uint32]*Animal),
	}
}

// CreateAnimal 创建动物（使用对象池）
func (pam *PooledAnimalManager) CreateAnimal(animalType pb.EAnimal, pathID uint32) *Animal {
	animal := pam.pool.GetAnimal()

	// 初始化动物
	animal.ID = atomic.AddUint32(&pam.idGenerator, 1)
	animal.Type = animalType
	animal.State = pb.EAnimalState_state_normal
	animal.PathID = pathID
	animal.Progress = 0
	animal.Speed = GetAnimalSpeed(animalType)
	animal.HP = GetAnimalHP(animalType)
	animal.MaxHP = animal.HP
	animal.BaseOdds = GetAnimalBaseOdds(animalType)
	animal.Odds = animal.BaseOdds
	animal.SpawnTime = time.Now()
	animal.HasRedBag = randFloat() < 0.1 // 10%概率有红包
	if animal.HasRedBag {
		animal.RedBagAmount = uint32(50 + randFloat()*100) // 50-150红包
	}
	atomic.StoreInt32(&animal.IsAlive, 1)

	// 设置特殊效果
	switch animalType {
	case pb.EAnimal_pikachu:
		animal.SpecialEffect = EffectLightning
	case pb.EAnimal_bomber:
		animal.SpecialEffect = EffectBomb
	}

	// 加入管理器
	pam.mu.Lock()
	pam.animals[animal.ID] = animal
	pam.mu.Unlock()

	return animal
}

// RemoveAnimal 移除动物（归还到对象池）
func (pam *PooledAnimalManager) RemoveAnimal(id uint32) {
	pam.mu.Lock()
	defer pam.mu.Unlock()

	if animal, exists := pam.animals[id]; exists {
		delete(pam.animals, id)
		pam.pool.PutAnimal(animal)
	}
}

// GetAnimal 获取动物
func (pam *PooledAnimalManager) GetAnimal(id uint32) *Animal {
	pam.mu.RLock()
	defer pam.mu.RUnlock()
	return pam.animals[id]
}

// GetAllAnimals 获取所有动物
func (pam *PooledAnimalManager) GetAllAnimals() []*Animal {
	pam.mu.RLock()
	defer pam.mu.RUnlock()

	animals := make([]*Animal, 0, len(pam.animals))
	for _, animal := range pam.animals {
		if animal.IsAliveAtomic() {
			animals = append(animals, animal)
		}
	}
	return animals
}

// UpdateAnimals 更新所有动物
func (pam *PooledAnimalManager) UpdateAnimals(deltaTime float32, pathManager *PathManager) []uint32 {
	pam.mu.RLock()
	animals := make([]*Animal, 0, len(pam.animals))
	for _, animal := range pam.animals {
		animals = append(animals, animal)
	}
	pam.mu.RUnlock()

	var removedIDs []uint32

	for _, animal := range animals {
		if !animal.IsAliveAtomic() {
			removedIDs = append(removedIDs, animal.ID)
			continue
		}

		// 获取路径
		path := pathManager.GetPath(animal.PathID)
		if path != nil {
			animal.Update(deltaTime, path)
		}

		// 检查是否需要移除
		if !animal.IsAliveAtomic() || animal.Progress >= 1.0 {
			removedIDs = append(removedIDs, animal.ID)
		}
	}

	// 批量移除死亡动物
	for _, id := range removedIDs {
		pam.RemoveAnimal(id)
	}

	return removedIDs
}

// Cleanup 清理所有动物
func (pam *PooledAnimalManager) Cleanup() {
	pam.mu.Lock()
	defer pam.mu.Unlock()

	for id, animal := range pam.animals {
		pam.pool.PutAnimal(animal)
		delete(pam.animals, id)
	}
}