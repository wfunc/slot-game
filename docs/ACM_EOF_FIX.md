# ACM EOF错误修复说明

## 🔴 修复的问题

### 1. **ACM连接后立即EOF错误**
**现象**：
- ACM设备连接成功后立即收到EOF错误
- 错误不断重复，无法正常通信

**原因**：
- ACM设备可能不兼容或需要特殊初始化
- 串口参数配置不匹配
- 设备驱动问题

**修复**：
- 正确处理EOF错误，标记连接断开并退出循环
- 避免EOF错误导致的无限循环

### 2. **程序无法正常关闭**
**现象**：
- Ctrl+C后程序无法退出
- 显示"关闭超时，强制退出"

**原因**：
- goroutine没有正确响应停止信号
- channel未初始化导致panic
- 串口读取阻塞

**修复**：
- 初始化所有必需的channel
- 改进Disconnect逻辑
- 添加channel关闭检查

## 📋 部署步骤

```bash
# 1. 复制到设备
scp release/slot-game-arm64.tar.gz sg@<设备IP>:/tmp/

# 2. SSH登录
ssh sg@<设备IP>

# 3. 停止服务
sudo systemctl stop slot-game

# 4. 安装新版本
cd /tmp
tar -xzf slot-game-arm64.tar.gz
cd slot-game-arm64
sudo ./install.sh

# 5. 重启服务
sudo systemctl restart slot-game

# 6. 查看日志
sudo journalctl -u slot-game -f
```

## 🔍 排查ACM设备问题

### 1. 检查设备信息
```bash
# 查看ACM设备详情
ls -la /dev/ttyACM*
dmesg | grep -i acm
lsusb -v | grep -A 10 -i acm
```

### 2. 测试串口通信
```bash
# 使用minicom测试
sudo apt-get install minicom
sudo minicom -D /dev/ttyACM0 -b 115200

# 或使用screen
sudo screen /dev/ttyACM0 115200

# 测试命令
help
ver
```

### 3. 检查串口参数
可能需要调整的参数：
- 波特率：9600, 19200, 38400, 57600, 115200
- 数据位：7 或 8
- 停止位：1 或 2
- 校验：None, Even, Odd

### 4. 修改配置文件
```yaml
serial:
  acm:
    enabled: true
    port: "/dev/ttyACM0"  # 指定具体设备而不是auto
    baud_rate: 115200      # 尝试不同的波特率
    data_bits: 8
    stop_bits: 1           # 或尝试2
    parity: "N"
```

## 🎯 预期日志

### 正常连接
```
ACM控制器已连接 port=/dev/ttyACM0
发送help命令查询支持的命令列表
ACM帮助信息 message=<help响应>
Algo定时器已启动
```

### EOF错误（设备不兼容）
```
ACM控制器已连接 port=/dev/ttyACM0
ACM连接断开(EOF)，退出读取循环
ACM读取循环已退出
```

### 正常关闭
```
收到退出信号
正在断开ACM连接...
收到停止信号，退出读取循环
收到停止信号，退出处理循环
ACM读取循环已退出
ACM处理循环已退出
ACM控制器已断开
```

## ⚠️ 如果仍有EOF错误

1. **尝试不同的串口工具确认设备是否正常**
2. **检查设备是否需要特殊的初始化序列**
3. **尝试禁用ACM，只使用STM32**：
   ```yaml
   serial:
     acm:
       enabled: false  # 暂时禁用ACM
   ```

4. **收集更多信息**：
   ```bash
   # 查看USB设备信息
   lsusb -t
   
   # 查看串口驱动
   ls /sys/class/tty/ttyACM*/device/driver/
   
   # 查看设备属性
   udevadm info -a -n /dev/ttyACM0
   ```

## 📝 后续改进建议

1. **添加重连机制**：EOF错误后自动重连
2. **设备兼容性检测**：连接前先检测设备类型
3. **配置验证**：自动测试不同的串口参数
4. **健康检查**：定期发送心跳命令确认连接状态