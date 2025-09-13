# 串口监听工具使用指南

## 概述
`serial-monitor` 是一个增强版的串口监听工具，能够同时解析：
1. **JSON格式消息** - 类似Android/Java系统使用的协议
2. **二进制协议帧** - STM32使用的0xAA...0x55格式

## 工具特性

### 双协议支持
根据Java代码（`Acmlib.java`）的分析，系统使用两种通信协议：
- **JSON协议**: 用于与Android/上位机通信，以`{`开始，以`\r\n`结束
- **二进制协议**: 用于STM32硬件通信，格式为 `0xAA | 长度 | 命令 | 数据 | CRC | 0x55`

### JSON消息类型
工具能识别以下JSON消息类型（基于Java代码）：
- **M1**: APP控制消息
- **M2**: ACM响应消息（包括算法algo消息）
- **M3**: WiFi配置/升级消息
- **M4**: VP（可能是固件）更新消息
- **M5**: 版本更新状态
- **M6**: MQTT消息

## 使用方法

### 基本使用
```bash
# 默认参数（/dev/ttyS3, 115200, 8-N-2）
./serial-monitor-arm64

# 自定义串口和波特率
./serial-monitor-arm64 -port /dev/ttyS3 -baud 115200

# 显示ASCII格式（适合JSON消息）
./serial-monitor-arm64 -ascii

# 同时显示HEX和ASCII
./serial-monitor-arm64 -hex -ascii

# 不同的串口参数
./serial-monitor-arm64 -port /dev/ttyS3 -baud 9600 -stop 1 -parity N
```

### 参数说明
- `-port`: 串口设备路径（默认 /dev/ttyS3）
- `-baud`: 波特率（默认 115200）
- `-stop`: 停止位 1 或 2（默认 2）
- `-parity`: 校验位 N/O/E（默认 N=无）
- `-hex`: 显示HEX格式（默认开启）
- `-ascii`: 显示ASCII格式（默认关闭）

## 输出格式示例

### JSON消息输出
```
📝 检测到 1 个JSON消息:
JSON消息 #1:
  类型: JSON对象
  消息类型: M2
  说明: ACM响应消息
  功能: algo
  中奖: 0.00
  HP30: 0
  内容:
  {
    "MsgType": "M2",
    "function": "algo",
    "win": 0,
    "hp30": 0,
    "idex": 1001
  }
```

### 二进制协议帧输出
```
🔍 检测到 1 个协议帧:
帧 #1:
  原始数据: AA000E31000301000001CRC55
  帧头: 0xAA
  长度: 14
  命令: 0x31 (心跳包)
  序列号: 3
  数据: 01000001
  CRC: 0xXXXX
  帧尾: 0x55
```

## 数据流分析

基于Java代码分析，数据流如下：
1. **ACM串口** (/dev/ttyACM*): 与算法模块通信，使用JSON格式
2. **ttyS3串口**: 与STM32硬件通信，可能使用两种格式：
   - JSON格式（当作为中间层时）
   - 二进制协议（直接与STM32通信时）

## 故障排查

### 收到乱码
1. 检查波特率是否正确（尝试9600, 115200等）
2. 检查停止位（尝试1或2）
3. 使用自动检测工具：
   ```bash
   ./serial-detect-arm64 -port /dev/ttyS3
   ```

### 未检测到协议帧
1. 确认数据格式（可能是纯JSON而非二进制）
2. 开启ASCII显示查看原始内容
3. 检查设备是否正确连接

### JSON解析失败
1. 确认消息以`\r\n`结尾
2. 检查JSON格式是否完整
3. 查看原始HEX数据排查问题

## 统计信息
工具会每5秒显示统计信息：
- 总接收字节数
- 检测到的协议帧数量
- 检测到的JSON消息数量

## 与其他工具配合

1. **serial-detect**: 先用此工具自动检测串口参数
2. **serial-test**: 测试STM32通信协议
3. **serial-debug**: 发送测试数据并分析响应
4. **serial-monitor**: 持续监听和分析数据

## 注意事项

1. Java代码显示系统使用 **115200, 8-N-2** 参数
2. STM32可能使用不同的参数，需要先检测
3. JSON消息必须以`\r\n`结尾才能被正确识别
4. 二进制协议帧必须有完整的头尾（0xAA...0x55）