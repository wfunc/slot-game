# WebSocket 协议集成文档

## 📝 概述

系统现在支持两种 WebSocket 协议格式，以满足不同客户端的需求。

## 🔌 WebSocket 端点

### 1. Protobuf 协议端点（原有）
```
ws://localhost:8080/ws/game
ws://localhost:8080/ws/slot
```
- **协议格式**: `[4字节长度][2字节消息ID][protobuf数据]`
- **用途**: 支持完整的游戏功能（老虎机、动物等）
- **消息ID范围**: 1901-1905（游戏）, 2001-2099（配置）

### 2. 二进制协议端点（新增）
```
ws://localhost:8080/ws/binary
```
- **协议格式**:
  - 客户端→服务端: `[2字节DataSize][1字节DataStatus][4字节Flag][2字节Cmd][Data]` (9字节头)
  - 服务端→客户端: `[2字节ErrorID][2字节DataSize][1字节DataStatus][4字节Flag][2字节Cmd][Data]` (11字节头)
- **用途**: 支持前端自定义二进制协议
- **字节序**: 大端序（Big-Endian）

## 📊 协议对比

| 特性 | Protobuf 协议 | 二进制协议 |
|------|--------------|------------|
| 端点 | `/ws/game`, `/ws/slot` | `/ws/binary` |
| 消息格式 | Protobuf | 自定义二进制 |
| 头部大小 | 6字节 | 9字节(C→S), 11字节(S→C) |
| 数据编码 | Protobuf序列化 | JSON或自定义 |
| 错误处理 | 通过消息体 | ErrorID字段 |

## 🎮 命令定义

### 通用命令（二进制协议）
```go
const (
    CmdLogin     uint16 = 1001 // 登录
    CmdHeartbeat uint16 = 1002 // 心跳
    CmdGame      uint16 = 1003 // 游戏

    // 游戏相关命令
    CmdSlotStart     uint16 = 2001 // 老虎机开始
    CmdSlotSpin      uint16 = 2002 // 老虎机转动
    CmdSlotSettle    uint16 = 2003 // 老虎机结算
    CmdSlotBatchSpin uint16 = 2004 // 批量转动

    // 钱包相关命令
    CmdWalletBalance  uint16 = 3001 // 查询余额
    CmdWalletDeposit  uint16 = 3002 // 充值
    CmdWalletWithdraw uint16 = 3003 // 提现
)
```

### 错误码定义
```go
const (
    ErrSuccess       uint16 = 0    // 成功
    ErrUnknown       uint16 = 1000 // 未知错误
    ErrDecode        uint16 = 1001 // 解码失败
    ErrProcess       uint16 = 1002 // 处理失败
    ErrAuth          uint16 = 2000 // 认证失败
    ErrInvalidData   uint16 = 2001 // 数据无效
    ErrGameError     uint16 = 3000 // 游戏错误
    ErrInvalidBet    uint16 = 3001 // 投注无效
    ErrInsufficientBalance uint16 = 3002 // 余额不足
)
```

## 🛠️ 使用示例

### JavaScript 客户端连接示例
```javascript
// 连接到二进制协议端点
const ws = new WebSocket('ws://localhost:8080/ws/binary');
ws.binaryType = 'arraybuffer';

// 发送消息
function sendMessage(cmd, flag, data) {
    const jsonData = JSON.stringify(data);
    const textEncoder = new TextEncoder();
    const dataBytes = textEncoder.encode(jsonData);

    const buffer = new ArrayBuffer(9 + dataBytes.length);
    const dataView = new DataView(buffer);

    // 写入头部
    dataView.setUint16(0, dataBytes.length); // DataSize
    dataView.setUint8(2, 0);                 // DataStatus
    dataView.setUint32(3, flag);             // Flag
    dataView.setUint16(7, cmd);              // Cmd

    // 写入数据
    const uint8Array = new Uint8Array(buffer);
    uint8Array.set(dataBytes, 9);

    ws.send(buffer);
}

// 接收消息
ws.onmessage = (event) => {
    const dataView = new DataView(event.data);

    // 解析头部
    const errorID = dataView.getUint16(0);
    const dataSize = dataView.getUint16(2);
    const dataStatus = dataView.getUint8(4);
    const flag = dataView.getUint32(5);
    const cmd = dataView.getUint16(9);

    // 解析数据
    const dataBytes = new Uint8Array(event.data, 11, dataSize);
    const textDecoder = new TextDecoder();
    const jsonStr = textDecoder.decode(dataBytes);
    const data = JSON.parse(jsonStr);

    console.log('收到消息:', { errorID, cmd, flag, data });
};
```

## 📁 相关文件

### 核心实现
- `/internal/websocket/protocol.go` - 二进制协议编解码
- `/internal/websocket/protocol_handler.go` - 协议客户端处理器
- `/internal/websocket/protocol_example.go` - 示例消息处理器
- `/internal/websocket/protocol_test.go` - 协议测试

### API集成
- `/internal/api/binary_websocket_handler.go` - 二进制协议WebSocket处理器
- `/internal/api/protobuf_websocket_handler.go` - Protobuf协议WebSocket处理器
- `/internal/api/router.go` - 路由配置

## 🔍 调试方法

### 启用调试日志
```go
protocol := ws.NewProtocol()
protocol.Debug = true // 开启调试日志
```

### 查看原始数据
在 `protobuf_codec.go` 中已添加详细的调试输出，可以查看：
- 接收到的原始字节数据
- 解析的协议字段
- 协议格式识别

## ⚠️ 注意事项

1. **协议选择**: 确保前端使用正确的端点
   - Protobuf游戏: `/ws/game` 或 `/ws/slot`
   - 二进制协议: `/ws/binary`

2. **字节序**: 所有多字节字段使用大端序（Big-Endian）

3. **消息大小**: 默认最大消息大小为 8192 字节

4. **心跳机制**: 建议每 30 秒发送一次心跳（Cmd=1002）

5. **错误处理**: 检查 ErrorID 字段，非零表示错误

## 🚀 下一步

1. 完善游戏逻辑处理器
2. 添加用户认证集成
3. 实现完整的游戏命令处理
4. 添加更多的测试用例
5. 优化性能和错误处理

---
*更新日期: 2025-09-18*
*版本: 1.0.0*