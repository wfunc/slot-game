package animal

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Bullet 子弹实体
type Bullet struct {
	// 基础属性
	ID        string    // 唯一ID（使用UUID格式）
	PlayerID  uint32    // 发射玩家ID
	BetAmount uint32    // 下注金额

	// 位置和运动
	StartX    float32   // 起始X坐标
	StartY    float32   // 起始Y坐标
	TargetX   float32   // 目标X坐标
	TargetY   float32   // 目标Y坐标
	CurrentX  float32   // 当前X坐标
	CurrentY  float32   // 当前Y坐标
	Speed     float32   // 飞行速度
	Direction float32   // 飞行方向（弧度）

	// 目标信息
	TargetID     uint32    // 目标动物ID（0表示无目标）
	IsLocking    bool      // 是否锁定目标
	LockStrength float32   // 锁定强度（0-1）

	// 子弹属性
	Damage       int32     // 伤害值
	Penetration  int32     // 穿透次数（可以击中多个目标）
	HitCount     int32     // 已命中次数
	ExplosiveRadius float32 // 爆炸半径（0表示无爆炸）

	// 技能增强
	HasIceEffect     bool      // 是否有冰冻效果
	IceDuration      time.Duration // 冰冻持续时间
	HasChainEffect   bool      // 是否有连锁效果
	ChainProbability float32   // 连锁概率
	DamageMultiplier float32   // 伤害倍率

	// 生命周期
	CreateTime   time.Time // 创建时间
	MaxLifetime  time.Duration // 最大生存时间
	IsAlive      int32     // 是否存活（原子操作）
	HitAnimals   []uint32  // 已击中的动物ID列表

	// 并发控制
	mu sync.RWMutex
}

// BulletManager 子弹管理器
type BulletManager struct {
	bullets      map[string]*Bullet  // 活跃子弹集合
	bulletPool   *sync.Pool          // 子弹对象池
	idGenerator  uint64              // ID生成器
	mu           sync.RWMutex
}

// NewBulletManager 创建子弹管理器
func NewBulletManager() *BulletManager {
	return &BulletManager{
		bullets: make(map[string]*Bullet),
		bulletPool: &sync.Pool{
			New: func() interface{} {
				return &Bullet{
					HitAnimals: make([]uint32, 0, 10),
				}
			},
		},
	}
}

// CreateBullet 创建新子弹
func (bm *BulletManager) CreateBullet(playerID uint32, betAmount uint32, startX, startY, targetX, targetY float32) *Bullet {
	// 从对象池获取子弹
	bullet := bm.bulletPool.Get().(*Bullet)

	// 生成唯一ID
	id := bm.generateBulletID()

	// 重置并初始化子弹
	bullet.Reset()
	bullet.ID = id
	bullet.PlayerID = playerID
	bullet.BetAmount = betAmount
	bullet.StartX = startX
	bullet.StartY = startY
	bullet.CurrentX = startX
	bullet.CurrentY = startY
	bullet.TargetX = targetX
	bullet.TargetY = targetY
	bullet.Speed = 500.0 // 像素/秒
	bullet.Damage = calculateDamage(betAmount)
	bullet.DamageMultiplier = 1.0
	bullet.Penetration = 1
	bullet.CreateTime = time.Now()
	bullet.MaxLifetime = 3 * time.Second
	atomic.StoreInt32(&bullet.IsAlive, 1)

	// 计算方向
	dx := targetX - startX
	dy := targetY - startY
	bullet.Direction = atan2(dy, dx)

	// 加入管理器
	bm.mu.Lock()
	bm.bullets[id] = bullet
	bm.mu.Unlock()

	return bullet
}

// CreateSkillBullet 创建带技能的子弹
func (bm *BulletManager) CreateSkillBullet(playerID uint32, betAmount uint32, startX, startY, targetX, targetY float32, skills []SkillType) *Bullet {
	bullet := bm.CreateBullet(playerID, betAmount, startX, startY, targetX, targetY)

	// 应用技能效果
	for _, skill := range skills {
		switch skill {
		case SkillTypeIce:
			bullet.HasIceEffect = true
			bullet.IceDuration = 5 * time.Second

		case SkillTypeLocking:
			bullet.IsLocking = true
			bullet.LockStrength = 0.8 // 80%的追踪强度

		case SkillTypeOddsBoost:
			bullet.DamageMultiplier = 2.0 // 双倍伤害

		case SkillTypeChain:
			bullet.HasChainEffect = true
			bullet.ChainProbability = 0.3 // 30%连锁概率
			bullet.Penetration = 3 // 可以穿透3个目标

		case SkillTypeExplosive:
			bullet.ExplosiveRadius = 100 // 100像素爆炸半径
			bullet.Damage *= 2 // 爆炸子弹基础伤害翻倍
		}
	}

	return bullet
}

