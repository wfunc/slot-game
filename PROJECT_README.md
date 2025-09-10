# 推币机游戏后端系统

电玩城推币机游戏控制系统，结合老虎机玩法，提供本地控制和远程管理功能。

## 🚀 项目状态

### 阶段1: 项目初始化 ✅ 完成

- [x] 初始化Go项目结构
- [x] 配置go.mod依赖管理
- [x] 创建配置文件模板
- [x] 实现配置管理器(viper)
- [x] 设置日志系统(zap)
- [x] 创建错误处理机制
- [x] 实现优雅关闭

## 📁 项目结构

```
slot-game/
├── cmd/
│   └── server/         # 主程序入口
│       └── main.go     # 服务器主文件（优雅关闭）
├── internal/
│   ├── config/         # 配置管理
│   │   └── config.go   # Viper配置管理器
│   ├── logger/         # 日志系统
│   │   └── logger.go   # Zap日志封装
│   ├── errors/         # 错误处理
│   │   └── errors.go   # 错误码和错误包装
│   ├── game/          # 游戏逻辑（待实现）
│   ├── hardware/      # 硬件控制（待实现）
│   ├── comm/          # 通信模块（待实现）
│   └── data/          # 数据层（待实现）
├── config/
│   └── config.yaml.example  # 配置文件模板
├── docs/
│   ├── prd/           # 产品需求文档
│   ├── architecture/  # 架构设计文档
│   └── development/   # 开发任务清单
├── go.mod             # Go模块定义
└── Makefile          # 构建脚本
```

## 🛠️ 技术栈

- **语言**: Go 1.21+
- **配置管理**: Viper
- **日志系统**: Zap
- **数据库**: GORM + SQLite
- **通信协议**: 
  - WebSocket (gorilla/websocket)
  - MQTT (paho.mqtt.golang)
  - Serial (tarm/serial)

## 📦 安装依赖

```bash
# 安装Go依赖
make deps

# 或手动安装
go mod download
go mod tidy
```

## 🚀 快速开始

### 1. 复制配置文件

```bash
cp config/config.yaml.example config/config.yaml
```

### 2. 编译项目

```bash
make build
```

### 3. 运行服务器

```bash
make run

# 或开发模式
make dev
```

### 4. 查看帮助

```bash
./bin/slot-game-server -help
```

## 📝 配置说明

配置文件 `config/config.yaml` 包含以下主要配置项：

- **server**: 服务器基础配置
- **database**: SQLite数据库配置
- **websocket**: WebSocket服务配置
- **serial**: 串口通信配置
- **mqtt**: MQTT远程控制配置
- **game**: 游戏参数配置
- **log**: 日志系统配置

## 🔧 Makefile命令

```bash
make build       # 构建项目
make run         # 运行项目
make dev         # 开发模式
make test        # 运行测试
make fmt         # 格式化代码
make lint        # 代码检查
make clean       # 清理构建
make help        # 查看帮助
```

## 📊 开发进度

- **阶段1**: 项目初始化 ✅ 100%
- **阶段2**: 数据层开发 ⏳ 0%
- **阶段3**: 游戏引擎开发 ⏳ 0%
- **阶段4**: 通信层开发 ⏳ 0%
- **阶段5**: 业务集成 ⏳ 0%
- **阶段6**: 监控与优化 ⏳ 0%
- **阶段7**: 测试与部署 ⏳ 0%

## 🔍 当前功能

### ✅ 已实现

1. **配置管理**
   - YAML配置文件加载
   - 环境变量覆盖
   - 配置热更新

2. **日志系统**
   - 结构化日志
   - 日志分级
   - 文件轮转
   - 模块化日志

3. **错误处理**
   - 错误码定义
   - 错误包装
   - 调用栈追踪

4. **优雅关闭**
   - 信号处理
   - 超时控制
   - 资源清理

### ⏳ 待实现

- 游戏逻辑引擎
- 串口硬件控制
- WebSocket通信
- MQTT远程控制
- 数据持久化
- 监控告警

