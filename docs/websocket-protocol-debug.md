# WebSocket Protocol Debug Summary

## 发现的问题和解决方案

### 1. 协议格式

前端和后端的二进制协议格式已确认：

#### 客户端 → 服务端（9字节头）
```
DataSize   (2字节) - 数据长度
DataStatus (1字节) - 数据状态
Flag       (4字节) - 消息标识
Cmd        (2字节) - 命令ID
Data       (n字节) - 数据内容
```

#### 服务端 → 客户端（11字节头）
```
ErrorID    (2字节) - 错误码（0=成功）
DataSize   (2字节) - 数据长度
DataStatus (1字节) - 数据状态
Flag       (4字节) - 消息标识
Cmd        (2字节) - 命令ID
Data       (n字节) - 数据内容
```

### 2. 当前实现状态

✅ **已完成**：
- 二进制协议的编解码（protocol.go）
- 命令路由系统（binary_protocol_router.go）
- WebSocket端点配置（/ws/binary）
- 详细的调试日志

⚠️ **待完成**：
- 实际的protobuf消息生成（目前返回空数据）
- SlotHandler和AnimalHandler的真实逻辑集成
- 数据格式转换（前端protobuf ↔ 后端处理）

### 3. 调试日志分析

从最新的日志可以看到完整的消息流程：

```
[1] 收到客户端消息: cmd=2099, flag=69, data_len=0
[2] 调用消息处理器: BinaryProtocolRouter
[路由] 开始路由消息
[路由] 处理2099命令 - 配置查询
[3] 准备发送响应: error_id=0, data_len=0
[4] 开始编码服务端消息
[5] 编码后的消息: 0000000000000000450833 (11 bytes)
[6] 将消息加入发送队列
[7] 消息已加入发送队列
[8] 通过WebSocket发送二进制数据
[9] 消息发送成功
```

发送的字节解析：
- `00 00` - ErrorID = 0（成功）
- `00 00` - DataSize = 0（无数据）
- `00` - DataStatus = 0
- `00 00 00 45` - Flag = 69
- `08 33` - Cmd = 2099
- 无Data部分

### 4. 前端报错原因

前端报错 "RangeError: index out of range" 的原因是：
- 前端期望接收protobuf格式的数据
- 后端目前返回空数据或JSON格式
- 前端尝试解析protobuf时失败

### 5. 解决方案

需要在BinaryProtocolRouter中实现：

1. **集成真实的Handler**：
   ```go
   // 调用真实的SlotHandler
   response := r.slotHandler.HandleMessage(msg)
   ```

2. **生成正确的protobuf响应**：
   ```go
   // 使用protobuf编码器生成响应
   protoData := r.codec.Encode(messageID, protoMessage)
   ```

3. **数据格式转换**：
   - 解析前端的protobuf请求
   - 调用对应的业务逻辑
   - 生成protobuf响应

### 6. 测试工具

创建了 `test/websocket_test.html` 用于测试：
- 可以发送各种命令
- 显示发送和接收的原始字节
- 解析消息头信息
- 验证协议格式

### 7. 下一步计划

1. 实现protobuf消息的序列化/反序列化
2. 集成真实的游戏逻辑处理器
3. 添加更多的命令支持
4. 完善错误处理和状态管理

## 使用说明

### 测试WebSocket连接

1. 启动服务器：
   ```bash
   go run cmd/server/main.go
   ```

2. 打开测试页面：
   ```
   test/websocket_test.html
   ```

3. 点击各个命令按钮测试

### 查看调试日志

服务端日志会显示详细的消息处理流程：
- `[1]` - 接收客户端消息
- `[2]` - 调用处理器
- `[路由]` - 路由决策
- `[3-9]` - 响应发送流程

---

更新时间：2025-09-18