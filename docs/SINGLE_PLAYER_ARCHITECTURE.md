# 🎮 单机版老虎机游戏架构设计方案

## 📋 项目概述

基于WebSocket + Protobuf的单机版老虎机游戏，专注于**Slot拉霸机**功能实现，简化多人房间和复杂社交功能。

### 核心特性
- ✅ 单人游戏体验
- ✅ 本地状态管理
- ✅ 快速响应（无网络延迟）
- ✅ 简化的架构设计
- ✅ 支持离线游戏

---

## 🏗️ 系统架构

### 整体架构图
```
┌─────────────────────────────────────────────────┐
│                   前端客户端                      │
│  (HTML5/Unity/Cocos)                            │
└──────────────────┬──────────────────────────────┘
                   │ WebSocket + Protobuf
┌──────────────────▼──────────────────────────────┐
│              WebSocket Handler                   │
│         (gorilla/websocket)                     │
├──────────────────────────────────────────────────┤
│            Protobuf Codec                        │
│         消息编解码 + 路由分发                      │
├──────────────────────────────────────────────────┤
│             Game Engine                          │
│         核心游戏逻辑 + 状态机                      │
├──────────────────────────────────────────────────┤
│          Session Manager                         │
│         单会话管理 + 状态持久化                    │
├──────────────────────────────────────────────────┤
│           Local Storage                          │
│         本地数据存储(SQLite/JSON)                 │
└──────────────────────────────────────────────────┘
```

### 简化的消息流
```
客户端 → [m_1901_tos 进入游戏] → 服务端
      ← [m_1901_toc 游戏配置] ←

客户端 → [m_1902_tos 开始拉霸] → 服务端  
      ← [m_1902_toc 游戏结果] ←

客户端 → [m_2099_tos 心跳包] → 服务端
      ← [m_2099_toc 心跳响应] ←
```

---

## 💻 详细实现方案

### 1. WebSocket连接管理（简化版）

```go
// internal/websocket/connection.go
package websocket

import (
    "github.com/gorilla/websocket"
    "sync"
    "time"
)

type Connection struct {
    ws         *websocket.Conn
    send       chan []byte
    session    *Session
    mu         sync.RWMutex
    lastPing   time.Time
}

type Session struct {
    PlayerID   uint32
    Balance    uint64  // 玩家余额
    TotalWin   uint64  // 累计赢分
    FreeSpins  uint32  // 免费次数
    CurrentBet uint32  // 当前下注
    GameState  *SlotGameState
}

type SlotGameState struct {
    BetLevel    uint32           // 下注档位
    IsFreeSpin  bool            // 是否免费游戏
    FreeCount   uint32          // 免费游戏计数
    FreeTotal   uint32          // 免费游戏总数
    LastResult  *SpinResult     // 上次结果
    JPValues    map[string]uint32 // JP累积值
}

func NewConnection(ws *websocket.Conn) *Connection {
    return &Connection{
        ws:   ws,
        send: make(chan []byte, 256),
        session: &Session{
            PlayerID: generatePlayerID(),
            Balance:  100000, // 初始金币
            GameState: &SlotGameState{
                JPValues: initJPValues(),
            },
        },
        lastPing: time.Now(),
    }
}

// 简化的消息处理循环
func (c *Connection) Run() {
    go c.writePump()
    c.readPump()
}

func (c *Connection) readPump() {
    defer c.ws.Close()
    
    for {
        _, message, err := c.ws.ReadMessage()
        if err != nil {
            return
        }
        
        // 解析Protobuf消息并处理
        c.handleMessage(message)
    }
}
```

### 2. Protobuf消息处理器

