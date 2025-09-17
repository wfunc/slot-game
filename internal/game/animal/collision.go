package animal

import (
	"math"
	"sync"
)

// CollisionResult 碰撞结果
type CollisionResult struct {
	BulletID     string   // 子弹ID
	AnimalID     uint32   // 动物ID
	Hit          bool     // 是否命中
	Damage       int32    // 造成伤害
	IsKill       bool     // 是否击杀
	ChainTargets []uint32 // 连锁目标
	IceTargets   []uint32 // 冰冻目标
}

// CollisionSystem 碰撞检测系统
type CollisionSystem struct {
	// 空间分区优化
	gridSize    float32           // 网格大小
	gridWidth   int               // 网格宽度
	gridHeight  int               // 网格高度
	animalGrids map[int][]*Animal // 动物空间索引
	bulletGrids map[int][]*Bullet // 子弹空间索引

	// 碰撞结果缓存
	results []CollisionResult

	// 并发控制
	mu sync.RWMutex
}

// NewCollisionSystem 创建碰撞检测系统
func NewCollisionSystem(worldWidth, worldHeight float32) *CollisionSystem {
	gridSize := float32(100) // 100像素一个格子
	return &CollisionSystem{
		gridSize:    gridSize,
		gridWidth:   int(worldWidth/gridSize) + 1,
		gridHeight:  int(worldHeight/gridSize) + 1,
		animalGrids: make(map[int][]*Animal),
		bulletGrids: make(map[int][]*Bullet),
		results:     make([]CollisionResult, 0, 100),
	}
}

// UpdateSpatialIndex 更新空间索引
func (cs *CollisionSystem) UpdateSpatialIndex(animals []*Animal, bullets []*Bullet) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 清空网格
	cs.animalGrids = make(map[int][]*Animal)
	cs.bulletGrids = make(map[int][]*Bullet)

	// 将动物加入网格
	for _, animal := range animals {
		if animal.IsAliveAtomic() {
			gridIndex := cs.getGridIndex(animal.X, animal.Y)
			cs.animalGrids[gridIndex] = append(cs.animalGrids[gridIndex], animal)
		}
	}

	// 将子弹加入网格
	for _, bullet := range bullets {
		if bullet.IsAliveAtomic() {
			gridIndex := cs.getGridIndex(bullet.CurrentX, bullet.CurrentY)
			cs.bulletGrids[gridIndex] = append(cs.bulletGrids[gridIndex], bullet)
		}
	}
}

// DetectCollisions 检测所有碰撞
func (cs *CollisionSystem) DetectCollisions() []CollisionResult {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cs.results = cs.results[:0] // 清空结果但保留容量

	// 遍历所有有子弹的网格
	for gridIndex, bullets := range cs.bulletGrids {
		if len(bullets) == 0 {
			continue
		}

		// 获取该网格及周围网格的动物
		nearbyAnimals := cs.getNearbyAnimals(gridIndex)

		// 检测该网格内的碰撞
		for _, bullet := range bullets {
			for _, animal := range nearbyAnimals {
				if cs.checkBulletAnimalCollision(bullet, animal) {
					result := cs.processCollision(bullet, animal)
					cs.results = append(cs.results, result)
				}
			}
		}
	}

	return cs.results
}

// CheckBulletAnimalCollision 检测子弹与动物的碰撞
func (cs *CollisionSystem) checkBulletAnimalCollision(bullet *Bullet, animal *Animal) bool {
	if !bullet.IsAliveAtomic() || !animal.IsAliveAtomic() {
		return false
	}

	// 检查是否已经击中过
	for _, hitID := range bullet.HitAnimals {
		if hitID == animal.ID {
			return false
		}
	}

	// 获取动物碰撞箱
	ax1, ay1, ax2, ay2 := animal.GetBoundingBox()

	// 基础碰撞检测：点与矩形
	if cs.pointInRect(bullet.CurrentX, bullet.CurrentY, ax1, ay1, ax2, ay2) {
		return true
	}

	// 如果子弹有爆炸半径
	if bullet.ExplosiveRadius > 0 {
		centerX := (ax1 + ax2) / 2
		centerY := (ay1 + ay2) / 2
		dist := cs.distance(bullet.CurrentX, bullet.CurrentY, centerX, centerY)
		return dist <= bullet.ExplosiveRadius
	}

	// 线段与矩形碰撞（子弹轨迹）
	if bullet.Speed > 200 { // 高速子弹需要连续碰撞检测
		return cs.lineRectCollision(
			bullet.CurrentX-bullet.Speed*0.016, // 上一帧位置
			bullet.CurrentY-bullet.Speed*0.016,
			bullet.CurrentX, bullet.CurrentY,
			ax1, ay1, ax2, ay2,
		)
	}

	return false
}

// processCollision 处理碰撞
func (cs *CollisionSystem) processCollision(bullet *Bullet, animal *Animal) CollisionResult {
	result := CollisionResult{
		BulletID: bullet.ID,
		AnimalID: animal.ID,
		Hit:      true,
		Damage:   bullet.GetDamage(),
	}

	// 应用伤害
	result.IsKill = animal.TakeDamage(result.Damage)

	// 记录命中
	bullet.Hit(animal)

	// 处理技能效果
	if bullet.HasIceEffect && !animal.IsFrozen {
		animal.Freeze(bullet.IceDuration)
		result.IceTargets = append(result.IceTargets, animal.ID)
	}

	// 处理连锁效果
	if bullet.HasChainEffect && result.IsKill {
		result.ChainTargets = cs.findChainTargets(animal, bullet.ChainProbability)
	}

	return result
}

