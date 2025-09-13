# ACM最终修复方案

## 🔴 核心问题诊断

通过对比工作正常的测试工具 (`/home/ztl/tools/serial_tester_linux_arm`) 和我们的ACM控制器实现，发现了**根本性的架构错误**：

### 关键发现

1. **processLoop逻辑颠倒**
   - ❌ 错误：把要发送到设备的命令（cmdCh）当作从设备接收的命令处理
   - ✅ 修复：cmdCh中的命令应该被写入串口，而不是被handleCommand处理

2. **行尾符不匹配**
   - ❌ 错误：使用`\n`作为命令结尾
   - ✅ 修复：使用`\r\n`，与测试工具和ACM设备期望的格式一致

3. **不必要的初始化命令**
   - ❌ 错误：连接时发送`ver`命令，设备不识别导致"Command not recognised"错误
   - ✅ 修复：移除初始命令，直接等待用户命令或定时器命令

4. **EOF处理正确但其他问题掩盖了效果**
   - ✅ 正确：已经正确忽略EOF错误（USB-CDC设备的正常行为）
   - 问题：由于命令无法发送，看起来像是连接问题

## 📝 具体修改

### 1. processLoop修复 (internal/hardware/acm_controller.go:558-598)
```go
// 修复前：错误地处理cmdCh中的命令
c.handleCommand(cmd)  // 错误！这是处理接收命令的函数

// 修复后：正确发送命令到串口
cmdBytes := []byte(cmd + "\r\n")  // 使用\r\n
if c.port != nil {
    n, err := c.port.Write(cmdBytes)
    if err != nil {
        c.logger.Error("发送命令失败", zap.Error(err))
    } else {
        c.logger.Debug("命令发送成功", zap.Int("bytes_written", n))
    }
}
```

### 2. SendAlgoCommand行尾符修复 (line 810)
```go
// 修复前
cmdBytes := []byte(cmd + "\n")

// 修复后
cmdBytes := []byte(cmd + "\r\n")  // 与测试工具一致
```

### 3. 移除初始化命令 (line 161-163)
```go
// 修复前
if err := c.SendCommand("ver"); err != nil {
    c.logger.Warn("发送测试命令失败", zap.Error(err))
}

// 修复后
c.logger.Info("ACM设备就绪，等待命令")
// 不发送初始命令，避免设备报错
```

## 🚀 部署步骤

```bash
# 1. 复制到设备
scp release/slot-game-arm64.tar.gz sg@192.168.10.113:/tmp/

# 2. SSH登录
ssh sg@192.168.10.113

# 3. 停止服务
sudo systemctl stop slot-game

# 4. 解压并安装
cd /tmp
tar -xzf slot-game-arm64.tar.gz
cd slot-game-arm64
sudo ./install.sh

# 5. 重启服务
sudo systemctl restart slot-game

# 6. 查看日志
sudo journalctl -u slot-game -f
```

## ✅ 预期结果

### 成功连接日志
```
ACM控制器已连接 port=/dev/ttyACM0 baudrate=115200 config=8N2
ACM设备就绪，等待命令
Algo定时器已启动 interval=5s bet=1 prize=100
发送命令到ACM设备 command=algo -b 1 -p 100
ACM JSON响应 message={"code":0,"msg":"success",...}
```

### 关键改进
1. **命令能够正确发送**：processLoop现在真正发送命令到串口
2. **格式兼容**：使用`\r\n`行尾符，与设备期望一致
3. **无错误启动**：不发送未知命令，避免初始错误
4. **稳定连接**：EOF正确处理，连接保持稳定

## 🔍 验证方法

```bash
# 1. 检查连接状态
sudo journalctl -u slot-game | grep "ACM控制器已连接"

# 2. 检查命令发送
sudo journalctl -u slot-game | grep "发送命令到ACM设备"

# 3. 检查响应接收
sudo journalctl -u slot-game | grep "ACM JSON响应"

# 4. 确认无错误消息
sudo journalctl -u slot-game | grep -v "Command not recognised"
```

## 📊 技术细节

### 为什么之前的修复没有解决问题？
1. **EOF处理是正确的**，但命令无法发送掩盖了这个修复的效果
2. **processLoop的根本性错误**导致所有通过SendCommand发送的命令都被错误处理
3. **行尾符不匹配**即使命令发送了，设备也可能无法正确解析

### 测试工具为什么能工作？
1. 直接使用`port.Write([]byte(input + "\r\n"))`发送命令
2. 不发送初始测试命令
3. 正确忽略EOF错误
4. 使用正确的行尾符

## 🎯 结论

此次修复解决了ACM控制器的**根本架构问题**，而不仅仅是表面的连接问题。通过仔细对比工作的测试工具和我们的实现，发现并修复了命令发送逻辑的根本错误。这个经验教训提醒我们：
- 深入理解参考实现的每个细节
- 不要假设函数名称就代表其功能
- 串口通信对格式（如行尾符）非常敏感