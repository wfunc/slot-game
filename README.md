# 老虎机游戏后端（Slot Game Backend）

高性能 Go 后端，提供老虎机逻辑、钱包账务、JWT 认证、WebSocket 实时通信，以及可选的串口/MQTT 集成能力。内置 OpenAPI 文档与 Swagger/Redoc 在线页面，支持离线预览。

• 技术栈：Go 1.21+、Gin、GORM（SQLite）、Zap、JWT、gorilla/websocket  
• 可选扩展：tarm/serial、paho.mqtt.golang、gin-swagger、swag

参考文档：
- 开发 TODO（最新）：docs/development/backend-todo.md
- 执行任务看板：docs/development/backend-task-board.md
- API 快速指南：docs/API_QUICK_START.md
- 服务器运维指南：docs/SERVER_GUIDE.md
- OpenAPI（手工维护）：docs/api/openapi.yaml

## 目录结构
```
slot-game/
├── cmd/server/              # 主程序入口
├── internal/
│   ├── api/                # HTTP API/路由/handler
│   ├── game/               # 游戏引擎、状态机、会话管理
│   ├── repository/         # GORM 仓储与分页/事务封装（含大量单测）
│   ├── service/            # 业务服务：认证/用户/统计
│   ├── websocket/          # Hub-Client 与游戏消息处理
│   ├── database/           # DB 初始化与迁移
│   ├── middleware/         # JWT 等中间件
│   └── hardware/           # 串口控制（可选）
├── docs/                    # 项目文档（含 OpenAPI 与 Swagger 产物）
│   ├── api/openapi.yaml    # 手工维护的 OpenAPI 3 文档
│   └── swagger/            # swag 生成（可选构建产物）
├── static/                  # 静态资源（前端、离线文档资源）
│   └── vendors/            # redoc/swagger-ui 离线脚本（可选下载）
├── Makefile                 # 构建/运行/测试/文档
└── config/                  # 配置（复制 example 后编辑）
```

## 环境要求
- Go 1.21+（推荐 1.21 或更高）
- 可访问 Go 模块代理（或配置 GOPROXY）

## 快速开始
```bash
# 1) 安装依赖（填充 go.sum）
make deps

# 2) 准备配置
cp config/config.yaml.example config/config.yaml

# 3) 运行（默认读取 config/config.yaml）
make run

# 打开 Web 界面（如果需要）
# http://localhost:8080/static/index.html
```

## 主要能力
- 老虎机引擎：赔率/图案规则/1024线/级联玩法、RTP 控制与统计
- 游戏会话与状态机：持久化（内存+DB，可扩展 Redis）、异常恢复（预留退款/幂等等）
- 认证与用户：注册/登录/刷新/登出/资料/改密（JWT）
- 钱包与交易：余额、充值（测试）、提现（模拟）、交易记录与统计
- WebSocket 实时：游戏状态、结果、余额推送（Hub-Client 架构）
- 可选硬件/远程：串口控制器（推币、传感器）、MQTT 远程控制（预留）

## 运行与配置
```bash
# 运行（release 版）
GIN_MODE=release make run

# 指定配置文件路径
bin/slot-game-server -config=config/config.yaml

# 开发模式（go run）
make dev
```
配置说明见 config/config.yaml.example（Server/Database/WebSocket/Security/Game/Log 等）。

## API 文档
- OpenAPI（YAML）：
  - 在线：GET `/openapi` 或 `/openapi.yaml`
  - 本地文件：`docs/api/openapi.yaml`

- Redoc 页面（默认可用，支持深浅主题切换）：
  - 在线：GET `/docs/redoc`
  - 离线：`make fetch-redoc`（下载到 `static/vendors/redoc/`）

- 增强版 Swagger UI（带导航，默认可用）：
  - 在线：GET `/docs/ui`（CDN 渲染 /openapi；离线优先读取 `static/vendors/swagger-ui/*`）
  - 离线：`make fetch-swagger-ui`

- 原生 Swagger UI（gin-swagger；按需启用）：
  - 运行：`make run-swagger`（使用 `-tags swagger` 启动）
  - 访问：GET `/swagger/index.html`

- 使用 swag 生成 `docs/swagger`（可选）：
  ```bash
  make docs      # swag init -g cmd/server/main.go -o docs/swagger
  ```
  注意：当前原生 Swagger 已直接以 `/openapi` 作为数据源，无需依赖 `docs/swagger` 包。

## Makefile 常用命令
```bash
make deps              # 下载依赖、整理 go.mod/go.sum
make build             # 构建 bin/slot-game-server
make run               # 构建并运行
make dev               # go run 开发模式
make test              # 运行所有测试（设置 GOCACHE/GOTMPDIR 到本地目录）
make test-quiet        # 运行测试并过滤链接器警告
make coverage          # 生成覆盖率报告 coverage.html
make docs              # 生成 docs/swagger（swag）
make run-swagger       # 以 -tags swagger 运行，启用 /swagger 原生页面
make fetch-redoc       # 下载 Redoc 离线脚本到 static/vendors/redoc/
make fetch-swagger-ui  # 下载 Swagger UI 离线资源到 static/vendors/swagger-ui/
```

## WebSocket
- 连接：GET `/ws/game`（可选携带 JWT；未携带则访客模式）
- 推送类型：`game_start`/`game_result`/`balance_update`/`game_state` 等
- 详情：`internal/websocket/*` 与 `internal/api/websocket_handler.go`

## 测试
```bash
make test          # 使用本地 .gocache/.gotmp，避免权限问题
make test-repo     # 仅仓储层测试
make coverage      # 生成 coverage.html
```
如网络原因导致 go test 获取依赖失败，可设置：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

## 离线文档资源
- Redoc：`static/vendors/redoc/redoc.standalone.js`（`make fetch-redoc` 获取）
- Swagger UI：`static/vendors/swagger-ui/*`（`make fetch-swagger-ui` 获取）
两者均“本地优先、无则回退 CDN”。

## 发行与部署
- 发行包与 systemd 示例：`release/slot-game-arm64/*`
- 运行脚本：`release/slot-game-arm64/*.sh`
- 服务文件：`slot-game-sg.service` / `slot-game-ztl.service`（示例）

## 进阶与路线
- 开发 TODO（最新）：`docs/development/backend-todo.md`（已对齐真实代码实现）
- 执行任务看板：`docs/development/backend-task-board.md`（P0/P1 验收标准）
- 串口重构方案：`docs/development/serial-port-refactoring.md`
- 游戏集成指南：`docs/development/game-integration-guide.md`

## 常见问题（FAQ）
- swag 生成报错（gorm.DeletedAt 等）：
  - 已为基础模型增加 `swaggerignore:"true"`，或升级 swag 版本；必要时手工维护 `openapi.yaml`。
- Mac 下 go test 沙箱权限：
  - Makefile 已设置 GOCACHE/GOTMPDIR 到本地目录，可直接 `make test`。
- gin-swagger 与自定义文档路由冲突：
  - 已使用 `/swagger/*any` 作为原生页面，`/docs` 保留给 Redoc 与增强版 UI。

—
如需更多帮助，请查看 `docs/` 目录与 `internal/*` 实现，或直接提 issue。

