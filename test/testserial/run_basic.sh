#!/bin/bash

# 运行基础版本串口测试

echo "🚀 编译并运行基础版本..."
cd cmd/basic

go run main.go "$@"