```go
// internal/protocol/handler.go
package protocol

import (
    "github.com/wfunc/slot-game/internal/pb"
    "google.golang.org/protobuf/proto"
)

type MessageHandler struct {
    conn      *Connection
    slotGame  *SlotGameEngine
}

func (h *MessageHandler) HandleMessage(msgID uint16, data []byte) error {
    switch msgID {
    case 1901: // 进入房间
        return h.handleEnterRoom(data)
    case 1902: // 开始游戏
        return h.handleStartGame(data)
    case 2001: // 获取信息
        return h.handleGetInfo(data)
    case 2099: // 心跳
        return h.handleHeartbeat(data)
    default:
        return ErrUnknownMessage
    }
}

func (h *MessageHandler) handleEnterRoom(data []byte) error {
    req := &pb.M_1901_Tos{}
    if err := proto.Unmarshal(data, req); err != nil {
        return err
    }
    
    // 单机模式：直接返回配置
    resp := &pb.M_1901_Toc{
        BetVal: []uint32{1, 2, 5, 10, 20, 50, 100},
        Odds:   h.slotGame.GetOddsTable(),
        Cfg: &pb.P_Config{
            DevId: "SINGLE_PLAYER",
            DevNo: 1,
        },
    }
    
    return h.sendMessage(1901, resp)
}

func (h *MessageHandler) handleStartGame(data []byte) error {
    req := &pb.M_1902_Tos{}
    if err := proto.Unmarshal(data, req); err != nil {
        return err
    }
    
    // 验证余额
    if h.conn.session.Balance < uint64(req.BetVal) {
        return ErrInsufficientBalance
    }
    
    // 执行游戏逻辑
    result := h.slotGame.Spin(req.BetVal, h.conn.session)
    
    // 更新会话状态
    h.conn.session.Balance -= uint64(req.BetVal)
    h.conn.session.Balance += result.WinAmount
    h.conn.session.TotalWin += result.WinAmount
    
    // 构建响应
    resp := &pb.M_1902_Toc{
        BetVal:      req.BetVal,
        Win:         uint32(result.WinAmount),
        TotalWin:    uint32(h.conn.session.TotalWin),
        IsFree:      result.IsFree,
        CurrentFree: result.CurrentFree,
        TotalFree:   result.TotalFree,
        Result:      convertToProtoResult(result),
    }
    
    return h.sendMessage(1902, resp)
}
```

### 3. 核心游戏引擎

