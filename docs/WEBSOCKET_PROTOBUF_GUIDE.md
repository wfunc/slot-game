# WebSocket + Protocol Buffers v2 使用指南

## 📋 概述

本项目使用 WebSocket + Protocol Buffers v2 实现高效的实时通信。

### 优势
- **高性能**: 二进制协议，体积小，解析快
- **强类型**: 消息格式明确，减少错误
- **跨语言**: 支持多种编程语言
- **实时性**: WebSocket 双向通信

## 🚀 快速开始

### 1. 安装依赖

```bash
# 安装 protobuf 编译器
brew install protobuf  # macOS
apt-get install protobuf-compiler  # Linux

# 安装 Go protobuf 插件
go get -u github.com/golang/protobuf/proto
go get -u github.com/golang/protobuf/protoc-gen-go
go get -u github.com/gorilla/websocket
```

### 2. 编译 Proto 文件

```bash
# 使用 Makefile
make proto

# 或手动编译
protoc --go_out=internal/pb --go_opt=paths=source_relative proto/*.proto
```

### 3. 运行示例

```bash
# 运行 WebSocket 服务器
go run examples/proto_websocket_example.go

# 访问测试页面
open http://localhost:8080
```

## 📦 消息格式

### 二进制消息结构
```
[4字节长度][2字节消息ID][protobuf数据]
```

- **长度**: 32位大端序，不包括长度字段本身
- **消息ID**: 16位大端序，标识消息类型
- **数据**: Protocol Buffers 序列化数据

### 消息ID映射
```go
// 老虎机游戏
1901: m_1901_tos/toc - 进入房间
1902: m_1902_tos/toc - 开始游戏  
1903: p_1903_toc     - 推送数据

// 动物园游戏
1801: m_1801_tos/toc - 进入房间
1802: m_1802_tos/toc - 离开房间
1803: m_1803_tos/toc - 下注
```

## 💻 服务端实现

### 1. 创建消息处理器

```go
import (
    "github.com/wfunc/slot-game/internal/websocket"
    "github.com/wfunc/slot-game/internal/pb"
)

// 创建处理器
handler := websocket.NewProtoHandler(conn, logger)

// 注册消息处理
handler.RegisterMessage(1901, &pb.M1901Tos{}, func(msg proto.Message) error {
    req := msg.(*pb.M1901Tos)
    // 处理进入房间请求
    return handleEnterRoom(req)
})

// 启动处理循环
handler.Start()
```

### 2. 发送消息

```go
// 构建响应消息
resp := &pb.M1901Toc{
    BetVal: []uint32{1, 5, 10, 50, 100},
    Odds:   getOdds(),
}

// 发送消息
err := handler.SendMessage(1901, resp)
```

### 3. 完整服务示例

```go
type GameService struct {
    logger *zap.Logger
}

func (s *GameService) HandleConnection(conn *websocket.Conn) {
    handler := websocket.NewProtoHandler(conn, s.logger)
    
    // 注册所有消息处理器
    handler.RegisterMessage(1901, &pb.M1901Tos{}, s.handleEnterRoom)
    handler.RegisterMessage(1902, &pb.M1902Tos{}, s.handleStartGame)
    
    // 启动处理
    handler.Start()
}

func (s *GameService) handleEnterRoom(msg proto.Message) error {
    req := msg.(*pb.M1901Tos)
    
    // 业务逻辑
    slotType := req.GetType()
    
    // 返回响应
    resp := &pb.M1901Toc{
        BetVal: []uint32{1, 5, 10, 50, 100},
    }
    
    return handler.SendMessage(1901, resp)
}
```

## 🌐 客户端实现

### JavaScript/TypeScript 客户端

```javascript
class ProtoWebSocket {
    constructor(url) {
        this.ws = new WebSocket(url);
        this.ws.binaryType = 'arraybuffer';
        this.handlers = new Map();
    }
    
    // 发送消息
    sendMessage(msgId, data) {
        const totalLen = 2 + data.length;
        const buffer = new ArrayBuffer(4 + totalLen);
        const view = new DataView(buffer);
        
        // 写入长度
        view.setUint32(0, totalLen);
        // 写入消息ID
        view.setUint16(4, msgId);
        // 写入数据
        new Uint8Array(buffer, 6).set(data);
        
        this.ws.send(buffer);
    }
    
    // 接收消息
    onMessage(event) {
        const view = new DataView(event.data);
        const length = view.getUint32(0);
        const msgId = view.getUint16(4);
        const data = new Uint8Array(event.data, 6);
        
        // 调用处理器
        const handler = this.handlers.get(msgId);
        if (handler) {
            handler(data);
        }
    }
    
    // 注册处理器
    registerHandler(msgId, handler) {
        this.handlers.set(msgId, handler);
    }
}

// 使用示例
const client = new ProtoWebSocket('ws://localhost:8080/ws/game');

// 注册处理器
client.registerHandler(1901, (data) => {
    // 解析 protobuf 数据
    const resp = M1901Toc.decode(data);
    console.log('进入房间响应:', resp);
});

// 发送消息
const req = M1901Tos.create({ type: 1 });
const data = M1901Tos.encode(req).finish();
client.sendMessage(1901, data);
```

