# 服务器启动指南

## 🚀 三种服务器启动方式对比

你的项目中有三个不同的服务器启动程序，各有不同的用途：

### 1. **cmd/server/main.go** - 完整企业级服务器 🏢

**用途**: 生产环境的完整服务器
**特点**:
- ✅ 完整的企业级架构
- ✅ 数据库集成 (MySQL/PostgreSQL)
- ✅ 用户认证系统
- ✅ 配置文件管理
- ✅ 日志系统
- ✅ 优雅关闭
- ✅ 健康检查
- ❌ 复杂配置要求

**启动命令**:
```bash
go run cmd/server/main.go
```

**适用场景**: 生产环境、需要用户管理、数据持久化的完整应用

---

### 2. **cmd/simple-api/main.go** - 简化API服务器 🎮

**用途**: 金色Wild麻将游戏API专用服务器  
**特点**:
- ✅ 专注游戏逻辑
- ✅ 即开即用，无需配置
- ✅ 内存会话管理
- ✅ RESTful API
- ✅ 金色Wild功能完整
- ❌ 无数据库持久化
- ❌ 无用户认证

**启动命令**:
```bash
go run cmd/simple-api/main.go
```

**API地址**: http://localhost:8080
**适用场景**: 游戏开发、前端集成测试、演示展示

---

### 3. **cmd/api/main.go** - 兼容性启动器 🔄

**用途**: 为了兼容性而存在，实际上调用简化API
**特点**:
- ✅ 与 simple-api 功能相同
- ✅ 为了兼容现有脚本而保留

**启动命令**:
```bash
go run cmd/api/main.go
```

**说明**: 这个文件与 `cmd/simple-api/main.go` 功能完全相同

---

## 🎯 **推荐使用方案**

### 对于你的金色Wild麻将游戏开发：

```bash
# 推荐使用这个 - 最简单直接
go run cmd/simple-api/main.go
```

**为什么推荐 simple-api**:
1. **专门为你的游戏设计** - 包含完整的金色Wild机制
2. **即开即用** - 无需复杂配置
3. **专注游戏逻辑** - 不包含不必要的企业功能
4. **完整API** - 支持前端所需的所有接口
5. **测试友好** - 内存存储，重启即重置

### API功能对比表

| 功能 | simple-api | server |
|------|------------|---------|
| 游戏会话管理 | ✅ | ✅ |
| 金色Wild旋转 | ✅ | ❌ |
| 1024线匹配 | ✅ | ❌ |
| 连锁消除 | ✅ | ❌ |
| 统计查询 | ✅ | ✅ |
| 用户注册/登录 | ❌ | ✅ |
| 数据库持久化 | ❌ | ✅ |
| 复杂配置 | ❌ | ✅ |

---

## 🛠️ **使用示例**

### 启动游戏API服务器
```bash
cd /Users/mini/Documents/GitHub/wfunc/slot-game
go run cmd/simple-api/main.go
```

### 测试API
```bash
# 健康检查
curl http://localhost:8080/health

# 创建游戏会话
curl -X POST http://localhost:8080/api/v1/session \
  -H "Content-Type: application/json" \
  -d '{"player_id":"player123","initial_balance":10000}'

# 游戏旋转
curl -X POST http://localhost:8080/api/v1/spin \
  -H "Content-Type: application/json" \
  -d '{"session_id":"session_player123_xxx","bet_amount":100}'
```

### 运行测试客户端
```bash
go run test/api_test_client.go
```

---

## 📁 **文件清理建议**

为了避免混淆，建议：

1. **保留** `cmd/simple-api/main.go` - 主要使用
2. **删除** `cmd/api/main.go` - 重复功能
3. **保留** `cmd/server/main.go` - 未来扩展用

```bash
# 可选：删除重复文件
rm cmd/api/main.go
```

---

## 🎮 **总结**

**对于你的金色Wild麻将游戏项目，使用这个命令启动服务器：**

```bash
go run cmd/simple-api/main.go
```

这个版本包含了你所有的游戏特性，API完整，易于使用，非常适合游戏开发和前端集成！🎰✨