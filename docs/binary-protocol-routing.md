# 二进制协议路由系统

## 📝 概述

系统现在支持将前端的二进制协议自动路由到对应的游戏处理器，实现了协议的统一管理和分发。

## 🔌 WebSocket 端点

```
ws://localhost:8080/ws/binary
```

## 🎯 命令路由规则

| 命令范围 | 处理器 | 说明 |
|----------|--------|------|
| 1800-1899 | AnimalHandler | 动物游戏相关命令 |
| 1900-1999 | SlotHandler | 老虎机游戏相关命令 |
| 2000-2099 | ConfigHandler | 配置相关命令 |
| 1002 | HeartbeatHandler | 心跳命令 |

## 📊 具体命令映射

### Slot游戏命令 (1900-1999)
- `1901` - 进入房间
- `1902` - 开始游戏
- `1903` - 执行转动
- `1904` - 彩金推送
- `1905` - 游戏结算

### Animal游戏命令 (1800-1899)
- `1801` - 进入动物园
- `1802` - 下注
- `1803` - 开始游戏
- `1804` - 游戏结果
- `1805` - 退出游戏

### 配置命令 (2000-2099)
- `2001` - 获取游戏配置
- `2002` - 更新配置
- `2099` - 特殊配置查询（返回空成功响应）

## 🏗️ 系统架构

```
前端 (JavaScript)
    ↓ [二进制协议: 9字节头 + 数据]
/ws/binary 端点
    ↓
BinaryWebSocketHandler
    ↓
ProtocolClient (协议解析)
    ↓
BinaryProtocolRouter (路由分发)
    ├── SlotHandler (1900-1999)
    ├── AnimalHandler (1800-1899)
    └── ConfigHandler (2000-2099)
```

## 💡 使用示例

### 前端发送 Slot 游戏命令
```javascript
// 发送1901命令 - 进入房间
sendMessage(1901, flag, {room_type: "normal"});

// 服务端自动路由到 SlotHandler
// 返回：{"status":"success","room_id":1,"game":"slot"}
```

### 前端发送 Animal 游戏命令
```javascript
// 发送1801命令 - 进入动物园
sendMessage(1801, flag, {zoo_type: 1});

// 服务端自动路由到 AnimalHandler
// 返回：{"status":"success","game":"animal"}
```

### 前端发送配置命令
```javascript
// 发送2099命令 - 配置查询
sendMessage(2099, flag, {});

// 服务端返回空的成功响应（ErrorID=0, 无数据）
// 这样前端不会报错
```

## 🔧 核心组件

### BinaryProtocolRouter
- **作用**: 消息路由中心
- **职责**: 根据命令ID分发到对应处理器
- **文件**: `/internal/websocket/binary_protocol_router.go`

### 路由逻辑
```go
switch {
case msg.Cmd >= 1900 && msg.Cmd < 2000:
    // 路由到 SlotHandler
case msg.Cmd >= 1800 && msg.Cmd < 1900:
    // 路由到 AnimalHandler
case msg.Cmd >= 2000 && msg.Cmd < 2100:
    // 路由到 ConfigHandler
case msg.Cmd == 1002:
    // 处理心跳
default:
    // 返回未知命令错误
}
```

## ✨ 特性

1. **自动路由**: 根据命令ID自动分发到对应处理器
2. **协议转换**: 支持前端二进制协议与内部protobuf协议的转换
3. **错误处理**: 对未知命令返回友好的错误响应
4. **调试支持**: 详细的日志记录每个消息的路由过程
5. **扩展性强**: 轻松添加新的命令范围和处理器

## 📝 注意事项

1. **命令ID规划**: 确保不同游戏的命令ID不重叠
2. **错误码约定**: ErrorID=0 表示成功，非零表示错误
3. **数据格式**: Data字段通常使用JSON格式
4. **心跳机制**: 建议每30秒发送一次心跳(Cmd=1002)

## 🚀 后续优化

1. **完整集成**: 将SlotHandler和AnimalHandler的真实逻辑集成
2. **协议桥接**: 实现二进制协议到protobuf的完整转换
3. **状态管理**: 添加用户会话和游戏状态管理
4. **性能优化**: 使用对象池减少内存分配

---
*更新日期: 2025-09-18*
*版本: 1.1.0*