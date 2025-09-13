#!/bin/bash

# STM32集成测试套件
# 用于系统化测试串口通信

echo "========================================="
echo "    STM32集成测试套件 v1.0"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果
PASSED=0
FAILED=0

# 函数：运行测试
run_test() {
    local test_name=$1
    local command=$2
    local expected=$3
    
    echo -n "测试: $test_name ... "
    
    result=$(eval $command 2>&1)
    
    if [[ "$result" == *"$expected"* ]]; then
        echo -e "${GREEN}✓ 通过${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ 失败${NC}"
        echo "  期望: $expected"
        echo "  实际: $result"
        ((FAILED++))
    fi
}

# 函数：检查程序
check_program() {
    local program=$1
    if [ ! -f "$program" ]; then
        echo -e "${RED}错误: $program 不存在${NC}"
        echo "请先编译: go build $program.go"
        return 1
    fi
    return 0
}

echo "【环境检查】"
echo "----------------------------------------"

# 检查必要的程序
programs=(
    "serial_tester"
    "protocol_tester"
    "serial_diagnose"
    "hardware_diagnose"
    "stm32_protocol"
)

all_exist=true
for prog in "${programs[@]}"; do
    if check_program "./$prog"; then
        echo "✓ $prog 存在"
    else
        all_exist=false
    fi
done

if [ "$all_exist" = false ]; then
    echo ""
    echo "编译所有程序..."
    make build
fi

echo ""
echo "【1. 设备检测测试】"
echo "----------------------------------------"

# 测试设备检测
run_test "检测串口设备" "ls /dev/tty* | grep -E 'ttyS3|ttyACM0' | wc -l" "0"

# 测试权限
if [ -e /dev/ttyACM0 ]; then
    run_test "ACM设备权限" "test -r /dev/ttyACM0 && echo 'readable'" "readable"
fi

if [ -e /dev/ttyS3 ]; then
    run_test "ttyS3设备权限" "test -r /dev/ttyS3 && echo 'readable'" "readable"
fi

echo ""
echo "【2. ACM设备测试】"
echo "----------------------------------------"

if [ -e /dev/ttyACM0 ]; then
    # 测试ACM基础连接
    timeout 2 ./serial_tester -d /dev/ttyACM0 -m send -msg "ver" > /tmp/acm_test.log 2>&1
    if grep -q "已发送" /tmp/acm_test.log; then
        echo -e "${GREEN}✓ ACM连接成功${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ ACM连接失败${NC}"
        ((FAILED++))
    fi
    
    # 测试ACM命令
    echo "测试ACM命令协议..."
    commands=("ver" "sta" "help")
    for cmd in "${commands[@]}"; do
        timeout 2 ./protocol_tester -d /dev/ttyACM0 -m custom -n 1 > /tmp/acm_cmd_$cmd.log 2>&1
        if grep -q "响应" /tmp/acm_cmd_$cmd.log; then
            echo "  $cmd命令: ✓"
        else
            echo "  $cmd命令: ✗"
        fi
    done
else
    echo -e "${YELLOW}⚠ ACM设备不存在，跳过测试${NC}"
fi

echo ""
echo "【3. STM32设备测试】"
echo "----------------------------------------"

if [ -e /dev/ttyS3 ]; then
    # 测试STM32基础连接
    timeout 2 ./serial_tester -d /dev/ttyS3 -m send -msg '{"MsgType":"M4"}' > /tmp/stm32_test.log 2>&1
    if grep -q "已发送" /tmp/stm32_test.log; then
        echo -e "${GREEN}✓ STM32端口打开成功${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ STM32端口打开失败${NC}"
        ((FAILED++))
    fi
    
    # 测试JSON协议
    echo "测试JSON消息协议..."
    timeout 2 ./protocol_tester -d /dev/ttyS3 -m version -n 1 > /tmp/stm32_json.log 2>&1
    if grep -q "发送 版本查询" /tmp/stm32_json.log; then
        echo "  JSON发送: ✓"
        if grep -q "接收" /tmp/stm32_json.log; then
            echo "  JSON响应: ✓"
        else
            echo "  JSON响应: ✗ (设备可能未连接)"
        fi
    else
        echo "  JSON发送: ✗"
    fi
else
    echo -e "${YELLOW}⚠ STM32设备不存在，跳过测试${NC}"
fi

