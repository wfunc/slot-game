# 游戏系统集成指南

## 概述
本文档介绍如何集成和使用游戏状态机、会话管理器和游戏服务。

## 架构设计
基于解耦架构，游戏逻辑独立于硬件接口：
- **游戏核心**：老虎机引擎、状态机、会话管理
- **通信层**：WebSocket、MQTT（与前端/管理端通信）
- **硬件接口**：推币机、串口（预留接口，后期对接）

## 核心组件

### 1. 游戏状态机 (StateMachine)
管理游戏生命周期和状态转换。

#### 状态流程
```
待机(idle) → 准备(ready) → 转动中(spinning) → 
计算(calculating) → 中奖展示(winning) → 结算(settlement) → 待机
```

#### 使用示例
```go
// 创建状态机
sm := NewStateMachine(sessionID, userID, logger, persister)

// 设置回调
sm.OnStateChange(func(from, to GameState) {
    logger.Info("状态变更", zap.String("from", from), zap.String("to", to))
})

// 触发事件
err := sm.Trigger(ctx, "insert_coin")  // 投币
err := sm.Trigger(ctx, "start_spin")   // 开始转动
err := sm.Trigger(ctx, "stop_spin")    // 停止转动
```

### 2. 会话管理器 (SessionManager)
管理游戏会话的创建、恢复和清理。

#### 功能特性
- 会话创建和恢复
- 自动清理超时会话
- 状态持久化
- 并发安全

#### 使用示例
```go
// 创建会话管理器
config := &SessionConfig{
    Logger:         logger,
    DB:             db,
    SessionTimeout: 30 * time.Minute,
    MaxSessions:    1000,
}
sessionManager := NewSessionManager(config)

// 创建或恢复会话
session, err := sessionManager.RecoverOrCreateSession(ctx, sessionID, userID)

// 获取会话
session, err := sessionManager.GetSession(sessionID)

// 启动清理任务
sessionManager.StartCleanupTask(ctx, 5*time.Minute)
```

### 3. 游戏服务 (GameService)
业务逻辑层，处理游戏流程和用户交互。

#### 主要功能
- 开始游戏（扣除余额）
- 执行转动（生成结果）
- 结算游戏（发放奖金）
- 查询历史和统计

#### 使用示例
```go
// 创建游戏服务
config := &GameServiceConfig{
    DB:             db,
    Logger:         logger,
    SessionTimeout: 30 * time.Minute,
    MaxSessions:    1000,
}
gameService := NewGameService(config)

// 启动服务
gameService.Start(ctx)

// 开始游戏
err := gameService.StartGame(ctx, userID, sessionID, betAmount)

// 执行转动
response, err := gameService.Spin(ctx, sessionID)

// 结算游戏
err := gameService.Settle(ctx, sessionID)

// 获取用户统计
stats, err := gameService.GetUserStats(ctx, userID)
```

## API集成示例

### Gin路由配置
```go
func SetupGameRoutes(router *gin.RouterGroup, gameService *game.GameService) {
    gameAPI := router.Group("/game")
    {
        gameAPI.POST("/start", handleGameStart(gameService))
        gameAPI.POST("/spin", handleGameSpin(gameService))
        gameAPI.POST("/settle", handleGameSettle(gameService))
        gameAPI.GET("/session/:id", handleGetSession(gameService))
        gameAPI.GET("/history/:userId", handleGetHistory(gameService))
    }
}
```

### 处理函数示例
```go
func handleGameStart(service *game.GameService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req game.GameStartRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // 生成会话ID
        sessionID := generateSessionID()
        
        // 开始游戏
        if err := service.StartGame(c.Request.Context(), 
            req.UserID, sessionID, req.BetAmount); err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, gin.H{
            "success": true,
            "session_id": sessionID,
        })
    }
}

func handleGameSpin(service *game.GameService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req game.GameSpinRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // 执行转动
        response, err := service.Spin(c.Request.Context(), req.SessionID)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, response)
    }
}
```

## WebSocket集成（待实现）

