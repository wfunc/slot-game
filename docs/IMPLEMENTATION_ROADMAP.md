# 🎮 Slot-Animal 游戏实现路线图

基于现有代码分析，项目已完成大部分基础功能，现提供后续开发建议。

## 📊 现有实现分析

### ✅ 已完成模块
1. **WebSocket通讯层** (90%完成)
   - ✓ Protobuf编解码器
   - ✓ 消息路由机制
   - ✓ 会话管理
   
2. **Slot游戏引擎** (85%完成)
   - ✓ 金色Wild消除算法
   - ✓ 1024线匹配
   - ✓ 连锁消除机制
   - ✓ JP池管理

3. **数据层** (80%完成)
   - ✓ GORM数据模型
   - ✓ 游戏记录存储
   - ✓ 钱包系统
   - ✓ JP累积机制

---

## 🚀 推荐实现方案：Slot→Animal桥接

### 架构设计
```
[Slot游戏] → 特殊符号触发 → [前端判断] → 切换到 → [Animal游戏]
```

### 实现步骤

## 第一阶段：完善Slot触发机制（2天）

### 1. 定义Animal触发符号
```go
// internal/game/slot/symbols.go
const (
    SYMBOL_ANIMAL_WILD = 8  // 动物Wild符号
    SYMBOL_ANIMAL_BONUS = 9 // 动物Bonus符号
)

type AnimalTriggerConfig struct {
    RequiredCount    int     // 需要的符号数量
    TriggerSymbols   []int   // 触发符号ID列表
    FreeRoundsBase   int     // 基础免费次数
    MultiplierBase   float64 // 基础倍率
}
```

### 2. 扩展Slot结果
```go
// internal/game/slot/types.go
type SlotResultExt struct {
    *GoldenWildResult
    
    // Animal触发信息
    TriggerAnimal   bool              `json:"trigger_animal"`
    AnimalTrigger   *AnimalTriggerData `json:"animal_trigger,omitempty"`
}

type AnimalTriggerData struct {
    Type        string  `json:"type"`
    FreeRounds  int     `json:"free_rounds"`
    Multiplier  float64 `json:"multiplier"`
    BonusPool   int64   `json:"bonus_pool"`
}
```

### 3. 修改slot_handler.go
```go
// 在handleStartGame方法中添加触发检测
func (h *SlotHandler) handleStartGame(session *SlotSessionSimple, data []byte) {
    // ... 现有代码 ...
    
    // 执行游戏
    result, err := engine.SpinWithGoldenWild(ctx, spinReq)
    
    // 检查Animal触发
    animalTrigger := h.checkAnimalTrigger(result)
    if animalTrigger != nil {
        // 在响应中添加触发信息
        resp.TriggerBonus = proto.Bool(true)
        resp.BonusType = proto.String("animal_game")
        resp.BonusData = &pb.PBridgeData{
            FreeRounds: proto.Uint32(uint32(animalTrigger.FreeRounds)),
            Multiplier: proto.Float32(float32(animalTrigger.Multiplier)),
        }
    }
}
```

---

## 第二阶段：实现Animal游戏模块（3天）

### 1. 创建Animal游戏处理器
```go
// internal/websocket/animal_handler.go
type AnimalHandler struct {
    sessions    map[string]*AnimalSession
    db          *gorm.DB
    walletRepo  repository.WalletRepository
    mu          sync.RWMutex
}

type AnimalSession struct {
    ID          string
    UserID      uint
    Conn        *websocket.Conn
    Codec       *ProtobufCodec
    
    // Animal游戏状态
    RoomType    pb.EZooType
    Animals     []*AnimalEntity
    Skills      []*SkillState
    FreeRounds  int
    Multiplier  float64
    TotalWin    int64
}
```

### 2. 实现Animal游戏逻辑
```go
// internal/game/animal/engine.go
type AnimalGameEngine struct {
    config      *AnimalConfig
    animalPool  *AnimalPool
    skillSystem *SkillSystem
}

func (e *AnimalGameEngine) FireBullet(ctx context.Context, req *FireRequest) (*FireResult, error) {
    // 发射子弹逻辑
}

func (e *AnimalGameEngine) HitAnimal(ctx context.Context, req *HitRequest) (*HitResult, error) {
    // 击中动物逻辑
}
```

---

## 第三阶段：前端切换控制（2天）

