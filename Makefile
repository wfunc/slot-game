# 推币机游戏服务器 Makefile

# 变量定义
APP_NAME=slot-game-server
MAIN_PATH=cmd/server/main.go
BUILD_DIR=build
BIN_DIR=bin
CONFIG_FILE=config/config.yaml

# 版本信息
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go编译参数
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# 颜色输出
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

.PHONY: all build run clean test deps fmt lint help

# 默认目标
all: build

# 构建项目
build:
	@echo "$(GREEN)正在构建 $(APP_NAME)...$(NC)"
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)构建完成: $(BIN_DIR)/$(APP_NAME)$(NC)"

# 运行项目
run: build
	@echo "$(GREEN)启动服务器...$(NC)"
	@if [ ! -f $(CONFIG_FILE) ]; then \
		echo "$(YELLOW)配置文件不存在，复制示例配置...$(NC)"; \
		cp config/config.yaml.example $(CONFIG_FILE); \
	fi
	$(BIN_DIR)/$(APP_NAME) -config=$(CONFIG_FILE)

# 开发模式运行（自动重载）
dev:
	@echo "$(GREEN)开发模式运行...$(NC)"
	@if [ ! -f $(CONFIG_FILE) ]; then \
		cp config/config.yaml.example $(CONFIG_FILE); \
	fi
	$(GO) run $(MAIN_PATH) -config=$(CONFIG_FILE)

# 安装依赖
deps:
	@echo "$(GREEN)安装依赖...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)依赖安装完成$(NC)"

# 更新依赖
update:
	@echo "$(GREEN)更新依赖...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "$(GREEN)依赖更新完成$(NC)"

# 运行测试（禁用缓存）
test:
	@echo "$(GREEN)运行测试...$(NC)"
	$(GO) test -v -race -cover -count=1 ./...

# 运行测试（过滤警告）
test-quiet:
	@echo "$(GREEN)运行测试（过滤链接器警告）...$(NC)"
	@$(GO) test -v -race -cover -count=1 ./... 2>&1 | ( grep -v "ld: warning.*LC_DYSYMTAB" || true )

# 测试仓储层
test-repo:
	@echo "$(GREEN)运行仓储层测试...$(NC)"
	$(GO) test -v -count=1 ./internal/repository/...

# 清理测试缓存
test-clean:
	@echo "$(YELLOW)清理测试缓存...$(NC)"
	$(GO) clean -testcache
	@echo "$(GREEN)测试缓存已清理$(NC)"

# 测试覆盖率
coverage:
	@echo "$(GREEN)生成测试覆盖率报告...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.out -count=1 ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)覆盖率报告: coverage.html$(NC)"

# 格式化代码
fmt:
	@echo "$(GREEN)格式化代码...$(NC)"
	$(GO) fmt ./...
	@echo "$(GREEN)代码格式化完成$(NC)"

# 代码检查
lint:
	@echo "$(GREEN)运行代码检查...$(NC)"
	@if ! which golangci-lint > /dev/null; then \
		echo "$(YELLOW)安装 golangci-lint...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

# 清理构建文件
clean:
	@echo "$(YELLOW)清理构建文件...$(NC)"
	rm -rf $(BIN_DIR) $(BUILD_DIR) coverage.* *.log
	@echo "$(GREEN)清理完成$(NC)"

# 创建必要的目录
init:
	@echo "$(GREEN)初始化项目目录...$(NC)"
	@mkdir -p $(BIN_DIR) $(BUILD_DIR) data logs backups
	@if [ ! -f $(CONFIG_FILE) ]; then \
		cp config/config.yaml.example $(CONFIG_FILE); \
		echo "$(GREEN)已创建配置文件: $(CONFIG_FILE)$(NC)"; \
	fi
	@echo "$(GREEN)初始化完成$(NC)"

# Docker构建
docker-build:
	@echo "$(GREEN)构建Docker镜像...$(NC)"
	docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)Docker镜像构建完成$(NC)"

# Docker运行
docker-run:
	@echo "$(GREEN)运行Docker容器...$(NC)"
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-p 8081:8081 \
		-v $(PWD)/config:/app/config \
		-v $(PWD)/data:/app/data \
		-v $(PWD)/logs:/app/logs \
		$(APP_NAME):$(VERSION)
	@echo "$(GREEN)Docker容器已启动$(NC)"

# 生成文档
docs:
	@echo "$(GREEN)生成API文档...$(NC)"
	@if ! which swag > /dev/null; then \
		echo "$(YELLOW)安装 swag...$(NC)"; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	swag init -g $(MAIN_PATH) -o docs/api

# 版本信息
version:
	@echo "$(GREEN)版本信息:$(NC)"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git提交: $(GIT_COMMIT)"
	@echo "  Go版本: $(shell go version)"

# 检查环境
check:
	@echo "$(GREEN)检查开发环境...$(NC)"
	@echo "Go版本: $(shell go version)"
	@echo "Git版本: $(shell git --version)"
	@echo "Make版本: $(shell make --version | head -n1)"
	@if which docker > /dev/null; then \
		echo "Docker版本: $(shell docker --version)"; \
	else \
		echo "$(YELLOW)Docker未安装$(NC)"; \
	fi
	@echo "$(GREEN)环境检查完成$(NC)"

# 帮助信息
help:
	@echo "$(GREEN)推币机游戏服务器 Makefile 使用说明$(NC)"
	@echo ""
	@echo "$(YELLOW)用法:$(NC)"
	@echo "  make [目标]"
	@echo ""
	@echo "$(YELLOW)可用目标:$(NC)"
	@echo "  $(GREEN)build$(NC)        - 构建项目"
	@echo "  $(GREEN)run$(NC)          - 构建并运行项目"
	@echo "  $(GREEN)dev$(NC)          - 开发模式运行"
	@echo "  $(GREEN)test$(NC)         - 运行所有测试（禁用缓存）"
	@echo "  $(GREEN)test-repo$(NC)    - 运行仓储层测试"
	@echo "  $(GREEN)test-clean$(NC)   - 清理测试缓存"
	@echo "  $(GREEN)coverage$(NC)     - 生成测试覆盖率报告"
	@echo "  $(GREEN)deps$(NC)         - 安装依赖"
	@echo "  $(GREEN)update$(NC)       - 更新依赖"
	@echo "  $(GREEN)fmt$(NC)          - 格式化代码"
	@echo "  $(GREEN)lint$(NC)         - 运行代码检查"
	@echo "  $(GREEN)clean$(NC)        - 清理构建文件"
	@echo "  $(GREEN)init$(NC)         - 初始化项目目录"
	@echo "  $(GREEN)docker-build$(NC) - 构建Docker镜像"
	@echo "  $(GREEN)docker-run$(NC)   - 运行Docker容器"
	@echo "  $(GREEN)docs$(NC)         - 生成API文档"
	@echo "  $(GREEN)version$(NC)      - 显示版本信息"
	@echo "  $(GREEN)check$(NC)        - 检查开发环境"
	@echo "  $(GREEN)help$(NC)         - 显示此帮助信息"
	@echo ""
	@echo "$(YELLOW)示例:$(NC)"
	@echo "  make build    # 构建项目"
	@echo "  make run      # 运行项目"
	@echo "  make test     # 运行测试"