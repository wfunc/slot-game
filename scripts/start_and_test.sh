#!/bin/bash

# 启动服务器并测试API

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 打印消息函数
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 清理函数
cleanup() {
    info "停止服务器..."
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null
        wait $SERVER_PID 2>/dev/null
    fi
    success "服务器已停止"
}

# 设置退出时清理
trap cleanup EXIT

# 1. 构建服务器
echo "========================================="
info "构建服务器..."
echo "========================================="
make build
if [ $? -ne 0 ]; then
    error "构建失败"
    exit 1
fi
success "构建成功"
echo ""

# 2. 启动服务器
echo "========================================="
info "启动服务器..."
echo "========================================="
./bin/server &
SERVER_PID=$!

# 等待服务器启动
info "等待服务器启动..."
sleep 3

# 检查服务器是否运行
if ! kill -0 $SERVER_PID 2>/dev/null; then
    error "服务器启动失败"
    exit 1
fi

# 检查健康状态
for i in {1..10}; do
    if curl -s http://localhost:8080/health | grep -q "healthy"; then
        success "服务器已启动并运行正常"
        break
    fi
    if [ $i -eq 10 ]; then
        error "服务器健康检查失败"
        exit 1
    fi
    sleep 1
done
echo ""

# 3. 运行API测试
echo "========================================="
info "运行API测试..."
echo "========================================="
sleep 2
./scripts/test_api.sh

# 测试完成
echo ""
echo "========================================="
success "测试完成！"
echo "========================================="
info "服务器仍在运行中..."
info "访问 http://localhost:8080/health 检查健康状态"
info "使用 Ctrl+C 停止服务器"
echo ""

# 保持服务器运行
wait $SERVER_PID