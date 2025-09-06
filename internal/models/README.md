# 数据库模型设计文档

## 📊 数据库架构概览

本项目使用GORM作为ORM框架，支持多种数据库（SQLite、MySQL、PostgreSQL），采用模块化的模型设计。

## 🗂️ 模型结构

### 基础模型 (base.go)
```go
BaseModel - 所有模型的基础结构
├── ID (主键)
├── CreatedAt (创建时间)
├── UpdatedAt (更新时间)
└── DeletedAt (软删除)
```

### 用户系统 (user.go)

#### User - 用户基础信息
- 用户名、昵称、手机、邮箱
- 头像、状态、等级、经验值
- VIP等级和过期时间
- 最后登录时间和IP

#### UserProfile - 用户详细信息
- 真实姓名、性别、生日
- 地理位置信息
- 设备信息（ID、类型、版本）

#### UserAuth - 用户认证信息
- 登录密码（加盐加密）
- 支付密码（加盐加密）
- 双因素认证
- 安全问题和答案
- 登录尝试和锁定机制

#### UserSession - 用户会话
- 会话ID、Token、刷新Token
- IP地址、User Agent
- 平台信息、设备ID
- 在线状态、活跃时间

### 游戏系统 (game.go)

#### Game - 游戏类型
- 游戏名称、类型（slot/pusher）
- 描述、图标、状态
- 最小/最大投注
- RTP（返还率）
- 配置信息（JSON）

#### GameRoom - 游戏房间
- 房间号、名称、类型
- 状态、最大玩家数
- 投注限制、等级限制
- 房间配置

#### GameSession - 游戏会话
- 用户ID、游戏ID、房间ID
- 会话ID、状态
- 开始/结束时间、持续时长
- 总投注、总赢取、总回合数
- 游戏数据（JSON）

#### GameResult - 游戏结果
- 回合ID、投注金额、赢取金额
- 倍率、结果详情（JSON）
- 是否大奖、是否奖励

### 老虎机系统

#### SlotMachine - 老虎机配置
- 机器ID、名称
- 转轮数、行数、赔付线数
- 符号配置、赔付表
- 奖励配置、大奖池

#### SlotSpin - 旋转记录
- 转轮停止位置
- 中奖线、奖励获得
- 免费旋转、倍率

#### SlotWinLine - 中奖线记录
- 线号、符号、数量
- 赢取金额、位置信息

### 推币机系统

#### PusherMachine - 推币机配置
- 机器ID、名称
- 平台宽度/深度
- 推力、速度
- 币值、特殊物品
- 当前币数

#### PusherSession - 推币机会话
- 插入币数、赢取币数
- 赢取物品（JSON）
- 推动次数
- 开始/结束状态

#### CoinDrop - 掉落记录
- 掉落时间、币数
- 物品类型、价值
- 位置信息

### 交易系统 (transaction.go)

#### Wallet - 用户钱包
- 余额（分）、游戏币、积分
- 冻结余额、冻结币
- 总充值、总提现
- 总投注、总赢取
- 每日统计

#### Transaction - 交易记录
- 订单号、类型、子类型
- 金额、前后余额
- 状态、关联ID
- 描述、备注、元数据

#### CoinPurchase - 币购买记录
- 订单号、套餐信息
- 币数量、奖励数量
- 价格、折扣
- 支付方式、支付账号
- 支付时间、状态

#### Withdrawal - 提现记录
- 订单号、金额、手续费
- 提现方式、账户信息
- 审核人、审核时间
- 处理时间、失败原因

### 系统管理

#### SystemConfig - 系统配置
- 配置键、值、类型
- 描述、分组
- 是否公开

#### SystemLog - 系统日志
- 用户ID、类型、动作
- 模块、IP、User Agent
- 请求、响应、状态
- 持续时间

#### ErrorLog - 错误日志
- 级别、模块、函数
- 消息、堆栈
- 文件、行号
- 上下文、解决状态

#### DeviceStatus - 设备状态
- 设备ID、名称、类型
- 状态、IP、位置
- 版本、最后心跳
- CPU、内存、磁盘使用率

## 🔧 数据库特性

### 索引优化
- 用户表：username、phone索引
- 游戏会话：user_id、status索引
- 交易表：user_id、type、created_at索引

### 软删除
所有核心模型都支持软删除（DeletedAt字段）

### JSON字段
使用JSONMap类型存储灵活的配置和元数据

### 事务支持
关键操作（充值、提现、游戏结算）使用事务确保数据一致性

### 钩子函数
- BeforeCreate：创建前的数据初始化
- 自定义验证和业务逻辑

## 📝 使用示例

```go
// 初始化数据库
database.Init(&config.Database)

// 自动迁移
database.AutoMigrate()

// 创建用户
user := &models.User{
    Username: "player001",
    Phone: "13800138000",
    Email: "player@example.com",
}
database.DB.Create(user)

// 查询用户
var user models.User
database.DB.Preload("Wallet").First(&user, "username = ?", "player001")

// 事务操作
database.Transaction(func(tx *gorm.DB) error {
    // 扣减余额
    // 创建交易记录
    // 更新游戏统计
    return nil
})
```

## 🚀 下一步计划

1. **性能优化**
   - 添加更多索引
   - 查询优化
   - 缓存策略

2. **功能扩展**
   - 排行榜系统
   - 活动系统
   - 社交功能

3. **数据分析**
   - 用户行为分析
   - 游戏数据统计
   - 财务报表