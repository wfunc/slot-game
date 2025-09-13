# Ubuntu设备串口测试部署指南

## 快速开始

### 1. 编译程序
```bash
cd examples/testserial

# 编译Linux版本
make linux

# 或编译所有Linux架构
make all-linux
```

### 2. 手动部署
```bash
# 复制到Ubuntu设备
scp serial_tester_linux_amd64 ubuntu@<设备IP>:/home/ubuntu/
ssh ubuntu@<设备IP>

# 在Ubuntu设备上
chmod +x serial_tester_linux_amd64
sudo ./serial_tester_linux_amd64 -h
```

### 3. 自动部署（推荐）
```bash
# 一键部署到Ubuntu设备
./deploy.sh <设备IP> [用户名] [远程路径]

# 示例
./deploy.sh 192.168.1.100
./deploy.sh 192.168.1.100 ubuntu /opt/serial_test
```

## 程序使用

### 基本测试
```bash
# 交互模式（推荐）
sudo ./serial_tester -m both

# 只发送
sudo ./serial_tester -m send -msg "TEST_MESSAGE" -i 1000

# 只接收
sudo ./serial_tester -m recv -v

# 回显模式
sudo ./serial_tester -m echo
```

### 参数说明
```
-d /dev/ttyS3   # 串口设备（默认）
-b 115200       # 波特率（默认）
-s 2            # 停止位（默认2，与Java代码一致）
-m both         # 模式：send/recv/both/echo
-i 1000         # 发送间隔（毫秒）
-msg "TEST"     # 测试消息
-v              # 详细输出（显示HEX）
```

### 交互模式命令
```
:test 10   # 发送10条测试消息
:hex       # 切换HEX显示
:stats     # 显示统计信息
:clear     # 清屏
:quit      # 退出
```

## 测试场景

### 1. 验证串口是否工作
```bash
# 检查设备
ls -l /dev/ttyS3

# 查看配置
stty -F /dev/ttyS3 -a

# 快速测试
echo "TEST" | sudo tee /dev/ttyS3
```

### 2. 与STM32通信测试
```bash
# 接收STM32消息
sudo ./serial_tester -m recv -v

# 发送JSON消息到STM32
sudo ./serial_tester -m both
> {"MsgType":"M2","idex":1001,"data":{"function":"algo"}}
```

### 3. 性能测试
```bash
# 高频发送测试
sudo ./serial_tester -m send -i 10 -msg "PERF_TEST"

# 接收统计
sudo ./serial_tester -m recv
# 会显示接收速率
```

### 4. 回环测试
如果TX和RX短接：
```bash
sudo ./serial_tester -m echo
```

## 故障排查

### 权限问题
```bash
# 添加用户到dialout组
sudo usermod -a -G dialout $USER

# 或直接使用sudo
sudo ./serial_tester
```

### 串口被占用
```bash
# 查看占用进程
sudo lsof /dev/ttyS3

# 结束占用进程
sudo kill -9 <PID>
```

### 串口配置
```bash
# 设置串口参数
stty -F /dev/ttyS3 115200 cs8 -cstopb -parenb

# 查看当前配置
stty -F /dev/ttyS3 -a
```

## 编译选项

### Makefile目标
- `make build` - 为当前平台编译
- `make linux` - Linux amd64
- `make linux-arm` - Linux ARM
- `make linux-arm64` - Linux ARM64
- `make all-linux` - 所有Linux版本
- `make clean` - 清理编译文件

## 日志示例

### 正常通信
```
[15:04:05.123] → 发送 14 字节
[15:04:05.234] ← 接收 28 字节: {"MsgType":"M1","code":0}
[15:04:06.345] → 发送 35 字节
[15:04:06.456] ← 接收 42 字节: {"MsgType":"M2","idex":1001,"data":{}}
```

### 统计信息
```
=== 统计信息 ===
发送: 1024 字节
接收: 2048 字节
错误: 0 次
```