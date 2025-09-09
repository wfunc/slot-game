# Java代码串口通信分析与测试

## 从Java代码提取的关键信息

### 1. 串口配置

#### ACM设备（硬件控制）
- **设备路径**: 自动检测包含"ACM"的设备
- **波特率**: 115200
- **数据位**: 8
- **停止位**: 2
- **校验位**: None
- **流控**: None
- **功能**: 控制硬件操作（LED、按钮等）

#### STM32芯片（加密芯片）
- **设备路径**: `/dev/ttyS3`
- **波特率**: 115200
- **数据位**: 8
- **停止位**: 2
- **校验位**: None
- **功能**: 读取加密芯片数据、游戏算法处理

### 2. 消息协议

#### 消息类型（MsgType）
```json
{
  "M1": "基础消息",
  "M2": "响应消息 - 包含code和function字段",
  "M3": "状态消息 - 包含state字段",
  "M4": "控制消息 - 包含action字段(wait/exit/start/fail)",
  "M5": "更新状态消息 - 包含upstate字段",
  "M6": "MQTT消息 - 包含toptype字段(0/1/2)"
}
```

#### 消息格式示例

**M1 - 基础消息**
```json
{
  "MsgType": "M1",
  "data": {"cmd": "get_version"},
  "timestamp": 1703123456
}
```

**M2 - 响应消息**
```json
{
  "MsgType": "M2",
  "code": 200,
  "function": "HP30",
  "timestamp": 1703123456
}
```

**M3 - 状态消息**
```json
{
  "MsgType": "M3",
  "state": 1,
  "timestamp": 1703123456
}
```

**M4 - 控制消息**
```json
{
  "MsgType": "M4",
  "action": "wait",  // 可选: wait, exit, start, fail
  "timestamp": 1703123456
}
```

**M5 - 更新状态**
```json
{
  "MsgType": "M5",
  "upstate": 1,
  "timestamp": 1703123456
}
```

**M6 - MQTT消息**
```json
{
  "MsgType": "M6",
  "toptype": 0,  // 0, 1, 或 2
  "data": {"key": "value"},
  "timestamp": 1703123456
}
```

### 3. 原始命令

Java代码中还使用了一些文本命令（非JSON）：

- `ver` - 查询版本
- `ver -u <path>` - 更新固件
- `sta` - 查询WiFi状态
- `sta -s <ssid> -p <password>` - 连接WiFi

这些命令后面都需要添加`\r\n`。

### 4. 通信流程

1. **初始化**
   - opencomACM() - 打开ACM设备
   - opencom2() - 打开STM32设备

2. **消息发送**
   - sendstr() - 发送到ACM设备
   - sendstr2() - 发送到STM32设备
   - 所有消息后添加`\r\n`

3. **消息接收**
   - 使用缓冲区接收数据
   - 解析JSON或文本响应
   - 根据MsgType处理不同类型消息

## 测试程序使用说明

### 运行增强版本

增强版本（`main_enhanced.go`）包含了从Java代码提取的所有配置和协议：

```bash
# 给脚本添加执行权限
chmod +x run_test.sh

# 运行增强版本
./run_test.sh enhanced

# 自动连接模式
./run_test.sh auto

# 直接运行测试序列
./run_test.sh test
```

### 交互式命令

启动程序后，可以使用以下命令：

#### 1. 自动连接
```
> auto
```
自动连接配置文件中的串口设备。

#### 2. 手动连接
```
> connect acm auto 115200     # 自动查找ACM设备
> connect stm32 /dev/ttyS3 115200  # 连接STM32
```

#### 3. 发送JSON消息
```
> send stm32 {"MsgType":"M4","action":"wait"}
> send stm32 {"MsgType":"M2","code":200,"function":"HP30"}
```

#### 4. 发送原始命令
```
> cmd acm ver          # 查询版本
> cmd acm sta          # 查询WiFi状态
```

#### 5. 快速测试消息
```
> m1    # 发送M1基础消息
> m2    # 发送M2响应消息
> m3    # 发送M3状态消息
> m4    # 发送M4控制消息(wait)
> m5    # 发送M5更新状态
> m6    # 发送M6 MQTT消息
```

#### 6. 运行完整测试序列
```
> test
```
运行预定义的测试序列，包含所有消息类型。

### 测试序列内容

测试序列按照以下顺序执行：

1. 查询ACM版本 (`ver`)
2. 查询WiFi状态 (`sta`)
3. 发送等待命令 (M4 - wait)
4. 获取STM32版本 (M1)
5. 启动游戏 (M4 - start)
6. 设置状态 (M3)
7. HP30游戏功能 (M2)
8. 退出游戏 (M4 - exit)

### 配置文件说明

`config.json`文件包含了从Java代码提取的实际配置：

- ACM设备自动检测（查找包含"ACM"的设备）
- STM32固定路径`/dev/ttyS3`
- 波特率115200，8数据位，2停止位
- 消息类型定义

### 注意事项

1. **权限问题**
   - Linux下可能需要添加用户到dialout组：
     ```bash
     sudo usermod -a -G dialout $USER
     ```

2. **设备路径差异**
   - Linux: `/dev/ttyUSB*`, `/dev/ttyACM*`, `/dev/ttyS*`
   - macOS: `/dev/cu.usbmodem*`, `/dev/tty.usbmodem*`
   - Windows: `COM1`, `COM2`, `COM3`...

3. **消息格式**
   - 所有JSON消息自动添加`\r\n`
   - 原始命令也会自动添加`\r\n`
   - 时间戳自动生成

4. **调试模式**
   - 配置文件中`test_mode: true`启用详细日志
   - 所有发送和接收的消息都会显示

## 与原Java代码的对应关系

| Java方法 | Go实现 | 说明 |
|---------|--------|------|
| opencomACM() | InitACMPort() | 打开ACM设备，自动检测 |
| opencom2() | InitSTM32Port() | 打开STM32设备，固定路径 |
| sendstr() | SendRawCommand(acmPort) | 发送到ACM |
| sendstr2() | SendGameMessage(stm32Port) | 发送到STM32 |
| chkresult() | handleReceivedData() | 处理接收数据 |

## 测试建议

1. **基础连接测试**
   ```bash
   ./run_test.sh auto
   ```

2. **消息类型测试**
   - 依次测试m1到m6命令
   - 观察响应格式

3. **完整流程测试**
   ```bash
   ./run_test.sh test
   ```

4. **性能测试**
   - 连续发送消息测试稳定性
   - 监控缓冲区处理

这个测试程序完全基于Java代码的实际配置和协议，可以用于验证Go版本的串口通信实现是否正确。