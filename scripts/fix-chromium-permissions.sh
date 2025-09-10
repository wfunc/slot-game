#!/bin/bash

# Chromium权限修复脚本 - 解决sg用户无法启动chromium的问题

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}Chromium 权限修复脚本${NC}"
echo "================================================"

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用sudo运行此脚本${NC}"
    exit 1
fi

# 1. 添加sg用户到必要的组
echo -e "\n${GREEN}1. 更新用户组权限...${NC}"
usermod -a -G video sg 2>/dev/null && echo "  ✓ 添加到 video 组"
usermod -a -G audio sg 2>/dev/null && echo "  ✓ 添加到 audio 组"
usermod -a -G dialout sg 2>/dev/null && echo "  ✓ 添加到 dialout 组"
usermod -a -G render sg 2>/dev/null && echo "  ✓ 添加到 render 组" || true
usermod -a -G input sg 2>/dev/null && echo "  ✓ 添加到 input 组" || true

# 2. 设置X11权限
echo -e "\n${GREEN}2. 配置X11权限...${NC}"
# 允许所有本地用户访问X服务器
xhost +local: 2>/dev/null || echo "  注意: xhost 未安装或X服务器未运行"

# 创建Xauthority文件
if [ -n "$DISPLAY" ]; then
    echo "  DISPLAY 已设置: $DISPLAY"
    # 为sg用户创建.Xauthority文件
    touch /home/sg/.Xauthority
    chown sg:sg /home/sg/.Xauthority
    chmod 600 /home/sg/.Xauthority
    
    # 复制当前用户的X权限到sg用户
    if [ -f "$HOME/.Xauthority" ]; then
        cp $HOME/.Xauthority /home/sg/.Xauthority
        chown sg:sg /home/sg/.Xauthority
        echo "  ✓ X权限已复制到sg用户"
    fi
else
    echo -e "  ${YELLOW}警告: DISPLAY未设置${NC}"
fi

# 3. 创建临时目录权限
echo -e "\n${GREEN}3. 设置临时目录权限...${NC}"
mkdir -p /tmp/chromium-kiosk
chown sg:sg /tmp/chromium-kiosk
chmod 755 /tmp/chromium-kiosk
echo "  ✓ 临时目录权限已设置"

# 4. 检查chromium路径
echo -e "\n${GREEN}4. 检查Chromium安装...${NC}"
if which chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH="chromium-browser"
    echo "  ✓ 找到 chromium-browser"
elif which chromium >/dev/null 2>&1; then
    CHROMIUM_PATH="chromium"
    echo "  ✓ 找到 chromium"
else
    echo -e "  ${RED}✗ 未找到chromium${NC}"
    echo "  请安装: sudo apt install chromium-browser"
    exit 1
fi

# 5. 更新服务文件（如果路径不同）
echo -e "\n${GREEN}5. 检查服务文件...${NC}"
SERVICE_FILE="/etc/systemd/system/chromium-kiosk.service"
if [ -f "$SERVICE_FILE" ]; then
    # 检查是否需要更新chromium路径
    if ! grep -q "$CHROMIUM_PATH" "$SERVICE_FILE"; then
        echo "  更新chromium路径为: $CHROMIUM_PATH"
        sed -i "s|/usr/bin/chromium|/usr/bin/$CHROMIUM_PATH|g" "$SERVICE_FILE"
        systemctl daemon-reload
    fi
    echo "  ✓ 服务文件已检查"
else
    echo -e "  ${YELLOW}服务文件不存在${NC}"
fi

# 6. 测试sg用户权限
echo -e "\n${GREEN}6. 测试用户权限...${NC}"
sudo -u sg bash -c 'echo "  ✓ sg用户可以执行命令"'
sudo -u sg bash -c "ls /home/sg/slot-game >/dev/null 2>&1" && echo "  ✓ sg用户可以访问应用目录"

# 7. 创建测试脚本
echo -e "\n${GREEN}7. 创建测试脚本...${NC}"
cat > /home/sg/test-chromium.sh << 'EOF'
#!/bin/bash
# 测试chromium是否可以启动

echo "测试Chromium启动..."
echo "DISPLAY=$DISPLAY"
echo "HOME=$HOME"
echo "USER=$USER"

# 尝试启动chromium（headless模式测试）
chromium --headless --disable-gpu --dump-dom http://127.0.0.1:8080 2>&1 | head -20

if [ $? -eq 0 ]; then
    echo "✓ Chromium可以启动"
else
    echo "✗ Chromium启动失败"
fi
EOF
chmod +x /home/sg/test-chromium.sh
chown sg:sg /home/sg/test-chromium.sh
echo "  ✓ 测试脚本已创建: /home/sg/test-chromium.sh"

# 8. 显示诊断信息
echo -e "\n${GREEN}诊断信息：${NC}"
echo "================================================"
echo "sg用户组: $(groups sg)"
echo "DISPLAY: ${DISPLAY:-未设置}"
echo "Chromium路径: $(which $CHROMIUM_PATH)"
echo "================================================"

echo -e "\n${GREEN}修复完成！${NC}"
echo ""
echo "下一步操作："
echo "1. 重启chromium-kiosk服务："
echo "   sudo systemctl restart chromium-kiosk"
echo ""
echo "2. 如果仍然失败，运行测试脚本："
echo "   sudo -u sg /home/sg/test-chromium.sh"
echo ""
echo "3. 查看错误日志："
echo "   sudo journalctl -u chromium-kiosk -n 50"
echo ""
echo "4. 如果是无头服务器（无图形界面），考虑："
echo "   - 安装 xvfb 虚拟显示器"
echo "   - 或者禁用 chromium-kiosk 服务"