# 疯狂动物园游戏 - Golang实现架构优化

## 一、现有代码分析

### 1.1 已实现功能
✅ **基础架构**
- WebSocket 通信层
- Protobuf 消息编解码
- 基础的房间管理器 (Manager)
- 玩家会话管理 (PlayerSession)
- 数据库集成 (GORM)

✅ **部分游戏功能**
- 进入/离开房间
- 基础的下注框架
- 房间类型定义
- 玩家数据结构

### 1.2 缺失的核心功能
❌ **游戏核心机制**
- 动物生成与移动系统
- 路径系统实现
- 碰撞检测算法
- 赔率计算引擎
- 技能系统完整实现
- 特殊动物效果（闪电链、全屏爆炸）

❌ **数据管理**
- 彩金池系统
- 游戏记录存储
- 排行榜功能
- 统计分析

❌ **系统优化**
- 定时器管理
- 对象池复用
- 消息批量发送
- 并发控制优化

## 二、优化后的项目结构

```
slot-game/
├── internal/
│   ├── game/
│   │   └── animal/
│   │       ├── manager.go       # 游戏管理器（已有，需扩展）
│   │       ├── types.go         # 数据结构（已有，需扩展）
│   │       ├── room.go          # 房间逻辑（新建）
│   │       ├── animal.go        # 动物逻辑（新建）
│   │       ├── path.go          # 路径系统（新建）
│   │       ├── skill.go         # 技能系统（新建）
│   │       ├── odds.go          # 赔率计算（新建）
│   │       ├── collision.go     # 碰撞检测（新建）
│   │       ├── effect.go        # 特殊效果（新建）
│   │       ├── jackpot.go       # 彩金池（新建）
│   │       └── timer.go         # 定时器管理（新建）
│   └── websocket/
│       └── animal_handler.go    # WebSocket处理（已有，需优化）
```

## 三、核心模块实现计划

### 3.1 第一阶段：完善基础系统（本周）

#### 3.1.1 动物系统 (animal.go)
```go
package animal

import (
    "sync"
    "time"
    "github.com/wfunc/slot-game/internal/pb"
)

// Animal 动物实体
type Animal struct {
    ID        uint32
    Type      pb.EAnimal
    LineID    uint32
    Position  float32     // 在路径上的位置 [0, 1]
    Speed     float32     // 移动速度
    State     pb.EAnimalState
    RedBag    bool
    SpawnTime time.Time

    // 位置信息
    X, Y      float32
    Direction float32

    // 状态效果
    Frozen      bool
    FreezeUntil time.Time

    mu sync.RWMutex
}

// Update 更新动物状态
func (a *Animal) Update(deltaTime float32, path *Path) {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.Frozen && time.Now().After(a.FreezeUntil) {
        a.Frozen = false
        a.State = pb.EAnimalState_normal
    }

    if !a.Frozen {
        a.Position += a.Speed * deltaTime
        if a.Position >= 1.0 {
            a.Position = 1.0
        }

        // 更新世界坐标
        a.X, a.Y = path.GetWorldPosition(a.Position)
        a.Direction = path.GetDirection(a.Position)
    }
}

// Freeze 冻结动物
func (a *Animal) Freeze(duration time.Duration) {
    a.mu.Lock()
    defer a.mu.Unlock()

    a.Frozen = true
    a.State = pb.EAnimalState_ice
    a.FreezeUntil = time.Now().Add(duration)
}

// IsAlive 检查动物是否还在场内
func (a *Animal) IsAlive() bool {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.Position < 1.0
}

// GetBoundingBox 获取碰撞盒
func (a *Animal) GetBoundingBox() BoundingBox {
    a.mu.RLock()
    defer a.mu.RUnlock()

    size := getAnimalSize(a.Type)
    return BoundingBox{
        Min: Point{X: a.X - size/2, Y: a.Y - size/2},
        Max: Point{X: a.X + size/2, Y: a.Y + size/2},
    }
}
```

