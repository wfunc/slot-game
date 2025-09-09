#!/bin/bash

# 构建两个版本的可执行文件

echo "🔧 构建串口测试程序..."
echo ""

echo "📦 构建基础版本..."
cd cmd/basic
go build -o ../../bin/serial_basic
if [ $? -eq 0 ]; then
    echo "✅ 基础版本构建成功: bin/serial_basic"
else
    echo "❌ 基础版本构建失败"
    exit 1
fi

echo ""
echo "📦 构建增强版本..."
cd ../enhanced
go build -o ../../bin/serial_enhanced
if [ $? -eq 0 ]; then
    echo "✅ 增强版本构建成功: bin/serial_enhanced"
else
    echo "❌ 增强版本构建失败"
    exit 1
fi

echo ""
echo "✨ 所有版本构建完成！"
echo ""
echo "运行方式："
echo "  基础版本: ./bin/serial_basic"
echo "  增强版本: ./bin/serial_enhanced"
echo "  或使用脚本: ./run_basic.sh 或 ./run_enhanced.sh"