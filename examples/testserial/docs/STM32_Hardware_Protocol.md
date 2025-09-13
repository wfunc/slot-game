# STM32硬件通信协议文档

## 概述

基于对Acmlib.java的分析和测试结果，系统采用双串口架构：
- **/dev/ttyACM0**: 命令行接口，接收文本命令，返回JSON格式结果
- **/dev/ttyS3**: STM32通信接口，使用JSON协议进行双向通信

## 串口配置

两个串口使用相同的配置：
- 波特率: 115200
- 数据位: 8
- 校验位: 无 (None)
- 停止位: 2
- 读超时: 100ms

## ACM设备协议 (/dev/ttyACM0)

### 命令格式
- 输入: 文本命令 + `\r\n`
- 输出: JSON结果 + `\n>`

### 支持的命令

| 命令 | 描述 | 示例响应 |
|------|------|----------|
| `ver` | 获取版本信息 | `{"ver":"1.0.0","build":"2024.01.01"}\n>` |
| `sta` | 获取状态信息 | `{"status":"ready","games":100}\n>` |
| `algo -b [bet] -p [param]` | 执行算法 | `{"result":[1,2,3],"win":100}\n>` |
| `help` | 获取帮助信息 | `{"commands":["ver","sta","algo"]}\n>` |

### 示例交互
```bash
# 发送
ver\r\n

# 接收
{"ver":"1.0.0","build":"2024.01.01"}\n>

# 发送
algo -b 1 -p 100\r\n

# 接收
{"result":[1,2,3,4,5],"win":50,"balance":150}\n>
```

## STM32设备协议 (/dev/ttyS3)

### 消息格式
- 所有消息使用JSON格式
- 消息以`\r\n`结束
- 消息类型通过`MsgType`字段标识

### 消息类型

#### M1 - 配置消息
```json
{
  "MsgType": "M1",
  "data": {
    "cfgData": {
      "hp30": 1,
      "other_params": "value"
    }
  }
}
```

#### M2 - 算法请求/响应
请求:
```json
{
  "MsgType": "M2",
  "idex": 1234,
  "data": {
    "function": "algo",
    "param1": 100,
    "param2": 200
  }
}
```

响应:
```json
{
  "MsgType": "M2",
  "idex": 1234,
  "code": 0,
  "data": {
    "result": [1,2,3,4,5],
    "win": 100
  }
}
```

#### M3 - WiFi配置
```json
{
  "MsgType": "M3",
  "wifiname": "TestWiFi",
  "wifipass": "password123",
  "path": "/update"
}
```

#### M4 - 版本/状态
请求:
```json
{
  "MsgType": "M4",
  "cVer": "1.0.0",
  "lVer": "1.0.0",
  "devType": "STM32",
  "uid": "DEVICE001"
}
```

响应:
```json
{
  "MsgType": "M4",
  "action": "wait",
  "ready": "1"
}
```

#### M5 - 版本更新
```json
{
  "MsgType": "M5",
  "upver": "1.2.1"
}
```

#### M6 - MQTT消息
```json
{
  "MsgType": "M6",
  "toptype": 0,  // 0=配置请求, 1=数据
  "data": {
    "topic": "device/status",
    "payload": "online"
  }
}
```

## 数据流

```
Android App (Acmlib.java)
     ↓
[命令输入] → /dev/ttyACM0 → [命令处理器]
                                   ↓
                            [转换为JSON]
                                   ↓
                            /dev/ttyS3
                                   ↓
                              [STM32]
                                   ↓
                            [处理&响应]
                                   ↓
                            /dev/ttyS3
                                   ↓
                            [JSON解析]
                                   ↓
/dev/ttyACM0 ← [格式化响应] ←
     ↓
Android App
```

## 错误处理

### ACM设备错误
- 未识别的命令: `Command not recognised\n>`
- 参数错误: `Invalid parameters\n>`
- 超时: 2秒无响应

### STM32设备错误
- JSON解析错误: 静默丢弃
- 超时: 2秒无响应
- 连接错误: 返回空响应

## 测试工具

### 1. 基础测试
```bash
# 测试ACM设备
./serial_tester -d /dev/ttyACM0 -m both

# 测试STM32设备
./serial_tester -d /dev/ttyS3 -m both
```

### 2. 协议测试
```bash
# 测试ACM命令
./protocol_tester -d /dev/ttyACM0 -m algo

# 测试STM32 JSON
./protocol_tester -d /dev/ttyS3 -m version
```

### 3. 诊断工具
```bash
# 完整诊断
./hardware_diagnose -v -test all

# 回环测试
./serial_diagnose -d /dev/ttyS3 -loopback

# 监控模式
./serial_diagnose -d /dev/ttyACM0 -monitor
```

### 4. 桥接测试
```bash
# 桥接模式（ACM ↔ STM32）
./stm32_protocol -mode bridge -duration 60

# 监控模式
./stm32_protocol -mode monitor -duration 60

# 测试模式
./stm32_protocol -mode test
```

### 5. 集成测试
```bash
# 运行完整测试套件
./integration_test.sh
```

## 故障排查

### 问题: ttyS3无响应
1. 检查STM32是否上电
2. 验证TX/RX接线（可能需要交叉）
3. 运行回环测试验证硬件
4. 检查STM32固件是否正确烧录

### 问题: ACM命令不识别
1. 检查命令格式是否正确
2. 确认以`\r\n`结束
3. 验证波特率设置

### 问题: JSON消息无响应
1. 验证JSON格式是否正确
2. 检查MsgType是否支持
3. 确认消息以`\r\n`结束

## 注意事项

1. **当前状态**: 根据测试，/dev/ttyACM0正常工作，但/dev/ttyS3无响应
2. **可能原因**: STM32未连接或未启动
3. **建议操作**: 先确保STM32硬件连接和固件运行正常