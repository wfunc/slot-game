# ACM连接成功指南

## ✅ 问题已定位

通过测试工具验证，ACM设备工作正常：
- 设备路径：`/dev/ttyACM0`
- 配置：115200波特率，8N2（8数据位，无校验，2停止位）
- 命令格式：`algo -b 1 -p 100`
- 响应格式：JSON

## 🔧 已修复的问题

### 1. 串口配置
- 确认使用8N2配置（与测试工具一致）
- 固定ReadTimeout为100ms
- 移除可能导致问题的Flush调用

### 2. 初始化序列
- 延长等待时间到500ms让设备完全就绪
- 使用ver命令代替help作为初始测试
- 连接后清空缓冲区

### 3. 消息处理
- 正确跳过"end"和">"标记
- 识别并记录JSON响应
- 改进错误消息处理

## 📋 部署步骤

```bash
# 1. 复制新版本
scp release/slot-game-arm64.tar.gz sg@<设备IP>:/tmp/

# 2. SSH登录
ssh sg@<设备IP>

# 3. 停止服务
sudo systemctl stop slot-game

# 4. 安装
cd /tmp
tar -xzf slot-game-arm64.tar.gz
cd slot-game-arm64
sudo ./install.sh

# 5. 重启服务
sudo systemctl restart slot-game

# 6. 查看日志
sudo journalctl -u slot-game -f
```

## 🎯 预期日志

成功连接后应该看到：
```
ACM控制器已连接 port=/dev/ttyACM0 baudrate=115200 config=8N2
发送测试命令
ACM JSON响应 message={"code":0,"msg":"success"...}
Algo定时器已启动 interval=5s
定时algo命令响应 response={...}
```

## 📊 ACM设备行为

从测试工具输出可以看到：
1. **第一次命令可能失败**：返回"Command not recognised"
2. **第二次开始正常**：返回JSON格式的响应
3. **响应格式**：
   ```json
   {
     "code": 0,
     "msg": "success",
     "ident": 47,
     "function": "algo",
     "prize": 0,
     "bet": 1,
     "algo": {...},
     "win": 0.0000,
     "chk": "..."
   }
   ```
4. **结束标记**：每个响应后有"end"和">"

## 🔍 验证连接

```bash
# 1. 检查串口是否被占用
lsof /dev/ttyACM0

# 2. 查看ACM相关日志
sudo journalctl -u slot-game | grep ACM

# 3. 查看algo命令响应
sudo journalctl -u slot-game | grep "algo命令响应"

# 4. 检查是否有EOF错误
sudo journalctl -u slot-game | grep EOF
```

## ⚠️ 注意事项

1. **设备初始化**：ACM设备需要时间初始化，不要太快发送命令
2. **错误处理**：第一个命令可能失败是正常的
3. **定时器间隔**：建议algo定时器间隔不要太短（>=5秒）

## 📝 配置文件确认

```yaml
serial:
  enabled: true
  
  acm:
    enabled: true
    port: "auto"  # 或指定 "/dev/ttyACM0"
    baud_rate: 115200
    algo_timer_enabled: true
    algo_timer_interval: 5s
    algo_bet: 1
    algo_prize: 100
```

## 🚀 测试命令

部署后可以用测试工具验证：
```bash
/home/ztl/tools/serial_tester_linux_arm -d /dev/ttyACM0
```

如果slot-game正在运行，测试工具会显示设备被占用，这说明连接成功。