# 项目概览

## 项目基本信息
- **项目名称**: 老虎机游戏后端系统 (Slot Game Backend)
- **技术栈**: Go 1.21 + Gin + GORM + WebSocket + MQTT + SQLite
- **开发周期**: 4周目标
- **目标平台**: Ubuntu Desktop

## 项目架构
- **语言**: Go 1.21
- **Web框架**: Gin (RESTful API)
- **ORM**: GORM (支持SQLite/MySQL/PostgreSQL)
- **实时通信**: WebSocket (gorilla/websocket)
- **设备通信**: MQTT + 串口通信
- **配置管理**: Viper (YAML配置)
- **日志系统**: Zap + Lumberjack
- **认证**: JWT
- **数据库**: 默认SQLite，支持多数据库

## 目录结构
```
cmd/server/main.go           # 主程序入口
internal/
├── config/                  # 配置管理
├── models/                  # 数据模型
├── database/                # 数据库连接
├── repository/              # 数据访问层
├── service/                 # 业务服务层
├── game/                    # 游戏引擎
├── api/                     # HTTP API处理器
├── websocket/               # WebSocket通信
├── hardware/                # 硬件串口通信
├── middleware/              # 中间件
├── logger/                  # 日志系统
├── utils/                   # 工具库
└── errors/                  # 错误处理

config/config.yaml           # 配置文件
static/                      # 静态文件(Web界面)
test/                        # 测试文件
docs/                        # 文档
```

## 运行环境
- **操作系统**: Darwin (macOS) - 开发环境
- **目标部署**: Ubuntu Desktop
- **默认端口**: 8080 (HTTP + WebSocket)
- **数据库**: ./data/slot-game.db (SQLite)