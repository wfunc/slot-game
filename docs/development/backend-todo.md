# 推币机游戏后端开发TODO List

## 项目概览
- **项目名称**: 推币机游戏后端系统
- **技术栈**: Go + GORM + SQLite + WebSocket + MQTT + Serial
- **开发周期**: 4周
- **目标平台**: Ubuntu Desktop

## 开发进度跟踪

### 📊 总体进度
- [ ] 阶段1: 项目初始化 (0/4)
- [ ] 阶段2: 数据层开发 (0/9)
- [ ] 阶段3: 游戏引擎开发 (0/12)
- [ ] 阶段4: 通信层开发 (0/15)
- [ ] 阶段5: 业务集成 (0/10)
- [ ] 阶段6: 监控与优化 (0/8)
- [ ] 阶段7: 测试与部署 (0/8)

---

## 阶段1: 项目初始化 [Week 1 - Day 1-2]

### 环境搭建
- [ ] 初始化Go项目结构
  ```bash
  mkdir -p cmd/server internal/{game,hardware,comm,data,config}
  go mod init github.com/wfunc/slot-game
  ```
- [ ] 配置go.mod依赖管理
  ```go
  // 需要添加的依赖
  // gorm.io/gorm
  // gorm.io/driver/sqlite
  // github.com/gorilla/websocket
  // github.com/eclipse/paho.mqtt.golang
  // github.com/tarm/serial
  // github.com/spf13/viper
  // go.uber.org/zap
  ```
- [ ] 设置开发环境变量
  ```bash
  export SLOT_GAME_ENV=development
  export SLOT_GAME_CONFIG=/path/to/config.yaml
  ```
- [ ] 创建配置文件模板 `config/config.yaml.example`

### 基础框架
- [ ] 实现配置管理器(viper)
  - 文件: `internal/config/config.go`
  - 功能: 加载yaml配置、环境变量覆盖、热更新
- [ ] 设置日志系统(zap)
  - 文件: `internal/logger/logger.go`
  - 功能: 分级日志、文件轮转、结构化日志
- [ ] 创建错误处理机制
  - 文件: `internal/errors/errors.go`
  - 功能: 错误码定义、错误包装、错误追踪
- [ ] 实现优雅关闭
  - 文件: `cmd/server/main.go`
  - 功能: 信号处理、资源清理、连接关闭

---

## 阶段2: 数据层开发 [Week 1 - Day 3-5]

### 数据库初始化
- [ ] 集成GORM框架
  - 文件: `internal/data/database.go`
  - 功能: 连接管理、连接池配置
- [ ] 创建数据库连接池
  ```go
  // 配置连接池参数
  // MaxIdleConns: 10
  // MaxOpenConns: 100
  // ConnMaxLifetime: time.Hour
  ```
- [ ] 实现自动迁移机制
  - 文件: `internal/data/migration.go`
  - 功能: 版本管理、升级降级
- [ ] 编写数据模型
  - 文件: `internal/data/models/*.go`
  - 模型: GameRecord, WinRecord, DeviceStatus, SystemConfig

### 数据访问层
- [ ] 实现GameRecord仓储
  - 文件: `internal/data/repository/game_record.go`
  - 方法: Create, Update, FindBySessionID, GetStatistics
- [ ] 实现WinRecord仓储
  - 文件: `internal/data/repository/win_record.go`
  - 方法: Create, FindByGameID, GetWinStatistics
- [ ] 实现DeviceStatus仓储
  - 文件: `internal/data/repository/device_status.go`
  - 方法: UpdateStatus, GetStatus, GetHealthReport
- [ ] 实现Config仓储
  - 文件: `internal/data/repository/config.go`
  - 方法: Get, Set, GetAll, Update
- [ ] 添加事务支持
  - 文件: `internal/data/transaction.go`
  - 功能: Begin, Commit, Rollback, WithTransaction

---

## 阶段3: 游戏引擎开发 [Week 2 - Day 1-3]

### 老虎机逻辑
- [ ] 实现随机数生成器
  - 文件: `internal/game/random.go`
  - 算法: MT19937、种子管理
- [ ] 创建中奖规则引擎
  - 文件: `internal/game/rules.go`
  - 规则: 图案组合、概率配置
- [ ] 开发赔率计算器
  - 文件: `internal/game/calculator.go`
  - 功能: 赔率计算、奖金结算
- [ ] 实现转轮动画数据生成
  - 文件: `internal/game/animation.go`
  - 数据: 转轮序列、停止位置

### 推币机控制
- [ ] 定义推币策略接口
  - 文件: `internal/game/push_strategy.go`
  - 接口: IPushStrategy
- [ ] 实现推币力度算法
  - 文件: `internal/game/push_force.go`
  - 算法: 基于中奖等级的力度计算
