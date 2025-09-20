#!/bin/bash

# 修复SQLite数据库锁定问题的脚本

DB_PATH="${1:-./game.db}"
SERVICE_NAME="slot-game"

echo "=========================================="
echo "SQLite数据库锁定问题修复脚本"
echo "数据库路径: $DB_PATH"
echo "=========================================="

# 1. 停止服务
echo "[1] 停止 $SERVICE_NAME 服务..."
if systemctl is-active --quiet $SERVICE_NAME; then
    sudo systemctl stop $SERVICE_NAME
    echo "服务已停止"
else
    echo "服务未运行"
fi

# 2. 查找并杀死所有访问数据库的进程
echo "[2] 查找并终止访问数据库的进程..."
if [ -f "$DB_PATH" ]; then
    # 使用lsof查找访问数据库文件的进程
    PIDS=$(lsof "$DB_PATH" 2>/dev/null | awk 'NR>1 {print $2}' | sort -u)

    if [ ! -z "$PIDS" ]; then
        echo "发现以下进程正在访问数据库:"
        for PID in $PIDS; do
            ps -p $PID -o pid,comm,args
        done

        echo "终止这些进程..."
        for PID in $PIDS; do
            sudo kill -9 $PID 2>/dev/null
        done
        echo "进程已终止"
    else
        echo "没有进程访问数据库"
    fi
fi

# 3. 清理WAL和SHM文件
echo "[3] 清理数据库临时文件..."
if [ -f "${DB_PATH}-wal" ]; then
    rm -f "${DB_PATH}-wal"
    echo "删除了 ${DB_PATH}-wal"
fi

if [ -f "${DB_PATH}-shm" ]; then
    rm -f "${DB_PATH}-shm"
    echo "删除了 ${DB_PATH}-shm"
fi

if [ -f "${DB_PATH}-journal" ]; then
    rm -f "${DB_PATH}-journal"
    echo "删除了 ${DB_PATH}-journal"
fi

# 4. 检查和修复数据库
echo "[4] 检查数据库完整性..."
if [ -f "$DB_PATH" ]; then
    # 运行完整性检查
    INTEGRITY_CHECK=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;" 2>&1)

    if [ "$INTEGRITY_CHECK" = "ok" ]; then
        echo "数据库完整性检查通过"
    else
        echo "警告：数据库可能存在问题"
        echo "$INTEGRITY_CHECK"

        # 尝试恢复
        echo "尝试恢复数据库..."
        sqlite3 "$DB_PATH" <<EOF
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 30000;
VACUUM;
EOF
        echo "恢复完成"
    fi

    # 设置正确的权限
    echo "[5] 设置数据库权限..."
    if [ -n "$SUDO_USER" ]; then
        # 如果是通过sudo运行，设置为原始用户
        chown $SUDO_USER:$SUDO_USER "$DB_PATH"
    elif id -u sg >/dev/null 2>&1; then
        # 如果sg用户存在，设置为sg用户
        chown sg:sg "$DB_PATH"
    fi
    chmod 644 "$DB_PATH"
    echo "权限设置完成"
fi

# 5. 优化数据库
echo "[6] 优化数据库..."
if [ -f "$DB_PATH" ]; then
    sqlite3 "$DB_PATH" <<EOF
-- 启用WAL模式
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -2000000;
PRAGMA busy_timeout = 30000;
PRAGMA foreign_keys = ON;

-- 分析和优化
ANALYZE;
REINDEX;
EOF
    echo "数据库优化完成"
fi

# 6. 重启服务
echo "[7] 启动 $SERVICE_NAME 服务..."
sudo systemctl start $SERVICE_NAME

# 等待几秒检查状态
sleep 3

if systemctl is-active --quiet $SERVICE_NAME; then
    echo "✅ 服务启动成功！"
    echo "查看服务状态: sudo systemctl status $SERVICE_NAME"
    echo "查看日志: sudo journalctl -u $SERVICE_NAME -f"
else
    echo "❌ 服务启动失败"
    echo "查看错误日志: sudo journalctl -u $SERVICE_NAME -n 50"
fi

echo "=========================================="
echo "修复完成！"
echo "=========================================="