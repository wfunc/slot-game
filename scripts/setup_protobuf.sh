#!/bin/bash

echo "安装 Protocol Buffers 编译器和 Go 插件..."

# 安装 protoc 编译器 (如果还没安装)
if ! command -v protoc &> /dev/null; then
    echo "安装 protoc..."
    # macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install protobuf
    # Linux
    else
        sudo apt-get update
        sudo apt-get install -y protobuf-compiler
    fi
fi

# 安装 Go protobuf 插件
echo "安装 Go protobuf 插件..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 确保 GOPATH/bin 在 PATH 中
export PATH="$PATH:$(go env GOPATH)/bin"

echo "✅ Protocol Buffers 环境设置完成"
echo ""
echo "使用方法："
echo "1. 编译 proto 文件: make proto"
echo "2. proto 文件位置: proto/*.proto"
echo "3. 生成的 Go 代码: internal/pb/*.pb.go"