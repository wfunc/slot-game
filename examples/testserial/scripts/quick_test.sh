#!/bin/bash

# 快速协议测试脚本
echo "==================================="
echo "    串口协议快速测试"
echo "==================================="
echo ""

# 检查程序文件
if [ ! -f "./protocol_tester_linux_arm" ]; then
    echo "❌ 错误: protocol_tester_linux_arm 不存在"
    echo "请先编译: make linux-arm"
    exit 1
fi

echo "请选择测试:"
echo "1. 算法协议测试 (M1/M2消息)"
echo "2. MQTT协议测试 (M6消息)"
echo "3. 版本协议测试 (M4消息)"
echo "4. 更新协议测试 (M3/M5消息)"
echo "5. 连续发送测试"
echo "6. 自定义JSON测试"
echo ""
read -p "选择 (1-6): " choice

case $choice in
    1)
        echo "【算法协议测试】"
        echo "发送M2算法请求消息..."
        sudo ./protocol_tester_linux_arm -m algo -n 3 -delay 1000
        ;;
    2)
        echo "【MQTT协议测试】"
        echo "发送M6 MQTT消息..."
        sudo ./protocol_tester_linux_arm -m mqtt
        ;;
    3)
        echo "【版本协议测试】"
        echo "发送M4版本查询消息..."
        sudo ./protocol_tester_linux_arm -m version
        ;;
    4)
        echo "【更新协议测试】"
        echo "发送M3/M5更新消息..."
        sudo ./protocol_tester_linux_arm -m update
        ;;
    5)
        echo "【连续发送测试】"
        read -p "发送次数: " count
        sudo ./protocol_tester_linux_arm -m algo -n $count -delay 500
        ;;
    6)
        echo "【自定义测试】"
        sudo ./protocol_tester_linux_arm -m custom
        ;;
    *)
        echo "无效选择"
        exit 1
        ;;
esac