// Update 更新子弹状态
func (b *Bullet) Update(deltaTime float32, animals []*Animal) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.IsAliveAtomic() {
		return
	}

	// 检查生存时间
	if time.Since(b.CreateTime) > b.MaxLifetime {
		atomic.StoreInt32(&b.IsAlive, 0)
		return
	}

	// 如果有锁定目标，更新目标位置
	if b.IsLocking && b.TargetID > 0 {
		for _, animal := range animals {
			if animal.ID == b.TargetID && animal.IsAliveAtomic() {
				animalX, animalY := animal.GetPosition()
				// 平滑追踪
				b.TargetX = b.TargetX + (animalX-b.TargetX)*b.LockStrength*deltaTime
				b.TargetY = b.TargetY + (animalY-b.TargetY)*b.LockStrength*deltaTime
				break
			}
		}
	}

	// 计算移动
	dx := b.TargetX - b.CurrentX
	dy := b.TargetY - b.CurrentY
	dist := sqrt(dx*dx + dy*dy)

	if dist < 5 { // 到达目标
		if b.HitCount >= b.Penetration {
			atomic.StoreInt32(&b.IsAlive, 0)
		}
		return
	}

	// 归一化方向并移动
	moveDistance := b.Speed * deltaTime
	if moveDistance > dist {
		moveDistance = dist
	}

	b.CurrentX += (dx / dist) * moveDistance
	b.CurrentY += (dy / dist) * moveDistance
	b.Direction = atan2(dy, dx)
}

// CheckCollision 检查与动物的碰撞
func (b *Bullet) CheckCollision(animal *Animal) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.IsAliveAtomic() || !animal.IsAliveAtomic() {
		return false
	}

	// 检查是否已经击中过这个动物
	for _, hitID := range b.HitAnimals {
		if hitID == animal.ID {
			return false
		}
	}

	// 获取动物碰撞箱
	ax1, ay1, ax2, ay2 := animal.GetBoundingBox()

	// 点与矩形碰撞检测
	if b.CurrentX >= ax1 && b.CurrentX <= ax2 &&
	   b.CurrentY >= ay1 && b.CurrentY <= ay2 {
		return true
	}

	// 如果有爆炸半径，检查爆炸范围
	if b.ExplosiveRadius > 0 {
		centerX := (ax1 + ax2) / 2
		centerY := (ay1 + ay2) / 2
		dist := distance(b.CurrentX, b.CurrentY, centerX, centerY)
		return dist <= b.ExplosiveRadius
	}

	return false
}

// Hit 命中处理
func (b *Bullet) Hit(animal *Animal) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 记录击中
	b.HitAnimals = append(b.HitAnimals, animal.ID)
	b.HitCount++

	// 达到穿透上限则销毁
	if b.HitCount >= b.Penetration {
		atomic.StoreInt32(&b.IsAlive, 0)
	}
}

// GetDamage 获取伤害值
func (b *Bullet) GetDamage() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return int32(float32(b.Damage) * b.DamageMultiplier)
}

// IsAliveAtomic 原子检查是否存活
func (b *Bullet) IsAliveAtomic() bool {
	return atomic.LoadInt32(&b.IsAlive) == 1
}

// Reset 重置子弹状态（用于对象池复用）
func (b *Bullet) Reset() {
	b.ID = ""
	b.PlayerID = 0
	b.BetAmount = 0
	b.TargetID = 0
	b.IsLocking = false
	b.LockStrength = 0
	b.Damage = 0
	b.Penetration = 1
	b.HitCount = 0
	b.ExplosiveRadius = 0
	b.HasIceEffect = false
	b.HasChainEffect = false
	b.ChainProbability = 0
	b.DamageMultiplier = 1.0
	b.HitAnimals = b.HitAnimals[:0] // 清空但保留容量
	atomic.StoreInt32(&b.IsAlive, 0)
}

// UpdateBullets 批量更新所有子弹
func (bm *BulletManager) UpdateBullets(deltaTime float32, animals []*Animal) {
	bm.mu.RLock()
	bullets := make([]*Bullet, 0, len(bm.bullets))
	for _, bullet := range bm.bullets {
		bullets = append(bullets, bullet)
	}
	bm.mu.RUnlock()

	// 更新每个子弹
	for _, bullet := range bullets {
		bullet.Update(deltaTime, animals)

		// 移除死亡子弹
		if !bullet.IsAliveAtomic() {
			bm.RemoveBullet(bullet.ID)
		}
	}
}

// RemoveBullet 移除子弹并回收到对象池
func (bm *BulletManager) RemoveBullet(id string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bullet, exists := bm.bullets[id]; exists {
		delete(bm.bullets, id)
		bullet.Reset()
		bm.bulletPool.Put(bullet)
	}
}

// GetBullet 获取子弹
func (bm *BulletManager) GetBullet(id string) *Bullet {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bullets[id]
}

// GetActiveBullets 获取所有活跃子弹
func (bm *BulletManager) GetActiveBullets() []*Bullet {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	bullets := make([]*Bullet, 0, len(bm.bullets))
	for _, bullet := range bm.bullets {
		if bullet.IsAliveAtomic() {
			bullets = append(bullets, bullet)
		}
	}
	return bullets
}

// generateBulletID 生成子弹ID
func (bm *BulletManager) generateBulletID() string {
	id := atomic.AddUint64(&bm.idGenerator, 1)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%d_%d", timestamp, id)
}

// 辅助函数

// calculateDamage 根据下注金额计算伤害
func calculateDamage(betAmount uint32) int32 {
	// 基础伤害 = 1，每100金币增加1点伤害
	return 1 + int32(betAmount/100)
}

// atan2 计算反正切
func atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

// sqrt 计算平方根
func sqrt(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