- [ ] 创建推币序列生成器
  - 文件: `internal/game/push_sequence.go`
  - 功能: 生成推币命令序列
- [ ] 开发币数管理器
  - 文件: `internal/game/coin_manager.go`
  - 功能: 币数统计、进出账管理

### 游戏状态机
- [ ] 设计状态转换图
  - 文件: `docs/state-machine.md`
  - 内容: 状态定义、转换规则
- [ ] 实现状态机框架
  - 文件: `internal/game/state_machine.go`
  - 功能: 状态管理、转换验证
- [ ] 添加状态持久化
  - 文件: `internal/game/state_persistence.go`
  - 功能: 状态保存、恢复
- [ ] 处理异常状态恢复
  - 文件: `internal/game/recovery.go`
  - 功能: 异常检测、自动恢复

---

## 阶段4: 通信层开发 [Week 2 - Day 4-5, Week 3 - Day 1-2]

### 串口通信
- [ ] 集成go-serial库
  ```go
  import "github.com/tarm/serial"
  ```
- [ ] 实现串口管理器
  - 文件: `internal/hardware/serial_manager.go`
  - 功能: 连接管理、重连机制
- [ ] 创建命令队列
  - 文件: `internal/hardware/command_queue.go`
  - 功能: 优先级队列、批处理
- [ ] 开发协议编解码器
  - 文件: `internal/hardware/protocol.go`
  - 协议: 命令格式、响应解析
- [ ] 添加重试机制
  - 文件: `internal/hardware/retry.go`
  - 策略: 指数退避、最大重试

### WebSocket服务
- [ ] 集成gorilla/websocket
  ```go
  import "github.com/gorilla/websocket"
  ```
- [ ] 实现连接管理器
  - 文件: `internal/comm/ws_manager.go`
  - 功能: 连接池、会话管理
- [ ] 创建消息路由器
  - 文件: `internal/comm/ws_router.go`
  - 路由: 消息分发、处理器注册
- [ ] 开发心跳机制
  - 文件: `internal/comm/ws_heartbeat.go`
  - 功能: 定期ping、超时检测
- [ ] 实现断线重连
  - 文件: `internal/comm/ws_reconnect.go`
  - 功能: 自动重连、状态恢复

### MQTT客户端
- [ ] 集成paho.mqtt.golang
  ```go
  import mqtt "github.com/eclipse/paho.mqtt.golang"
  ```
- [ ] 实现MQTT连接管理
  - 文件: `internal/comm/mqtt_client.go`
  - 功能: 连接建立、断线重连
- [ ] 创建主题订阅器
  - 文件: `internal/comm/mqtt_subscriber.go`
  - 主题: 设备控制、配置更新
- [ ] 开发消息处理器
  - 文件: `internal/comm/mqtt_handler.go`
  - 处理: 指令解析、响应生成
- [ ] 添加QoS保证
  - 文件: `internal/comm/mqtt_qos.go`
  - 级别: QoS 0/1/2配置

---

## 阶段5: 业务集成 [Week 3 - Day 3-4]

### 游戏流程
- [ ] 实现投币检测
  - 文件: `internal/game/coin_detector.go`
  - 功能: 信号接收、验证
- [ ] 开发游戏启动流程
  - 文件: `internal/game/game_starter.go`
  - 流程: 初始化、状态设置
- [ ] 集成老虎机转动
  - 文件: `internal/game/slot_spin.go`
  - 功能: 转动控制、结果生成
- [ ] 实现中奖结算
  - 文件: `internal/game/settlement.go`
  - 功能: 奖金计算、记录保存
- [ ] 触发推币动作
  - 文件: `internal/game/push_trigger.go`
  - 功能: 推币指令生成、执行

### 远程控制
- [ ] 实现设备注册
  - 文件: `internal/remote/device_register.go`
  - 功能: 设备ID生成、注册上报
- [ ] 开发指令解析器
  - 文件: `internal/remote/command_parser.go`
  - 指令: 参数更新、控制命令
- [ ] 创建参数更新机制
  - 文件: `internal/remote/config_updater.go`
  - 功能: 热更新、验证
- [ ] 实现状态上报
  - 文件: `internal/remote/status_reporter.go`
  - 周期: 定时上报、事件触发
- [ ] 添加远程诊断
  - 文件: `internal/remote/diagnostics.go`
  - 功能: 日志上传、远程调试

---

## 阶段6: 监控与优化 [Week 4 - Day 1-2]

### 系统监控
- [ ] 添加性能指标采集
  - 文件: `internal/monitor/metrics.go`
  - 指标: CPU、内存、goroutine
- [ ] 实现健康检查接口
  - 文件: `internal/monitor/health.go`
  - 接口: /health、/ready
- [ ] 创建告警机制
  - 文件: `internal/monitor/alert.go`
  - 告警: 阈值设置、通知发送