#### 3.1.2 路径系统 (path.go)
```go
package animal

import "math"

// Path 动物移动路径
type Path struct {
    ID     uint32
    Points []PathPoint
    Length float32
}

// PathPoint 路径点
type PathPoint struct {
    X, Y      float32
    Direction float32
}

// GetWorldPosition 根据路径进度获取世界坐标
func (p *Path) GetWorldPosition(progress float32) (x, y float32) {
    if progress <= 0 {
        return p.Points[0].X, p.Points[0].Y
    }
    if progress >= 1 {
        last := p.Points[len(p.Points)-1]
        return last.X, last.Y
    }

    targetDistance := progress * p.Length
    accumulated := float32(0)

    for i := 0; i < len(p.Points)-1; i++ {
        segmentLength := p.getSegmentLength(i)

        if accumulated+segmentLength >= targetDistance {
            t := (targetDistance - accumulated) / segmentLength
            x = lerp(p.Points[i].X, p.Points[i+1].X, t)
            y = lerp(p.Points[i].Y, p.Points[i+1].Y, t)
            return
        }
        accumulated += segmentLength
    }

    last := p.Points[len(p.Points)-1]
    return last.X, last.Y
}

// GetDirection 获取当前方向
func (p *Path) GetDirection(progress float32) float32 {
    if len(p.Points) < 2 {
        return 0
    }

    targetDistance := progress * p.Length
    accumulated := float32(0)

    for i := 0; i < len(p.Points)-1; i++ {
        segmentLength := p.getSegmentLength(i)

        if accumulated+segmentLength >= targetDistance {
            dx := p.Points[i+1].X - p.Points[i].X
            dy := p.Points[i+1].Y - p.Points[i].Y
            return float32(math.Atan2(float64(dy), float64(dx)))
        }
        accumulated += segmentLength
    }

    return 0
}

func (p *Path) getSegmentLength(index int) float32 {
    if index >= len(p.Points)-1 {
        return 0
    }
    dx := p.Points[index+1].X - p.Points[index].X
    dy := p.Points[index+1].Y - p.Points[index].Y
    return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

func lerp(a, b, t float32) float32 {
    return a + (b-a)*t
}

// PathManager 路径管理器
var PathManager = &pathManager{
    paths: map[uint32]*Path{
        1: {
            ID: 1,
            Points: []PathPoint{
                {X: 0, Y: 100},
                {X: 200, Y: 100},
                {X: 400, Y: 200},
                {X: 600, Y: 200},
                {X: 800, Y: 100},
            },
            Length: 847.2, // 预计算的总长度
        },
        2: {
            ID: 2,
            Points: []PathPoint{
                {X: 0, Y: 300},
                {X: 300, Y: 300},
                {X: 500, Y: 400},
                {X: 800, Y: 400},
            },
            Length: 916.2,
        },
        // 更多路径...
    },
}

type pathManager struct {
    paths map[uint32]*Path
}

func (pm *pathManager) GetPath(id uint32) *Path {
    return pm.paths[id]
}

func (pm *pathManager) GetRandomPath() *Path {
    // 随机选择一条路径
    for _, path := range pm.paths {
        return path // 简化实现，实际应该随机选择
    }
    return nil
}
```

