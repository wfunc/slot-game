# SQLite CGO 问题解决方案

## 问题描述
ARM64编译的二进制文件在运行时报错：
```
Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work
```

这是因为纯Go编译模式下无法使用依赖C库的SQLite驱动。

## 解决方案

### 方案1：在目标服务器上安装SQLite并直接编译（推荐）

在Ubuntu ARM64服务器上执行：

```bash
# 1. 安装必要的开发工具和SQLite
sudo apt-get update
sudo apt-get install -y build-essential gcc sqlite3 libsqlite3-dev golang

# 2. 克隆或复制源代码到服务器
git clone https://github.com/wfunc/slot-game.git
cd slot-game

# 3. 直接在服务器上编译（使用CGO）
CGO_ENABLED=1 go build -v -ldflags="-s -w" -o slot-game ./cmd/server

# 4. 运行服务
./slot-game
```

### 方案2：使用交叉编译工具链（需要正确的工具链）

在macOS上安装Linux交叉编译工具链：

```bash
# 安装正确的Linux工具链（不是embedded工具链）
brew tap messense/macos-cross-toolchains
brew install aarch64-unknown-linux-gnu

# 使用CGO编译
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
  CC=aarch64-unknown-linux-gnu-gcc \
  go build -v -ldflags="-s -w" -o slot-game ./cmd/server
```

### 方案3：临时使用其他数据库（如MySQL）

修改配置文件 `config/config.yaml`：

```yaml
database:
  driver: "mysql"
  dsn: "user:password@tcp(localhost:3306)/slot_game?charset=utf8mb4&parseTime=True&loc=Local"
```

### 方案4：手动下载依赖（网络问题时）

如果是网络问题导致无法下载Go依赖：

```bash
# 设置Go代理
export GOPROXY=https://goproxy.cn,direct
# 或
export GOPROXY=https://goproxy.io,direct

# 然后重新下载依赖
go mod download
```

## 当前状态

由于网络问题，无法下载纯Go的SQLite驱动（modernc.org/sqlite），因此：
1. ARM64二进制使用纯Go编译，无法使用SQLite
2. 需要在目标服务器上直接编译，或使用其他数据库

## 长期解决方案

待网络恢复后，执行以下操作：

```bash
# 安装纯Go的SQLite驱动
go get github.com/glebarez/sqlite
go get modernc.org/sqlite

# 然后重新编译ARM64版本
make build-arm64
```

这样就可以在不需要CGO的情况下使用SQLite了。