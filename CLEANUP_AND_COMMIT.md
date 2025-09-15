# Git清理和提交建议

## 不需要提交的文件（应该删除或忽略）

### 1. 测试脚本（本地测试用）
```bash
# 这些是临时测试脚本，不需要提交
rm -f test_*.sh
rm -f quick_test.sh
rm -f deploy_to_device.sh
rm -f test_acm_debug.go
```

### 2. 临时修复文档
```bash
# 这些是调试过程中的文档，保留一个总结文档即可
rm -f docs/*_FIX*.md
rm -f docs/GIT_COMMIT_GUIDE.md
```

### 3. 临时补丁文件
```bash
rm -f internal/hardware/reconnect_patch.go
```

### 4. 本地同步脚本
```bash
rm -f sync_proto.sh
```

### 5. 已删除的测试文件
```bash
# 这些已经被删除，可以从git中移除
git rm test_binary test_build examples/test_acm_timer.go
```

## 需要保留并提交的重要文件

### 核心功能文件
- ✅ `internal/hardware/serial_reconnect.go` - 串口重连管理器
- ✅ `internal/hardware/acm_controller.go` - ACM控制器（已修复）
- ✅ `internal/database/migration.go` - 数据库迁移（添加了game_states表）
- ✅ `cmd/server/main.go` - 主程序（修复了关闭流程）

### Protobuf相关（新功能）
- ✅ `proto/` - Protobuf定义文件
- ✅ `internal/pb/` - 生成的Protobuf Go代码
- ✅ `internal/api/protobuf_websocket_handler.go` - Protobuf WebSocket处理
- ✅ `internal/websocket/protobuf_codec.go` - Protobuf编解码
- ✅ `internal/websocket/slot_handler.go` - Slot游戏处理

### 重要文档（应该保留）
- ✅ `docs/SLOT_PROTOBUF_TESTING.md` - Protobuf测试文档
- ✅ `docs/WEBSOCKET_PROTOBUF_GUIDE.md` - WebSocket Protobuf指南

### 其他重要文件
- ✅ `internal/repository/jackpot.go` - JP奖池功能
- ✅ `scripts/setup_protobuf.sh` - Protobuf安装脚本（可能需要保留）

## 建议的提交顺序

### 1. 清理不需要的文件
```bash
# 删除测试脚本
rm -f test_*.sh quick_test.sh deploy_to_device.sh test_acm_debug.go

# 删除临时文档
rm -f docs/*_FIX*.md docs/GIT_COMMIT_GUIDE.md

# 删除补丁文件
rm -f internal/hardware/reconnect_patch.go

# 删除同步脚本
rm -f sync_proto.sh

# 从git中移除已删除的文件
git rm test_binary test_build examples/test_acm_timer.go
```

### 2. 提交核心修复
```bash
git add internal/hardware/serial_reconnect.go
git add internal/hardware/acm_controller.go
git add internal/database/migration.go
git add cmd/server/main.go
git commit -m "fix: 修复ACM设备重连、Algo定时器和程序关闭问题

- 添加串口自动重连管理器
- 修复ACM控制器死锁和Algo定时器问题
- 修复程序关闭时挂起问题
- 添加game_states表到数据库迁移
- 改进JSON消息处理，避免误识别"
```

### 3. 提交Protobuf功能
```bash
git add proto/
git add internal/pb/
git add internal/api/protobuf_websocket_handler.go
git add internal/websocket/protobuf_codec.go
git add internal/websocket/slot_handler.go
git add docs/SLOT_PROTOBUF_TESTING.md
git add docs/WEBSOCKET_PROTOBUF_GUIDE.md
git commit -m "feat: 添加Protobuf支持和WebSocket二进制通信

- 添加slot和animal protobuf定义
- 实现WebSocket Protobuf处理器
- 添加Protobuf编解码器
- 完善测试文档"
```

### 4. 提交其他功能改进
```bash
git add internal/repository/jackpot.go
git add internal/repository/wallet.go
git add internal/models/game.go
git add internal/models/transaction.go
git add internal/api/router.go
git add static/websocket-test.html
git add Makefile sg.sh go.mod
git commit -m "feat: 添加JP奖池功能和其他改进

- 实现JP奖池repository
- 更新钱包和交易模型
- 改进WebSocket测试页面
- 更新构建脚本"
```

### 5. 更新.gitignore
```bash
git add .gitignore
git commit -m "chore: 更新.gitignore，忽略测试脚本和临时文件"
```

## 执行清理脚本
```bash
#!/bin/bash
# cleanup.sh

echo "开始清理不需要的文件..."

# 删除测试脚本
echo "删除测试脚本..."
rm -f test_*.sh quick_test.sh deploy_to_device.sh test_acm_debug.go

# 删除临时文档
echo "删除临时修复文档..."
rm -f docs/*_FIX*.md docs/GIT_COMMIT_GUIDE.md

# 删除补丁文件
echo "删除补丁文件..."
rm -f internal/hardware/reconnect_patch.go

# 删除同步脚本
echo "删除同步脚本..."
rm -f sync_proto.sh

echo "清理完成！"
echo ""
echo "现在可以运行 git status 查看状态"
```

## 注意事项
1. 在提交前先运行 `git status` 确认文件状态
2. 使用 `git diff` 查看具体修改内容
3. 考虑是否需要保留 `scripts/setup_protobuf.sh` 供其他开发者使用
4. 确保重要的文档（如Protobuf使用指南）被保留