```go
// internal/game/slot_engine.go
package game

import (
    "math/rand"
    "time"
)

type SlotGameEngine struct {
    config     *GameConfig
    rng        *rand.Rand
    payTable   *PayTable
    jpPool     *JackpotPool
}

type GameConfig struct {
    Reels      [][]Symbol     // 转轴配置
    Paylines   [][]Position   // 赢线配置
    WildSymbol Symbol         // Wild符号
    FreeSymbol Symbol         // Free符号
    RTP        float64        // 返还率
}

type SpinResult struct {
    Grid        [][]Symbol      // 5x3或5x4的格子
    WinLines    []WinLine       // 中奖线
    WinAmount   uint64          // 赢分
    IsFree      bool            // 是否触发免费
    CurrentFree uint32          // 当前免费次数
    TotalFree   uint32          // 总免费次数
    JPWin       *JackpotWin     // JP中奖
    Features    []Feature       // 特殊功能
}

type WinLine struct {
    Symbol    Symbol
    Count     int
    Positions []Position
    Payout    uint64
}

func NewSlotGameEngine(config *GameConfig) *SlotGameEngine {
    return &SlotGameEngine{
        config:   config,
        rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
        payTable: loadPayTable(),
        jpPool:   NewJackpotPool(),
    }
}

// 核心旋转逻辑
func (e *SlotGameEngine) Spin(betAmount uint32, session *Session) *SpinResult {
    result := &SpinResult{
        Grid: e.generateGrid(),
    }
    
    // 检查免费游戏
    if session.GameState.IsFreeSpin {
        result.IsFree = true
        result.CurrentFree = session.GameState.FreeCount + 1
        result.TotalFree = session.GameState.FreeTotal
        
        session.GameState.FreeCount++
        if session.GameState.FreeCount >= session.GameState.FreeTotal {
            session.GameState.IsFreeSpin = false
            session.GameState.FreeCount = 0
        }
    }
    
    // 计算中奖
    result.WinLines = e.calculateWins(result.Grid, betAmount)
    result.WinAmount = e.calculateTotalWin(result.WinLines)
    
    // 检查是否触发免费游戏
    if !session.GameState.IsFreeSpin {
        freeCount := e.checkFreeGames(result.Grid)
        if freeCount > 0 {
            session.GameState.IsFreeSpin = true
            session.GameState.FreeTotal = freeCount
            session.GameState.FreeCount = 0
            result.TotalFree = freeCount
        }
    }
    
    // 检查JP
    result.JPWin = e.checkJackpot(result.Grid, betAmount)
    if result.JPWin != nil {
        result.WinAmount += result.JPWin.Amount
        e.jpPool.Reset(result.JPWin.Type)
    }
    
    // 更新JP池（每次下注贡献）
    e.jpPool.Contribute(betAmount)
    
    return result
}

// 生成转轴结果
func (e *SlotGameEngine) generateGrid() [][]Symbol {
    grid := make([][]Symbol, 5) // 5列
    for col := 0; col < 5; col++ {
        grid[col] = make([]Symbol, 3) // 3行（可配置为4行）
        for row := 0; row < 3; row++ {
            // 根据权重随机选择符号
            grid[col][row] = e.getRandomSymbol(col)
        }
    }
    return grid
}

// 根据权重获取随机符号
func (e *SlotGameEngine) getRandomSymbol(reel int) Symbol {
    weights := e.config.Reels[reel]
    totalWeight := 0
    for _, w := range weights {
        totalWeight += int(w)
    }
    
    r := e.rng.Intn(totalWeight)
    for symbol, weight := range weights {
        r -= int(weight)
        if r < 0 {
            return symbol
        }
    }
    return Symbol(0)
}

// 计算中奖线
func (e *SlotGameEngine) calculateWins(grid [][]Symbol, betAmount uint32) []WinLine {
    var wins []WinLine
    
    // 检查每条赢线
    for _, payline := range e.config.Paylines {
        symbols := make([]Symbol, len(payline))
        positions := make([]Position, len(payline))
        
        for i, pos := range payline {
            symbols[i] = grid[pos.Col][pos.Row]
            positions[i] = pos
        }
        
        // 从左到右连续相同符号
        if win := e.checkLineWin(symbols, positions, betAmount); win != nil {
            wins = append(wins, *win)
        }
    }
    
    return wins
}

// 检查单条线中奖
func (e *SlotGameEngine) checkLineWin(symbols []Symbol, positions []Position, bet uint32) *WinLine {
    firstSymbol := symbols[0]
    if firstSymbol == Symbol(0) { // 空符号
        return nil
    }
    
    count := 1
    for i := 1; i < len(symbols); i++ {
        if symbols[i] == firstSymbol || symbols[i] == e.config.WildSymbol {
            count++
        } else {
            break
        }
    }
    
    // 至少3个相同才中奖
    if count >= 3 {
        payout := e.payTable.GetPayout(firstSymbol, count) * uint64(bet)
        return &WinLine{
            Symbol:    firstSymbol,
            Count:     count,
            Positions: positions[:count],
            Payout:    payout,
        }
    }
    
    return nil
}
```

### 4. 数据持久化（本地存储）

