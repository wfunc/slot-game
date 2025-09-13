#!/bin/bash

# 推币机游戏一键自动安装脚本
# 完全自动化，无需任何用户交互
# slot-game 运行在 sg 用户下
# chromium-kiosk 运行在 ztl 用户下

set -e  # 遇到错误立即退出

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
SG_USER="sg"
SG_GROUP="sg"
SG_HOME="/home/sg"
SG_APP_DIR="${SG_HOME}/slot-game"
ZTL_USER="ztl"
ZTL_APP_DIR="/home/ztl/slot-game-arm64"

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

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    log_error "请使用sudo运行此脚本"
    echo "用法: sudo $0"
    exit 1
fi

# 检查程序文件是否存在（而不是检查tar.gz包）
if [ ! -f "slot-game" ]; then
    log_error "未找到程序文件 slot-game"
    echo "请确保在正确的目录下运行此脚本"
    echo "当前目录: $(pwd)"
    exit 1
fi

echo "================================================"
echo -e "${GREEN}    推币机游戏一键自动安装程序${NC}"
echo "================================================"
echo ""
log_info "开始自动安装..."
echo ""

# ========================================
# 步骤1: 创建 sg 用户
# ========================================
log_step "创建 sg 用户..."

if id "$SG_USER" &>/dev/null; then
    log_warn "用户 ${SG_USER} 已存在，跳过创建"
else
    useradd -m -s /bin/bash -d ${SG_HOME} ${SG_USER}
    log_info "用户 ${SG_USER} 创建成功"
fi

# 创建目录结构
log_step "创建 sg 用户目录结构..."
mkdir -p ${SG_APP_DIR}/{data,logs,config,static,backups}
chown -R ${SG_USER}:${SG_GROUP} ${SG_HOME}
chmod 755 ${SG_HOME}
chmod 755 ${SG_APP_DIR}
chmod 700 ${SG_APP_DIR}/data
chmod 755 ${SG_APP_DIR}/logs
chmod 755 ${SG_APP_DIR}/config
chmod 755 ${SG_APP_DIR}/static
chmod 700 ${SG_APP_DIR}/backups

# 添加到必要的组
log_step "配置 sg 用户组权限..."
usermod -a -G dialout ${SG_USER} 2>/dev/null || true
usermod -a -G video ${SG_USER} 2>/dev/null || true
usermod -a -G audio ${SG_USER} 2>/dev/null || true

# ========================================
# 步骤2: 创建 ztl 用户目录
# ========================================
log_step "准备 ztl 用户环境..."

# 检查 ztl 用户是否存在
if ! id "$ZTL_USER" &>/dev/null; then
    log_warn "用户 ${ZTL_USER} 不存在，创建中..."
    useradd -m -s /bin/bash ${ZTL_USER}
fi

# 创建 ztl 应用目录
mkdir -p ${ZTL_APP_DIR}
chown ${ZTL_USER}:${ZTL_USER} ${ZTL_APP_DIR}

# ========================================
# 步骤3: 部署文件
# ========================================
log_step "部署文件..."

# 获取当前目录（安装包解压后的目录）
CURRENT_DIR=$(pwd)