#### 3.1.3 房间逻辑优化 (room.go)
```go
package animal

import (
    "sync"
    "time"
    "math/rand"
    "github.com/wfunc/slot-game/internal/pb"
)

// Room 游戏房间
type Room struct {
    Type      pb.EZooType
    BetValues []uint32
    MaxPlayer uint32
    MinVIP    uint32

    mu            sync.RWMutex
    animals       map[uint32]*Animal
    nextAnimalID  uint32
    players       map[uint32]*PlayerSession

    // 游戏状态
    jackpot       uint64
    redBag        bool
    lastSpawnTime time.Time
    spawnInterval time.Duration

    // 定时器
    updateTicker  *time.Ticker
    spawnTicker   *time.Ticker

    // 消息通道
    msgChan       chan RoomMessage
    closeChan     chan struct{}
}

// NewRoom 创建房间
func NewRoom(zooType pb.EZooType) *Room {
    room := &Room{
        Type:          zooType,
        BetValues:     defaultBetValues[zooType],
        MaxPlayer:     100,
        MinVIP:        vipRequirement[zooType],
        animals:       make(map[uint32]*Animal),
        nextAnimalID:  1,
        players:       make(map[uint32]*PlayerSession),
        jackpot:       5000000,
        redBag:        true,
        spawnInterval: 3 * time.Second,
        msgChan:       make(chan RoomMessage, 100),
        closeChan:     make(chan struct{}),
    }

    return room
}

// Start 启动房间
func (r *Room) Start() {
    r.updateTicker = time.NewTicker(100 * time.Millisecond)
    r.spawnTicker = time.NewTicker(r.spawnInterval)

    go r.run()
}

// run 房间主循环
func (r *Room) run() {
    for {
        select {
        case <-r.updateTicker.C:
            r.update()

        case <-r.spawnTicker.C:
            r.spawnRandomAnimal()

        case msg := <-r.msgChan:
            r.handleMessage(msg)

        case <-r.closeChan:
            r.updateTicker.Stop()
            r.spawnTicker.Stop()
            return
        }
    }
}

// update 更新房间状态
func (r *Room) update() {
    r.mu.Lock()
    defer r.mu.Unlock()

    deltaTime := float32(0.1) // 100ms

    // 更新所有动物
    deadAnimals := []uint32{}
    for id, animal := range r.animals {
        path := PathManager.GetPath(animal.LineID)
        animal.Update(deltaTime, path)

        if !animal.IsAlive() {
            deadAnimals = append(deadAnimals, id)
        }
    }

    // 移除死亡动物
    for _, id := range deadAnimals {
        delete(r.animals, id)
        r.broadcastAnimalLeave(id)
    }

    // 更新技能效果
    r.updateSkills()
}

// spawnRandomAnimal 生成随机动物
func (r *Room) spawnRandomAnimal() {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 随机选择动物类型
    animalType := defaultAnimalOrder[rand.Intn(len(defaultAnimalOrder))]

    // 随机选择路径
    path := PathManager.GetRandomPath()
    if path == nil {
        return
    }

    animal := &Animal{
        ID:        r.nextAnimalID,
        Type:      animalType,
        LineID:    path.ID,
        Position:  0,
        Speed:     getAnimalSpeed(animalType),
        State:     pb.EAnimalState_normal,
        RedBag:    rand.Float32() < 0.1, // 10%概率携带红包
        SpawnTime: time.Now(),
    }

    r.nextAnimalID++
    r.animals[animal.ID] = animal

    r.broadcastAnimalEnter(animal)
}

// broadcastAnimalEnter 广播动物进入
func (r *Room) broadcastAnimalEnter(animal *Animal) {
    route := &pb.PRoute{
        Id:       &animal.ID,
        Bet:      animal.Type.Enum(),
        LineId:   &animal.LineID,
        Point:    proto.Uint32(uint32(animal.Position * 100)),
        RedState: &animal.RedBag,
        Status:   animal.State.Enum(),
    }

    msg := &pb.M_1887Toc{
        Animal: []*pb.PRoute{route},
    }

    for _, session := range r.players {
        session.SendMessage(1887, msg)
    }
}

// broadcastAnimalLeave 广播动物离开
func (r *Room) broadcastAnimalLeave(id uint32) {
    msg := &pb.M_1888Toc{
        Id: &id,
    }

    for _, session := range r.players {
        session.SendMessage(1888, msg)
    }
}

// updateSkills 更新技能效果
func (r *Room) updateSkills() {
    now := time.Now()

    for _, session := range r.players {
        for skillType, endTime := range session.SkillEnds {
            if now.After(endTime) {
                delete(session.SkillEnds, skillType)
                // 通知客户端技能结束
                session.SendSkillEnd(skillType)
            }
        }
    }
}
```

### 3.2 第二阶段：实现游戏核心机制（第2周）