// findChainTargets 查找连锁目标
func (cs *CollisionSystem) findChainTargets(sourceAnimal *Animal, probability float32) []uint32 {
	var targets []uint32
	sourceX, sourceY := sourceAnimal.GetPosition()
	chainRadius := float32(150) // 连锁范围150像素

	// 获取源动物所在网格
	gridIndex := cs.getGridIndex(sourceX, sourceY)
	nearbyAnimals := cs.getNearbyAnimals(gridIndex)

	for _, animal := range nearbyAnimals {
		if animal.ID == sourceAnimal.ID || !animal.IsAliveAtomic() {
			continue
		}

		animalX, animalY := animal.GetPosition()
		dist := cs.distance(sourceX, sourceY, animalX, animalY)

		if dist <= chainRadius && randFloat() < probability {
			targets = append(targets, animal.ID)
		}
	}

	return targets
}

// getNearbyAnimals 获取网格及周围的动物
func (cs *CollisionSystem) getNearbyAnimals(gridIndex int) []*Animal {
	var animals []*Animal

	// 获取3x3网格范围内的动物
	x := gridIndex % cs.gridWidth
	y := gridIndex / cs.gridWidth

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx := x + dx
			ny := y + dy

			if nx >= 0 && nx < cs.gridWidth && ny >= 0 && ny < cs.gridHeight {
				neighborIndex := ny*cs.gridWidth + nx
				if gridAnimals, exists := cs.animalGrids[neighborIndex]; exists {
					animals = append(animals, gridAnimals...)
				}
			}
		}
	}

	return animals
}

// DetectAreaDamage 区域伤害检测
func (cs *CollisionSystem) DetectAreaDamage(centerX, centerY, radius float32, damage int32, source uint32) []CollisionResult {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var results []CollisionResult

	// 获取中心点所在网格
	gridIndex := cs.getGridIndex(centerX, centerY)
	nearbyAnimals := cs.getNearbyAnimals(gridIndex)

	for _, animal := range nearbyAnimals {
		if !animal.IsAliveAtomic() {
			continue
		}

		animalX, animalY := animal.GetPosition()
		dist := cs.distance(centerX, centerY, animalX, animalY)

		if dist <= radius {
			// 根据距离计算伤害衰减
			damageFactor := 1.0 - (dist / radius) * 0.5
			actualDamage := int32(float32(damage) * damageFactor)

			isKill := animal.TakeDamage(actualDamage)

			results = append(results, CollisionResult{
				AnimalID: animal.ID,
				Hit:      true,
				Damage:   actualDamage,
				IsKill:   isKill,
			})
		}
	}

	return results
}

// getGridIndex 计算网格索引
func (cs *CollisionSystem) getGridIndex(x, y float32) int {
	gridX := int(x / cs.gridSize)
	gridY := int(y / cs.gridSize)

	// 边界检查
	if gridX < 0 {
		gridX = 0
	} else if gridX >= cs.gridWidth {
		gridX = cs.gridWidth - 1
	}

	if gridY < 0 {
		gridY = 0
	} else if gridY >= cs.gridHeight {
		gridY = cs.gridHeight - 1
	}

	return gridY*cs.gridWidth + gridX
}

// pointInRect 点与矩形碰撞检测
func (cs *CollisionSystem) pointInRect(px, py, rx1, ry1, rx2, ry2 float32) bool {
	return px >= rx1 && px <= rx2 && py >= ry1 && py <= ry2
}

// lineRectCollision 线段与矩形碰撞检测
func (cs *CollisionSystem) lineRectCollision(x1, y1, x2, y2, rx1, ry1, rx2, ry2 float32) bool {
	// 检查线段端点是否在矩形内
	if cs.pointInRect(x1, y1, rx1, ry1, rx2, ry2) ||
	   cs.pointInRect(x2, y2, rx1, ry1, rx2, ry2) {
		return true
	}

	// 检查线段是否与矩形的四条边相交
	return cs.lineLineCollision(x1, y1, x2, y2, rx1, ry1, rx2, ry1) || // 上边
	       cs.lineLineCollision(x1, y1, x2, y2, rx1, ry2, rx2, ry2) || // 下边
	       cs.lineLineCollision(x1, y1, x2, y2, rx1, ry1, rx1, ry2) || // 左边
	       cs.lineLineCollision(x1, y1, x2, y2, rx2, ry1, rx2, ry2)    // 右边
}

// lineLineCollision 线段与线段碰撞检测
func (cs *CollisionSystem) lineLineCollision(x1, y1, x2, y2, x3, y3, x4, y4 float32) bool {
	denom := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
	if math.Abs(float64(denom)) < 0.0001 {
		return false // 平行或共线
	}

	t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / denom
	u := -((x1-x2)*(y1-y3) - (y1-y2)*(x1-x3)) / denom

	return t >= 0 && t <= 1 && u >= 0 && u <= 1
}

// distance 计算两点距离
func (cs *CollisionSystem) distance(x1, y1, x2, y2 float32) float32 {
	dx := x2 - x1
	dy := y2 - y1
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// CircleCollision 圆形碰撞检测
func (cs *CollisionSystem) CircleCollision(x1, y1, r1, x2, y2, r2 float32) bool {
	dist := cs.distance(x1, y1, x2, y2)
	return dist <= (r1 + r2)
}

// GetCollisionStats 获取碰撞统计
func (cs *CollisionSystem) GetCollisionStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	animalCount := 0
	for _, animals := range cs.animalGrids {
		animalCount += len(animals)
	}

	bulletCount := 0
	for _, bullets := range cs.bulletGrids {
		bulletCount += len(bullets)
	}

	return map[string]interface{}{
		"grid_size":     cs.gridSize,
		"grid_count":    cs.gridWidth * cs.gridHeight,
		"animal_count":  animalCount,
		"bullet_count":  bulletCount,
		"result_count":  len(cs.results),
	}
}