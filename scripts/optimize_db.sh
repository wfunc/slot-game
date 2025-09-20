#!/bin/bash

# 优化SQLite数据库脚本
# 用于生产环境的数据库性能优化

DB_PATH="${1:-./game.db}"

if [ ! -f "$DB_PATH" ]; then
    echo "错误：数据库文件不存在: $DB_PATH"
    exit 1
fi

echo "开始优化数据库: $DB_PATH"
echo "数据库大小（优化前）: $(du -h $DB_PATH | cut -f1)"

# 备份数据库
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
echo "创建备份: $BACKUP_PATH"
cp "$DB_PATH" "$BACKUP_PATH"

# 运行SQLite优化命令
sqlite3 "$DB_PATH" <<EOF
-- 设置优化参数
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -2000000;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 30000000000;
PRAGMA page_size = 4096;
PRAGMA busy_timeout = 30000;
PRAGMA foreign_keys = ON;

-- 分析和优化表
ANALYZE;

-- 重建索引
REINDEX;

-- 清理未使用的空间
VACUUM;

-- 输出统计信息
SELECT 'serial_logs表记录数:', COUNT(*) FROM serial_logs;
SELECT '数据库页数:', page_count FROM pragma_page_count();
SELECT '页大小:', page_size FROM pragma_page_size();
EOF

echo "数据库大小（优化后）: $(du -h $DB_PATH | cut -f1)"
echo "优化完成！"

# 如果是在systemd服务中运行，可以选择性地重启服务
if systemctl is-active --quiet slot-game; then
    echo "检测到slot-game服务正在运行"
    echo "建议重启服务以应用优化: sudo systemctl restart slot-game"
fi