#### 3.2.1 碰撞检测系统 (collision.go)
```go
package animal

import "math"

// Point 二维点
type Point struct {
    X, Y float32
}

// BoundingBox 边界盒
type BoundingBox struct {
    Min, Max Point
}

// Ray 射线
type Ray struct {
    Origin    Point
    Direction Point
}

// CollisionSystem 碰撞检测系统
type CollisionSystem struct{}

// CheckRayBoxIntersection 射线与边界盒相交检测
func (cs *CollisionSystem) CheckRayBoxIntersection(ray Ray, box BoundingBox) bool {
    // AABB射线相交算法
    tmin := (box.Min.X - ray.Origin.X) / ray.Direction.X
    tmax := (box.Max.X - ray.Origin.X) / ray.Direction.X

    if tmin > tmax {
        tmin, tmax = tmax, tmin
    }

    tymin := (box.Min.Y - ray.Origin.Y) / ray.Direction.Y
    tymax := (box.Max.Y - ray.Origin.Y) / ray.Direction.Y

    if tymin > tymax {
        tymin, tymax = tymax, tymin
    }

    if tmin > tymax || tymin > tmax {
        return false
    }

    if tymin > tmin {
        tmin = tymin
    }

    if tymax < tmax {
        tmax = tymax
    }

    return tmax >= 0
}

// GetNearbyAnimals 获取范围内的动物
func (cs *CollisionSystem) GetNearbyAnimals(center Point, radius float32, animals map[uint32]*Animal) []*Animal {
    result := []*Animal{}
    radiusSq := radius * radius

    for _, animal := range animals {
        dx := animal.X - center.X
        dy := animal.Y - center.Y
        distSq := dx*dx + dy*dy

        if distSq <= radiusSq {
            result = append(result, animal)
        }
    }

    return result
}

// CheckHit 检查射击是否命中
func (cs *CollisionSystem) CheckHit(shooter *PlayerSession, animal *Animal) bool {
    // 获取玩家瞄准角度
    angle := shooter.GetAimAngle()

    // 创建射线
    ray := Ray{
        Origin: Point{
            X: shooter.GetX(),
            Y: shooter.GetY(),
        },
        Direction: Point{
            X: float32(math.Cos(float64(angle))),
            Y: float32(math.Sin(float64(angle))),
        },
    }

    // 获取动物碰撞盒
    box := animal.GetBoundingBox()

    // 检测相交
    return cs.CheckRayBoxIntersection(ray, box)
}
```