### 消息格式
```json
// 客户端 -> 服务器
{
    "type": "game.start",
    "data": {
        "user_id": 123,
        "bet_amount": 100
    }
}

// 服务器 -> 客户端
{
    "type": "game.state_change",
    "data": {
        "session_id": "xxx",
        "from": "idle",
        "to": "ready"
    }
}

// 转动结果推送
{
    "type": "game.spin_result",
    "data": {
        "session_id": "xxx",
        "result": {...},
        "state": "winning",
        "total_win": 500
    }
}
```

## 状态持久化

### 支持的持久化方式
1. **内存持久化**（用于测试）
2. **数据库持久化**（生产环境）
3. **Redis持久化**（预留接口）
4. **缓存装饰器**（两级缓存）

### 配置示例
```go
// 使用数据库持久化
persister := NewDatabaseStatePersister(db)

// 使用缓存装饰器
cachePersister := NewMemoryStatePersister()
storagePersister := NewDatabaseStatePersister(db)
persister := NewCacheStatePersister(cachePersister, storagePersister, 5*time.Minute)
```

## 异常恢复

### 恢复策略
- **待机状态**：无需特殊处理
- **准备状态**：检查超时，退还投注
- **转动中**：继续到计算阶段
- **计算中**：重新计算结果
- **中奖展示**：直接进入结算
- **结算状态**：完成游戏
- **错误状态**：重置到待机

### 使用恢复管理器
```go
// 创建恢复管理器
recoveryManager := NewRecoveryManager(logger, persister, 30*time.Minute)

// 恢复会话
sm, err := recoveryManager.RecoverSession(ctx, sessionID)
if err != nil {
    // 会话无法恢复，创建新会话
    sm = NewStateMachine(sessionID, userID, logger, persister)
}
```

## 测试建议

### 单元测试
```go
func TestStateMachine(t *testing.T) {
    // 创建内存持久化器
    persister := NewMemoryStatePersister()
    
    // 创建状态机
    sm := NewStateMachine("test-session", 1, logger, persister)
    
    // 测试状态转换
    assert.Equal(t, StateIdle, sm.GetState())
    
    sm.SetBetAmount(100)
    err := sm.Trigger(ctx, "insert_coin")
    assert.NoError(t, err)
    assert.Equal(t, StateReady, sm.GetState())
}
```

### 集成测试
```go
func TestGameFlow(t *testing.T) {
    // 初始化服务
    gameService := NewGameService(config)
    
    // 完整游戏流程
    err := gameService.StartGame(ctx, userID, sessionID, 100)
    assert.NoError(t, err)
    
    response, err := gameService.Spin(ctx, sessionID)
    assert.NoError(t, err)
    assert.NotNil(t, response.Result)
    
    err = gameService.Settle(ctx, sessionID)
    assert.NoError(t, err)
}
```

## 监控指标

### 建议监控的指标
- 活跃会话数
- 平均会话时长
- 游戏RTP（实际返还率）
- 状态转换耗时
- 错误率和恢复率

### Prometheus指标示例
```go
var (
    activeSessionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "game_active_sessions",
        Help: "Number of active game sessions",
    })
    
    gameStartCounter = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "game_starts_total",
        Help: "Total number of games started",
    })
    
    stateTransitionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name: "game_state_transition_duration_seconds",
        Help: "Duration of state transitions",
    }, []string{"from", "to"})
)
```

## 下一步开发计划

### 第一优先级
1. **WebSocket服务**：实现实时通信
2. **游戏流程API**：完整的REST接口
3. **前端集成**：与前端对接测试

### 第二优先级
1. **MQTT远程控制**：管理端功能
2. **数据统计**：报表和分析
3. **性能优化**：缓存和并发

### 第三优先级（硬件对接）
1. **接口适配器模式**：预留硬件接口
2. **Mock硬件服务**：模拟测试
3. **协议文档**：定义通信协议

## 注意事项

1. **并发安全**：所有公共方法都是线程安全的
2. **事务处理**：涉及金额的操作使用数据库事务
3. **错误恢复**：所有状态都可以恢复
4. **性能考虑**：使用缓存减少数据库访问
5. **日志记录**：关键操作都有日志记录

## 联系方式

如有问题，请联系后端开发团队。