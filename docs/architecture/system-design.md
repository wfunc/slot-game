# 推币机游戏系统架构设计

## 1. 系统架构概览

### 1.1 架构愿景
构建一个模块化、可扩展、高可靠的推币机游戏控制系统，支持本地运行和远程管理。

### 1.2 架构原则
- **模块化设计**：各模块职责单一，接口清晰
- **高内聚低耦合**：模块内部高度聚合，模块间松耦合
- **可扩展性**：易于添加新功能和硬件支持
- **容错性**：单点故障不影响整体运行
- **可维护性**：代码结构清晰，易于理解和修改

## 2. 技术架构

### 2.1 整体架构图
```
┌─────────────────────────────────────────────────────────┐
│                     远程管理平台                          │
│                    (MQTT Broker)                        │
└────────────────────┬───────────────────────────────────┘
                     │ MQTT Protocol
┌────────────────────┴───────────────────────────────────┐
│                   后端核心系统 (Go)                      │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────┐  │
│  │                  应用层 (Application)             │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │游戏控制器 │  │设备管理器 │  │远程控制器 │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘      │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │                  业务层 (Business)                │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │游戏引擎   │  │推币控制   │  │数据统计   │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘      │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │                  基础设施层 (Infrastructure)       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │通信管理   │  │数据访问   │  │配置管理   │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘      │  │
│  └──────────────────────────────────────────────────┘  │
└─────────┬──────────────┬──────────────┬───────────────┘
          │              │              │
      WebSocket      串口通信         SQLite
          │              │              │
    ┌─────┴──────┐ ┌─────┴──────┐ ┌────┴─────┐
    │Cocos前端   │ │推币机硬件   │ │ 数据库    │
    └────────────┘ └────────────┘ └──────────┘
```

### 2.2 分层架构

#### 应用层 (Application Layer)
负责协调各个模块，处理外部请求，实现业务流程。

| 组件 | 职责 | 依赖 |
|------|------|------|
| GameController | 游戏流程控制、状态管理 | GameEngine, DeviceManager |
| DeviceManager | 硬件设备管理、状态监控 | SerialManager, StatusMonitor |
| RemoteController | 远程指令处理、配置更新 | MQTTClient, ConfigManager |

#### 业务层 (Business Layer)
实现核心业务逻辑，不依赖具体技术实现。

| 组件 | 职责 | 接口 |
|------|------|------|
| GameEngine | 老虎机逻辑、中奖计算 | IGameEngine |
| PushController | 推币策略、力度控制 | IPushController |
| DataAnalyzer | 数据统计、报表生成 | IDataAnalyzer |

#### 基础设施层 (Infrastructure Layer)
提供技术支撑，屏蔽底层实现细节。

| 组件 | 职责 | 技术选型 |
|------|------|----------|
| CommunicationManager | 通信协议管理 | WebSocket, MQTT, Serial |
| DataAccessLayer | 数据持久化 | GORM + SQLite |
| ConfigManager | 配置加载和管理 | Viper |

## 3. 详细设计

### 3.1 游戏引擎设计

#### 状态机设计
```go
// 游戏状态定义
type GameState int

const (
    StateIdle GameState = iota      // 空闲
    StateWaitingCoin                 // 等待投币
    StateSpinning                    // 转轮中
    StateCalculating                 // 计算结果
    StateWinAnimation                // 中奖动画
    StatePushing                     // 推币中
    StateSettlement                  // 结算
    StateError                       // 错误
)

// 状态转换图
/*
    Idle ──投币──> WaitingCoin ──确认──> Spinning
      ↑                                    │
      │                                    ↓
    Settlement <── Pushing <── WinAnimation ── Calculating
      │                                    │
      └──────────── Error <────────────────┘
*/
```

#### 中奖算法
```go
// 中奖规则配置
type WinRule struct {
    Pattern     []int   // 中奖图案组合
    Probability float64 // 中奖概率
    Reward      int     // 奖励倍数
    PushCount   int     // 推币次数
}

// 随机数生成器
type RandomGenerator interface {
    Next() int
    SetSeed(seed int64)
}

// 中奖判定器
type WinChecker interface {
    Check(result []int) *WinRule
    Calculate(rule *WinRule, bet int) int
}
```

### 3.2 通信架构设计

#### WebSocket通信
```go
// WebSocket连接管理
type WSConnectionManager struct {
    connections map[string]*WSConnection
    mutex       sync.RWMutex
    hub         *WSHub
}

// 消息路由
type MessageRouter struct {
    handlers map[string]MessageHandler
}

// 消息处理接口
type MessageHandler interface {
    Handle(conn *WSConnection, msg *Message) error
}
```