#### 3.2.2 赔率计算系统 (odds.go)
```go
package animal

import (
    "math/rand"
    "sync"
    "github.com/wfunc/slot-game/internal/pb"
)

// OddsCalculator 赔率计算器
type OddsCalculator struct {
    mu       sync.RWMutex
    baseOdds map[pb.EAnimal][2]float32 // [最小, 最大]
}

// NewOddsCalculator 创建赔率计算器
func NewOddsCalculator() *OddsCalculator {
    return &OddsCalculator{
        baseOdds: map[pb.EAnimal][2]float32{
            pb.EAnimal_turtle:   {1.2, 2.0},
            pb.EAnimal_cock:     {1.5, 2.5},
            pb.EAnimal_dog:      {2.0, 3.0},
            pb.EAnimal_monkey:   {2.5, 3.5},
            pb.EAnimal_horse:    {3.0, 4.0},
            pb.EAnimal_ox:       {3.5, 4.5},
            pb.EAnimal_panda:    {5.0, 8.0},
            pb.EAnimal_hippo:    {4.0, 6.0},
            pb.EAnimal_lion:     {4.5, 7.0},
            pb.EAnimal_elephant: {6.0, 10.0},
            pb.EAnimal_pikachu:  {8.0, 15.0}, // 特殊动物，高赔率
            pb.EAnimal_bomber:   {10.0, 20.0}, // 特殊动物，超高赔率
            pb.EAnimal_tiger:    {5.0, 8.0},
            pb.EAnimal_sheep:    {2.0, 3.0},
            pb.EAnimal_bear:     {4.0, 6.0},
            pb.EAnimal_tuzi:     {1.8, 2.8},
            pb.EAnimal_lv:       {2.2, 3.2},
            pb.EAnimal_baozi:    {6.0, 9.0},
            pb.EAnimal_zhu:      {3.0, 4.5},
            pb.EAnimal_hema:     {4.0, 6.0},
        },
    }
}

// Calculate 计算实际赔率
func (oc *OddsCalculator) Calculate(
    animalType pb.EAnimal,
    session *PlayerSession,
) float32 {
    oc.mu.RLock()
    defer oc.mu.RUnlock()

    // 获取基础赔率范围
    odds, exists := oc.baseOdds[animalType]
    if !exists {
        return 1.0
    }

    // 在范围内随机
    baseOdds := odds[0] + rand.Float32()*(odds[1]-odds[0])

    // 应用技能加成
    multiplier := float32(1.0)
    if skill, exists := session.SkillEnds[pb.EAnimalSkillType_improve_odds]; exists {
        if !skill.IsZero() && time.Now().Before(skill) {
            multiplier = 1.5 // 倍率提升技能效果
        }
    }

    // 应用VIP加成
    vipBonus := 1.0 + float32(session.Player.VIP)*0.02 // VIP每级2%加成

    return baseOdds * multiplier * vipBonus
}

// GetDisplayOdds 获取显示用的赔率范围
func (oc *OddsCalculator) GetDisplayOdds() []*pb.PAnimalOdds {
    oc.mu.RLock()
    defer oc.mu.RUnlock()

    result := []*pb.PAnimalOdds{}

    for animal, odds := range oc.baseOdds {
        result = append(result, &pb.PAnimalOdds{
            Bet:  animal.Enum(),
            Odds: []uint32{uint32(odds[0] * 10), uint32(odds[1] * 10)},
        })
    }

    return result
}
```

#### 3.2.3 技能系统 (skill.go)
```go
package animal

import (
    "errors"
    "time"
    "github.com/wfunc/slot-game/internal/pb"
)

var (
    ErrSkillOnCooldown = errors.New("skill is on cooldown")
    ErrSkillNoStock    = errors.New("skill has no stock")
)

// SkillManager 技能管理器
type SkillManager struct {
    prices map[pb.EAnimalSkillType]uint32
}

// NewSkillManager 创建技能管理器
func NewSkillManager() *SkillManager {
    return &SkillManager{
        prices: map[pb.EAnimalSkillType]uint32{
            pb.EAnimalSkillType_ice:          100,
            pb.EAnimalSkillType_locking:      500,
            pb.EAnimalSkillType_improve_odds: 1000,
        },
    }
}

// UseSkill 使用技能
func (sm *SkillManager) UseSkill(
    session *PlayerSession,
    skillType pb.EAnimalSkillType,
    room *Room,
) error {
    // 检查技能是否在冷却中
    if endTime, exists := session.SkillEnds[skillType]; exists {
        if time.Now().Before(endTime) {
            return ErrSkillOnCooldown
        }
    }

    // 检查技能库存
    skill, exists := session.Skills[skillType]
    if !exists || skill.Count == 0 {
        return ErrSkillNoStock
    }

    // 扣除技能数量
    skill.Count--

    // 应用技能效果
    switch skillType {
    case pb.EAnimalSkillType_ice:
        sm.applyIceSkill(session, room)

    case pb.EAnimalSkillType_locking:
        sm.applyLockingSkill(session)

    case pb.EAnimalSkillType_improve_odds:
        sm.applyOddsSkill(session)
    }

    // 设置技能持续时间
    duration := time.Duration(skill.Time) * time.Second
    session.SkillEnds[skillType] = time.Now().Add(duration)

    return nil
}

// applyIceSkill 应用冰冻技能
func (sm *SkillManager) applyIceSkill(session *PlayerSession, room *Room) {
    room.mu.Lock()
    defer room.mu.Unlock()

    // 获取瞄准的动物
    targetID := session.GetTarget()
    if animal, exists := room.animals[targetID]; exists {
        // 冻结动物
        animal.Freeze(5 * time.Second)

        // 广播冰冻效果
        room.broadcastSkillEffect(session.Player.ID, pb.EAnimalSkillType_ice, []uint32{targetID})
    }
}

// applyLockingSkill 应用锁定技能
func (sm *SkillManager) applyLockingSkill(session *PlayerSession) {
    // 设置自动瞄准标记
    session.AutoAim = true
    session.AutoAimUntil = time.Now().Add(30 * time.Second)
}

// applyOddsSkill 应用倍率提升技能
func (sm *SkillManager) applyOddsSkill(session *PlayerSession) {
    // 倍率提升效果已在赔率计算中处理
    session.OddsMultiplier = 1.5
    session.OddsBoostUntil = time.Now().Add(60 * time.Second)
}

// BuySkill 购买技能
func (sm *SkillManager) BuySkill(
    player *Player,
    skillType pb.EAnimalSkillType,
    count uint32,
) error {
    price := sm.prices[skillType]
    totalCost := uint64(price * count)

    if player.Balance < totalCost {
        return ErrInsufficientFunds
    }

    // 扣除金币
    player.Balance -= totalCost

    // 增加技能库存
    if skill, exists := player.Skills[skillType]; exists {
        skill.Count += count
    } else {
        player.Skills[skillType] = &PlayerSkill{
            Type:  skillType,
            Value: 1,
            Count: count,
            Time:  getSkillDuration(skillType),
        }
    }

    return nil
}

func getSkillDuration(skillType pb.EAnimalSkillType) uint32 {
    switch skillType {
    case pb.EAnimalSkillType_ice:
        return 5
    case pb.EAnimalSkillType_locking:
        return 30
    case pb.EAnimalSkillType_improve_odds:
        return 60
    default:
        return 10
    }
}
```

