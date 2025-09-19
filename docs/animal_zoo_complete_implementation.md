# 疯狂动物园完整功能移植实现文档

## 一、缺失功能清单

基于 Erlang 源码分析，当前 Go 实现缺失以下核心功能：

### 1. 房间类型系统
- **体验场 (free)**: 使用体验币，有任务系统
- **平民场 (civilian)**: 低门槛，100/500/1000下注档位
- **小资场 (petty)**: 中等门槛
- **富豪场 (rich)**: 高门槛，10000/50000/100000下注档位
- **黄金场 (gold)**: VIP专属
- **钻石场 (diamond)**: 顶级VIP
- **单人场 (single)**: 练习模式

### 2. 动态赔率系统
- 赔率范围而非固定值（如 [7,14] 实际是 0.7~1.4）
- 根据房间盈亏动态调整
- VIP加成计算

### 3. 特殊动物效果
- **皮卡丘闪电链**: 链式伤害，影响附近动物
- **炸弹人全屏爆炸**: AOE伤害，清屏效果
- 精确的概率计算和范围判定

### 4. 任务系统
- 动物园任务（zoo_task）
- 体验场任务（free_zoo_task）
- 每日任务统计
- 任务完成记录

### 5. 活动系统
- 准备阶段（pre_time）
- 活动阶段（work_time）
- 发奖阶段
- 排行榜系统
- 活动奖励配置

### 6. 红包系统
- 基于动物类型的红包概率
- 红包金额算法
- 红包与金豆转换

### 7. 彩金池系统
- 彩金累积机制
- 触发条件
- 中奖广播

### 8. 一击必杀功能
- 设置特定玩家必杀
- 时间限制
- 权限控制

### 9. 房间盈亏控制
- P值（投注总额）和W值（赢取总额）
- 动态控制机制
- 防止房间亏损过大

### 10. 冰冻系统优化
- 累计冰冻次数
- 冰冻时间叠加
- 群体冰冻效果

## 二、核心数据结构补充

### 赔率配置（从 Erlang 迁移）
```go
// 正式场赔率（需要除以10）
var AnimalOddsNormal = map[pb.EAnimal][2]int{
    pb.EAnimal_turtle:   {7, 14},     // 0.7-1.4
    pb.EAnimal_cock:     {12, 12},    // 1.2
    pb.EAnimal_dog:      {20, 20},    // 2.0
    pb.EAnimal_monkey:   {40, 40},    // 4.0
    pb.EAnimal_horse:    {60, 60},    // 6.0
    pb.EAnimal_tuzi:     {80, 80},    // 8.0
    pb.EAnimal_ox:       {100, 100},  // 10.0
    pb.EAnimal_panda:    {200, 200},  // 20.0
    pb.EAnimal_lv:       {300, 300},  // 30.0
    pb.EAnimal_baozi:    {400, 400},  // 40.0
    pb.EAnimal_zhu:      {600, 600},  // 60.0
    pb.EAnimal_hema:     {800, 800},  // 80.0
    pb.EAnimal_hippo:    {1000, 1000},// 100.0
    pb.EAnimal_pikachu:  {500, 500},  // 50.0
    pb.EAnimal_lion:     {2000, 2000},// 200.0
    pb.EAnimal_elephant: {10000, 10000},// 1000.0
    pb.EAnimal_bomber:   {0, 0},      // 特殊
}

// 体验场赔率（固定值）
var AnimalOddsFree = map[pb.EAnimal]int{
    pb.EAnimal_turtle:   20,
    pb.EAnimal_cock:     40,
    pb.EAnimal_dog:      60,
    pb.EAnimal_monkey:   100,
    pb.EAnimal_horse:    200,
    pb.EAnimal_ox:       300,
    pb.EAnimal_panda:    500,
    pb.EAnimal_hippo:    1000,
    pb.EAnimal_lion:     2000,
    pb.EAnimal_elephant: 3000,
}
```

### 房间配置
```go
var RoomTypeConfig = map[pb.EZooType]*RoomConfig{
    pb.EZooType_free: {
        BetValues: []uint32{100, 500, 1000},
        MinVIP:    0,
        MaxPlayer: 100,
        UseFreeGold: true,
    },
    pb.EZooType_civilian: {
        BetValues: []uint32{100, 500, 1000},
        MinVIP:    0,
        MaxPlayer: 100,
    },
    pb.EZooType_petty: {
        BetValues: []uint32{1000, 5000, 10000},
        MinVIP:    1,
        MaxPlayer: 80,
    },
    pb.EZooType_rich: {
        BetValues: []uint32{10000, 50000, 100000},
        MinVIP:    5,
        MaxPlayer: 50,
    },
}
```

## 三、功能实现细节

### 1. 皮卡丘闪电链效果
```go
// 从 Erlang calcPikachu 函数迁移
func (r *Room) ProcessPikachuLightning(targetID uint32, betVal uint32, roleID uint32) *BetOutcome {
    // 1. 首先击杀目标动物
    // 2. 查找附近动物（同一条线路或距离范围内）
    // 3. 按概率触发连锁（30%概率）
    // 4. 最多连锁3-5只动物
    // 5. 连锁伤害递减
}
```

