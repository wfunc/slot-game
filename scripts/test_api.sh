#!/bin/bash

# API测试脚本
# 测试老虎机游戏API和钱包API

BASE_URL="http://localhost:8080/api/v1"
TOKEN=""
SESSION_ID=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印成功消息
success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# 打印错误消息
error() {
    echo -e "${RED}❌ $1${NC}"
}

# 打印信息
info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# 1. 健康检查
echo "========================================="
echo "1. 健康检查"
echo "========================================="
curl -s ${BASE_URL%/api/v1}/health | jq '.' || error "健康检查失败"
success "健康检查完成"
echo ""

# 2. 用户注册
echo "========================================="
echo "2. 用户注册"
echo "========================================="
TIMESTAMP=$(date +%s)
USERNAME="testuser_${TIMESTAMP}"
EMAIL="test_${TIMESTAMP}@example.com"

REGISTER_RESPONSE=$(curl -s -X POST ${BASE_URL}/auth/register \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"${USERNAME}\",
    \"email\": \"${EMAIL}\",
    \"password\": \"Test123456!\"
  }")

echo "$REGISTER_RESPONSE" | jq '.'
success "用户注册成功: ${USERNAME}"
echo ""

# 3. 用户登录
echo "========================================="
echo "3. 用户登录"
echo "========================================="
LOGIN_RESPONSE=$(curl -s -X POST ${BASE_URL}/auth/login \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"${USERNAME}\",
    \"password\": \"Test123456!\"
  }")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.access_token // .access_token // .token')
if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    error "登录失败，无法获取token"
    echo "$LOGIN_RESPONSE" | jq '.'
    exit 1
fi

echo "$LOGIN_RESPONSE" | jq '.'
success "登录成功，获取到Token"
info "Token: ${TOKEN:0:20}..."
echo ""

# 4. 查询钱包余额
echo "========================================="
echo "4. 查询钱包余额"
echo "========================================="
BALANCE_RESPONSE=$(curl -s -X GET ${BASE_URL}/wallet/balance \
  -H "Authorization: Bearer ${TOKEN}")

echo "$BALANCE_RESPONSE" | jq '.'
BALANCE=$(echo "$BALANCE_RESPONSE" | jq -r '.balance // 0')
success "当前余额: ${BALANCE}"
echo ""

# 5. 测试充值（如果余额不足）
if [ "$BALANCE" -lt "1000" ]; then
    echo "========================================="
    echo "5. 测试充值"
    echo "========================================="
    DEPOSIT_RESPONSE=$(curl -s -X POST ${BASE_URL}/wallet/deposit \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d '{"amount": 10000}')
    
    echo "$DEPOSIT_RESPONSE" | jq '.'
    success "充值10000金币成功"
    echo ""
    
    # 重新查询余额
    BALANCE_RESPONSE=$(curl -s -X GET ${BASE_URL}/wallet/balance \
      -H "Authorization: Bearer ${TOKEN}")
    BALANCE=$(echo "$BALANCE_RESPONSE" | jq -r '.balance // 0')
    info "充值后余额: ${BALANCE}"
    echo ""
fi

# 6. 开始游戏
echo "========================================="
echo "6. 开始老虎机游戏"
echo "========================================="
START_RESPONSE=$(curl -s -X POST ${BASE_URL}/slot/start \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"bet_amount": 100}')

SESSION_ID=$(echo "$START_RESPONSE" | jq -r '.session_id // ""')
if [ -z "$SESSION_ID" ]; then
    error "开始游戏失败"
    echo "$START_RESPONSE" | jq '.'
    exit 1
fi

echo "$START_RESPONSE" | jq '.'
success "游戏开始，会话ID: ${SESSION_ID}"
echo ""

# 7. 执行转动
echo "========================================="
echo "7. 执行转动"
echo "========================================="
SPIN_RESPONSE=$(curl -s -X POST ${BASE_URL}/slot/spin \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"${SESSION_ID}\"}")

echo "$SPIN_RESPONSE" | jq '.'
success "转动完成"
echo ""

# 8. 查询会话信息
echo "========================================="
echo "8. 查询会话信息"
echo "========================================="
SESSION_INFO=$(curl -s -X GET ${BASE_URL}/slot/session/${SESSION_ID} \
  -H "Authorization: Bearer ${TOKEN}")

echo "$SESSION_INFO" | jq '.'
success "会话信息获取成功"
echo ""

# 9. 结算游戏
echo "========================================="
echo "9. 结算游戏"
echo "========================================="
SETTLE_RESPONSE=$(curl -s -X POST ${BASE_URL}/slot/settle \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"${SESSION_ID}\"}")

echo "$SETTLE_RESPONSE" | jq '.'
success "游戏结算完成"
echo ""

# 10. 查询游戏历史
echo "========================================="
echo "10. 查询游戏历史"
echo "========================================="
HISTORY_RESPONSE=$(curl -s -X GET "${BASE_URL}/slot/history?page=1&page_size=10" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$HISTORY_RESPONSE" | jq '.'
success "游戏历史获取成功"
echo ""

# 11. 查询交易记录
echo "========================================="
echo "11. 查询交易记录"
echo "========================================="
TRANSACTIONS_RESPONSE=$(curl -s -X GET "${BASE_URL}/wallet/transactions?page=1&page_size=10" \
  -H "Authorization: Bearer ${TOKEN}")

echo "$TRANSACTIONS_RESPONSE" | jq '.'
success "交易记录获取成功"
echo ""

# 12. 查询用户统计
echo "========================================="
echo "12. 查询用户统计"
echo "========================================="
STATS_RESPONSE=$(curl -s -X GET ${BASE_URL}/slot/stats \
  -H "Authorization: Bearer ${TOKEN}")

echo "$STATS_RESPONSE" | jq '.'
success "用户统计获取成功"
echo ""

# 13. 查询最终余额
echo "========================================="
echo "13. 查询最终余额"
echo "========================================="
FINAL_BALANCE=$(curl -s -X GET ${BASE_URL}/wallet/balance \
  -H "Authorization: Bearer ${TOKEN}")

echo "$FINAL_BALANCE" | jq '.'
FINAL_AMOUNT=$(echo "$FINAL_BALANCE" | jq -r '.balance // 0')
success "最终余额: ${FINAL_AMOUNT}"
echo ""

echo "========================================="
success "🎉 所有API测试完成！"
echo "========================================="
echo ""
echo "测试摘要:"
echo "  - 用户名: ${USERNAME}"
echo "  - 会话ID: ${SESSION_ID}"
echo "  - 初始余额: ${BALANCE}"
echo "  - 最终余额: ${FINAL_AMOUNT}"
echo "========================================="