### 3.3 第三阶段：特殊效果与优化（第3周）

#### 3.3.1 特殊动物效果 (effect.go)
```go
package animal

import (
    "github.com/wfunc/slot-game/internal/pb"
)

// SpecialEffect 特殊效果接口
type SpecialEffect interface {
    Trigger(room *Room, killer *PlayerSession, target *Animal) []*Animal
}

// PikachuLightning 皮卡丘闪电链
type PikachuLightning struct {
    Radius    float32
    MaxChain  int
    Damage    float32
}

// Trigger 触发闪电链
func (pl *PikachuLightning) Trigger(room *Room, killer *PlayerSession, target *Animal) []*Animal {
    affected := []*Animal{}
    collision := &CollisionSystem{}

    // 找到附近的动物
    center := Point{X: target.X, Y: target.Y}
    nearbyAnimals := collision.GetNearbyAnimals(center, pl.Radius, room.animals)

    // 限制连锁数量
    chainCount := 0
    for _, animal := range nearbyAnimals {
        if animal.ID == target.ID {
            continue
        }

        affected = append(affected, animal)
        chainCount++

        if chainCount >= pl.MaxChain {
            break
        }
    }

    return affected
}

// BomberExplosion 炸弹人全屏爆炸
type BomberExplosion struct {
    Radius float32
}

// Trigger 触发全屏爆炸
func (be *BomberExplosion) Trigger(room *Room, killer *PlayerSession, target *Animal) []*Animal {
    affected := []*Animal{}

    // 全屏爆炸，影响所有动物
    for _, animal := range room.animals {
        if animal.ID != target.ID {
            affected = append(affected, animal)
        }
    }

    return affected
}

// EffectManager 特效管理器
type EffectManager struct {
    effects map[pb.EAnimal]SpecialEffect
}

// NewEffectManager 创建特效管理器
func NewEffectManager() *EffectManager {
    return &EffectManager{
        effects: map[pb.EAnimal]SpecialEffect{
            pb.EAnimal_pikachu: &PikachuLightning{
                Radius:   200,
                MaxChain: 5,
                Damage:   0.5,
            },
            pb.EAnimal_bomber: &BomberExplosion{
                Radius: 1000,
            },
        },
    }
}

// ProcessKill 处理击杀及特殊效果
func (em *EffectManager) ProcessKill(
    room *Room,
    killer *PlayerSession,
    target *Animal,
) []*Animal {
    // 检查是否有特殊效果
    if effect, exists := em.effects[target.Type]; exists {
        return effect.Trigger(room, killer, target)
    }

    return []*Animal{}
}
```

