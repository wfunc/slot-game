# 串口日志系统文档

## 概述

串口日志系统用于记录和分析ACM和STM32设备的所有串口通信数据，便于调试和问题排查。

## 功能特性

### 1. 数据采集
- **双向记录**：记录所有发送(send)和接收(receive)的数据
- **多格式存储**：
  - 原始数据 (ASCII)
  - 十六进制数据 (HEX)
  - JSON格式数据（如果可解析）
- **设备支持**：
  - ACM控制器通信日志
  - STM32控制器通信日志

### 2. 游戏数据提取
从algo命令中自动提取并索引：
- bet（下注金额）
- prize（奖励金额）
- win（赢取金额）

### 3. 异步批量写入
- 使用缓冲通道实现异步写入
- 每5秒或缓冲区满（100条）时批量写入数据库
- 优化性能，减少数据库写入压力

### 4. 请求追踪
- 自动生成RequestID关联请求和响应
- 方便追踪完整的通信流程

## API接口

所有API接口位于 `/api/v1/admin/serial-logs` 路径下（需要管理员权限）。

### 查询日志列表
```
GET /api/v1/admin/serial-logs
```

查询参数：
- `device_type`: 设备类型 (acm/stm32)
- `direction`: 方向 (send/receive)
- `command`: 命令名称
- `start_time`: 开始时间 (RFC3339格式)
- `end_time`: 结束时间
- `min_bet`: 最小下注金额
- `max_bet`: 最大下注金额
- `has_error`: 是否有错误 (true/false)
- `limit`: 返回条数 (默认20)
- `offset`: 偏移量

### 获取最新日志
```
GET /api/v1/admin/serial-logs/latest
```

参数：
- `limit`: 返回条数
- `device_type`: 设备类型

### 获取统计信息
```
GET /api/v1/admin/serial-logs/stats
```

返回各设备、命令的统计数据。

### 获取algo命令日志
```
GET /api/v1/admin/serial-logs/algo
```

专门查询algo命令相关日志，包含bet、prize、win数据。

### 获取错误日志
```
GET /api/v1/admin/serial-logs/errors
```

查询所有错误级别的日志。

### 清理旧日志
```
POST /api/v1/admin/serial-logs/cleanup
```

参数：
- `retention_days`: 保留天数 (默认30)

### 导出日志
```
GET /api/v1/admin/serial-logs/export
```

导出日志为JSON格式文件。

## 数据库结构

### serial_logs 表
```sql
CREATE TABLE serial_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    device_type VARCHAR(20) NOT NULL,     -- 'acm' 或 'stm32'
    direction VARCHAR(10) NOT NULL,       -- 'send' 或 'receive'
    command VARCHAR(255),                 -- 命令名称
    raw_data TEXT,                       -- 原始数据
    hex_data TEXT,                       -- 十六进制数据
    json_data JSON,                      -- JSON格式数据
    bet DECIMAL(10,4),                   -- 下注金额
    prize DECIMAL(10,4),                 -- 奖励金额
    win DECIMAL(10,4),                   -- 赢取金额
    level VARCHAR(20),                   -- 日志级别
    error_message TEXT,                  -- 错误信息
    request_id VARCHAR(100),             -- 请求ID
    response_time BIGINT,                -- 响应时间(毫秒)
    session_id VARCHAR(100),             -- 会话ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_device_type (device_type),
    INDEX idx_direction (direction),
    INDEX idx_command (command),
    INDEX idx_request_id (request_id),
    INDEX idx_session_id (session_id),
    INDEX idx_created_at (created_at)
);
```

## 使用示例

### 查询特定时间段的algo命令
```bash
curl -X GET "http://localhost:8080/api/v1/admin/serial-logs/algo?start_time=2024-01-01T00:00:00Z&limit=100" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 查询错误日志
```bash
curl -X GET "http://localhost:8080/api/v1/admin/serial-logs/errors?limit=50" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 清理30天前的旧日志
```bash
curl -X POST "http://localhost:8080/api/v1/admin/serial-logs/cleanup" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d "retention_days=30"
```

## 性能优化

1. **索引优化**：对常用查询字段建立索引
2. **批量写入**：使用缓冲区批量写入，减少数据库操作
3. **异步处理**：日志写入不阻塞主业务流程
4. **自动清理**：定期清理旧日志，防止数据库膨胀

## 监控建议

1. 定期检查错误日志数量
2. 监控响应时间分布
3. 关注algo命令的bet/win比率
4. 设置日志存储容量告警