# 部署到 sg 用户目录
log_step "部署文件到 sg 用户目录..."
cp -a ${CURRENT_DIR}/slot-game ${SG_APP_DIR}/ 2>/dev/null || true
cp -a ${CURRENT_DIR}/*.sh ${SG_APP_DIR}/ 2>/dev/null || true
if [ -d "${CURRENT_DIR}/static" ]; then
    cp -a ${CURRENT_DIR}/static/* ${SG_APP_DIR}/static/ 2>/dev/null || true
fi
if [ -d "${CURRENT_DIR}/config" ]; then
    cp -a ${CURRENT_DIR}/config/* ${SG_APP_DIR}/config/ 2>/dev/null || true
fi

# 部署到 ztl 用户目录（保留一份副本用于兼容性）
log_step "部署文件到 ztl 用户目录..."
cp -a ${CURRENT_DIR}/* ${ZTL_APP_DIR}/ 2>/dev/null || true

# 设置权限
chmod +x ${SG_APP_DIR}/slot-game 2>/dev/null || true
chmod +x ${SG_APP_DIR}/*.sh 2>/dev/null || true
chown -R ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}

chmod +x ${ZTL_APP_DIR}/slot-game 2>/dev/null || true
chmod +x ${ZTL_APP_DIR}/*.sh 2>/dev/null || true
chown -R ${ZTL_USER}:${ZTL_USER} ${ZTL_APP_DIR}

# ========================================
# 步骤4: 检测 Chromium 路径
# ========================================
log_step "检测 Chromium 浏览器..."

CHROMIUM_PATH=""
if command -v chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium-browser)
    log_info "找到 Chromium: $CHROMIUM_PATH"
elif command -v chromium >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium)
    log_info "找到 Chromium: $CHROMIUM_PATH"
else
    log_warn "未找到 Chromium 浏览器"
    log_warn "chromium-kiosk 服务将无法启动"
    log_info "请手动安装: sudo apt install chromium-browser 或 sudo apt install chromium"
    CHROMIUM_PATH="/usr/bin/chromium"  # 设置默认值
fi

# ========================================
# 步骤5: 配置 systemd 服务
# ========================================
log_step "安装 systemd 服务..."

# 创建 slot-game 服务（sg 用户）
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

[Install]
WantedBy=multi-user.target
EOF

# 创建 chromium-kiosk 服务（ztl 用户）
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

# 等待 slot-game 服务就绪（支持多种检测方式）
ExecStartPre=/bin/bash -c 'timeout=30; while [ $timeout -gt 0 ]; do \
    if command -v curl >/dev/null 2>&1 && curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \
    elif command -v wget >/dev/null 2>&1 && wget -q -O- http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \
    elif command -v nc >/dev/null 2>&1 && nc -zv 127.0.0.1 8080 2>/dev/null; then exit 0; \
    elif [ -f /proc/net/tcp ] && grep -q ":1F90" /proc/net/tcp 2>/dev/null; then exit 0; \
    fi; \
    sleep 2; timeout=$((timeout-2)); done; \
    [ $timeout -gt 0 ]'

# 启动 Chromium（使用您验证过的配置）
ExecStart=/usr/bin/chromium \
    --user-data-dir=/tmp/chromium-kiosk \
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

# 资源限制
LimitNOFILE=65536

[Install]
WantedBy=graphical.target
EOF

# ========================================
# 步骤5: 配置 sudo 权限
# ========================================
log_step "配置 sudo 权限..."

cat > /etc/sudoers.d/sg-user << EOF
# Allow sg user to manage services
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/journalctl -u slot-game *
EOF
chmod 440 /etc/sudoers.d/sg-user

# ========================================
# 步骤6: 启用服务
# ========================================
log_step "启用并启动服务..."

# 重新加载 systemd
systemctl daemon-reload

# 启用服务（开机自启）
systemctl enable slot-game.service
systemctl enable chromium-kiosk.service

# 尝试启动服务
log_step "启动服务..."
systemctl start slot-game.service || log_warn "slot-game 服务启动失败，将在重启后自动启动"

# 等待 slot-game 启动
sleep 3

# 检查是否有图形界面
if [ -n "$DISPLAY" ] || [ -n "$WAYLAND_DISPLAY" ]; then
    systemctl start chromium-kiosk.service || log_warn "chromium-kiosk 服务启动失败，将在重启后自动启动"
else
    log_warn "未检测到图形界面，chromium-kiosk 将在图形界面启动后自动运行"
fi

# ========================================
# 步骤7: 创建管理脚本
# ========================================
log_step "创建管理脚本..."

cat > /usr/local/bin/slot-game-manage << 'SCRIPT_EOF'
#!/bin/bash
# Slot Game 管理脚本

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

case "$1" in
    start)
        echo -e "${GREEN}启动服务...${NC}"
        sudo systemctl start slot-game
        sleep 2
        sudo systemctl start chromium-kiosk
        ;;
    stop)
        echo -e "${YELLOW}停止服务...${NC}"
        sudo systemctl stop chromium-kiosk
        sudo systemctl stop slot-game
        ;;
    restart)
        echo -e "${YELLOW}重启服务...${NC}"
        sudo systemctl restart slot-game
        sleep 2
        sudo systemctl restart chromium-kiosk
        ;;
    status)
        echo -e "${GREEN}服务状态：${NC}"
        sudo systemctl status slot-game --no-pager
        echo ""
        sudo systemctl status chromium-kiosk --no-pager
        ;;
    logs)
        echo -e "${GREEN}查看日志（Ctrl+C退出）：${NC}"
        sudo journalctl -f -u slot-game -u chromium-kiosk
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs}"
        exit 1
        ;;
esac
SCRIPT_EOF

chmod +x /usr/local/bin/slot-game-manage

# ========================================
# 步骤8: 完成安装
# ========================================
echo ""
echo "================================================"
echo -e "${GREEN}    安装完成！${NC}"
echo "================================================"
echo ""
log_info "系统配置："
echo "  • slot-game 服务: 运行在 ${SG_USER} 用户下"
echo "  • chromium-kiosk 服务: 运行在 ${ZTL_USER} 用户下"
echo "  • 数据目录: ${SG_APP_DIR}/data"
echo "  • 日志目录: ${SG_APP_DIR}/logs"
echo ""
log_info "服务状态："
systemctl is-active slot-game >/dev/null 2>&1 && echo -e "  • slot-game: ${GREEN}运行中${NC}" || echo -e "  • slot-game: ${YELLOW}未运行${NC}"
systemctl is-active chromium-kiosk >/dev/null 2>&1 && echo -e "  • chromium-kiosk: ${GREEN}运行中${NC}" || echo -e "  • chromium-kiosk: ${YELLOW}未运行${NC}"
echo ""
log_info "管理命令："
echo "  • 查看状态: slot-game-manage status"
echo "  • 重启服务: slot-game-manage restart"
echo "  • 查看日志: slot-game-manage logs"
echo ""
echo -e "${YELLOW}建议重启系统以确保所有服务正常启动：${NC}"
echo "  sudo reboot"
echo ""
echo "================================================"

# 安装成功
exit 0