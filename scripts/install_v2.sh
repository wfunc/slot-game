#!/bin/bash

# 推币机游戏智能安装/更新脚本 V2
# 支持：
# - 全新安装
# - 更新升级
# - 数据库迁移处理
# - 数据库锁定修复

set -e  # 遇到错误立即退出

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# 配置变量
SG_USER="sg"
SG_GROUP="sg"
SG_HOME="/home/sg"
SG_APP_DIR="${SG_HOME}/slot-game"
ZTL_USER="ztl"
ZTL_APP_DIR="/home/ztl/slot-game-arm64"
DB_PATH="${SG_APP_DIR}/data/game.db"
BACKUP_DIR="${SG_APP_DIR}/backups"

# 版本检测
INSTALL_TYPE="new"  # new | update
OLD_VERSION=""
NEW_VERSION=""

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_debug() {
    [ "${DEBUG:-0}" = "1" ] && echo -e "${MAGENTA}[DEBUG]${NC} $1" >&2
}

# 检查是否为root用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用sudo运行此脚本"
        echo "用法: sudo $0"
        exit 1
    fi
}

# 检测安装类型
detect_install_type() {
    log_step "检测安装类型..."

    if [ -f "${SG_APP_DIR}/slot-game" ]; then
        INSTALL_TYPE="update"
        # 尝试获取旧版本
        if [ -f "${SG_APP_DIR}/VERSION" ]; then
            OLD_VERSION=$(cat "${SG_APP_DIR}/VERSION")
        fi
        log_info "检测到已安装版本: ${OLD_VERSION:-未知}"
    else
        INSTALL_TYPE="new"
        log_info "这是全新安装"
    fi

    # 获取新版本
    if [ -f "./VERSION" ]; then
        NEW_VERSION=$(cat "./VERSION")
    elif [ -f "./slot-game" ]; then
        # 尝试从二进制文件获取版本
        NEW_VERSION=$(./slot-game --version 2>/dev/null | head -1 || echo "未知")
    fi
    log_info "新版本: ${NEW_VERSION:-未知}"
}

# 备份数据库
backup_database() {
    if [ ! -f "$DB_PATH" ]; then
        return 0
    fi

    log_step "备份数据库..."

    # 创建备份目录
    mkdir -p "$BACKUP_DIR"

    # 生成备份文件名
    BACKUP_FILE="${BACKUP_DIR}/game_$(date +%Y%m%d_%H%M%S).db"

    # 停止服务以确保数据一致性
    systemctl stop slot-game 2>/dev/null || true

    # 备份数据库和相关文件
    cp "$DB_PATH" "$BACKUP_FILE"
    [ -f "${DB_PATH}-wal" ] && cp "${DB_PATH}-wal" "${BACKUP_FILE}-wal"
    [ -f "${DB_PATH}-shm" ] && cp "${DB_PATH}-shm" "${BACKUP_FILE}-shm"

    # 设置权限
    chown -R ${SG_USER}:${SG_GROUP} "$BACKUP_DIR"

    log_success "数据库已备份到: $BACKUP_FILE"

    # 保留最近10个备份
    ls -t ${BACKUP_DIR}/game_*.db 2>/dev/null | tail -n +11 | xargs -r rm -f
}

# 修复数据库锁定
fix_database_lock() {
    log_step "检查并修复数据库锁定..."

    # 只在更新时停止服务
    if [ "$INSTALL_TYPE" = "update" ]; then
        systemctl stop slot-game 2>/dev/null || true
        sleep 2
    fi

    # 查找并终止访问数据库的进程
    if [ -f "$DB_PATH" ]; then
        PIDS=$(lsof "$DB_PATH" 2>/dev/null | awk 'NR>1 {print $2}' | sort -u)
        if [ ! -z "$PIDS" ]; then
            log_warn "发现进程正在访问数据库，正在终止..."
            for PID in $PIDS; do
                kill -9 $PID 2>/dev/null || true
            done
            sleep 1
        fi
    fi

    # 清理WAL和SHM文件
    if [ -f "${DB_PATH}-wal" ] || [ -f "${DB_PATH}-shm" ] || [ -f "${DB_PATH}-journal" ]; then
        log_info "清理数据库临时文件..."
        rm -f "${DB_PATH}-wal" "${DB_PATH}-shm" "${DB_PATH}-journal"
    fi

    log_success "数据库锁定检查完成"
}

