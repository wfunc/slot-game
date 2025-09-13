# ACM连接调试日志指南

## 🎯 问题诊断流程

根据我的深度分析（--ultrathink），ACM连接失败可能有以下原因：

### 1. **配置加载问题**
- ACM配置没有被正确读取
- 配置文件结构不正确
- 配置的启用标志为false

### 2. **初始化失败**
- ACM控制器对象创建失败
- 控制器为nil

### 3. **连接失败**
- 设备路径错误
- 权限问题
- 设备被占用

## 📋 新增的调试日志

### 初始化阶段日志

```
1. "串口配置详情" - 显示完整的串口配置
   - serial.enabled
   - serial.stm32.enabled
   - serial.acm.enabled  ← 重点关注这个
   - serial.acm.port
   - serial.acm.algo_timer_enabled

2. "检查ACM配置" - 专门检查ACM配置
   - acm.enabled ← 如果是false，说明配置没加载
   - acm_config (完整配置对象)

3. "开始初始化ACM控制器" - ACM初始化开始
   - 显示所有ACM参数

4. "创建ACM控制器对象" - 创建控制器
   - 如果失败会显示"ACM控制器创建失败：返回nil"
   - 成功会显示"ACM控制器创建成功"
```

### 连接阶段日志

```
1. "控制器状态检查" - 显示控制器对象状态
   - stm32_controller_exists
   - acm_controller_exists ← 如果是false，说明初始化失败
   - stm32_connected
   - acm_connected

2. "尝试连接ACM..." - 开始连接
   - configured_port
   - auto_detect

3. 连接结果：
   - 成功："✅ ACM连接成功"
   - 失败："ACM连接失败" + error_detail
   - 控制器为空："❌ ACM控制器为nil，无法连接！"
```

## 🔍 请检查以下日志输出

运行服务后，请查找这些关键日志：

### 第一步：检查配置是否加载
```
"串口配置详情" ... serial.acm.enabled=??? 
```
**预期**：`serial.acm.enabled=true`
**如果是false**：配置文件有问题

### 第二步：检查ACM初始化
```
"检查ACM配置" acm.enabled=???
```
**预期**：`acm.enabled=true`
**如果是false或没有这条日志**：ACM配置没有被正确读取

### 第三步：检查控制器创建
```
"创建ACM控制器对象"
```
后面应该跟着：
- ✅ "ACM控制器创建成功"
- ❌ "ACM控制器创建失败：返回nil"

### 第四步：检查连接尝试
```
"控制器状态检查" acm_controller_exists=???
```
**预期**：`acm_controller_exists=true`
**如果是false**：控制器没有被创建

### 第五步：检查连接结果
```
"尝试连接ACM..."
```
后面应该有：
- ✅ "✅ ACM连接成功"
- ❌ "ACM连接失败" + 具体错误
- ❌ "❌ ACM控制器为nil，无法连接！"

## 🚨 重点关注

1. **如果看到 "ACM未启用，跳过初始化"**
   - 说明配置文件中 ACM.Enabled = false
   - 检查 `/home/sg/slot-game/config/config.yaml`

2. **如果看到 "❌ ACM控制器为nil"**
   - 说明初始化阶段就失败了
   - 查看前面的初始化日志

3. **如果看到 ACM连接失败的错误详情**
   - 查看 error_detail 具体内容
   - 可能是设备路径、权限等问题

## 📝 配置文件检查清单

请确认 `/home/sg/slot-game/config/config.yaml`：

```yaml
serial:
  enabled: true  # ← 必须是true
  
  acm:
    enabled: true  # ← 必须是true
    port: "auto"   # 或指定 "/dev/ttyACM0"
    algo_timer_enabled: true
```

## 🔧 测试命令

```bash
# 重新编译
make build-arm64

# 查看当前配置
cat /home/sg/slot-game/config/config.yaml | grep -A 10 "acm:"

# 重启服务并查看日志
sudo systemctl restart slot-game && sudo journalctl -u slot-game -f

# 检查ACM设备是否存在
ls -la /dev/ttyACM*

# 检查设备权限
ls -la /dev/ttyACM0

# 检查sg用户是否在dialout组
groups sg
```

## 📊 问题定位决策树

```
串口配置详情 serial.acm.enabled=false?
  └─> 是：配置文件问题 → 检查config.yaml
  └─> 否：继续↓

检查ACM配置 acm.enabled=false?
  └─> 是：配置加载问题 → 检查配置结构
  └─> 否：继续↓

ACM控制器创建失败？
  └─> 是：初始化问题 → 查看错误日志
  └─> 否：继续↓

控制器状态检查 acm_controller_exists=false?
  └─> 是：控制器未创建 → 检查初始化逻辑
  └─> 否：继续↓

ACM连接失败？
  └─> 查看error_detail
      ├─> "未找到ACM设备"：设备不存在或路径错误
      ├─> "permission denied"：权限问题
      └─> 其他：具体分析错误信息
```

请运行服务后，把包含这些新日志的输出发给我，我可以精确定位问题所在。