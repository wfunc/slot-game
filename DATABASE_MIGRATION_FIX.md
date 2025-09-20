# 数据库迁移性能优化和锁定问题修复

## 问题描述

在Ubuntu系统部署时，出现两个主要问题：

### 1. 数据库迁移性能问题
- serial_logs表有176,951条记录
- GORM的AutoMigrate尝试重建整个表，导致迁移时间超过53秒
- 服务启动失败或超时

### 2. SQLite数据库锁定问题
- 错误信息："database table is locked"
- 服务无法启动，systemd不断重试
- 影响jackpots等表的迁移

## 根本原因

### 迁移性能问题
GORM的AutoMigrate在SQLite中检测到表结构变化时，会：
1. 创建新的临时表
2. 复制所有数据到临时表
3. 删除原表
4. 重命名临时表

对于大数据量的表，这个过程极其缓慢。

### 数据库锁定问题
SQLite默认使用DELETE模式的journal，在多进程访问时容易产生锁定：
1. systemd重启服务时，旧进程可能还未完全释放数据库
2. WAL模式未启用，导致写操作互斥
3. 迁移操作需要独占访问，与其他连接冲突

## 解决方案

### 1. 优化迁移策略

在`internal/database/migration.go`中实现了智能迁移：

```go
// 检查大表是否需要迁移
func shouldSkipMigration(tableName string) bool {
    if tableName == "serial_logs" {
        // 检查表是否存在
        // 检查数据量
        // 如果超过10000条记录，跳过AutoMigrate
        // 仅创建缺失的索引
    }
}
```

### 2. SQLite性能优化（立即启用WAL模式）

在`internal/database/database.go`中，连接后立即启用WAL模式：

```go
func optimizeSQLiteImmediately() error {
    // 最关键：立即启用WAL模式以避免锁问题
    PRAGMA journal_mode = WAL;
    // 设置忙等待超时为30秒
    PRAGMA busy_timeout = 30000;
    // 降低同步级别提高写入性能
    PRAGMA synchronous = NORMAL;
    // 增大缓存到2GB
    PRAGMA cache_size = -2000000;
    // 使用内存存储临时表
    PRAGMA temp_store = MEMORY;
    // 启用内存映射
    PRAGMA mmap_size = 30000000000;
}
```

### 3. 索引管理

对大表单独管理索引，避免触发表重建：

```go
func ensureIndexesForLargeTable(tableName string) {
    // 仅创建不存在的索引
    // 使用CREATE INDEX IF NOT EXISTS
    // 避免重建表结构
}
```

## 使用方法

### 开发环境

正常运行即可，迁移会自动优化：
```bash
make run
```

### 生产环境（Ubuntu）

#### 方法1：预防性优化（推荐）

首次部署前，运行优化脚本：
```bash
chmod +x scripts/optimize_db.sh
./scripts/optimize_db.sh ./game.db
```

#### 方法2：修复锁定问题

如果遇到"database table is locked"错误：
```bash
chmod +x scripts/fix_db_lock.sh
sudo ./scripts/fix_db_lock.sh /home/sg/slot-game/game.db
```

该脚本会：
1. 停止服务
2. 终止访问数据库的进程
3. 清理WAL/SHM/Journal文件
4. 检查并修复数据库完整性
5. 启用WAL模式和优化参数
6. 重启服务

#### 方法3：手动处理

```bash
# 1. 停止服务
sudo systemctl stop slot-game

# 2. 清理锁文件
rm -f game.db-wal game.db-shm game.db-journal

# 3. 优化数据库
sqlite3 game.db <<EOF
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 30000;
VACUUM;
EOF

# 4. 重启服务
sudo systemctl start slot-game
```

## 性能对比

优化前：
- serial_logs表迁移时间：53+ 秒
- 服务启动：失败或超时

优化后：
- serial_logs表迁移时间：< 1秒（跳过重建，仅创建索引）
- 服务启动：正常

## 注意事项

1. 首次部署时建议备份数据库
2. 大表（>10000条记录）会自动跳过AutoMigrate
3. 索引会正常创建和维护
4. 不影响新表的创建和小表的迁移

## 监控建议

监控以下指标：
- 数据库文件大小
- serial_logs表记录数
- 迁移执行时间
- 服务启动时间

当serial_logs表超过100万条记录时，考虑：
- 归档历史数据
- 分表存储
- 使用专门的时序数据库