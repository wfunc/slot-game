#!/bin/bash

echo "🔄 重启游戏服务器..."

# 停止现有服务器
pkill -f animal-game

# 编译
echo "🔨 编译服务器..."
go build -o animal-game cmd/server/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

# 启动服务器
echo "🚀 启动服务器..."
./animal-game &
SERVER_PID=$!

echo "✅ 服务器已启动 PID: $SERVER_PID"
echo ""
echo "📝 查看日志: tail -f server.log | grep 1803"
echo "🛑 停止服务器: kill $SERVER_PID"
echo ""
echo "💡 测试步骤："
echo "1. 前端点击发射按钮（发送1815）"
echo "2. 子弹击中动物（发送1803）"
echo "3. 观察是否还有报错"