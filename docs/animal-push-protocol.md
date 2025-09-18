# 动物推送协议实现文档

## 概述

本文档描述了动物游戏中动物进入/离开房间时的实时推送协议实现。

## 协议定义

### 1. 动物进入推送 (m_1887_toc)

当新动物进入房间时，服务器会推送此消息给房间内所有玩家。

**消息ID**: 1887

**Protobuf定义**:
```protobuf
// 推送动物进来
message m_1887_toc{
    repeated    p_route     animal  = 1; // 动物进来房间的路径
}

message p_route{
    required    uint32      id          = 1; // 唯一ID
    required    e_animal    bet         = 2; // 动物名字
    required    uint32      line_id     = 3; // 路线
    required    uint32      point       = 4; // 路径
    required    bool        red_state   = 5; // 红包状态
    required    e_animal_state  status  = 6; // 动物当前的状态
}
```

### 2. 动物离开推送 (m_1888_toc)

当动物离开房间时，服务器会推送此消息。

**消息ID**: 1888

**Protobuf定义**:
```protobuf
// 推送动物离开
message m_1888_toc{
    required    uint32      id      = 1; // 动物ID
}
```

## 系统架构

```
动物房间 (AnimalRoom)
    ↓ [生成/移除动物]
推送回调 (PushCallback)
    ↓ [创建推送消息]
推送管理器 (PushManager)
    ↓ [序列化protobuf]
客户端管理器 (ClientManager)
    ↓ [广播到房间]
WebSocket客户端
```

## 核心组件

### 1. AnimalRoom (动物房间)

- **位置**: `/internal/game/animal/animal_room.go`
- **职责**: 管理动物的生成、移除和状态
- **推送触发**:
  - `generateAnimal()` - 生成新动物时调用 `pushAnimalEnter()`
  - `removeAnimal()` - 移除动物时调用 `pushAnimalLeave()`

### 2. PushManager (推送管理器)

- **位置**: `/internal/websocket/push_manager.go`
- **职责**: 处理protobuf消息的序列化和路由
- **主要方法**:
  - `HandleAnimalPush()` - 处理动物推送消息
  - `encodeProtobufMessage()` - 序列化protobuf消息
  - `CreatePushCallback()` - 创建推送回调函数

### 3. ClientManager (客户端管理器)

- **位置**: `/internal/websocket/client_manager.go`
- **职责**: 管理WebSocket客户端连接和房间
- **主要方法**:
  - `JoinRoom()` - 客户端加入房间
  - `LeaveRoom()` - 客户端离开房间
  - `BroadcastToRoom()` - 向房间内所有客户端广播消息

### 4. BinaryProtocolRouter (协议路由器)

- **位置**: `/internal/websocket/binary_protocol_router.go`
- **职责**: 处理前端的二进制协议请求
- **集成点**:
  - 初始化时创建默认动物房间
  - 处理1801命令时将客户端加入房间
  - 自动接收并转发动物推送消息

## 消息流程

### 动物生成并推送

1. **定时器触发生成**
   ```go
   AnimalRoom.generateAnimal()
   ```

2. **创建动物实体**
   ```go
   newAnimal := r.generator.GenerateAnimal(excludeTypes)
   r.animals[newAnimal.ID] = newAnimal
   ```

3. **推送动物进入消息**
   ```go
   r.pushAnimalEnter(newAnimal)
   ```

4. **构建protobuf消息**
   ```go
   msg := &pb.M_1887Toc{
       Animal: []*pb.PRoute{route},
   }
   ```

5. **调用推送回调**
   ```go
   r.pushCallback(&PushMessage{
       MsgID:   1887,
       Message: msg,
   })
   ```

6. **序列化并广播**
   ```go
   data, _ := proto.Marshal(message)
   clientManager.BroadcastToRoom(roomID, msgID, data)
   ```

## 测试方法

### 使用测试页面

1. 打开 `test/websocket_test.html`
2. 点击 "Connect" 连接到服务器
3. 点击 "1801 - Enter Zoo" 进入动物房间
4. 观察接收日志中的推送消息：
   - 🦁 动物进入推送 (1887)
   - 🐢 动物离开推送 (1888)

### 日志输出示例

```
[PushManager] 处理动物推送 room_id:1 msg_id:1887 zoo_type:civilian
[PushManager] 推送动物进入 animal_id:1 animal_type:tuzi line_id:19 point:10
[ClientManager] 向房间广播消息 room_id:1 msg_id:1887 client_count:1
```

## 消息格式

### 二进制协议格式

服务端推送消息格式（11字节头 + protobuf数据）:
```
ErrorID    (2字节) = 0x0000 (成功)
DataSize   (2字节) = protobuf长度
DataStatus (1字节) = 0x00
Flag       (4字节) = 0x00000000 (推送消息通常为0)
Cmd        (2字节) = 0x075B (1887) 或 0x0760 (1888)
Data       (n字节) = protobuf序列化数据
```

### 示例数据

动物进入推送 (1887):
```
00 00  // ErrorID = 0
00 0E  // DataSize = 14
00     // DataStatus = 0
00 00 00 00  // Flag = 0
07 5B  // Cmd = 1887
[protobuf数据]
```

## 配置说明

### 房间配置

默认房间ID: 1
默认房间类型: civilian (平民场)

### 动物生成配置

- 初始动物数量: 15-25只
- 生成间隔: 1-5秒
- 动物存活时间: 基于路径点数（20-45秒）

## 注意事项

1. **客户端连接管理**: 客户端断开时需要清理会话
2. **房间生命周期**: 房间为空时不会自动销毁（保持动物继续运行）
3. **消息序列化**: 使用 google.golang.org/protobuf 进行序列化
4. **并发安全**: ClientManager 使用读写锁保护并发访问

## 后续优化

1. **多房间支持**: 根据玩家数量动态创建房间
2. **消息压缩**: 对大量推送消息进行批量压缩
3. **断线重连**: 支持客户端断线重连后的状态恢复
4. **性能优化**: 使用对象池减少内存分配

---

更新时间: 2025-09-18
版本: 1.0.0