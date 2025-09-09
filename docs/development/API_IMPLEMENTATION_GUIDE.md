# API实现指南

## 当前状态
✅ **已完成的功能模块**
- 游戏引擎核心逻辑（100%完成）
- 状态机和会话管理（100%完成）
- 数据库层和仓储模式（100%完成）
- 用户认证系统（登录、注册、JWT）

❌ **尚未实现的API接口**
- 游戏相关API（0%）
- 钱包相关API（0%）
- WebSocket实时通信（0%）
- 管理后台API（0%）

## 立即需要实现的API清单

### 1. 老虎机游戏API (`/api/v1/slot`)

#### POST `/api/v1/slot/start`
开始游戏会话
```go
// handler: internal/api/slot_handler.go
type StartRequest struct {
    BetAmount int64 `json:"bet_amount" binding:"required,min=100"`
}

type StartResponse struct {
    SessionID string `json:"session_id"`
    Balance   int64  `json:"balance"`
}
```

#### POST `/api/v1/slot/spin`
执行转动
```go
type SpinRequest struct {
    SessionID string `json:"session_id" binding:"required"`
}

type SpinResponse struct {
    Result    *slot.SpinResult `json:"result"`
    Balance   int64           `json:"balance"`
    State     string          `json:"state"`
}
```

#### POST `/api/v1/slot/settle`
结算游戏
```go
type SettleRequest struct {
    SessionID string `json:"session_id" binding:"required"`
}

type SettleResponse struct {
    TotalBet  int64 `json:"total_bet"`
    TotalWin  int64 `json:"total_win"`
    Balance   int64 `json:"balance"`
}
```

#### GET `/api/v1/slot/history`
获取游戏历史
```go
type HistoryResponse struct {
    Records []GameRecord `json:"records"`
    Total   int         `json:"total"`
}
```

### 2. 钱包API (`/api/v1/wallet`)

#### GET `/api/v1/wallet/balance`
查询余额
```go
type BalanceResponse struct {
    Balance      int64 `json:"balance"`
    FrozenAmount int64 `json:"frozen_amount"`
    Available    int64 `json:"available"`
}
```

#### GET `/api/v1/wallet/transactions`
交易记录
```go
type TransactionListResponse struct {
    Transactions []Transaction `json:"transactions"`
    Total       int          `json:"total"`
    Page        int          `json:"page"`
}
```

#### POST `/api/v1/wallet/deposit`
充值（测试用）
```go
type DepositRequest struct {
    Amount int64 `json:"amount" binding:"required,min=100"`
}

type DepositResponse struct {
    TransactionID string `json:"transaction_id"`
    Balance      int64  `json:"balance"`
}
```

### 3. WebSocket实时通信

#### 连接端点: `/ws/game`
```javascript
// 客户端连接示例
const ws = new WebSocket('ws://localhost:8080/ws/game');

// 消息类型
{
    "type": "spin_result",
    "data": {
        "session_id": "xxx",
        "result": {...},
        "balance": 10000
    }
}

{
    "type": "state_change", 
    "data": {
        "session_id": "xxx",
        "from": "ready",
        "to": "spinning"
    }
}
```

## 实现步骤

### Step 1: 创建Handler文件（立即执行）
```bash
# 创建handler文件
touch internal/api/slot_handler.go
touch internal/api/wallet_handler.go
touch internal/api/websocket_handler.go
```

### Step 2: 实现基础Handler结构
```go
// internal/api/slot_handler.go
package api

import (
    "github.com/gin-gonic/gin"
    "github.com/wfunc/slot-game/internal/game"
    "github.com/wfunc/slot-game/internal/middleware"
)

type SlotHandler struct {
    gameService *game.GameService
}

func NewSlotHandler(gameService *game.GameService) *SlotHandler {
    return &SlotHandler{
        gameService: gameService,
    }
}

func (h *SlotHandler) Start(c *gin.Context) {
    userID := middleware.GetUserID(c)
    
    var req StartRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 创建会话ID
    sessionID := generateSessionID()
    
    // 调用游戏服务
    err := h.gameService.StartGame(c.Request.Context(), userID, sessionID, req.BetAmount)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 获取余额
    // ...
    
    c.JSON(200, StartResponse{
        SessionID: sessionID,
        Balance:   balance,
    })
}

func (h *SlotHandler) Spin(c *gin.Context) {
    // 实现转动逻辑
}

func (h *SlotHandler) Settle(c *gin.Context) {
    // 实现结算逻辑
}

func (h *SlotHandler) GetHistory(c *gin.Context) {
    // 实现历史记录查询
}
```

### Step 3: 注册路由
```go
// internal/api/router.go
func (r *Router) setupRoutes() {
    // ... existing code ...
    
    // Slot game routes
    slot := v1.Group("/slot")
    slot.Use(middleware.AuthRequired())
    {
        slot.POST("/start", r.slotHandler.Start)
        slot.POST("/spin", r.slotHandler.Spin)
        slot.POST("/settle", r.slotHandler.Settle)
        slot.GET("/history", r.slotHandler.GetHistory)
    }
    
    // Wallet routes
    wallet := v1.Group("/wallet")
    wallet.Use(middleware.AuthRequired())
    {
        wallet.GET("/balance", r.walletHandler.GetBalance)
        wallet.GET("/transactions", r.walletHandler.GetTransactions)
        wallet.POST("/deposit", r.walletHandler.Deposit)
    }
}
```

### Step 4: WebSocket实现
```go
// internal/api/websocket_handler.go
package api

import (
    "github.com/gorilla/websocket"
    "github.com/gin-gonic/gin"
)

type WebSocketHandler struct {
    upgrader websocket.Upgrader
    hub      *Hub
}

func (h *WebSocketHandler) GameWebSocket(c *gin.Context) {
    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    
    client := &Client{
        conn: conn,
        send: make(chan []byte, 256),
    }
    
    h.hub.register <- client
    
    go client.writePump()
    go client.readPump()
}
```

## 测试用例

### 测试游戏流程
```bash
# 1. 登录获取token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'

# 2. 开始游戏
curl -X POST http://localhost:8080/api/v1/slot/start \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bet_amount":100}'

# 3. 执行转动
curl -X POST http://localhost:8080/api/v1/slot/spin \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"SESSION_ID"}'

# 4. 结算
curl -X POST http://localhost:8080/api/v1/slot/settle \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"SESSION_ID"}'
```

## 开发优先级

1. **第一天**: 实现基础游戏API（start, spin, settle）
2. **第二天**: 实现钱包API和交易记录
3. **第三天**: 实现WebSocket实时通信
4. **第四天**: 集成测试和bug修复
5. **第五天**: 性能优化和文档完善

## 注意事项

1. **事务处理**: 所有涉及金额变动的操作必须使用数据库事务
2. **并发安全**: 使用锁机制防止并发问题
3. **错误处理**: 统一的错误响应格式
4. **日志记录**: 记录所有关键操作
5. **安全验证**: JWT认证和权限校验

## 相关文件路径

- 游戏服务: `internal/game/game_service.go`
- 会话管理: `internal/game/session_manager.go`
- 状态机: `internal/game/state_machine.go`
- 老虎机引擎: `internal/game/slot/engine.go`
- 钱包仓储: `internal/repository/wallet.go`
- 交易仓储: `internal/repository/transaction.go`