### 前端 Protobuf 集成

1. 安装 protobufjs
```bash
npm install protobufjs
```

2. 编译 proto 文件为 JavaScript
```bash
npx pbjs -t static-module -w commonjs -o proto.js proto/*.proto
npx pbts -o proto.d.ts proto.js
```

3. 使用示例
```javascript
import { M1901Tos, M1901Toc } from './proto';

// 编码
const message = M1901Tos.create({ type: 1 });
const buffer = M1901Tos.encode(message).finish();

// 解码
const decoded = M1901Toc.decode(buffer);
```

## 🎮 游戏流程示例

### 老虎机游戏流程

```sequence
客户端->服务器: 1901 进入房间请求
服务器->客户端: 1901 进入房间响应（下注档位、赔率）
客户端->服务器: 1902 开始游戏（下注金额）
服务器->客户端: 1902 游戏结果（中奖信息）
服务器->客户端: 1903 推送数据（金币、JP等）
```

### 代码示例

```go
// 1. 进入房间
enterReq := &pb.M1901Tos{
    Type: proto.Int32(pb.ESlotType_E_SLOT_TYPE_MAHJONG),
}
handler.SendMessage(1901, enterReq)

// 2. 开始游戏
gameReq := &pb.M1902Tos{
    BetVal: proto.Uint32(100), // 下注100
}
handler.SendMessage(1902, gameReq)

// 3. 接收结果
handler.RegisterMessage(1902, &pb.M1902Toc{}, func(msg proto.Message) error {
    resp := msg.(*pb.M1902Toc)
    fmt.Printf("赢得: %d\n", resp.GetWin())
    fmt.Printf("总赢: %d\n", resp.GetTotalWin())
    fmt.Printf("免费游戏: %v\n", resp.GetIsFree())
    return nil
})
```

## 🔧 调试技巧

### 1. 日志记录

```go
// 启用详细日志
handler := websocket.NewProtoHandler(conn, logger)
handler.EnableDebug(true)
```

### 2. 消息监控

```go
// 添加消息拦截器
handler.AddInterceptor(func(msgId uint16, data []byte, isOutgoing bool) {
    direction := "收到"
    if isOutgoing {
        direction = "发送"
    }
    log.Printf("%s消息 ID:%d 大小:%d", direction, msgId, len(data))
})
```

### 3. Chrome DevTools

```javascript
// 在浏览器控制台监控 WebSocket
const ws = new WebSocket('ws://localhost:8080/ws/game');

// 监控所有消息
ws.addEventListener('message', (event) => {
    if (event.data instanceof ArrayBuffer) {
        const view = new DataView(event.data);
        const msgId = view.getUint16(4);
        console.log('收到消息:', msgId, event.data);
    }
});
```

## 📊 性能优化

### 1. 消息批处理

```go
// 批量发送多个消息
batch := handler.NewBatch()
batch.Add(1903, pushData1)
batch.Add(1903, pushData2)
batch.Send()
```

### 2. 消息压缩

```go
// 启用压缩（适合大消息）
handler.EnableCompression(true)
```

### 3. 连接池

```go
// 管理多个连接
type ConnectionPool struct {
    connections map[string]*websocket.ProtoHandler
    mu          sync.RWMutex
}

func (p *ConnectionPool) Broadcast(msgId uint16, msg proto.Message) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    for _, handler := range p.connections {
        go handler.SendMessage(msgId, msg)
    }
}
```

## 🛠️ 故障处理

### 1. 重连机制

```javascript
class ReconnectingWebSocket {
    constructor(url) {
        this.url = url;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
        this.connect();
    }
    
    connect() {
        this.ws = new WebSocket(this.url);
        
        this.ws.onclose = () => {
            setTimeout(() => {
                this.reconnectDelay = Math.min(
                    this.reconnectDelay * 2, 
                    this.maxReconnectDelay
                );
                this.connect();
            }, this.reconnectDelay);
        };
        
        this.ws.onopen = () => {
            this.reconnectDelay = 1000;
        };
    }
}
```

### 2. 心跳检测

```go
// 服务端心跳
func (h *ProtoHandler) startHeartbeat() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := h.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        case <-h.closeChan:
            return
        }
    }
}
```

## 📝 最佳实践

1. **消息版本控制**: 在 proto 文件中添加版本字段
2. **错误处理**: 始终处理消息发送和接收错误
3. **资源清理**: 确保连接关闭时释放所有资源
4. **限流保护**: 实现消息频率限制
5. **安全验证**: 添加身份验证和授权机制

## 🔗 相关资源

- [Protocol Buffers 官方文档](https://developers.google.com/protocol-buffers)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [protobuf.js](https://github.com/protobufjs/protobuf.js)
- [示例代码](examples/proto_websocket_example.go)