## 📚 文档

- [产品需求文档](docs/prd/backend-prd.md)
- [系统架构设计](docs/architecture/system-design.md)
- [开发任务清单](docs/development/backend-todo.md)
- [OpenAPI 文档](docs/api/openapi.yaml)

## 📘 预览 API 文档（OpenAPI）

方法一：使用 swagger-ui 容器（无需安装依赖）

```bash
docker run --rm -p 8081:8080 \
  -e SWAGGER_JSON=/spec/openapi.yaml \
  -v $(pwd)/docs/api/openapi.yaml:/spec/openapi.yaml \
  swaggerapi/swagger-ui

# 打开浏览器访问 http://localhost:8081
```

方法二：使用 Redocly（需要 Node 环境）

```bash
npx @redocly/cli preview-docs docs/api/openapi.yaml
# 打开输出的本地预览地址
```

方法三：VSCode 插件

- 安装 "OpenAPI (Swagger) Editor" 或 "Redocly" 插件，直接打开 `docs/api/openapi.yaml` 即可预览。

备注：如果需要基于源码注解自动生成文档，可安装 `swag` 并使用：`make docs`（输出到 `docs/swagger/`）。

## 🧭 内置文档页面（gin-swagger）

通过构建标签启用 swagger 路由（避免默认构建引入额外依赖）。

```bash
# 1) 生成 swagger 文档（基于源码注解）
make docs            # 产物：docs/swagger（Go 包，含 doc.json）

# 2) 运行服务并启用 /swagger 页面
make run-swagger     # 等价于：go run -tags swagger cmd/server/main.go -config=config/config.yaml

# 3) 浏览器访问
http://localhost:8080/swagger/index.html

增强版 Swagger UI（带导航）

```text
http://localhost:8080/docs/ui
```
说明：该页面基于 swagger-ui-dist（CDN 或本地）渲染 /openapi，并在顶部提供跳转入口到 Redoc 与 YAML。
```

说明：
- 非 swagger 构建不依赖 gin-swagger；仅在 `-tags swagger` 时才编译相关依赖与路由。
- 仍可使用 `docs/api/openapi.yaml` 进行手工维护与预览。

## 🌐 在线查看（无 swag 环境）

无需生成 swagger 文档，也可直接访问：

- OpenAPI YAML: `GET /openapi` 或 `GET /openapi.yaml`
- Redoc 页面: `GET /docs/redoc`（浏览器加载 CDN 脚本渲染）

说明：服务器需能访问 Redoc CDN 才能渲染 UI；若离线环境，可将 Redoc 脚本打包到 `static/` 并修改路由引用。

## 📴 离线 Redoc 使用

1) 从有网络环境下载 Redoc 脚本到本项目：

```bash
make fetch-redoc
# 或手动：
# curl -fsSL https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js \
#   -o static/vendors/redoc/redoc.standalone.js
```

2) 启动服务后访问：`http://localhost:8080/docs/redoc`

说明：路由会优先使用 `static/vendors/redoc/redoc.standalone.js`，若不存在则回退到 CDN。

## 📴 离线 Swagger UI 使用（增强版 /docs/ui）

1) 下载静态资源到项目：

```bash
make fetch-swagger-ui
# 或手动：
# curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui.css \
#   -o static/vendors/swagger-ui/swagger-ui.css
# curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js \
#   -o static/vendors/swagger-ui/swagger-ui-bundle.js
# curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js \
#   -o static/vendors/swagger-ui/swagger-ui-standalone-preset.js
```

2) 启动服务后访问：`http://localhost:8080/docs/ui`

说明：该页面会优先加载 `static/vendors/swagger-ui/*`，不存在时回退到 CDN。

## 🔗 下一步计划

1. 实现数据层（GORM + SQLite）
2. 开发游戏引擎核心逻辑
3. 集成串口通信
4. 实现WebSocket服务
5. 添加MQTT远程控制

## 📄 许可证

私有项目，版权所有。
