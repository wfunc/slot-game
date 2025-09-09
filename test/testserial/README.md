# 串口通信测试程序

这是一个用于测试串口通信功能的独立Go程序，支持同时管理两个串口设备（ACM设备和STM32芯片）。

## 功能特性

- ✅ 支持多串口同时管理
- ✅ JSON消息格式处理
- ✅ 交互式命令行界面
- ✅ 自动重连机制
- ✅ 消息收发日志
- ✅ 测试序列功能

## 安装和运行

### 1. 安装依赖

```bash
cd test/testserial
go mod download
```

### 2. 配置串口

编辑 `config.json` 文件，设置正确的串口设备路径：

```json
{
  "acm_port": {
    "device": "/dev/ttyUSB0",  // Linux下的设备路径
    "baud_rate": 115200
  },
  "stm32_port": {
    "device": "/dev/ttyUSB1",
    "baud_rate": 115200
  }
}
```

**不同操作系统的串口设备路径：**
- Linux: `/dev/ttyUSB0`, `/dev/ttyACM0`
- macOS: `/dev/tty.usbserial-xxx`, `/dev/cu.usbmodem-xxx`
- Windows: `COM1`, `COM2`, `COM3`...

### 3. 运行程序

```bash
# 直接运行，进入交互模式
go run main.go

# 或者指定串口设备
go run main.go /dev/ttyUSB0 /dev/ttyUSB1
```

## 使用方法

### 交互式命令

程序启动后，可以使用以下命令：

#### 连接串口
```
> connect acm /dev/ttyUSB0 115200
> connect stm32 /dev/ttyUSB1 115200
```

#### 发送消息
```
# 发送JSON格式消息
> send acm {"type":"command","command":"status"}

# 发送简单文本（会自动包装为JSON）
> send stm32 get_chip_id
```

#### 查看状态
```
> status
```

#### 运行测试序列
```
> test
```

#### 断开连接
```
> disconnect acm
> disconnect stm32
> disconnect all
```

#### 退出程序
```
> exit
```

### 消息格式

程序使用JSON格式进行消息通信：

```json
{
  "type": "command",
  "command": "led_control",
  "data": {
    "led": 1,
    "state": "on"
  },
  "timestamp": 1703123456
}
```

支持的消息类型：
- `command` - 命令消息
- `response` - 响应消息
- `event` - 事件通知
- `error` - 错误消息

## 测试用例

### 1. 基础连接测试
```bash
# 连接ACM设备
connect acm /dev/ttyUSB0 115200

# 查询状态
send acm {"type":"command","command":"status"}

# 断开连接
disconnect acm
```

### 2. LED控制测试
```bash
# 连接设备
connect acm /dev/ttyUSB0 115200

# 打开LED
send acm {"type":"command","command":"led_control","data":{"led":1,"state":"on"}}

# 关闭LED
send acm {"type":"command","command":"led_control","data":{"led":1,"state":"off"}}
```

### 3. 芯片通信测试
```bash
# 连接STM32
connect stm32 /dev/ttyUSB1 115200

# 获取芯片ID
send stm32 {"type":"command","command":"get_chip_id"}

# 读取加密数据
send stm32 {"type":"command","command":"read_encrypted","data":{"address":"0x1000","length":32}}
```

## 故障排除

### 串口无法打开
- 检查设备路径是否正确
- 确认串口设备已连接
- 检查用户权限（Linux下可能需要添加用户到dialout组）
  ```bash
  sudo usermod -a -G dialout $USER
  ```

### 无法接收数据
- 检查波特率设置是否正确
- 确认设备端的通信协议
- 查看设备是否正常工作

### 程序权限问题
- macOS可能需要授予终端访问USB设备的权限
- Windows下可能需要安装串口驱动

## 开发说明

### 代码结构
```
testserial/
├── main.go       # 主程序
├── config.json   # 配置文件
├── go.mod        # Go模块定义
├── go.sum        # 依赖版本锁定
└── README.md     # 本文档
```

### 扩展功能

如需添加新的命令处理，可以在 `handleReceivedData` 函数中添加：

```go
func (sp *SerialPort) handleReceivedData(data []byte) {
    // 添加自定义处理逻辑
    switch msg.Type {
    case "custom_type":
        // 处理自定义消息类型
    }
}
```

## 注意事项

1. 确保串口设备没有被其他程序占用
2. 不同操作系统的串口设备命名规则不同
3. 某些操作系统需要特殊权限才能访问串口
4. 测试完成后记得断开串口连接

## 相关文档

- [串口通信系统重构分析文档](../../docs/development/serial-port-refactoring.md)
- [Go serial库文档](https://github.com/tarm/serial)