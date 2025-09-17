#!/bin/bash

# 疯狂动物园游戏启动脚本

echo "🎮 启动疯狂动物园游戏服务器..."

# 切换到项目根目录
cd "$(dirname "$0")/.."

# 检查是否需要安装依赖
if [ ! -d "vendor" ]; then
    echo "📦 安装依赖..."
    go mod tidy
    go mod vendor
fi

# 生成 protobuf 代码
echo "🔧 生成 Protobuf 代码..."
protoc --go_out=internal/pb --go_opt=paths=source_relative \
    proto/animal.proto proto/bridge.proto proto/cfg.proto proto/slot.proto

# 运行测试
echo "🧪 运行测试..."
go test ./internal/game/animal/... -v

# 编译
echo "🔨 编译服务器..."
go build -o bin/animal-server cmd/server/main.go

# 启动服务器
echo "🚀 启动服务器..."
./bin/animal-server

echo "✅ 服务器已启动，访问 http://localhost:8080"