```go
// internal/storage/local_storage.go
package storage

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "encoding/json"
    "io/ioutil"
)

type LocalStorage struct {
    db *sql.DB
}

func NewLocalStorage(dbPath string) (*LocalStorage, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }
    
    // 创建表
    if err := createTables(db); err != nil {
        return nil, err
    }
    
    return &LocalStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
    schema := `
    CREATE TABLE IF NOT EXISTS player_data (
        player_id INTEGER PRIMARY KEY,
        balance INTEGER NOT NULL,
        total_win INTEGER NOT NULL,
        total_bet INTEGER NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS game_records (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        player_id INTEGER NOT NULL,
        bet_amount INTEGER NOT NULL,
        win_amount INTEGER NOT NULL,
        is_free BOOLEAN NOT NULL,
        result TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS jp_pool (
        type TEXT PRIMARY KEY,
        value INTEGER NOT NULL,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `
    
    _, err := db.Exec(schema)
    return err
}

// 保存游戏记录
func (s *LocalStorage) SaveGameRecord(playerID uint32, record *GameRecord) error {
    resultJSON, _ := json.Marshal(record.Result)
    
    _, err := s.db.Exec(`
        INSERT INTO game_records (player_id, bet_amount, win_amount, is_free, result)
        VALUES (?, ?, ?, ?, ?)
    `, playerID, record.BetAmount, record.WinAmount, record.IsFree, string(resultJSON))
    
    return err
}

// 更新玩家数据
func (s *LocalStorage) UpdatePlayerData(playerID uint32, balance, totalWin uint64) error {
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO player_data (player_id, balance, total_win, updated_at)
        VALUES (?, ?, ?, CURRENT_TIMESTAMP)
    `, playerID, balance, totalWin)
    
    return err
}

// 获取玩家数据
func (s *LocalStorage) GetPlayerData(playerID uint32) (*PlayerData, error) {
    var data PlayerData
    err := s.db.QueryRow(`
        SELECT player_id, balance, total_win, total_bet 
        FROM player_data 
        WHERE player_id = ?
    `, playerID).Scan(&data.PlayerID, &data.Balance, &data.TotalWin, &data.TotalBet)
    
    if err == sql.ErrNoRows {
        // 新玩家，返回默认值
        return &PlayerData{
            PlayerID: playerID,
            Balance:  100000, // 初始金币
            TotalWin: 0,
            TotalBet: 0,
        }, nil
    }
    
    return &data, err
}
```

### 5. JP（Jackpot）系统

```go
// internal/game/jackpot.go
package game

import (
    "sync"
    "math/rand"
)

type JackpotPool struct {
    pools map[string]*JPData
    mu    sync.RWMutex
}

type JPData struct {
    Type       string
    Value      uint64
    BaseValue  uint64
    Increment  float64 // 每次下注的贡献比例
    TriggerOdds float64 // 触发概率
}

func NewJackpotPool() *JackpotPool {
    return &JackpotPool{
        pools: map[string]*JPData{
            "JP1": {
                Type:        "JP1",
                Value:       1000,
                BaseValue:   1000,
                Increment:   0.01,  // 1%贡献
                TriggerOdds: 0.001, // 0.1%概率
            },
            "JP2": {
                Type:        "JP2",
                Value:       5000,
                BaseValue:   5000,
                Increment:   0.02,
                TriggerOdds: 0.0005,
            },
            "JP3": {
                Type:        "JP3",
                Value:       10000,
                BaseValue:   10000,
                Increment:   0.03,
                TriggerOdds: 0.0001,
            },
            "JPALL": {
                Type:        "JPALL",
                Value:       50000,
                BaseValue:   50000,
                Increment:   0.05,
                TriggerOdds: 0.00001,
            },
        },
    }
}

// 贡献到JP池
func (jp *JackpotPool) Contribute(betAmount uint32) {
    jp.mu.Lock()
    defer jp.mu.Unlock()
    
    for _, pool := range jp.pools {
        contribution := uint64(float64(betAmount) * pool.Increment)
        pool.Value += contribution
    }
}

// 检查是否触发JP
func (jp *JackpotPool) CheckTrigger(betAmount uint32) *JackpotWin {
    jp.mu.RLock()
    defer jp.mu.RUnlock()
    
    // 按优先级检查（从小到大）
    for _, jpType := range []string{"JP1", "JP2", "JP3", "JPALL"} {
        pool := jp.pools[jpType]
        if rand.Float64() < pool.TriggerOdds {
            return &JackpotWin{
                Type:   jpType,
                Amount: pool.Value,
            }
        }
    }
    
    return nil
}

// 重置JP池
func (jp *JackpotPool) Reset(jpType string) {
    jp.mu.Lock()
    defer jp.mu.Unlock()
    
    if pool, exists := jp.pools[jpType]; exists {
        pool.Value = pool.BaseValue
    }
}

// 获取当前JP值（用于显示）
func (jp *JackpotPool) GetValues() map[string]uint64 {
    jp.mu.RLock()
    defer jp.mu.RUnlock()
    
    values := make(map[string]uint64)
    for k, v := range jp.pools {
        values[k] = v.Value
    }
    return values
}
```

### 6. 配置文件

```yaml
# config/game_config.yaml
game:
  type: "slot_mahjong"
  rtp: 0.96  # 返还率96%
  
  # 下注档位
  bet_levels: [1, 2, 5, 10, 20, 50, 100]
  
  # 符号权重配置（每列）
  reels:
    - [10, 10, 10, 10, 10, 10, 10, 10, 5, 5, 5, 5, 3, 3, 2, 1]  # 第1列
    - [10, 10, 10, 10, 10, 10, 10, 10, 5, 5, 5, 5, 3, 3, 2, 1]  # 第2列
    - [10, 10, 10, 10, 10, 10, 10, 10, 5, 5, 5, 5, 3, 3, 2, 1]  # 第3列
    - [10, 10, 10, 10, 10, 10, 10, 10, 5, 5, 5, 5, 3, 3, 2, 1]  # 第4列
    - [10, 10, 10, 10, 10, 10, 10, 10, 5, 5, 5, 5, 3, 3, 2, 1]  # 第5列
  
  # 赔率表
  paytable:
    symbol_0:
      3: 5
      4: 10
      5: 20
    symbol_1:
      3: 10
      4: 20
      5: 50
    symbol_2:
      3: 15
      4: 30
      5: 75
    wild:
      3: 20
      4: 50
      5: 100
    free:
      3: 10  # 触发10次免费游戏
      4: 20  # 触发20次免费游戏
      5: 30  # 触发30次免费游戏
  
  # JP配置
  jackpot:
    jp1:
      base: 1000
      contribution: 0.01
      odds: 0.001
    jp2:
      base: 5000
      contribution: 0.02
      odds: 0.0005
    jp3:
      base: 10000
      contribution: 0.03
      odds: 0.0001
    jpall:
      base: 50000
      contribution: 0.05
      odds: 0.00001

# 服务器配置
server:
  host: "localhost"
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  
# 存储配置
storage:
  type: "sqlite"
  path: "./data/game.db"
  
# 日志配置
logging:
  level: "info"
  file: "./logs/game.log"
  max_size: 100  # MB
  max_age: 7     # days
  max_backups: 3
```

---

## 🚀 启动流程

### 主程序入口

```go
// cmd/server/main.go
package main

import (
    "flag"
    "log"
    "net/http"
    
    "github.com/gorilla/websocket"
    "github.com/wfunc/slot-game/internal/config"
    "github.com/wfunc/slot-game/internal/game"
    "github.com/wfunc/slot-game/internal/storage"
    "github.com/wfunc/slot-game/internal/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 单机版允许所有来源
    },
}

