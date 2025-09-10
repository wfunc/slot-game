#!/bin/bash

# Chromium 路径修复脚本 - 修复chromium-kiosk服务的浏览器路径问题

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "================================================"
echo -e "${GREEN}Chromium 路径修复脚本${NC}"
echo "================================================"

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用sudo运行此脚本${NC}"
    echo "用法: sudo $0"
    exit 1
fi

# 检测Chromium路径
echo -e "\n${GREEN}检测 Chromium 浏览器路径...${NC}"

CHROMIUM_PATH=""
if command -v chromium-browser >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium-browser)
    echo -e "${GREEN}✓ 找到 chromium-browser: $CHROMIUM_PATH${NC}"
elif command -v chromium >/dev/null 2>&1; then
    CHROMIUM_PATH=$(which chromium)
    echo -e "${GREEN}✓ 找到 chromium: $CHROMIUM_PATH${NC}"
else
    echo -e "${RED}✗ 未找到 Chromium 浏览器${NC}"
    echo ""
    echo "请先安装 Chromium:"
    echo "  Ubuntu/Debian: sudo apt install chromium-browser"
    echo "  或者: sudo apt install chromium"
    exit 1
fi

# 备份原服务文件
SERVICE_FILE="/etc/systemd/system/chromium-kiosk.service"
if [ -f "$SERVICE_FILE" ]; then
    echo -e "\n${GREEN}备份原服务文件...${NC}"
    cp $SERVICE_FILE ${SERVICE_FILE}.backup.$(date +%Y%m%d-%H%M%S)
    
    # 停止服务
    echo -e "${GREEN}停止 chromium-kiosk 服务...${NC}"
    systemctl stop chromium-kiosk 2>/dev/null || true
    
    # 修改服务文件
    echo -e "${GREEN}更新服务文件中的 Chromium 路径...${NC}"
    sed -i "s|ExecStart=/usr/bin/chromium-browser|ExecStart=$CHROMIUM_PATH|g" $SERVICE_FILE
    sed -i "s|ExecStart=/usr/bin/chromium|ExecStart=$CHROMIUM_PATH|g" $SERVICE_FILE
    
    # 重新加载systemd
    echo -e "${GREEN}重新加载 systemd 配置...${NC}"
    systemctl daemon-reload
    
    # 尝试启动服务
    echo -e "${GREEN}启动 chromium-kiosk 服务...${NC}"
    systemctl start chromium-kiosk
    
    # 检查服务状态
    sleep 2
    if systemctl is-active chromium-kiosk >/dev/null 2>&1; then
        echo -e "${GREEN}✓ chromium-kiosk 服务启动成功！${NC}"
        systemctl status chromium-kiosk --no-pager | head -10
    else
        echo -e "${YELLOW}⚠ chromium-kiosk 服务启动失败${NC}"
        echo "查看错误日志："
        journalctl -u chromium-kiosk -n 20 --no-pager
    fi
else
    echo -e "${RED}未找到 chromium-kiosk.service 文件${NC}"
    echo "请先运行安装脚本: sudo ./install.sh"
    exit 1
fi

echo ""
echo "================================================"
echo -e "${GREEN}修复完成${NC}"
echo "================================================"
echo ""
echo "管理命令："
echo "  查看状态: systemctl status chromium-kiosk"
echo "  查看日志: journalctl -u chromium-kiosk -f"
echo "  重启服务: systemctl restart chromium-kiosk"
echo "  停止服务: systemctl stop chromium-kiosk"