### 2. 炸弹人全屏爆炸
```go
// 从 Erlang calcBomber 函数迁移
func (r *Room) ProcessBomberExplosion(targetID uint32, betVal uint32, roleID uint32) *BetOutcome {
    // 1. 触发全屏爆炸
    // 2. 清除所有普通动物
    // 3. 特殊动物（皮卡丘、炸弹人）免疫
    // 4. 计算总奖励
}
```

### 3. 动态赔率计算
```go
func (r *Room) CalculateDynamicOdds(animal pb.EAnimal, vipLevel uint32) float32 {
    // 1. 获取基础赔率范围
    // 2. 根据房间盈亏调整
    // 3. 应用VIP加成
    // 4. 随机浮动±20%

    baseRange := AnimalOddsNormal[animal]
    minOdds := float32(baseRange[0]) / 10.0
    maxOdds := float32(baseRange[1]) / 10.0

    // 在范围内随机
    odds := minOdds + rand.Float32()*(maxOdds-minOdds)

    // VIP加成（每级VIP增加2%赔率）
    vipBonus := 1.0 + float32(vipLevel)*0.02

    return odds * vipBonus
}
```

### 4. 房间盈亏控制
```go
type RoomProfitControl struct {
    TotalBet uint64  // P值：总投注
    TotalWin uint64  // W值：总赢取

    // 控制策略
    // 当 W - P > 10000000 时，降低中奖率
    // 当 P - W > 10000000 时，提高中奖率
}

func (r *Room) ShouldAdjustWinRate() bool {
    diff := int64(r.profitControl.TotalWin) - int64(r.profitControl.TotalBet)
    if diff > 10000000 {
        // 房间亏损过大，30分之1概率不中奖
        return rand.Intn(30) == 0
    }
    return false
}
```

### 5. 任务系统
```go
type ZooTask struct {
    ID          uint32
    Type        string // "daily", "weekly", "achievement"
    Target      uint32 // 目标数量
    Progress    uint32 // 当前进度
    Reward      uint64 // 奖励金额
    Status      string // "active", "completed", "claimed"
}

type TaskManager struct {
    dailyTasks  map[uint32]*ZooTask
    weeklyTasks map[uint32]*ZooTask
    achievements map[uint32]*ZooTask
}
```

### 6. 活动系统
```go
type Activity struct {
    ID         uint32
    PreTime    uint32 // 准备时间（秒）
    WorkTime   uint32 // 活动时间（秒）
    MaxPlayers uint32
    BetValues  []uint32
    Status     string // "prepare", "active", "settlement", "idle"
    Rankings   []*PlayerRank
    Rewards    []*RankReward
}

type ActivityManager struct {
    currentActivity *Activity
    scheduler       *time.Timer
}
```

### 7. 彩金池系统
```go
type JackpotPool struct {
    CurrentAmount uint64
    Contributors  map[uint32]uint64 // playerID -> contribution
    TriggerRatio  float32 // 触发概率
}

func (j *JackpotPool) Accumulate(amount uint64) {
    // 每笔下注的1%进入彩金池
    contribution := amount / 100
    j.CurrentAmount += contribution
}

func (j *JackpotPool) TryTrigger(playerID uint32) (bool, uint64) {
    // 击杀大象有机会触发彩金
    if rand.Float32() < j.TriggerRatio {
        reward := j.CurrentAmount
        j.CurrentAmount = 0
        return true, reward
    }
    return false, 0
}
```

### 8. 红包系统优化
```go
func CalculateRedPacket(animal pb.EAnimal, winAmount uint32) (redBag uint32, gold uint32) {
    // 基于动物类型的红包概率和金额
    redBagChance := map[pb.EAnimal]float32{
        pb.EAnimal_panda:    0.3,
        pb.EAnimal_elephant: 0.5,
        pb.EAnimal_pikachu:  0.2,
        // ...
    }

    if rand.Float32() < redBagChance[animal] {
        // 红包金额为赢取金额的5%-10%
        redBag = uint32(float32(winAmount) * (0.05 + rand.Float32()*0.05))
        // 红包转金豆（1:1200）
        gold = winAmount + redBag*1200
        return redBag, gold
    }

    return 0, winAmount
}
```

## 四、实现优先级

1. **P0 - 立即实现**
   - 房间类型系统
   - 动态赔率系统
   - 完整的特殊动物效果

2. **P1 - 重要功能**
   - 任务系统
   - 房间盈亏控制
   - 红包系统优化

3. **P2 - 增强功能**
   - 活动系统
   - 彩金池系统
   - 一击必杀功能

## 五、测试要点

1. 赔率计算准确性
2. 特殊动物效果触发
3. 房间盈亏平衡
4. 并发安全性
5. 消息同步正确性