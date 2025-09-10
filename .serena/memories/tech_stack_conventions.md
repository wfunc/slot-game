# 技术栈和代码规范

## 核心技术栈
- **Go版本**: 1.21
- **Web框架**: Gin (RESTful API路由)
- **ORM**: GORM v1.25.5 (数据库ORM)
- **WebSocket**: gorilla/websocket v1.5.1
- **配置管理**: Viper v1.18.2 (YAML配置)
- **日志**: Zap v1.26.0 + Lumberjack v2.2.1
- **JWT**: golang-jwt/jwt/v5 v5.2.0
- **测试**: testify v1.8.4
- **加密**: golang.org/x/crypto v0.16.0
- **数据库驱动**: SQLite/MySQL/PostgreSQL

## 代码规范和风格

### 命名规范
- **包名**: 小写，简洁，如`config`, `models`, `service`
- **文件名**: 小写下划线，如`user_service.go`, `game_handler.go`
- **结构体**: 大驼峰，如`UserService`, `GameSession`
- **方法/函数**: 大驼峰(导出)，小驼峰(私有)
- **常量**: 大写下划线，如`DEFAULT_PAGE_SIZE`
- **变量**: 小驼峰，如`userID`, `gameSession`

### 项目结构规范
- **internal/**: 内部包，不对外暴露
- **cmd/**: 可执行程序入口
- **config/**: 配置文件
- **test/**: 测试文件和测试工具
- **docs/**: 项目文档
- **static/**: 静态资源文件

### 注释规范
- **包注释**: 每个包都有包级别注释
- **结构体注释**: 导出的结构体必须有注释
- **方法注释**: 导出的方法必须有注释，说明功能和参数
- **复杂逻辑**: 内部复杂逻辑要有行内注释

### 错误处理
- 使用自定义错误类型: `internal/errors/errors.go`
- 错误包装: 使用`errors.Wrap()`进行错误包装
- 错误码: 定义统一的错误码系统
- 错误日志: 重要错误必须记录日志

### 测试规范
- **测试文件**: `*_test.go`后缀
- **测试覆盖率**: 目标>80%，核心模块>90%
- **测试工具**: 使用testify/assert和testify/mock
- **集成测试**: 重要模块需要集成测试

### 配置管理
- **配置文件**: YAML格式，结构化配置
- **环境变量**: 支持环境变量覆盖配置
- **默认值**: 所有配置项都有合理默认值
- **热更新**: 支持配置文件热更新(通过fsnotify)