func main() {
    configPath := flag.String("config", "./config/game_config.yaml", "配置文件路径")
    flag.Parse()
    
    // 加载配置
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatal("加载配置失败:", err)
    }
    
    // 初始化存储
    store, err := storage.NewLocalStorage(cfg.Storage.Path)
    if err != nil {
        log.Fatal("初始化存储失败:", err)
    }
    
    // 初始化游戏引擎
    gameEngine := game.NewSlotGameEngine(cfg.Game)
    
    // WebSocket处理
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        ws, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Print("upgrade失败:", err)
            return
        }
        
        conn := websocket.NewConnection(ws)
        conn.SetGameEngine(gameEngine)
        conn.SetStorage(store)
        conn.Run()
    })
    
    // 静态文件服务（游戏客户端）
    http.Handle("/", http.FileServer(http.Dir("./static")))
    
    log.Printf("游戏服务器启动在 %s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port), nil))
}
```

---

## 📦 项目结构

```
slot-game/
├── cmd/
│   └── server/
│       └── main.go           # 主程序入口
├── config/
│   └── game_config.yaml      # 游戏配置
├── internal/
│   ├── config/               # 配置加载
│   ├── game/                 # 游戏逻辑
│   │   ├── slot_engine.go    # 老虎机引擎
│   │   ├── jackpot.go        # JP系统
│   │   └── paytable.go       # 赔率表
│   ├── pb/                   # Protobuf生成代码
│   ├── protocol/             # 协议处理
│   │   ├── handler.go        # 消息处理器
│   │   └── codec.go          # 编解码
│   ├── storage/              # 数据存储
│   │   └── local_storage.go  # 本地存储
│   └── websocket/            # WebSocket管理
│       └── connection.go     # 连接管理
├── proto/                    # Protobuf定义
│   ├── slot.proto
│   └── cfg.proto
├── static/                   # 静态资源
│   └── index.html           # 游戏客户端
├── data/                    # 数据目录
│   └── game.db             # SQLite数据库
├── logs/                    # 日志目录
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🎮 客户端集成示例

