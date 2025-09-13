#!/bin/bash

# 检查日志中的关键字段
# 这些字段会出现在日志中，即使中文被strip了

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}    检查二进制中的日志字段${NC}"
echo -e "${GREEN}========================================${NC}"

BINARY="release/slot-game-arm64/slot-game"

if [ ! -f "$BINARY" ]; then
    echo -e "${RED}未找到编译文件: $BINARY${NC}"
    exit 1
fi

echo -e "${YELLOW}检查关键日志字段...${NC}"
echo ""

# 这些是日志中的字段名，会在journalctl中显示
FIELDS=(
    "serial.acm.enabled"
    "acm_controller_exists"  
    "acm_connected"
    "configured_port"
    "auto_detect"
    "acm.enabled"
    "algo_timer_enabled"
)

FOUND=0
NOT_FOUND=0

for field in "${FIELDS[@]}"; do
    if strings "$BINARY" 2>/dev/null | grep -q "$field"; then
        echo -e "${GREEN}✅ 找到字段: $field${NC}"
        FOUND=$((FOUND + 1))
    else
        echo -e "${RED}❌ 未找到字段: $field${NC}"
        NOT_FOUND=$((NOT_FOUND + 1))
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"

if [ $NOT_FOUND -eq 0 ]; then
    echo -e "${GREEN}✅ 所有关键字段都存在！${NC}"
    echo -e "${GREEN}代码已更新，包含新的调试日志。${NC}"
    echo ""
    echo -e "${YELLOW}部署后在日志中查找这些字段：${NC}"
    echo -e "${YELLOW}sudo journalctl -u slot-game -f | grep -E 'serial.acm|acm_controller|acm_connected'${NC}"
else
    echo -e "${RED}⚠️ 缺少 $NOT_FOUND 个字段！${NC}"
    echo -e "${RED}代码可能不是最新的。${NC}"
fi

echo -e "${GREEN}========================================${NC}"