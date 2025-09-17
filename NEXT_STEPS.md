# 🎯 疯狂动物园游戏 - 下一步执行指南

## 📋 当前状态分析

### ✅ 已完成
- 基础项目结构搭建
- WebSocket 通信层
- Protobuf 消息定义
- 基础房间管理器
- 部分游戏逻辑

### ⚠️ 需要改进
- 动物移动和路径系统不完整
- 缺少碰撞检测
- 技能系统未实现
- 彩金池系统缺失
- 性能优化待做

## 🚀 立即执行步骤（今天）

### Step 1: 提交当前代码
```bash
cd /Users/mini/Documents/GitHub/wfunc/slot-game
git add .
git commit -m "feat: add animal game core structure and room logic"
git push
```

### Step 2: 创建缺失的核心文件
```bash
# 创建动物系统
cat > internal/game/animal/animal.go << 'EOF'
package animal

import (
    "sync"
    "time"
    "github.com/wfunc/slot-game/internal/pb"
)

type Animal struct {
    ID        uint32
    Type      pb.EAnimal
    LineID    uint32
    Position  float32
    Speed     float32
    State     pb.EAnimalState
    RedBag    bool
    SpawnTime time.Time

    X, Y      float32
    Direction float32

    Frozen      bool
    FreezeUntil time.Time

    mu sync.RWMutex
}

func (a *Animal) Update(deltaTime float32) {
    a.mu.Lock()
    defer a.mu.Unlock()

    if a.Frozen && time.Now().After(a.FreezeUntil) {
        a.Frozen = false
        a.State = pb.EAnimalState_normal
    }

    if !a.Frozen {
        a.Position += a.Speed * deltaTime
    }
}

func (a *Animal) IsAlive() bool {
    return a.Position < 1.0
}
EOF

# 创建路径系统
cat > internal/game/animal/path.go << 'EOF'
package animal

import "math"

type Path struct {
    ID     uint32
    Points []Point
    Length float32
}

type Point struct {
    X, Y float32
}

func (p *Path) GetPosition(progress float32) (x, y float32) {
    if len(p.Points) == 0 {
        return 0, 0
    }

    if progress <= 0 {
        return p.Points[0].X, p.Points[0].Y
    }

    if progress >= 1 {
        last := p.Points[len(p.Points)-1]
        return last.X, last.Y
    }

    // 简化实现：线性插值第一段和最后一段
    idx := int(progress * float32(len(p.Points)-1))
    if idx >= len(p.Points)-1 {
        idx = len(p.Points) - 2
    }

    localProgress := progress * float32(len(p.Points)-1) - float32(idx)
    x = p.Points[idx].X + (p.Points[idx+1].X-p.Points[idx].X)*localProgress
    y = p.Points[idx].Y + (p.Points[idx+1].Y-p.Points[idx].Y)*localProgress

    return x, y
}

var DefaultPaths = []*Path{
    {
        ID: 1,
        Points: []Point{{0, 100}, {200, 100}, {400, 200}, {600, 200}, {800, 100}},
        Length: 800,
    },
    {
        ID: 2,
        Points: []Point{{0, 300}, {300, 300}, {500, 400}, {800, 400}},
        Length: 800,
    },
    {
        ID: 3,
        Points: []Point{{0, 200}, {400, 300}, {800, 200}},
        Length: 800,
    },
}
EOF
```

### Step 3: 运行测试
```bash
# 给脚本添加执行权限
chmod +x scripts/run_animal_game.sh

# 运行测试
go test ./internal/game/animal/... -v
```

## 📅 本周任务计划

### 周一-周二：完善核心系统
- [ ] 完成动物生成和移动逻辑
- [ ] 实现路径系统
- [ ] 添加基础碰撞检测
- [ ] 完成赔率计算

### 周三-周四：实现游戏特性
- [ ] 实现技能系统（冰冻、锁定、倍率提升）
- [ ] 添加特殊动物效果（皮卡丘闪电、炸弹人爆炸）
- [ ] 实现彩金池系统
- [ ] 完善消息推送

### 周五：优化和测试
- [ ] 性能优化（对象池、批量消息）
- [ ] 添加单元测试
- [ ] 压力测试（100玩家并发）
- [ ] 修复发现的问题

## 🔧 技术实现要点

### 1. 并发控制
```go
// 使用 channel 进行房间消息传递
type RoomController struct {
    msgChan chan RoomMessage
    ticker  *time.Ticker
}

func (rc *RoomController) Run() {
    for {
        select {
        case msg := <-rc.msgChan:
            rc.handleMessage(msg)
        case <-rc.ticker.C:
            rc.updateRoom()
        }
    }
}
```

### 2. 性能优化
```go
// 使用对象池复用动物对象
var animalPool = sync.Pool{
    New: func() interface{} {
        return &Animal{}
    },
}

// 批量发送消息
type MessageBatcher struct {
    messages []Message
    mu       sync.Mutex
}

func (mb *MessageBatcher) Flush() {
    mb.mu.Lock()
    defer mb.mu.Unlock()
    // 批量发送所有消息
}
```

### 3. 定时器管理
```go
// 统一管理所有定时器
type TimerManager struct {
    timers map[string]*time.Timer
    mu     sync.Mutex
}
```

## 🧪 测试命令

### 运行所有测试
```bash
go test ./... -v
```

### 运行基准测试
```bash
go test -bench=. ./internal/game/animal/...
```

### 运行覆盖率测试
```bash
go test -cover ./internal/game/animal/...
```

## 📊 监控指标

需要监控的关键指标：
- 房间内动物数量
- 消息延迟（目标 <100ms）
- 内存使用
- CPU使用率
- 并发连接数

## ⚠️ 注意事项

1. **并发安全**：所有共享数据访问都要加锁
2. **内存泄漏**：及时清理断开的连接和过期数据
3. **错误恢复**：使用 recover 防止 panic 崩溃
4. **日志记录**：关键操作都要记录日志

## 🎯 今日目标

1. ✅ 提交当前代码
2. ✅ 创建核心文件（animal.go, path.go）
3. ✅ 运行基础测试
4. ✅ 修复发现的编译错误
5. ✅ 实现一个完整的游戏流程

## 💬 问题反馈

如遇到问题，可以：
1. 查看错误日志：`tail -f logs/animal-game.log`
2. 运行调试模式：`go run cmd/server/main.go -debug`
3. 查看文档：`docs/architecture/animal-game-implementation.md`

## 🚦 下一步行动

```bash
# 1. 先提交代码
git add . && git commit -m "feat: implement animal game core"

# 2. 创建分支进行开发
git checkout -b feature/animal-game

# 3. 开始实现核心功能
vim internal/game/animal/animal.go

# 4. 运行测试验证
go test ./internal/game/animal/...

# 5. 启动服务器测试
./scripts/run_animal_game.sh
```

---

💡 **提示**：建议按照顺序逐步实现，每完成一个模块就进行测试，确保稳定后再继续下一步。