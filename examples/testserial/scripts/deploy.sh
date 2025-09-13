#!/bin/bash

# 部署脚本 - 编译并传输到Ubuntu设备

echo "==================================="
echo "    串口测试程序部署脚本"
echo "==================================="
echo ""

# 默认配置
DEFAULT_HOST=""
DEFAULT_USER="ubuntu"
DEFAULT_PATH="/home/ubuntu/serial_test"

# 读取参数
if [ $# -lt 1 ]; then
    echo "用法: ./deploy.sh <host> [user] [remote_path]"
    echo "示例: ./deploy.sh 192.168.1.100"
    echo "      ./deploy.sh 192.168.1.100 ubuntu /home/ubuntu/test"
    exit 1
fi

HOST=$1
USER=${2:-$DEFAULT_USER}
REMOTE_PATH=${3:-$DEFAULT_PATH}

echo "目标设备: $USER@$HOST"
echo "部署路径: $REMOTE_PATH"
echo ""

# 1. 编译程序
echo "1. 编译Linux版本..."
make clean
make linux

if [ ! -f "serial_tester_linux_amd64" ]; then
    echo "✗ 编译失败"
    exit 1
fi
echo "✓ 编译成功"
echo ""

# 2. 创建部署包
echo "2. 创建部署包..."
mkdir -p deploy_temp
cp serial_tester_linux_amd64 deploy_temp/serial_tester
cp test_serial.sh deploy_temp/
chmod +x deploy_temp/*

# 创建测试脚本
cat > deploy_temp/test_serial.sh << 'EOF'
#!/bin/bash

echo "==================================="
echo "    /dev/ttyS3 串口测试"
echo "==================================="
echo ""

# 检查串口设备
if [ ! -e "/dev/ttyS3" ]; then
    echo "✗ 错误: /dev/ttyS3 不存在"
    echo ""
    echo "可用的串口设备:"
    ls /dev/tty* | grep -E "(ttyS|ttyUSB|ttyACM)"
    exit 1
fi

echo "✓ 检测到 /dev/ttyS3"
echo ""

# 显示串口信息
echo "串口信息:"
stty -F /dev/ttyS3 -a 2>/dev/null | head -1
echo ""

# 选择测试
echo "请选择测试模式:"
echo "1. 发送测试 (持续发送)"
echo "2. 接收测试 (等待接收)"
echo "3. 回显测试 (接收后发回)"
echo "4. 交互模式 (手动收发)"
echo "5. 快速测试 (发送10条消息)"
echo ""
read -p "选择 (1-5): " choice

case $choice in
    1)
        echo "启动发送测试..."
        sudo ./serial_tester -m send -i 1000 -msg "Hello_from_Ubuntu"
        ;;
    2)
        echo "启动接收测试..."
        sudo ./serial_tester -m recv -v
        ;;
    3)
        echo "启动回显测试..."
        sudo ./serial_tester -m echo
        ;;
    4)
        echo "启动交互模式..."
        sudo ./serial_tester -m both
        ;;
    5)
        echo "快速测试..."
        sudo ./serial_tester -m send -i 500 -msg "QUICK_TEST" &
        PID=$!
        sleep 6
        kill $PID 2>/dev/null
        echo "测试完成"
        ;;
    *)
        echo "无效选择"
        exit 1
        ;;
esac
EOF

chmod +x deploy_temp/test_serial.sh
echo "✓ 部署包准备完成"
echo ""

# 3. 传输文件
echo "3. 传输文件到目标设备..."
ssh $USER@$HOST "mkdir -p $REMOTE_PATH"
scp deploy_temp/* $USER@$HOST:$REMOTE_PATH/

if [ $? -eq 0 ]; then
    echo "✓ 文件传输成功"
else
    echo "✗ 文件传输失败"
    exit 1
fi
echo ""

# 4. 清理临时文件
rm -rf deploy_temp
echo "✓ 清理临时文件"
echo ""

# 5. 显示使用说明
echo "==================================="
echo "    部署完成！"
echo "==================================="
echo ""
echo "在目标设备上运行:"
echo "  ssh $USER@$HOST"
echo "  cd $REMOTE_PATH"
echo "  ./test_serial.sh          # 运行测试脚本"
echo ""
echo "或直接使用:"
echo "  sudo ./serial_tester -h    # 查看帮助"
echo "  sudo ./serial_tester -m both  # 交互模式"
echo "  sudo ./serial_tester -m send -msg TEST  # 发送模式"
echo "  sudo ./serial_tester -m recv  # 接收模式"
echo ""
echo "测试参数:"
echo "  -d /dev/ttyS3   # 串口设备"
echo "  -b 115200       # 波特率"
echo "  -s 2            # 停止位"
echo "  -v              # 详细输出"
echo ""