#### 3.3.2 彩金池系统 (jackpot.go)
```go
package animal

import (
    "math/rand"
    "sync"
    "time"
)

// JackpotPool 彩金池
type JackpotPool struct {
    mu              sync.RWMutex
    current         uint64
    minTrigger      uint64
    triggerChance   float32
    contributionRate float32
}

// NewJackpotPool 创建彩金池
func NewJackpotPool() *JackpotPool {
    return &JackpotPool{
        current:          5000000, // 初始500万
        minTrigger:       1000000, // 最低100万才能触发
        triggerChance:    0.001,   // 0.1%触发概率
        contributionRate: 0.01,     // 1%贡献率
    }
}

// Contribute 贡献彩金
func (jp *JackpotPool) Contribute(amount uint64) {
    jp.mu.Lock()
    defer jp.mu.Unlock()

    contribution := uint64(float64(amount) * float64(jp.contributionRate))
    jp.current += contribution
}

// TryTrigger 尝试触发彩金
func (jp *JackpotPool) TryTrigger() (bool, uint64) {
    jp.mu.Lock()
    defer jp.mu.Unlock()

    // 检查最低触发金额
    if jp.current < jp.minTrigger {
        return false, 0
    }

    // 随机触发
    if rand.Float32() > jp.triggerChance {
        return false, 0
    }

    // 计算奖励金额（50%-100%的彩金池）
    ratio := 0.5 + rand.Float64()*0.5
    reward := uint64(float64(jp.current) * ratio)

    // 扣除彩金池
    jp.current -= reward

    // 确保彩金池不会为0
    if jp.current < 100000 {
        jp.current = 100000
    }

    return true, reward
}

// GetCurrent 获取当前彩金池金额
func (jp *JackpotPool) GetCurrent() uint64 {
    jp.mu.RLock()
    defer jp.mu.RUnlock()
    return jp.current
}

// JackpotRecord 彩金记录
type JackpotRecord struct {
    ID       uint64
    PlayerID uint32
    Name     string
    Icon     string
    Amount   uint64
    Time     time.Time
    Animal   pb.EAnimal
}

// JackpotHistory 彩金历史记录
type JackpotHistory struct {
    mu      sync.RWMutex
    records []*JackpotRecord
    maxSize int
}

// NewJackpotHistory 创建历史记录
func NewJackpotHistory() *JackpotHistory {
    return &JackpotHistory{
        records: make([]*JackpotRecord, 0, 100),
        maxSize: 100,
    }
}

// AddRecord 添加记录
func (jh *JackpotHistory) AddRecord(record *JackpotRecord) {
    jh.mu.Lock()
    defer jh.mu.Unlock()

    jh.records = append(jh.records, record)

    // 限制记录数量
    if len(jh.records) > jh.maxSize {
        jh.records = jh.records[1:]
    }
}

// GetRecent 获取最近记录
func (jh *JackpotHistory) GetRecent(count int) []*JackpotRecord {
    jh.mu.RLock()
    defer jh.mu.RUnlock()

    start := len(jh.records) - count
    if start < 0 {
        start = 0
    }

    return jh.records[start:]
}
```

## 四、实施步骤

### 第1步：完善基础架构（今天）
```bash
# 1. 创建新文件
touch internal/game/animal/{room.go,animal.go,path.go}

# 2. 实现基础动物系统
# 3. 实现路径系统
# 4. 完善房间逻辑
```