- [ ] 开发诊断工具
  - 文件: `internal/monitor/diagnostic.go`
  - 工具: pprof集成、trace

### 稳定性保障
- [ ] 实现熔断器模式
  - 文件: `internal/resilience/circuit_breaker.go`
  - 功能: 故障隔离、自动恢复
- [ ] 添加限流控制
  - 文件: `internal/resilience/rate_limiter.go`
  - 算法: 令牌桶、滑动窗口
- [ ] 创建降级策略
  - 文件: `internal/resilience/degradation.go`
  - 策略: 功能降级、优先级
- [ ] 开发故障恢复
  - 文件: `internal/resilience/recovery.go`
  - 机制: 自动重启、状态恢复

---

## 阶段7: 测试与部署 [Week 4 - Day 3-5]

### 单元测试
- [ ] 游戏逻辑测试
  - 文件: `internal/game/*_test.go`
  - 覆盖率: > 80%
- [ ] 通信模块测试
  - 文件: `internal/comm/*_test.go`
  - 场景: 正常、异常、边界
- [ ] 数据层测试
  - 文件: `internal/data/*_test.go`
  - 测试: CRUD、事务
- [ ] 集成测试
  - 文件: `test/integration/*_test.go`
  - 流程: 端到端测试

### 部署准备
- [ ] 创建systemd服务
  - 文件: `scripts/slot-game.service`
  ```ini
  [Unit]
  Description=Slot Game Backend Service
  After=network.target
  
  [Service]
  Type=simple
  User=slot-game
  WorkingDirectory=/opt/slot-game
  ExecStart=/opt/slot-game/bin/slot-game-server
  Restart=always
  
  [Install]
  WantedBy=multi-user.target
  ```
- [ ] 编写部署脚本
  - 文件: `scripts/deploy.sh`
  - 功能: 编译、打包、安装
- [ ] 配置日志轮转
  - 文件: `/etc/logrotate.d/slot-game`
  - 策略: 按大小、按时间
- [ ] 准备监控脚本
  - 文件: `scripts/monitor.sh`
  - 监控: 进程、端口、日志

---

## 代码规范检查清单

### 代码质量
- [ ] 所有函数都有注释
- [ ] 错误处理完整
- [ ] 没有魔法数字
- [ ] 变量命名清晰
- [ ] 函数职责单一

### 测试覆盖
- [ ] 单元测试覆盖率 > 80%
- [ ] 关键路径100%覆盖
- [ ] 异常场景测试完整
- [ ] 性能测试通过
- [ ] 压力测试通过

### 文档完整性
- [ ] API文档完整
- [ ] 部署文档详细
- [ ] 配置说明清晰
- [ ] 故障排查指南
- [ ] 维护手册完善

---

## 里程碑追踪

| 里程碑 | 预计完成日期 | 实际完成日期 | 状态 |
|--------|-------------|-------------|------|
| M1: 项目框架搭建 | Week 1 - Day 2 | - | ⏳ |
| M2: 数据层完成 | Week 1 - Day 5 | - | ⏳ |
| M3: 游戏引擎完成 | Week 2 - Day 3 | - | ⏳ |
| M4: 通信层完成 | Week 3 - Day 2 | - | ⏳ |
| M5: 业务集成完成 | Week 3 - Day 4 | - | ⏳ |
| M6: 测试通过 | Week 4 - Day 4 | - | ⏳ |
| M7: 正式交付 | Week 4 - Day 5 | - | ⏳ |

---

## 风险跟踪

| 风险项 | 当前状态 | 缓解措施 | 负责人 |
|--------|---------|----------|--------|
| 串口通信不稳定 | 🟢 未发生 | 重试机制、缓冲队列 | - |
| 游戏状态丢失 | 🟢 未发生 | 定期持久化 | - |
| 网络断线 | 🟢 未发生 | 自动重连 | - |
| 硬件故障 | 🟢 未发生 | 故障检测 | - |
| 性能瓶颈 | 🟢 未发生 | 性能优化 | - |

---

## 备注

### 优先级说明
- 🔴 P0: 必须完成，核心功能
- 🟡 P1: 应该完成，重要功能
- 🟢 P2: 可以延后，优化功能

### 时间估算
- 每个任务预估时间：2-4小时
- 每日有效工作时间：6小时
- 缓冲时间：20%

### 依赖关系
1. 数据层必须先于业务层
2. 通信层可以并行开发
3. 测试贯穿全程
4. 部署在最后阶段

---

## 每日站会模板

```markdown
### 日期：2024-XX-XX

#### 昨天完成
- [ ] 任务1
- [ ] 任务2

#### 今天计划
- [ ] 任务3
- [ ] 任务4

#### 遇到问题
- 问题描述
- 解决方案

#### 需要支持
- 资源需求
- 技术支持
```