#### 串口通信
```go
// 串口管理器
type SerialManager struct {
    port        io.ReadWriteCloser
    commandQueue chan Command
    responseMap  map[string]chan Response
    mutex        sync.Mutex
}

// 命令队列
type CommandQueue struct {
    queue    []Command
    priority map[CommandType]int
    mutex    sync.Mutex
}

// 协议编解码器
type ProtocolCodec interface {
    Encode(cmd Command) []byte
    Decode(data []byte) Response
}
```

#### MQTT通信
```go
// MQTT客户端封装
type MQTTClientWrapper struct {
    client   mqtt.Client
    handlers map[string]TopicHandler
    config   *MQTTConfig
}

// 主题处理器
type TopicHandler interface {
    Handle(topic string, payload []byte) error
}

// 消息发布器
type Publisher interface {
    Publish(topic string, qos byte, retained bool, payload interface{}) error
}
```

### 3.3 数据模型设计

#### 实体模型
```go
// 游戏记录
type GameRecord struct {
    ID         uint      `gorm:"primarykey"`
    SessionID  string    `gorm:"size:50;index"`
    StartTime  time.Time
    EndTime    *time.Time
    CoinsIn    int
    CoinsOut   int
    TotalWins  int
    MaxWin     int
    Status     string    `gorm:"size:20"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// 中奖记录
type WinRecord struct {
    ID          uint      `gorm:"primarykey"`
    GameID      uint      `gorm:"index"`
    WinType     string    `gorm:"size:50"`
    WinPattern  string    `gorm:"size:20"`
    WinAmount   int
    PushCount   int
    TriggerTime time.Time
}

// 设备状态
type DeviceStatus struct {
    ID           uint      `gorm:"primarykey"`
    DeviceType   string    `gorm:"size:50"`
    DeviceName   string    `gorm:"size:100"`
    Status       string    `gorm:"size:20"`
    LastHeartbeat time.Time
    ErrorCount   int
    ErrorMsg     string    `gorm:"type:text"`
    UpdatedAt    time.Time
}

// 系统配置
type SystemConfig struct {
    ID          uint      `gorm:"primarykey"`
    ConfigKey   string    `gorm:"size:100;uniqueIndex"`
    ConfigValue string    `gorm:"type:text"`
    Description string    `gorm:"type:text"`
    UpdatedAt   time.Time
}
```

#### 仓储接口
```go
// 通用仓储接口
type Repository[T any] interface {
    Create(entity *T) error
    Update(entity *T) error
    Delete(id uint) error
    FindByID(id uint) (*T, error)
    FindAll() ([]*T, error)
    FindByCondition(condition interface{}) ([]*T, error)
}

// 游戏记录仓储
type GameRecordRepository interface {
    Repository[GameRecord]
    FindBySessionID(sessionID string) (*GameRecord, error)
    GetStatistics(startTime, endTime time.Time) (*Statistics, error)
}

// 事务管理
type TransactionManager interface {
    Begin() *gorm.DB
    Commit() error
    Rollback() error
}
```

## 4. 模块交互设计

### 4.1 游戏流程时序图
```
玩家      Cocos前端    WebSocket    GameEngine    PushController    SerialPort    硬件
 │          │            │            │              │               │           │
 │─投币──>│            │            │              │               │           │
 │          │──start──>│            │              │               │           │
 │          │            │──process─>│              │               │           │
 │          │            │            │──spin────>内部处理          │           │
 │          │            │<─result───│              │               │           │
 │          │<──notify──│            │              │               │           │
 │<─动画───│            │            │              │               │           │
 │          │            │            │──trigger──>│               │           │
 │          │            │            │              │──command──>│           │
 │          │            │            │              │               │──PUSH──>│
 │          │            │            │              │               │<──OK────│
 │          │            │            │              │<──response──│           │
 │          │            │<──status──│              │               │           │
 │          │<──update──│            │              │               │           │
 │<─反馈───│            │            │              │               │           │
```

### 4.2 远程控制流程
```
管理平台    MQTT Broker    MQTTClient    ConfigManager    GameEngine
   │           │              │              │              │
   │──publish─>│              │              │              │
   │           │──forward──>│              │              │
   │           │              │──parse────>│              │
   │           │              │              │──update───>│
   │           │              │              │<──result───│
   │           │              │<──response──│              │
   │           │<──publish───│              │              │
   │<──notify──│              │              │              │
