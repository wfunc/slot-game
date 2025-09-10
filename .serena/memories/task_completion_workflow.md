# 任务完成工作流程

## 代码开发完成后必须执行的步骤

### 1. 代码质量检查
```bash
# 格式化代码
make fmt

# 运行代码检查
make lint
```

### 2. 测试验证
```bash
# 运行所有测试
make test

# 检查测试覆盖率
make coverage

# 运行特定模块测试
make test-repo  # 仓储层测试
```

### 3. 构建验证
```bash
# 构建项目
make build

# 验证构建成功
make run
```

### 4. 功能验证
```bash
# 运行功能测试脚本
./test_features.sh

# 测试WebSocket连接
./test-websocket.sh
```

### 5. 清理工作
```bash
# 清理临时文件
make clean

# 清理测试缓存
make test-clean
```

## Git工作流程
1. 确保在feature分支工作
2. 提交前先pull最新代码
3. 运行完整测试套件
4. 写清晰的commit message
5. 推送代码并创建PR

## 部署前检查清单
- [ ] 所有测试通过
- [ ] 代码覆盖率达标
- [ ] Lint检查通过
- [ ] 配置文件正确
- [ ] 依赖项完整
- [ ] 文档更新