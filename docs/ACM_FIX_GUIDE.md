# ACM无限循环问题修复指南

## 🔧 已修复的问题

### 问题描述
ACM设备不认识 `algo -b 1 -p 100` 命令，返回错误消息：
```
Command not recognised. Enter 'help' to view a list of available commands.
```

程序把这个错误消息当作新命令处理，又发送错误响应，导致无限循环。

### 修复方案
1. **防止循环处理**：忽略包含 "Command not recognised" 或 "Enter 'help'" 的消息
2. **查询支持的命令**：连接后自动发送 `help` 命令
3. **延迟启动定时器**：等待help响应完成后再启动algo定时器

## 📝 部署新版本

```bash
# 1. 部署到设备
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

## 🔍 观察要点

### 1. 查看help命令响应
```bash
sudo journalctl -u slot-game | grep "ACM帮助信息"
```
这会显示ACM设备支持的命令列表

### 2. 检查是否还有循环
```bash
sudo journalctl -u slot-game -f | grep "Command not recognised"
```
不应该再有大量重复的错误消息

### 3. 查看algo命令
如果ACM设备支持不同格式的algo命令，需要根据help响应调整

## 🎯 可能的命令格式

根据不同的ACM设备，algo命令可能是：
- `algo -b 1 -p 100` （当前格式）
- `algo 1 100` （简单格式）
- `ALGO,1,100` （CSV格式）
- JSON格式：`{"cmd":"algo","bet":1,"prize":100}`

## 📊 预期日志输出

```
ACM控制器已连接 port=/dev/ttyACM0
发送help命令查询支持的命令列表
ACM帮助信息 message=<help响应内容>
忽略ACM错误响应 message=Command not recognised
Algo定时器已启动 interval=5s
```

## ⚠️ 如果问题仍然存在

1. **检查ACM设备型号**
   ```bash
   ls -la /dev/ttyACM*
   dmesg | grep ACM
   ```

2. **手动测试ACM通信**
   ```bash
   # 使用minicom或screen测试
   sudo minicom -D /dev/ttyACM0 -b 115200
   # 或
   sudo screen /dev/ttyACM0 115200
   
   # 输入help查看支持的命令
   help
   ```

3. **根据实际命令格式修改代码**
   - 修改 `SendAlgoCommand` 函数
   - 调整命令格式以匹配设备要求

## 📌 后续优化建议

1. **命令格式配置化**
   - 在配置文件中定义algo命令格式
   - 支持不同设备的命令格式

2. **命令验证**
   - 解析help响应，自动识别支持的命令
   - 验证命令格式是否正确

3. **错误处理改进**
   - 更智能的错误消息识别
   - 重试机制和错误恢复