# 优化数据库
optimize_database() {
    if [ ! -f "$DB_PATH" ]; then
        return 0
    fi

    log_step "优化数据库性能..."

    # 运行SQLite优化（添加timeout防止卡住）
    timeout 30 sqlite3 "$DB_PATH" <<EOF 2>/dev/null || true
-- 启用WAL模式（最重要）
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 30000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -2000000;
PRAGMA temp_store = MEMORY;
PRAGMA foreign_keys = ON;

-- 分析和优化
ANALYZE;
REINDEX;
EOF

    # 检查serial_logs表大小（添加timeout）
    if [ -f "$DB_PATH" ]; then
        SERIAL_LOGS_COUNT=$(timeout 5 sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM serial_logs;" 2>/dev/null || echo "0")
        if [ "$SERIAL_LOGS_COUNT" -gt 100000 ]; then
            log_warn "serial_logs表有 ${SERIAL_LOGS_COUNT} 条记录"
            log_info "建议定期清理历史数据：sqlite3 $DB_PATH \"DELETE FROM serial_logs WHERE created_at < date('now','-30 day');\""
        fi
    fi

    log_success "数据库优化完成"
}

# 创建用户和目录
setup_users_and_dirs() {
    log_step "设置用户和目录..."

    # 创建 sg 用户
    if ! id "$SG_USER" &>/dev/null; then
        useradd -m -s /bin/bash -d ${SG_HOME} ${SG_USER}
        log_success "用户 ${SG_USER} 创建成功"
    fi

    # 创建 ztl 用户
    if ! id "$ZTL_USER" &>/dev/null; then
        useradd -m -s /bin/bash ${ZTL_USER}
        log_success "用户 ${ZTL_USER} 创建成功"
    fi

    # 创建目录结构
    mkdir -p ${SG_APP_DIR}/{data,logs,config,static,backups}
    mkdir -p ${ZTL_APP_DIR}

    # 添加到必要的组
    usermod -a -G dialout ${SG_USER} 2>/dev/null || true
    usermod -a -G video ${SG_USER} 2>/dev/null || true
    usermod -a -G audio ${SG_USER} 2>/dev/null || true
}

# 部署文件
deploy_files() {
    log_step "部署程序文件..."

    CURRENT_DIR=$(pwd)

    # 备份旧文件（如果是更新）
    if [ "$INSTALL_TYPE" = "update" ] && [ -f "${SG_APP_DIR}/slot-game" ]; then
        log_info "备份旧程序文件..."
        cp "${SG_APP_DIR}/slot-game" "${SG_APP_DIR}/slot-game.old"
    fi

    # 部署到 sg 用户目录
    cp -a ${CURRENT_DIR}/slot-game ${SG_APP_DIR}/ 2>/dev/null || true
    cp -a ${CURRENT_DIR}/*.sh ${SG_APP_DIR}/ 2>/dev/null || true

    # 复制静态文件和配置（保留用户修改）
    if [ -d "${CURRENT_DIR}/static" ]; then
        cp -a ${CURRENT_DIR}/static/* ${SG_APP_DIR}/static/ 2>/dev/null || true
    fi

    if [ -d "${CURRENT_DIR}/config" ]; then
        # 如果是更新，备份用户配置
        if [ "$INSTALL_TYPE" = "update" ] && [ -f "${SG_APP_DIR}/config/config.yaml" ]; then
            cp "${SG_APP_DIR}/config/config.yaml" "${SG_APP_DIR}/config/config.yaml.backup"
            log_info "用户配置已备份"
        fi
        # 复制新配置（如果不存在）
        for config_file in ${CURRENT_DIR}/config/*; do
            filename=$(basename "$config_file")
            if [ ! -f "${SG_APP_DIR}/config/${filename}" ]; then
                cp "$config_file" "${SG_APP_DIR}/config/"
            fi
        done
    fi

    # 保存版本信息
    [ ! -z "$NEW_VERSION" ] && echo "$NEW_VERSION" > "${SG_APP_DIR}/VERSION"

    # 部署到 ztl 用户目录
    cp -a ${CURRENT_DIR}/* ${ZTL_APP_DIR}/ 2>/dev/null || true

    # 设置权限
    chmod +x ${SG_APP_DIR}/slot-game 2>/dev/null || true
    chmod +x ${SG_APP_DIR}/*.sh 2>/dev/null || true
    chown -R ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}

    chmod +x ${ZTL_APP_DIR}/slot-game 2>/dev/null || true
    chmod +x ${ZTL_APP_DIR}/*.sh 2>/dev/null || true
    chown -R ${ZTL_USER}:${ZTL_USER} ${ZTL_APP_DIR}

    # 设置数据库权限
    if [ -f "$DB_PATH" ]; then
        chown ${SG_USER}:${SG_GROUP} "$DB_PATH"
        chmod 644 "$DB_PATH"
    fi

    log_success "文件部署完成"
}

# 配置systemd服务
setup_services() {
    log_step "配置系统服务..."

    # 创建 slot-game 服务
    cat > /etc/systemd/system/slot-game.service << 'EOF'
[Unit]
Description=Slot Game Server (sg user)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=sg
Group=sg
WorkingDirectory=/home/sg/slot-game
ExecStart=/home/sg/slot-game/slot-game -config=/home/sg/slot-game/config/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=slot-game

# 环境变量
Environment="HOME=/home/sg"
Environment="USER=sg"

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 超时设置（给数据库迁移更多时间）
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

    # 创建 chromium-kiosk 服务
    cat > /etc/systemd/system/chromium-kiosk.service << 'EOF'
[Unit]
Description=Chromium Kiosk Mode (ztl user)
After=slot-game.service graphical-session.target
Wants=slot-game.service
Requires=slot-game.service

[Service]
Type=simple
User=ztl
Group=ztl
Environment="DISPLAY=:0"
Environment="HOME=/home/ztl"
Environment="USER=ztl"

# 等待 slot-game 服务就绪
ExecStartPre=/bin/bash -c 'timeout=60; while [ $timeout -gt 0 ]; do \
    if command -v curl >/dev/null 2>&1 && curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \
    elif command -v wget >/dev/null 2>&1 && wget -q -O- http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \
    elif command -v nc >/dev/null 2>&1 && nc -zv 127.0.0.1 8080 2>/dev/null; then exit 0; \
    fi; \
    sleep 2; timeout=$((timeout-2)); done; \
    [ $timeout -gt 0 ]'

# 启动 Chromium
ExecStart=/usr/bin/chromium \
    --user-data-dir=/tmp/chromium-kiosk \
    --autoplay-policy=no-user-gesture-required \
    --kiosk \
    --start-fullscreen \
    --new-window \
    --use-gl=egl \
    --enable-gpu-rasterization \
    --ignore-gpu-blocklist \
    --disable-software-rasterizer \
    --canvas-oop-rasterization=disabled \
    --enable-accelerated-video-decode \
    --enable-features=VaapiVideoDecoder,VaapiVideoEncoder \
    --ozone-platform=x11 \
    --no-first-run \
    --no-default-browser-check \
    --password-store=basic \
    --disable-password-manager-reauth \
    --disable-features=BackForwardCache,LowPriorityIframes \
    --disable-background-timer-throttling \
    --disable-renderer-backgrounding \
    --no-sandbox \
    --disable-dev-shm-usage \
    --disable-setuid-sandbox \
    "http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo"

Restart=always
RestartSec=5

[Install]
WantedBy=graphical.target
EOF

    # 重新加载 systemd
    systemctl daemon-reload

    log_success "服务配置完成"
}

# 启动服务
start_services() {
    log_step "启动服务..."

    # 启用服务
    systemctl enable slot-game.service 2>/dev/null || true

    # 仅在图形界面存在时启用chromium-kiosk
    if [ -n "$DISPLAY" ] || [ -n "$WAYLAND_DISPLAY" ]; then
        systemctl enable chromium-kiosk.service 2>/dev/null || true
    fi

    # 启动 slot-game
    log_info "启动 slot-game 服务..."
    if timeout 30 systemctl start slot-game.service; then
        log_success "slot-game 服务启动命令已执行"

        # 等待服务真正启动
        log_debug "等待服务完全启动..."
        for i in {1..10}; do
            if systemctl is-active slot-game >/dev/null 2>&1; then
                log_success "slot-game 服务已成功运行"
                break
            fi
            log_debug "等待服务启动... ($i/10)"
            sleep 2
        done

        # 最终检查
        if ! systemctl is-active slot-game >/dev/null 2>&1; then
            log_warn "slot-game 服务启动后未能保持运行状态"
            log_info "查看错误日志: journalctl -u slot-game -n 50"
        fi
    else
        log_warn "slot-game 服务启动超时，请检查日志: journalctl -u slot-game -n 50"
    fi

    # 检查是否有图形界面
    if [ -n "$DISPLAY" ] || [ -n "$WAYLAND_DISPLAY" ]; then
        log_info "检测到图形界面，尝试启动 chromium-kiosk..."
        # 使用timeout防止卡住
        if timeout 10 systemctl start chromium-kiosk.service 2>/dev/null; then
            log_success "chromium-kiosk 服务启动成功"
        else
            log_warn "chromium-kiosk 服务启动失败（可在图形界面手动启动）"
        fi
    else
        log_info "未检测到图形界面，chromium-kiosk 将在图形界面启动后自动运行"
    fi
}

# 显示安装摘要
show_summary() {
    log_debug "开始显示安装摘要..."
    echo ""
    echo "================================================"
    if [ "$INSTALL_TYPE" = "update" ]; then
        echo -e "${GREEN}    更新完成！${NC}"
        echo "================================================"
        echo ""
        log_info "版本信息："
        echo "  • 旧版本: ${OLD_VERSION:-未知}"
        echo "  • 新版本: ${NEW_VERSION:-未知}"
    else
        echo -e "${GREEN}    安装完成！${NC}"
        echo "================================================"
    fi
    echo ""
    log_info "系统配置："
    echo "  • slot-game 服务: ${SG_USER} 用户"
    echo "  • chromium-kiosk 服务: ${ZTL_USER} 用户"
    echo "  • 数据目录: ${SG_APP_DIR}/data"
    echo "  • 备份目录: ${SG_APP_DIR}/backups"
    echo ""
    log_info "服务状态："
    sleep 2  # 给服务更多时间完全启动
    timeout 2 systemctl is-active slot-game >/dev/null 2>&1 && \
        echo -e "  • slot-game: ${GREEN}运行中${NC}" || \
        echo -e "  • slot-game: ${YELLOW}未运行${NC}"
    timeout 2 systemctl is-active chromium-kiosk >/dev/null 2>&1 && \
        echo -e "  • chromium-kiosk: ${GREEN}运行中${NC}" || \
        echo -e "  • chromium-kiosk: ${YELLOW}未运行${NC}"
    echo ""
    log_info "管理命令："
    echo "  • 查看日志: journalctl -u slot-game -f"
    echo "  • 重启服务: systemctl restart slot-game"
    echo "  • 数据库优化: sqlite3 ${DB_PATH} 'VACUUM;'"

    if [ "$INSTALL_TYPE" = "update" ]; then
        echo ""
        log_info "备份位置："
        echo "  • 数据库备份: ${BACKUP_DIR}"
        echo "  • 旧程序: ${SG_APP_DIR}/slot-game.old"
        echo "  • 配置备份: ${SG_APP_DIR}/config/config.yaml.backup"
    fi

    echo ""
    echo "================================================"
    log_debug "安装摘要显示完成"
}

# 主函数
main() {
    echo "================================================"
    echo -e "${MAGENTA}    推币机游戏智能安装程序 V2${NC}"
    echo "================================================"
    echo ""

    # 检查root权限
    check_root

    # 检查程序文件
    if [ ! -f "slot-game" ]; then
        log_error "未找到程序文件 slot-game"
        exit 1
    fi

    # 检测安装类型
    detect_install_type

    if [ "$INSTALL_TYPE" = "update" ]; then
        log_info "执行更新流程..."

        # 备份数据库
        backup_database

        # 修复可能的锁定问题
        fix_database_lock
    else
        log_info "执行全新安装流程..."
    fi

    # 设置用户和目录
    setup_users_and_dirs

    # 部署文件
    deploy_files

    # 配置服务（放在优化数据库之前，避免服务影响数据库操作）
    setup_services

    # 优化数据库
    optimize_database

    # 启动服务
    log_debug "调用 start_services..."
    start_services

    # 显示摘要
    log_debug "调用 show_summary..."
    show_summary

    log_success "安装脚本执行完成"
    exit 0
}

# 运行主函数
main "$@"