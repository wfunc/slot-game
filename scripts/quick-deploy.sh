#!/bin/bash

# 快速部署脚本 - 编译并部署到ARM64设备
# 使用方法: ./scripts/quick-deploy.sh

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}    快速部署 Slot Game 到 ARM64${NC}"
echo -e "${GREEN}========================================${NC}"

# 1. 编译
echo -e "\n${YELLOW}步骤1: 编译ARM64版本...${NC}"
make build-arm64
if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败！${NC}"
    exit 1
fi

# 2. 检查编译结果
if [ ! -f "release/slot-game-arm64.tar.gz" ]; then
    echo -e "${RED}未找到编译包！${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 编译成功！${NC}"
echo -e "${GREEN}编译包: release/slot-game-arm64.tar.gz${NC}"

# 3. 提示部署步骤
echo -e "\n${YELLOW}请在目标设备上执行以下步骤：${NC}"
echo ""
echo "1. 复制编译包到目标设备："
echo -e "   ${GREEN}scp release/slot-game-arm64.tar.gz sg@<设备IP>:/tmp/${NC}"
echo ""
echo "2. SSH登录到设备："
echo -e "   ${GREEN}ssh sg@<设备IP>${NC}"
echo ""
echo "3. 解压并安装："
echo -e "   ${GREEN}cd /tmp${NC}"
echo -e "   ${GREEN}tar -xzf slot-game-arm64.tar.gz${NC}"
echo -e "   ${GREEN}cd slot-game-arm64${NC}"
echo -e "   ${GREEN}sudo ./install.sh${NC}"
echo ""
echo "4. 重启服务："
echo -e "   ${GREEN}sudo systemctl restart slot-game${NC}"
echo ""
echo "5. 查看日志："
echo -e "   ${GREEN}sudo journalctl -u slot-game -f${NC}"
echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}重要：请确保设备上的配置文件正确！${NC}"
echo -e "${YELLOW}配置文件位置: /home/sg/slot-game/config/config.yaml${NC}"
echo -e "${YELLOW}========================================${NC}"