echo ""
echo "【4. 协议兼容性测试】"
echo "----------------------------------------"

# 测试消息格式
echo "验证消息格式..."

# M1消息格式
msg1='{"MsgType":"M1","data":{"cfgData":{"hp30":1}}}'
echo -n "  M1消息格式: "
if echo "$msg1" | jq . > /dev/null 2>&1; then
    echo "✓ 有效JSON"
else
    echo "✗ 无效JSON"
fi

# M2消息格式
msg2='{"MsgType":"M2","idex":1000,"data":{"algo":"test"}}'
echo -n "  M2消息格式: "
if echo "$msg2" | jq . > /dev/null 2>&1; then
    echo "✓ 有效JSON"
else
    echo "✗ 无效JSON"
fi

echo ""
echo "【5. 性能测试】"
echo "----------------------------------------"

if [ -e /dev/ttyACM0 ]; then
    echo "测试ACM响应时间..."
    start_time=$(date +%s%N)
    timeout 1 ./serial_tester -d /dev/ttyACM0 -m send -msg "ver" > /dev/null 2>&1
    end_time=$(date +%s%N)
    elapsed=$((($end_time - $start_time) / 1000000))
    echo "  响应时间: ${elapsed}ms"
    
    if [ $elapsed -lt 100 ]; then
        echo -e "  ${GREEN}✓ 响应速度良好${NC}"
    elif [ $elapsed -lt 500 ]; then
        echo -e "  ${YELLOW}⚠ 响应速度一般${NC}"
    else
        echo -e "  ${RED}✗ 响应速度慢${NC}"
    fi
fi

echo ""
echo "【6. 诊断工具测试】"
echo "----------------------------------------"

# 运行完整诊断
echo "运行硬件诊断..."
timeout 5 ./hardware_diagnose -v -test all > /tmp/diagnose.log 2>&1

if grep -q "诊断报告" /tmp/diagnose.log; then
    echo -e "${GREEN}✓ 诊断工具运行正常${NC}"
    
    # 提取关键信息
    echo "诊断结果摘要:"
    grep -E "测试总数|成功率|ACM设备|STM32" /tmp/diagnose.log | head -5
else
    echo -e "${RED}✗ 诊断工具运行失败${NC}"
fi

echo ""
echo "========================================="
echo "           测试报告"
echo "========================================="
echo ""
echo -e "通过测试: ${GREEN}$PASSED${NC}"
echo -e "失败测试: ${RED}$FAILED${NC}"
total=$((PASSED + FAILED))
if [ $total -gt 0 ]; then
    success_rate=$((PASSED * 100 / total))
    echo "成功率: ${success_rate}%"
fi

echo ""
echo "【建议】"
echo "----------------------------------------"

# 根据测试结果给出建议
if [ ! -e /dev/ttyACM0 ] && [ ! -e /dev/ttyS3 ]; then
    echo "❌ 未检测到任何串口设备"
    echo "  1. 检查硬件连接"
    echo "  2. 检查USB线缆"
    echo "  3. 运行 dmesg | grep tty 查看系统日志"
elif [ -e /dev/ttyACM0 ] && [ ! -e /dev/ttyS3 ]; then
    echo "⚠️  只检测到ACM设备"
    echo "  1. STM32可能未连接"
    echo "  2. 检查STM32电源和接线"
    echo "  3. 验证STM32固件是否正确烧录"
elif [ ! -e /dev/ttyACM0 ] && [ -e /dev/ttyS3 ]; then
    echo "⚠️  只检测到ttyS3设备"
    echo "  1. ACM设备可能未连接"
    echo "  2. 检查USB连接"
    echo "  3. 可能需要安装驱动"
else
    echo "✓ 两个设备都已检测到"
    
    # 检查日志中的响应情况
    if grep -q "JSON响应: ✗" /tmp/stm32_json.log 2>/dev/null; then
        echo ""
        echo "但STM32设备无响应，建议:"
        echo "  1. 检查STM32是否正在运行"
        echo "  2. 验证波特率设置(115200)"
        echo "  3. 检查TX/RX接线是否正确"
        echo "  4. 运行回环测试: ./serial_diagnose -d /dev/ttyS3 -loopback"
    fi
fi

echo ""
echo "详细日志保存在 /tmp/ 目录"
echo "查看完整诊断: ./hardware_diagnose -v -test all"
echo ""