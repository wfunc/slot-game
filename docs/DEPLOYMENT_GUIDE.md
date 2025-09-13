# 部署指南 - ACM调试版本

## ✅ 代码验证完成

已确认编译的二进制文件包含所有调试代码。虽然中文日志消息被strip了，但日志字段名都在，可以通过字段名查找日志。

## 🚀 快速部署步骤

```bash
# 1. 复制到设备
scp release/slot-game-arm64.tar.gz sg@<设备IP>:/tmp/

# 2. SSH登录
ssh sg@<设备IP>

# 3. 解压
cd /tmp
tar -xzf slot-game-arm64.tar.gz
cd slot-game-arm64

# 4. 停止旧服务
sudo systemctl stop slot-game

# 5. 安装新版本
sudo ./install.sh

# 6. 重启服务
sudo systemctl restart slot-game

# 7. 查看日志
sudo journalctl -u slot-game -f
```

## 🔍 查找调试日志

由于中文被strip，使用字段名搜索：

```bash
# 查看ACM相关日志
sudo journalctl -u slot-game -f | grep -E 'serial.acm|acm_controller|acm_connected'

# 查看所有串口配置
sudo journalctl -u slot-game -f | grep "serial\."

# 查看ACM初始化
sudo journalctl -u slot-game -f | grep "acm"
```

## 📋 关键日志字段

部署后应该能看到这些字段：

1. **串口配置**
   - `serial.enabled` - 串口是否启用
   - `serial.acm.enabled` - ACM是否启用
   - `serial.stm32.enabled` - STM32是否启用

2. **ACM状态**
   - `acm.enabled` - ACM配置状态
   - `acm_controller_exists` - 控制器是否存在
   - `acm_connected` - 是否已连接

3. **连接信息**
   - `configured_port` - 配置的端口
   - `auto_detect` - 是否自动检测

4. **定时器**
   - `algo_timer_enabled` - Algo定时器是否启用

## 🎯 问题诊断步骤

### 1. 检查配置加载
```bash
sudo journalctl -u slot-game | grep "serial.acm.enabled"
```
应该看到 `serial.acm.enabled=true`

### 2. 检查ACM初始化
```bash
sudo journalctl -u slot-game | grep "acm.enabled"
```
应该看到 `acm.enabled=true`

### 3. 检查控制器状态
```bash
sudo journalctl -u slot-game | grep "acm_controller_exists"
```
应该看到 `acm_controller_exists=true`

### 4. 检查连接状态
```bash
sudo journalctl -u slot-game | grep "acm_connected"
```
成功应该看到 `acm_connected=true`

## ⚠️ 重要提醒

1. **确保停止旧服务** - 在安装前必须停止旧服务
2. **检查配置文件** - 确认 `/home/sg/slot-game/config/config.yaml` 中 ACM 已启用
3. **检查设备权限** - 确认 sg 用户在 dialout 组
4. **验证设备存在** - 确认 `/dev/ttyACM0` 存在

## 🔧 配置文件示例

```yaml
serial:
  enabled: true
  
  acm:
    enabled: true  # 必须为true
    port: "auto"   # 或指定 "/dev/ttyACM0"
    algo_timer_enabled: true
    algo_timer_interval: 5s
    algo_bet: 1
    algo_prize: 100
```

## 📝 验证部署

部署后运行以下命令验证：

```bash
# 检查服务状态
sudo systemctl status slot-game

# 检查进程
ps aux | grep slot-game

# 检查端口占用
ls -la /dev/ttyACM* 
lsof /dev/ttyACM0

# 实时日志
sudo journalctl -u slot-game -f --since "1 minute ago"
```

如果看到包含上述字段的日志输出，说明新版本已成功部署。