```javascript
// static/js/game.js
class SlotGameClient {
    constructor() {
        this.ws = null;
        this.connected = false;
    }
    
    connect() {
        this.ws = new WebSocket('ws://localhost:8080/ws');
        this.ws.binaryType = 'arraybuffer';
        
        this.ws.onopen = () => {
            console.log('连接成功');
            this.connected = true;
            this.enterRoom();
        };
        
        this.ws.onmessage = (event) => {
            this.handleMessage(event.data);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket错误:', error);
        };
    }
    
    // 进入游戏
    enterRoom() {
        const msg = {
            type: 1  // e_slot_type_mahjong
        };
        this.sendMessage(1901, msg);
    }
    
    // 开始游戏
    startGame(betAmount) {
        const msg = {
            bet_val: betAmount
        };
        this.sendMessage(1902, msg);
    }
    
    // 发送消息
    sendMessage(msgId, data) {
        // 编码Protobuf消息
        const encoded = encodeProtobuf(msgId, data);
        this.ws.send(encoded);
    }
    
    // 处理消息
    handleMessage(data) {
        const { msgId, message } = decodeProtobuf(data);
        
        switch(msgId) {
            case 1901: // 进入房间响应
                this.handleEnterRoom(message);
                break;
            case 1902: // 游戏结果
                this.handleGameResult(message);
                break;
            case 1903: // 推送数据
                this.handlePushData(message);
                break;
            case 1904: // JP中奖
                this.handleJPReward(message);
                break;
        }
    }
    
    handleGameResult(result) {
        // 更新UI显示
        updateReels(result.result);
        updateBalance(result.total_win);
        
        // 检查免费游戏
        if (result.is_free) {
            showFreeSpins(result.current_free, result.total_free);
        }
        
        // 显示赢分
        if (result.win > 0) {
            showWinAnimation(result.win);
        }
    }
}

// 初始化游戏
const game = new SlotGameClient();
game.connect();
```

---

## 🔧 开发步骤

### 第一阶段：基础框架（1-2天）
1. ✅ 搭建项目结构
2. ✅ 实现WebSocket连接
3. ✅ 集成Protobuf编解码
4. ✅ 基础消息路由

### 第二阶段：游戏逻辑（2-3天）
1. ✅ 实现老虎机核心算法
2. ✅ 配置赔率表
3. ✅ 实现中奖判定
4. ✅ 免费游戏逻辑

### 第三阶段：数据存储（1天）
1. ✅ SQLite集成
2. ✅ 游戏记录保存
3. ✅ 玩家数据管理

### 第四阶段：JP系统（1天）
1. ✅ JP累积逻辑
2. ✅ JP触发机制
3. ✅ JP重置和显示

### 第五阶段：测试优化（1-2天）
1. ✅ 单元测试
2. ✅ 性能优化
3. ✅ 客户端集成测试

---

## 🎯 优化建议

### 性能优化
- 使用对象池减少GC压力
- 预生成随机数序列
- 缓存常用计算结果

### 安全考虑
- 所有游戏逻辑在服务端
- 验证客户端输入
- 防止重放攻击

### 扩展性
- 模块化设计便于添加新游戏
- 配置驱动便于调整参数
- 预留多语言支持接口

---

## 📊 监控指标

### 关键指标
- RTP（返还率）监控
- JP触发频率
- 玩家留存率
- 平均游戏时长

### 日志记录
- 每次下注和结果
- JP触发事件
- 异常情况
- 性能指标