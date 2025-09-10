#!/bin/bash

# 快速修复 Chromium 服务配置
# 使用您验证过的工作配置

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}快速修复 Chromium 配置${NC}"
echo "================================================"

# 检查是否为root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用sudo运行此脚本${NC}"
    exit 1
fi

# 检测Chromium路径
CHROMIUM_PATH=""
if command -v chromium >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium)
elif command -v chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium-browser)
else
    echo -e "${RED}未找到 Chromium${NC}"
    exit 1
fi

echo -e "${GREEN}找到 Chromium: $CHROMIUM_PATH${NC}"

# 停止服务
echo "停止 chromium-kiosk 服务..."
systemctl stop chromium-kiosk 2>/dev/null || true

# 创建新的服务文件（使用您的工作配置）
cat > /etc/systemd/system/chromium-kiosk.service << EOF
[Unit]
Description=Chromium Kiosk Mode (ztl user)
After=graphical-session.target slot-game.service
Wants=graphical-session.target
Requires=slot-game.service

[Service]
Type=simple
User=ztl
Group=ztl
Environment="DISPLAY=:0"
Environment="HOME=/home/ztl"
Environment="USER=ztl"

# 等待 slot-game 服务就绪
ExecStartPre=/bin/bash -c 'timeout=30; while [ \$timeout -gt 0 ]; do \\
  if command -v curl >/dev/null 2>&1 && curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \\
  elif command -v wget >/dev/null 2>&1 && wget -q -O- http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; \\
  elif command -v nc >/dev/null 2>&1 && nc -zv 127.0.0.1 8080 2>/dev/null; then exit 0; \\
  elif [ -f /proc/net/tcp ] && grep -q ":1F90" /proc/net/tcp 2>/dev/null; then exit 0; \\
  fi; \\
  sleep 2; timeout=\$((timeout-2)); done; \\
  [ \$timeout -gt 0 ]'

# 使用您验证过的工作配置
ExecStart=$CHROMIUM_PATH \\
  --user-data-dir=/tmp/chromium-kiosk \\
  --kiosk \\
  --start-fullscreen \\
  --new-window \\
  --use-gl=egl \\
  --enable-gpu-rasterization \\
  --ignore-gpu-blocklist \\
  --disable-software-rasterizer \\
  --canvas-oop-rasterization=disabled \\
  --enable-accelerated-video-decode \\
  --enable-features=VaapiVideoDecoder,VaapiVideoEncoder \\
  --ozone-platform=x11 \\
  --no-first-run \\
  --no-default-browser-check \\
  --password-store=basic \\
  --disable-password-manager-reauth \\
  --disable-features=BackForwardCache,LowPriorityIframes \\
  --disable-background-timer-throttling \\
  --disable-renderer-backgrounding \\
  --no-sandbox \\
  --disable-dev-shm-usage \\
  --disable-setuid-sandbox \\
  "http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo"

Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 资源限制
LimitNOFILE=65536

[Install]
WantedBy=graphical.target
EOF

# 重新加载配置
echo "重新加载 systemd 配置..."
systemctl daemon-reload

# 启动服务
echo "启动 chromium-kiosk 服务..."
systemctl start chromium-kiosk

# 检查状态
sleep 3
if systemctl is-active chromium-kiosk >/dev/null 2>&1; then
    echo -e "${GREEN}✓ chromium-kiosk 服务启动成功！${NC}"
    echo ""
    systemctl status chromium-kiosk --no-pager | head -10
else
    echo -e "${RED}✗ chromium-kiosk 服务启动失败${NC}"
    echo "查看错误日志："
    journalctl -u chromium-kiosk -n 20 --no-pager
fi

echo ""
echo "================================================"
echo -e "${GREEN}配置完成${NC}"
echo "================================================"
echo ""
echo "这个配置使用了您验证过的参数："
echo "- EGL 硬件加速"
echo "- GPU 光栅化"
echo "- 硬件视频解码"
echo "- 禁用后台节流"
echo ""
echo "如需调整，编辑："
echo "  sudo nano /etc/systemd/system/chromium-kiosk.service"