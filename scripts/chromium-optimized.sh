#!/bin/bash

# Chromium 优化启动配置脚本
# 针对 ARM64 Ubuntu 环境优化

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}Chromium 优化配置生成器${NC}"
echo "================================================"

# 检测 Chromium 路径
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

# 检测系统架构
ARCH=$(uname -m)
echo "系统架构: $ARCH"

# 生成优化的服务文件
cat > chromium-kiosk-optimized.service << EOF
[Unit]
Description=Chromium Kiosk Mode (Optimized for ARM64)
After=graphical-session.target slot-game.service
Wants=graphical-session.target
Requires=slot-game.service

[Service]
Type=simple
User=ztl
Group=ztl
Environment="DISPLAY=:0"
Environment="HOME=/home/ztl"

# 等待 slot-game 服务就绪
ExecStartPre=/bin/bash -c 'timeout=30; while [ \$timeout -gt 0 ]; do \\
  if curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then exit 0; fi; \\
  sleep 2; timeout=\$((timeout-2)); done; \\
  [ \$timeout -gt 0 ]'

# 优化的 Chromium 启动命令
ExecStart=$CHROMIUM_PATH \\
  --kiosk \\
  --start-fullscreen \\
  --window-position=0,0 \\
  --window-size=1920,1080 \\
  --user-data-dir=/tmp/chromium-kiosk \\
  --disk-cache-dir=/tmp/chromium-cache \\
  --no-sandbox \\
  --disable-dev-shm-usage \\
  --disable-gpu-sandbox \\
  --disable-setuid-sandbox \\
  --disable-breakpad \\
  --disable-crash-reporter \\
  --disable-cloud-import \\
  --disable-gesture-typing \\
  --disable-offer-store-unmasked-wallet-cards \\
  --disable-offer-upload-credit-cards \\
  --disable-print-preview \\
  --disable-voice-input \\
  --disable-wake-on-wifi \\
  --disable-cookie-encryption \\
  --ignore-gpu-blocklist \\
  --enable-gpu-rasterization \\
  --enable-accelerated-2d-canvas \\
  --enable-smooth-scrolling \\
  --disable-features=TranslateUI,BlinkGenPropertyTrees,BackForwardCache \\
  --disable-background-timer-throttling \\
  --disable-backgrounding-occluded-windows \\
  --disable-renderer-backgrounding \\
  --disable-features=LazyFrameLoading \\
  --disable-ipc-flooding-protection \\
  --disable-pinch \\
  --overscroll-history-navigation=0 \\
  --noerrdialogs \\
  --disable-infobars \\
  --check-for-update-interval=31536000 \\
  --autoplay-policy=no-user-gesture-required \\
  --password-store=basic \\
  --disable-component-extensions-with-background-pages \\
  --disable-features=OptimizationHints \\
  --disable-features=PrivacySandboxSettings4 \\
  --disable-search-engine-choice-screen \\
  --no-default-browser-check \\
  --no-first-run \\
  --disable-blink-features=AutomationControlled \\
  --use-fake-ui-for-media-stream \\
  --disable-sync \\
  --no-pings \\
  --disable-default-apps \\
  --disable-features=InterestFeedContentSuggestions \\
  --disable-features=CrostiniUseBusterImage \\
  --disable-features=UserAgentClientHint \\
  --disable-hang-monitor \\
  http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo

Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 资源限制
LimitNOFILE=65536
MemoryLimit=1G
CPUQuota=80%

[Install]
WantedBy=graphical.target
EOF

echo ""
echo -e "${GREEN}优化配置已生成！${NC}"
echo ""
echo "主要优化点："
echo "1. 移除了可能导致问题的 GPU 参数："
echo "   - 移除 --use-gl=egl (让系统自动选择)"
echo "   - 移除 --canvas-oop-rasterization=disabled"
echo "   - 移除 VaapiVideoDecoder (ARM64可能不支持)"
echo ""
echo "2. 添加了稳定性参数："
echo "   - --disable-gpu-sandbox (避免权限问题)"
echo "   - --disk-cache-dir (指定缓存目录)"
echo "   - --disable-hang-monitor (避免假死检测)"
echo ""
echo "3. 性能优化："
echo "   - --enable-gpu-rasterization (GPU加速)"
echo "   - --enable-accelerated-2d-canvas (2D加速)"
echo "   - --enable-smooth-scrolling (平滑滚动)"
echo ""
echo "4. 资源限制："
echo "   - 内存限制 1GB"
echo "   - CPU 限制 80%"
echo ""
echo "安装新配置："
echo "  sudo cp chromium-kiosk-optimized.service /etc/systemd/system/chromium-kiosk.service"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl restart chromium-kiosk"
echo ""
echo "测试不同的 GPU 模式："
echo "  1. 默认模式（推荐）：不指定 --use-gl"
echo "  2. 软件渲染：添加 --disable-gpu"
echo "  3. EGL模式：添加 --use-gl=egl"
echo "  4. GLES模式：添加 --use-gl=gles"
echo ""
echo "如果仍有问题，尝试最小化模式："
echo "  $CHROMIUM_PATH --kiosk --no-sandbox --disable-gpu --disable-software-rasterizer http://127.0.0.1:8080"