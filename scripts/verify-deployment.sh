#!/bin/bash

# 验证部署版本脚本
# 用于确认部署的代码是否包含最新的调试日志

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}    验证 Slot Game 部署版本${NC}"
echo -e "${GREEN}========================================${NC}"

# 检查编译后的二进制文件是否包含新的日志字符串
echo -e "\n${YELLOW}检查二进制文件中的调试日志标记...${NC}"

# 检查关键字符串
KEYWORDS=(
    "串口配置详情"
    "检查ACM配置"
    "控制器状态检查"
    "尝试连接ACM"
    "ACM控制器创建成功"
    "ACM控制器为nil"
)

BINARY="release/slot-game-arm64/slot-game"

if [ ! -f "$BINARY" ]; then
    echo -e "${RED}未找到编译文件: $BINARY${NC}"
    echo -e "${YELLOW}请先运行: make build-arm64${NC}"
    exit 1
fi

echo -e "${GREEN}检查文件: $BINARY${NC}"
echo ""

FOUND=0
NOT_FOUND=0

for keyword in "${KEYWORDS[@]}"; do
    if strings "$BINARY" 2>/dev/null | grep -q "$keyword"; then
        echo -e "${GREEN}✅ 找到: $keyword${NC}"
        FOUND=$((FOUND + 1))
    else
        echo -e "${RED}❌ 未找到: $keyword${NC}"
        NOT_FOUND=$((NOT_FOUND + 1))
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"

if [ $NOT_FOUND -eq 0 ]; then
    echo -e "${GREEN}✅ 所有调试日志都存在！代码是最新的。${NC}"
    echo -e "${YELLOW}请部署这个版本到目标设备。${NC}"
else
    echo -e "${RED}⚠️ 缺少 $NOT_FOUND 个调试日志！${NC}"
    echo -e "${RED}代码可能不是最新的，请重新编译：${NC}"
    echo -e "${YELLOW}  1. git pull (确保代码最新)${NC}"
    echo -e "${YELLOW}  2. make clean${NC}"
    echo -e "${YELLOW}  3. make build-arm64${NC}"
fi

echo -e "${GREEN}========================================${NC}"

# 显示二进制文件信息
echo -e "\n${YELLOW}二进制文件信息：${NC}"
ls -lh "$BINARY" 2>/dev/null
file "$BINARY" 2>/dev/null

# 检查最后修改时间
echo -e "\n${YELLOW}编译时间：${NC}"
stat -c "最后修改: %y" "$BINARY" 2>/dev/null || stat -f "最后修改: %Sm" "$BINARY" 2>/dev/null

echo ""