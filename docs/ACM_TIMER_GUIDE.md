# ACM Algo定时器使用指南

## 功能概述

ACM控制器现在支持定时发送`algo`命令功能，可以按照配置的时间间隔自动向ACM设备发送算法请求命令。

## 配置说明

### 1. 配置文件设置

在 `config/config.yaml` 中配置ACM定时器：

```yaml
serial:
  acm:
    enabled: true         # 启用ACM设备
    port: "auto"          # 自动检测ACM设备
    baud_rate: 115200
    
    # Algo命令定时器配置
    algo_timer_enabled: true    # 是否启用定时发送algo命令
    algo_timer_interval: 5s     # 发送间隔时间
    algo_bet: 1                 # algo命令的bet参数
    algo_prize: 100             # algo命令的prize参数
```

### 2. 配置参数说明

- `algo_timer_enabled`: 是否启用定时器功能
- `algo_timer_interval`: 发送命令的时间间隔（支持 1s, 5s, 30s, 1m 等格式）
- `algo_bet`: algo命令中的投注参数
- `algo_prize`: algo命令中的奖金参数

## 使用方式

### 1. 自动启动

当配置文件中 `algo_timer_enabled: true` 时，ACM控制器连接成功后会自动启动定时器。

### 2. 手动控制

通过代码控制定时器：

```go
// 启动定时器
err := acmController.StartAlgoTimer()

// 停止定时器
acmController.StopAlgoTimer()

// 动态修改定时器参数
acmController.SetAlgoTimer(
    true,              // enabled
    10*time.Second,    // interval
    2,                 // bet
    200,               // prize
)
```

## 命令格式

定时器发送的命令格式：
```
algo -b 1 -p 100
```

其中：
- `-b` 参数对应配置的 `algo_bet`
- `-p` 参数对应配置的 `algo_prize`

## 响应处理

定时器会自动记录每次命令的响应，响应格式示例：

```json
{
  "code": 0,
  "msg": "success",
  "ident": 1234,
  "function": "algo",
  "prize": 100,
  "bet": 1,
  "algo": {
    "part": [
      [],
      {
        "l1": [1, 2, 3, 4, 5],
        "l2": [2, 3, 4, 5, 6],
        "l3": [3, 4, 5, 6, 7],
        "l4": [4, 5, 6, 7, 0],
        "l5": [5, 6, 7, 0, 1]
      }
    ]
  },
  "hp30": 0,
  "win": 2.5,
  "chk": "checksum_string"
}
```

## 日志输出

定时器运行时会输出以下日志：

1. **启动日志**：
```
INFO: Algo定时器已启动 interval=5s bet=1 prize=100
```

2. **发送日志**（Debug级别）：
```
DEBUG: 定时发送algo命令 command="algo -b 1 -p 100" timestamp=2024-12-30T10:00:00Z
```

3. **响应日志**：
```
INFO: 定时algo命令响应 response={...} win=2.5 hp30=0
```

4. **错误日志**：
```
ERROR: 定时algo命令执行失败 error="connection timeout" command="algo -b 1 -p 100"
```

## 测试程序

运行测试程序验证定时器功能：

```bash
# 编译测试程序
go build -o test_acm_timer examples/test_acm_timer.go

# 运行测试
./test_acm_timer
```

测试程序功能：
1. 自动连接ACM设备
2. 启动定时器（每3秒发送一次）
3. 运行30秒后停止定时器
4. 5秒后使用新参数重启定时器
5. 按Ctrl+C退出

## 注意事项

1. **连接要求**：定时器只在ACM设备连接成功后才会启动
2. **资源管理**：断开连接时会自动停止定时器
3. **并发安全**：定时器操作是线程安全的
4. **异步执行**：命令发送在独立协程中执行，不会阻塞主流程
5. **错误处理**：发送失败不会停止定时器，会在下次间隔继续尝试

## 故障排查

### 定时器未启动
- 检查配置文件中 `algo_timer_enabled` 是否为 `true`
- 检查ACM设备是否连接成功
- 查看日志中是否有错误信息

### 命令发送失败
- 检查ACM设备连接状态
- 验证串口配置是否正确
- 查看设备是否支持algo命令

### 响应解析错误
- 检查ACM设备固件版本
- 验证响应格式是否符合预期
- 查看日志中的原始响应数据