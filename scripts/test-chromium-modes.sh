#!/bin/bash

# Chromium GPU模式测试脚本
# 用于找到最适合您的ARM64设备的配置

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}Chromium GPU 模式测试工具${NC}"
echo "================================================"
echo ""
echo "此脚本将测试不同的GPU渲染模式，找到最适合您设备的配置"
echo ""

# 检测Chromium
CHROMIUM_PATH=""
if command -v chromium >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium)
elif command -v chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium-browser)
else
    echo -e "${RED}未找到 Chromium${NC}"
    exit 1
fi

echo -e "${GREEN}Chromium 路径: $CHROMIUM_PATH${NC}"
echo ""

# 基础参数
BASE_ARGS="--kiosk --start-fullscreen --window-size=1920,1080 --user-data-dir=/tmp/test-chromium --no-sandbox --disable-dev-shm-usage"
URL="http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo"

# 测试函数
test_mode() {
    local MODE_NAME=$1
    local EXTRA_ARGS=$2
    
    echo -e "${BLUE}测试模式: ${MODE_NAME}${NC}"
    echo "参数: $EXTRA_ARGS"
    echo ""
    
    # 启动Chromium
    timeout 10 $CHROMIUM_PATH $BASE_ARGS $EXTRA_ARGS $URL 2>/tmp/chromium-test.log &
    PID=$!
    
    # 等待5秒
    sleep 5
    
    # 检查进程是否还在运行
    if kill -0 $PID 2>/dev/null; then
        echo -e "${GREEN}✓ 成功启动${NC}"
        # 检查GPU信息
        if [ -f /tmp/chromium-test.log ]; then
            echo "GPU信息:"
            grep -i "gpu\|gl\|egl\|gles" /tmp/chromium-test.log | head -5
        fi
        # 停止进程
        kill $PID 2>/dev/null
        wait $PID 2>/dev/null
        return 0
    else
        echo -e "${RED}✗ 启动失败${NC}"
        if [ -f /tmp/chromium-test.log ]; then
            echo "错误信息:"
            tail -5 /tmp/chromium-test.log
        fi
        return 1
    fi
}

# 测试不同模式
echo "================================================"
echo -e "${GREEN}开始测试不同的GPU模式${NC}"
echo "================================================"
echo ""

# 模式1：默认（自动选择）
echo "1. 默认模式（推荐）"
echo "----------------------------------------"
if test_mode "默认自动选择" ""; then
    MODE1_OK=true
else
    MODE1_OK=false
fi
echo ""

# 模式2：禁用GPU（纯软件渲染）
echo "2. 软件渲染模式"
echo "----------------------------------------"
if test_mode "纯软件渲染" "--disable-gpu --disable-software-rasterizer"; then
    MODE2_OK=true
else
    MODE2_OK=false
fi
echo ""

# 模式3：EGL模式（您之前使用的）
echo "3. EGL模式"
echo "----------------------------------------"
if test_mode "EGL硬件加速" "--use-gl=egl --enable-gpu-rasterization"; then
    MODE3_OK=true
else
    MODE3_OK=false
fi
echo ""

# 模式4：GLES模式
echo "4. GLES模式"
echo "----------------------------------------"
if test_mode "GLES硬件加速" "--use-gl=gles --enable-gpu-rasterization"; then
    MODE4_OK=true
else
    MODE4_OK=false
fi
echo ""

# 模式5：SwiftShader（软件3D渲染）
echo "5. SwiftShader模式"
echo "----------------------------------------"
if test_mode "SwiftShader" "--use-gl=swiftshader"; then
    MODE5_OK=true
else
    MODE5_OK=false
fi
echo ""

# 模式6：您原始的完整配置
echo "6. 原始配置（完整参数）"
echo "----------------------------------------"
ORIGINAL_ARGS="--use-gl=egl --enable-gpu-rasterization --ignore-gpu-blocklist --disable-software-rasterizer --canvas-oop-rasterization=disabled --enable-accelerated-video-decode --enable-features=VaapiVideoDecoder,VaapiVideoEncoder --ozone-platform=x11"
if test_mode "原始完整配置" "$ORIGINAL_ARGS"; then
    MODE6_OK=true
else
    MODE6_OK=false
fi
echo ""

# 总结
echo "================================================"
echo -e "${GREEN}测试结果总结${NC}"
echo "================================================"
echo ""

if [ "$MODE1_OK" = true ]; then
    echo -e "${GREEN}✓ 模式1：默认自动选择 - 成功${NC}"
fi
if [ "$MODE2_OK" = true ]; then
    echo -e "${GREEN}✓ 模式2：软件渲染 - 成功${NC}"
fi
if [ "$MODE3_OK" = true ]; then
    echo -e "${GREEN}✓ 模式3：EGL硬件加速 - 成功${NC}"
fi
if [ "$MODE4_OK" = true ]; then
    echo -e "${GREEN}✓ 模式4：GLES硬件加速 - 成功${NC}"
fi
if [ "$MODE5_OK" = true ]; then
    echo -e "${GREEN}✓ 模式5：SwiftShader - 成功${NC}"
fi
if [ "$MODE6_OK" = true ]; then
    echo -e "${GREEN}✓ 模式6：原始完整配置 - 成功${NC}"
fi

echo ""
echo -e "${YELLOW}建议：${NC}"
echo "1. 优先使用成功的模式中的第一个"
echo "2. 如果所有模式都失败，检查："
echo "   - slot-game 服务是否运行: systemctl status slot-game"
echo "   - 图形环境是否正常: echo \$DISPLAY"
echo "   - Chromium是否正确安装: $CHROMIUM_PATH --version"
echo ""
echo "3. 选择合适的模式后，更新服务文件："
echo "   sudo nano /etc/systemd/system/chromium-kiosk.service"
echo "   修改 ExecStart 行添加对应的GPU参数"
echo "   sudo systemctl daemon-reload"
echo "   sudo systemctl restart chromium-kiosk"