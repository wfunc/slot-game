#!/bin/bash

echo "============================================"
echo "清理不需要提交的文件"
echo "============================================"
echo ""

# 确认操作
read -p "确定要清理这些文件吗？(y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "取消清理"
    exit 1
fi

echo ""
echo "开始清理..."

# 删除测试脚本
echo "1. 删除测试脚本..."
rm -fv test_*.sh quick_test.sh deploy_to_device.sh test_acm_debug.go

# 删除临时文档
echo ""
echo "2. 删除临时修复文档..."
rm -fv docs/*_FIX*.md docs/GIT_COMMIT_GUIDE.md

# 删除补丁文件
echo ""
echo "3. 删除补丁文件..."
rm -fv internal/hardware/reconnect_patch.go

# 删除同步脚本
echo ""
echo "4. 删除同步脚本..."
rm -fv sync_proto.sh

echo ""
echo "============================================"
echo "清理完成！"
echo "============================================"
echo ""
echo "现在运行 'git status' 查看状态："
echo ""
git status --short