### 1. 前端游戏管理器
```javascript
// static/js/game_manager.js
class GameManager {
    constructor() {
        this.currentGame = 'slot';
        this.slotHandler = new SlotGameHandler();
        this.animalHandler = new AnimalGameHandler();
    }
    
    handleSlotResult(result) {
        if (result.trigger_bonus && result.bonus_type === 'animal_game') {
            this.switchToAnimal(result.bonus_data);
        }
    }
    
    switchToAnimal(data) {
        // 保存Slot状态
        this.slotHandler.saveState();
        
        // 切换UI
        this.showTransition('slot_to_animal');
        
        // 初始化Animal游戏
        this.animalHandler.init({
            freeRounds: data.free_rounds,
            multiplier: data.multiplier
        });
        
        // 发送进入Animal房间请求
        this.sendMessage(1801, { type: 6 }); // 单人场
    }
}
```

### 2. 消息路由扩展
```javascript
// static/js/message_router.js
class MessageRouter {
    route(msgId, data) {
        // Slot消息 (1900-1999)
        if (msgId >= 1900 && msgId < 2000) {
            return this.slotHandler.handle(msgId, data);
        }
        
        // Animal消息 (1800-1899)
        if (msgId >= 1800 && msgId < 1900) {
            return this.animalHandler.handle(msgId, data);
        }
        
        // 配置消息 (2000-2099)
        if (msgId >= 2000 && msgId < 2100) {
            return this.configHandler.handle(msgId, data);
        }
    }
}
```

---

## 第四阶段：数据桥接与状态管理（1天）

### 1. 会话状态共享
```go
// internal/session/manager.go
type SessionManager struct {
    sessions map[string]*UnifiedSession
    mu       sync.RWMutex
}

type UnifiedSession struct {
    ID         string
    UserID     uint
    
    // 共享数据
    Balance    int64
    TotalWin   int64
    
    // 游戏特定状态
    SlotState  *SlotSessionState
    AnimalState *AnimalSessionState
    
    // 桥接数据
    BridgeData *BridgeData
}
```

### 2. 状态持久化
```go
// internal/repository/game_state.go
func (r *GameStateRepository) SaveBridgeData(userID uint, data *BridgeData) error {
    return r.db.Create(&models.GameBridge{
        UserID:     userID,
        FromGame:   data.FromGame,
        ToGame:     data.ToGame,
        TriggerData: data.TriggerData,
        CreatedAt:  time.Now(),
    }).Error
}
```

---

## 🎯 关键实现要点

### 1. 符号配置
- 在Slot中定义特殊的Animal触发符号（ID=8或9）
- 配置触发条件（3个相同符号触发普通，5个触发超级）

### 2. 前端切换
- 前端检测`trigger_bonus`字段
- 显示过渡动画
- 切换WebSocket消息处理器

### 3. 数据传递
- 通过`BridgeData`在游戏间传递信息
- 保持用户余额和统计数据一致性

### 4. 独立开发
- Slot和Animal完全独立的消息处理
- 各自的游戏引擎和状态管理
- 通过前端协调切换

---

## 📅 开发时间表

| 阶段 | 任务 | 预计时间 | 优先级 |
|------|------|----------|--------|
| 1 | 完善Slot触发机制 | 2天 | P0 |
| 2 | 实现Animal游戏模块 | 3天 | P0 |
| 3 | 前端切换控制 | 2天 | P1 |
| 4 | 数据桥接与状态管理 | 1天 | P1 |
| 5 | 测试与优化 | 2天 | P2 |

**总计：10天**

---

## ✅ 下一步行动

### 立即可做：
1. 在Slot符号配置中添加Animal触发符号
2. 扩展`pb.M_1902Toc`消息，添加触发字段
3. 创建`animal_handler.go`基础框架

### 需要决策：
1. Animal游戏是否需要独立的WebSocket端口？
2. 是否需要在数据库中记录游戏切换历史？
3. Animal游戏结束后的奖励如何结算？

---

## 🔧 技术建议

1. **保持模块独立**：Slot和Animal使用独立的handler和engine
2. **复用基础设施**：共享WebSocket连接、Codec、数据库连接
3. **前端主导切换**：由前端控制游戏场景切换，后端只提供触发信息
4. **渐进式开发**：先实现基础功能，再添加高级特性

这样的设计既保持了游戏模块的独立性，又能实现流畅的游戏切换体验。