### 第2步：测试基础功能（明天）
```go
// 创建测试文件
// internal/game/animal/room_test.go

func TestRoomUpdate(t *testing.T) {
    room := NewRoom(pb.EZooType_civilian)
    room.Start()

    // 测试动物生成
    time.Sleep(5 * time.Second)

    room.mu.RLock()
    animalCount := len(room.animals)
    room.mu.RUnlock()

    assert.Greater(t, animalCount, 0, "Should have spawned animals")
}
```

### 第3步：实现核心游戏机制
- 碰撞检测
- 赔率计算
- 技能系统

### 第4步：添加特殊效果
- 皮卡丘闪电链
- 炸弹人爆炸
- 其他特殊动物效果

### 第5步：性能优化
- 对象池
- 批量消息
- 并发优化

## 五、配置文件

创建 `config/animal_game.yaml`:
```yaml
game:
  update_interval: 100ms
  spawn_interval: 3s
  max_animals_per_room: 50

animals:
  turtle:
    speed: 0.05
    odds: [1.2, 2.0]
    size: 30

  panda:
    speed: 0.03
    odds: [5.0, 8.0]
    size: 40

  pikachu:
    speed: 0.08
    odds: [8.0, 15.0]
    size: 35
    special: lightning_chain

  bomber:
    speed: 0.06
    odds: [10.0, 20.0]
    size: 35
    special: full_screen_explosion

paths:
  - id: 1
    points:
      - {x: 0, y: 100}
      - {x: 200, y: 100}
      - {x: 400, y: 200}
      - {x: 600, y: 200}
      - {x: 800, y: 100}

  - id: 2
    points:
      - {x: 0, y: 300}
      - {x: 300, y: 300}
      - {x: 500, y: 400}
      - {x: 800, y: 400}

skills:
  ice:
    price: 100
    duration: 5s
    cooldown: 10s

  locking:
    price: 500
    duration: 30s
    cooldown: 60s

  improve_odds:
    price: 1000
    duration: 60s
    cooldown: 120s
    multiplier: 1.5

jackpot:
  initial: 5000000
  contribution_rate: 0.01
  trigger_chance: 0.001
  min_trigger: 1000000
```

## 六、数据库迁移

创建数据库迁移文件：
```sql
-- migrations/001_animal_game.sql

-- 玩家技能库存表
CREATE TABLE player_skills (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL,
    skill_type INTEGER NOT NULL,
    count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 游戏记录表
CREATE TABLE animal_game_records (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL,
    room_type INTEGER NOT NULL,
    animal_type INTEGER NOT NULL,
    bet_amount INTEGER NOT NULL,
    win_amount INTEGER NOT NULL,
    odds DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 彩金记录表
CREATE TABLE jackpot_records (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL,
    amount BIGINT NOT NULL,
    animal_type INTEGER,
    room_type INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_player_skills_player ON player_skills(player_id);
CREATE INDEX idx_game_records_player ON animal_game_records(player_id);
CREATE INDEX idx_game_records_time ON animal_game_records(created_at);
CREATE INDEX idx_jackpot_records_amount ON jackpot_records(amount);
```

## 七、下一步行动计划

### 立即执行（今天）
1. ✅ 创建缺失的核心文件
2. ✅ 实现动物系统和路径系统
3. ✅ 完善房间逻辑
4. ✅ 添加基础测试

### 明天任务
1. 实现碰撞检测系统
2. 完成赔率计算逻辑
3. 添加技能系统
4. 测试基础游戏流程

### 本周完成
1. 特殊动物效果
2. 彩金池系统
3. 性能优化
4. 完整测试

### 注意事项
1. **并发安全**：所有共享数据必须加锁
2. **内存管理**：使用对象池复用动物对象
3. **性能监控**：添加性能指标收集
4. **错误处理**：完善的错误处理和恢复机制
5. **日志记录**：关键操作都要记录日志

## 八、测试策略

### 单元测试
- 路径系统测试
- 碰撞检测测试
- 赔率计算测试
- 技能效果测试

### 集成测试
- 房间完整流程测试
- WebSocket通信测试
- 并发压力测试

### 性能测试
- 100个玩家同时在线
- 每房间50个动物
- 消息广播延迟 < 100ms