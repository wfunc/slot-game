#!/bin/bash

# 运行增强版本串口测试（包含Java协议支持）

echo "🚀 编译并运行增强版本..."
cd cmd/enhanced

go run main.go "$@"