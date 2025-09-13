# 串口测试程序

完整的串口通信测试套件，用于验证 `/dev/ttyS3` @ 115200 的通信，包含STM32和Android端协议模拟。

## 功能特性
- ✅ 串口参数：115200, 8N2 (与Acmlib.java一致)
- ✅ 双向通信：同时支持读写
- ✅ JSON协议：完整模拟STM32-Android通信
- ✅ 多种测试模式：回环、模拟器、交互式

## 安装依赖
```bash
go mod download
```

## 运行程序
```bash
# 需要串口权限
sudo go run serial_test.go

# 或编译后运行
go build -o serial_test
sudo ./serial_test
```

## 程序文件说明

| 文件 | 功能描述 |
|------|----------|
| `test1.go` | 原始简单测试程序 |
| `simple_test.go` | 极简版本，单次发送接收 |
| `serial_test.go` | 交互式串口终端 |
| `stm32_simulator.go` | 模拟STM32端发送JSON消息 |
| `android_simulator.go` | 模拟Android端(Acmlib.java)通信 |
| `test_bidirectional.sh` | 自动化测试脚本 |

## 快速测试

### 使用测试脚本（推荐）
```bash
./test_bidirectional.sh
```
选择测试模式：
1. **回环测试** - TX/RX短接自测
2. **STM32模拟器** - 模拟STM32发送消息
3. **Android模拟器** - 模拟Android端通信
4. **双窗口测试** - 同时运行两端模拟器
5. **简单测试** - 交互式终端

### 手动运行
```bash
# STM32端模拟
sudo go run stm32_simulator.go

# Android端模拟
sudo go run android_simulator.go

# 简单测试
sudo go run simple_test.go
```

## 测试建议
1. **回环测试**：将串口的 TX 和 RX 短接，发送的数据应该能收到
2. **设备测试**：连接实际的串口设备进行通信测试
3. **监控工具**：可以使用 `minicom` 或 `screen` 作为对端测试

## 协议说明

### 消息格式
所有消息采用JSON格式，以`\r\n`结尾：
```json
{"MsgType":"M1","data":{...}}\r\n
```

### 消息类型
- **M1**: 数据传输消息（配置、信息）
- **M2**: 带索引的确认消息（算法请求/响应）
- **M3**: WiFi配置消息
- **M4**: 版本控制消息
- **M5**: 升级状态消息
- **M6**: MQTT相关消息

### 通信流程示例
```
Android → STM32: {"MsgType":"M2","idex":1001,"data":{"function":"algo"}}
STM32 → Android: {"MsgType":"M1","data":{"function":"algo","win":100.5,"hp30":1}}
Android → STM32: {"MsgType":"M2","idex":1002,"code":0}
```

## 故障排查
- 权限问题：使用 `sudo` 或将用户加入 `dialout` 组
- 设备不存在：检查 `ls /dev/tty*` 确认设备名称
- 波特率不匹配：确认对端设备也是 115200
- 停止位配置：注意Java代码使用2个停止位(8N2)