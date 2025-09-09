#!/bin/bash

# 测试新功能的脚本
# 确保服务器正在运行在 http://localhost:8080

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== 老虎机游戏新功能测试 ===${NC}"
echo ""

# 1. 测试用户登录
echo -e "${YELLOW}1. 测试用户登录...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}❌ 登录失败${NC}"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
else
  echo -e "${GREEN}✅ 登录成功${NC}"
  echo "Token: ${TOKEN:0:20}..."
fi
echo ""

# 2. 测试WebSocket连接
echo -e "${YELLOW}2. 测试WebSocket连接...${NC}"
# 注意：这里只是检查端点是否存在，实际WebSocket测试需要特殊工具
WS_CHECK=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Upgrade: websocket" \
  -H "Connection: Upgrade" \
  "$BASE_URL/ws/game")

if [ "$WS_CHECK" = "400" ] || [ "$WS_CHECK" = "426" ]; then
  echo -e "${GREEN}✅ WebSocket端点存在${NC}"
else
  echo -e "${RED}❌ WebSocket端点不可用 (HTTP $WS_CHECK)${NC}"
fi
echo ""

# 3. 测试获取余额
echo -e "${YELLOW}3. 测试获取钱包余额...${NC}"
BALANCE_RESPONSE=$(curl -s -X GET "$API_URL/wallet/balance" \
  -H "Authorization: Bearer $TOKEN")

if echo "$BALANCE_RESPONSE" | grep -q "balance"; then
  echo -e "${GREEN}✅ 获取余额成功${NC}"
  echo "Response: $BALANCE_RESPONSE"
else
  echo -e "${RED}❌ 获取余额失败${NC}"
  echo "Response: $BALANCE_RESPONSE"
fi
echo ""

# 4. 测试开始游戏
echo -e "${YELLOW}4. 测试开始游戏...${NC}"
START_RESPONSE=$(curl -s -X POST "$API_URL/slot/start" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bet_amount":100}')

SESSION_ID=$(echo $START_RESPONSE | grep -o '"session_id":"[^"]*' | cut -d'"' -f4)

if [ -z "$SESSION_ID" ]; then
  echo -e "${RED}❌ 开始游戏失败${NC}"
  echo "Response: $START_RESPONSE"
else
  echo -e "${GREEN}✅ 开始游戏成功${NC}"
  echo "Session ID: $SESSION_ID"
fi
echo ""

# 5. 测试单次转动
if [ ! -z "$SESSION_ID" ]; then
  echo -e "${YELLOW}5. 测试单次转动...${NC}"
  SPIN_RESPONSE=$(curl -s -X POST "$API_URL/slot/spin" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$SESSION_ID\"}")
  
  if echo "$SPIN_RESPONSE" | grep -q "result"; then
    echo -e "${GREEN}✅ 单次转动成功${NC}"
    echo "Response: ${SPIN_RESPONSE:0:100}..."
  else
    echo -e "${RED}❌ 单次转动失败${NC}"
    echo "Response: $SPIN_RESPONSE"
  fi
  echo ""
fi

# 6. 测试批量转动API
if [ ! -z "$SESSION_ID" ]; then
  echo -e "${YELLOW}6. 测试批量转动API...${NC}"
  BATCH_RESPONSE=$(curl -s -X POST "$API_URL/slot/batch-spin" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"session_id\":\"$SESSION_ID\",
      \"spin_count\":5,
      \"auto_stop\":false,
      \"stop_on_big_win\":false,
      \"big_win_amount\":5000
    }")
  
  if echo "$BATCH_RESPONSE" | grep -q "spin_results"; then
    echo -e "${GREEN}✅ 批量转动成功${NC}"
    TOTAL_SPINS=$(echo $BATCH_RESPONSE | grep -o '"total_spins":[0-9]*' | cut -d':' -f2)
    TOTAL_WIN=$(echo $BATCH_RESPONSE | grep -o '"total_win":[0-9]*' | cut -d':' -f2)
    echo "总转动次数: $TOTAL_SPINS"
    echo "总赢取: $TOTAL_WIN"
  else
    echo -e "${RED}❌ 批量转动失败${NC}"
    echo "Response: $BATCH_RESPONSE"
  fi
  echo ""
fi

# 7. 测试游戏结算
if [ ! -z "$SESSION_ID" ]; then
  echo -e "${YELLOW}7. 测试游戏结算...${NC}"
  SETTLE_RESPONSE=$(curl -s -X POST "$API_URL/slot/settle" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"session_id\":\"$SESSION_ID\"}")
  
  if echo "$SETTLE_RESPONSE" | grep -q "total_bet"; then
    echo -e "${GREEN}✅ 游戏结算成功${NC}"
    echo "Response: $SETTLE_RESPONSE"
  else
    echo -e "${RED}❌ 游戏结算失败${NC}"
    echo "Response: $SETTLE_RESPONSE"
  fi
  echo ""
fi

# 8. 测试Web界面
echo -e "${YELLOW}8. 测试Web界面访问...${NC}"
WEB_CHECK=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/static/index.html")

if [ "$WEB_CHECK" = "200" ]; then
  echo -e "${GREEN}✅ Web界面可访问${NC}"
  echo "URL: $BASE_URL/static/index.html"
else
  echo -e "${RED}❌ Web界面不可访问 (HTTP $WEB_CHECK)${NC}"
fi
echo ""

echo -e "${GREEN}=== 测试完成 ===${NC}"
echo ""
echo "提示："
echo "1. 可以通过浏览器访问 $BASE_URL/static/index.html 查看Web界面"
echo "2. WebSocket会在Web界面中自动连接并显示实时更新"
echo "3. 批量转动功能支持自动停止和大奖停止选项"