```

## 5. 部署架构

### 5.1 部署拓扑
```
┌──────────────────────────────────────┐
│         Ubuntu Desktop PC             │
│  ┌──────────────────────────────┐    │
│  │    推币机游戏后端服务          │    │
│  │    (systemd service)         │    │
│  │    Port: 8080 (WebSocket)    │    │
│  └──────────────────────────────┘    │
│  ┌──────────────────────────────┐    │
│  │    Cocos游戏前端             │    │
│  │    (Electron/Browser)        │    │
│  └──────────────────────────────┘    │
│  ┌──────────────────────────────┐    │
│  │    SQLite数据库              │    │
│  │    /var/lib/slot-game/       │    │
│  └──────────────────────────────┘    │
└──────────┬───────────┬──────────────┘
           │           │
      串口(USB)    网络(MQTT)
           │           │
    ┌──────┴─────┐ ┌──┴──────┐
    │ 推币机硬件  │ │MQTT服务器│
    └────────────┘ └─────────┘
```

### 5.2 目录结构
```
/opt/slot-game/
├── bin/
│   └── slot-game-server      # 主程序
├── config/
│   ├── config.yaml           # 配置文件
│   └── config.yaml.example   # 配置模板
├── data/
│   └── slot-game.db         # SQLite数据库
├── logs/
│   ├── app.log              # 应用日志
│   └── error.log            # 错误日志
├── scripts/
│   ├── install.sh           # 安装脚本
│   ├── start.sh             # 启动脚本
│   └── stop.sh              # 停止脚本
└── docs/
    └── README.md            # 部署文档
```

## 6. 性能设计

### 6.1 性能指标
| 指标 | 目标值 | 测量方法 |
|------|--------|----------|
| 响应时间 | < 100ms | 从请求到响应的时间 |
| 吞吐量 | > 100 req/s | 每秒处理请求数 |
| CPU使用率 | < 30% | top命令监控 |
| 内存使用 | < 512MB | 进程内存占用 |
| 并发连接 | > 100 | WebSocket连接数 |

### 6.2 性能优化策略
- **连接池**：数据库连接池，复用连接
- **缓存机制**：热点数据内存缓存
- **异步处理**：使用goroutine处理IO操作
- **批量操作**：数据批量写入
- **消息队列**：命令队列缓冲

## 7. 安全设计

### 7.1 安全威胁分析
| 威胁 | 影响 | 防护措施 |
|------|------|----------|
| 通信劫持 | 高 | TLS加密通信 |
| 数据篡改 | 高 | 消息签名验证 |
| 恶意操控 | 高 | 访问控制、鉴权 |
| 拒绝服务 | 中 | 限流、熔断 |
| 数据泄露 | 中 | 敏感数据加密 |

### 7.2 安全架构
```
┌─────────────────────────────────────┐
│         安全层 (Security Layer)      │
├─────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐        │
│  │ 认证鉴权  │  │ 加密通信  │        │
│  └──────────┘  └──────────┘        │
│  ┌──────────┐  ┌──────────┐        │
│  │ 访问控制  │  │ 审计日志  │        │
│  └──────────┘  └──────────┘        │
└─────────────────────────────────────┘
```

## 8. 监控设计

### 8.1 监控指标
- **系统指标**：CPU、内存、磁盘、网络
- **应用指标**：请求量、响应时间、错误率
- **业务指标**：游戏次数、中奖率、收益
- **设备指标**：在线状态、故障次数

### 8.2 监控架构
```go
// 指标收集器
type MetricsCollector interface {
    CollectSystem() *SystemMetrics
    CollectApplication() *AppMetrics
    CollectBusiness() *BusinessMetrics
}

// 健康检查
type HealthChecker interface {
    Check() *HealthStatus
    RegisterCheck(name string, check CheckFunc)
}

// 告警管理
type AlertManager interface {
    SendAlert(level AlertLevel, message string)
    SetThreshold(metric string, threshold float64)
}
```

## 9. 扩展性设计

### 9.1 扩展点
- **游戏类型**：支持添加新的游戏玩法
- **硬件设备**：支持新的硬件设备接入
- **通信协议**：支持新的通信协议
- **数据存储**：支持切换数据库类型

### 9.2 插件机制
```go
// 插件接口
type Plugin interface {
    Name() string
    Version() string
    Init(config interface{}) error
    Start() error
    Stop() error
}

// 插件管理器
type PluginManager interface {
    Register(plugin Plugin) error
    Load(name string) error
    Unload(name string) error
    List() []Plugin
}
```

## 10. 技术决策记录

### 10.1 为什么选择Go语言
- 高性能：编译型语言，运行效率高
- 并发支持：goroutine天然支持高并发
- 部署简单：单文件部署，无需运行时
- 生态丰富：有成熟的串口、WebSocket、MQTT库

### 10.2 为什么选择SQLite
- 轻量级：无需独立数据库服务
- 可靠性：ACID事务支持
- 性能足够：单机游戏数据量不大
- 维护简单：单文件数据库

### 10.3 为什么选择WebSocket
- 实时性：双向通信，低延迟
- 效率高：长连接，减少握手开销
- 兼容性：浏览器原生支持
- 成熟度：协议标准化，库支持好