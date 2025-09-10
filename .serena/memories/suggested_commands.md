# 项目开发常用命令

## 构建和运行
```bash
# 构建项目
make build

# 运行项目
make run

# 开发模式运行(自动重载)
make dev

# 构建所有平台版本
make build-all

# 构建ARM64版本(Ubuntu)
make build-arm64
```

## 测试相关
```bash
# 运行所有测试
make test

# 运行测试(过滤链接器警告)
make test-quiet

# 运行仓储层测试
make test-repo

# 清理测试缓存
make test-clean

# 生成测试覆盖率报告
make coverage
```

## 代码质量
```bash
# 格式化代码
make fmt

# 代码检查(需要golangci-lint)
make lint

# 清理构建文件
make clean
```

## 依赖管理
```bash
# 安装依赖
make deps

# 更新依赖
make update

# 查看依赖
go mod tidy
go mod download
```

## 初始化和检查
```bash
# 初始化项目目录
make init

# 检查开发环境
make check

# 显示版本信息
make version

# 显示帮助
make help
```

## Docker相关
```bash
# 构建Docker镜像
make docker-build

# 运行Docker容器
make docker-run
```

## 开发工具
```bash
# 生成API文档(需要swag)
make docs

# 直接运行Go程序
go run cmd/server/main.go -config=config/config.yaml
```

## 系统工具(Darwin)
```bash
# 查看文件
ls -la
find . -name "*.go" -type f
grep -r "pattern" .

# Git操作
git status
git log --oneline -10
git diff

# 进程管理
ps aux | grep slot-game
kill -TERM <pid>
```