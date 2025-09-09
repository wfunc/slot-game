#!/bin/bash

# ARM64构建和部署脚本
# 用于编译适用于Ubuntu ARM64架构的程序包

set -e

echo "================================================"
echo "  老虎机游戏 ARM64 构建脚本"
echo "================================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 检查并选择交叉编译工具
CC=""
if command -v aarch64-linux-gnu-gcc &> /dev/null; then
    CC="aarch64-linux-gnu-gcc"
    echo -e "${GREEN}使用编译器: aarch64-linux-gnu-gcc${NC}"
elif command -v aarch64-unknown-linux-gnu-gcc &> /dev/null; then
    CC="aarch64-unknown-linux-gnu-gcc"
    echo -e "${GREEN}使用编译器: aarch64-unknown-linux-gnu-gcc${NC}"
elif command -v aarch64-none-elf-gcc &> /dev/null; then
    echo -e "${YELLOW}警告: 检测到 aarch64-none-elf-gcc (裸机工具链)${NC}"
    echo -e "${YELLOW}尝试使用纯Go编译（禁用CGO）${NC}"
    CC=""
else
    echo -e "${YELLOW}警告: 未找到ARM64交叉编译工具${NC}"
    echo "可选方案："
    echo "  1. macOS安装Linux工具链: brew tap messense/macos-cross-toolchains && brew install aarch64-unknown-linux-gnu"
    echo "  2. 使用纯Go编译（禁用CGO，SQLite将使用纯Go实现）"
    echo ""
    read -p "是否使用纯Go编译？(y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
    CC=""
fi

# 创建发布目录
RELEASE_DIR="release/slot-game-arm64"
echo -e "${GREEN}创建发布目录: ${RELEASE_DIR}${NC}"
rm -rf $RELEASE_DIR
mkdir -p $RELEASE_DIR

# 编译程序
echo -e "${GREEN}编译ARM64程序...${NC}"

if [ -n "$CC" ]; then
    # 使用CGO编译（C编译器可用）
    echo -e "${GREEN}使用CGO编译（支持原生SQLite）${NC}"
    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=$CC \
        go build -v -ldflags="-s -w" -o $RELEASE_DIR/slot-game ./cmd/server
else
    # 纯Go编译（无C编译器）
    echo -e "${YELLOW}使用纯Go编译（CGO禁用模式）${NC}"
    echo -e "${YELLOW}注意：使用标准Go编译，不依赖C库${NC}"
    
    # 纯Go编译
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
        go build -v -ldflags="-s -w" -o $RELEASE_DIR/slot-game ./cmd/server
fi

if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败！${NC}"
    exit 1
fi

# 复制配置文件
echo -e "${GREEN}复制配置文件...${NC}"
cp -r config $RELEASE_DIR/

# 创建必要的目录
echo -e "${GREEN}创建必要目录...${NC}"
mkdir -p $RELEASE_DIR/data
mkdir -p $RELEASE_DIR/logs
mkdir -p $RELEASE_DIR/static

# 复制静态文件
if [ -d "static" ]; then
    echo -e "${GREEN}复制静态文件...${NC}"
    cp -r static/* $RELEASE_DIR/static/
fi

# 创建启动脚本
echo -e "${GREEN}创建启动脚本...${NC}"
cat > $RELEASE_DIR/start.sh << 'EOF'
#!/bin/bash

# 老虎机游戏启动脚本

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 检查并创建必要目录
echo -e "${GREEN}检查运行环境...${NC}"
mkdir -p ./data
mkdir -p ./logs

# 设置执行权限
chmod +x ./slot-game

# 检查是否已经运行
if pgrep -f "slot-game" > /dev/null; then
    echo -e "${RED}服务已经在运行！${NC}"
    echo "使用 ./stop.sh 停止服务"
    exit 1
fi

# 启动服务
echo -e "${GREEN}启动老虎机游戏服务...${NC}"
nohup ./slot-game > logs/startup.log 2>&1 &

# 等待服务启动
sleep 2

# 检查服务状态
if pgrep -f "slot-game" > /dev/null; then
    echo -e "${GREEN}服务启动成功！${NC}"
    echo "访问地址: http://$(hostname -I | awk '{print $1}'):8080"
    echo "WebSocket: ws://$(hostname -I | awk '{print $1}'):8080/ws/game"
    echo "日志文件: logs/app.log"
    tail -n 20 logs/startup.log
else
    echo -e "${RED}服务启动失败！${NC}"
    echo "请查看日志: logs/startup.log"
    tail -n 50 logs/startup.log
fi
EOF

# 创建停止脚本
cat > $RELEASE_DIR/stop.sh << 'EOF'
#!/bin/bash

# 停止服务脚本

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PID=$(pgrep -f "slot-game")

if [ -z "$PID" ]; then
    echo -e "${RED}服务未运行${NC}"
else
    echo -e "${GREEN}停止服务 (PID: $PID)...${NC}"
    kill $PID
    sleep 2
    
    # 检查是否成功停止
    if pgrep -f "slot-game" > /dev/null; then
        echo -e "${RED}正常停止失败，强制终止...${NC}"
        kill -9 $PID
    fi
    
    echo -e "${GREEN}服务已停止${NC}"
fi
EOF

# 创建服务状态检查脚本
cat > $RELEASE_DIR/status.sh << 'EOF'
#!/bin/bash

# 服务状态检查

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

PID=$(pgrep -f "slot-game")

if [ -z "$PID" ]; then
    echo -e "${RED}服务状态: 未运行${NC}"
else
    echo -e "${GREEN}服务状态: 运行中${NC}"
    echo -e "进程ID: $PID"
    echo -e "内存使用:"
    ps aux | grep -E "PID|slot-game" | grep -v grep
    echo -e "\n端口监听:"
    netstat -tlnp 2>/dev/null | grep -E "8080|8081" || ss -tlnp | grep -E "8080|8081"
fi

# 检查数据库文件
if [ -f "./data/slot-game.db" ]; then
    echo -e "\n${GREEN}数据库文件存在${NC}"
    ls -lh ./data/slot-game.db
else
    echo -e "\n${YELLOW}数据库文件不存在（首次运行会自动创建）${NC}"
fi

# 检查日志
if [ -d "./logs" ]; then
    echo -e "\n最近日志:"
    tail -n 5 ./logs/app.log 2>/dev/null || echo "暂无日志"
fi
EOF

# 创建systemd服务文件
cat > $RELEASE_DIR/slot-game.service << 'EOF'
[Unit]
Description=Slot Game Server
After=network.target

[Service]
Type=simple
User=ztl
Group=ztl
WorkingDirectory=/home/ztl/slot-game
ExecStart=/home/ztl/slot-game/slot-game
Restart=on-failure
RestartSec=5
StandardOutput=append:/home/ztl/slot-game/logs/service.log
StandardError=append:/home/ztl/slot-game/logs/service-error.log

# 资源限制
LimitNOFILE=65535
LimitNPROC=4096

# 环境变量
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

# 创建安装脚本
cat > $RELEASE_DIR/install.sh << 'EOF'
#!/bin/bash

# 系统服务安装脚本

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用sudo运行此脚本${NC}"
    exit 1
fi

# 复制服务文件
echo -e "${GREEN}安装systemd服务...${NC}"
cp slot-game.service /etc/systemd/system/

# 重载systemd
systemctl daemon-reload

# 启用服务
systemctl enable slot-game.service

echo -e "${GREEN}服务安装完成！${NC}"
echo "使用以下命令管理服务:"
echo "  启动: sudo systemctl start slot-game"
echo "  停止: sudo systemctl stop slot-game"
echo "  重启: sudo systemctl restart slot-game"
echo "  状态: sudo systemctl status slot-game"
echo "  日志: sudo journalctl -u slot-game -f"
EOF

# 创建README
cat > $RELEASE_DIR/README.md << 'EOF'
# 老虎机游戏服务部署说明

## 快速开始

1. **解压文件**
   ```bash
   tar -xzf slot-game-arm64.tar.gz
   cd slot-game-arm64
   ```

2. **设置权限**
   ```bash
   chmod +x *.sh slot-game
   ```

3. **启动服务**
   ```bash
   ./start.sh
   ```

## 脚本说明

- `start.sh` - 启动服务
- `stop.sh` - 停止服务
- `status.sh` - 查看服务状态
- `install.sh` - 安装为系统服务（需要sudo）

## 配置文件

配置文件位于 `config/config.yaml`，主要配置项：

- 数据库：默认使用SQLite，数据文件在`data/slot-game.db`
- 端口：HTTP服务默认8080，WebSocket通过/ws/game路径访问
- 日志：日志文件保存在`logs/`目录

## 目录结构

```
slot-game-arm64/
├── slot-game           # 主程序
├── config/            # 配置文件目录
│   └── config.yaml    # 主配置文件
├── data/              # 数据目录（SQLite数据库）
├── logs/              # 日志目录
├── static/            # 静态文件（Web界面）
└── *.sh              # 管理脚本
```

## 系统要求

- Ubuntu 18.04+ (ARM64架构)
- 可用内存：至少512MB
- 磁盘空间：至少100MB

## 故障排查

1. **服务无法启动**
   - 检查端口是否被占用：`netstat -tlnp | grep 8080`
   - 查看日志：`tail -f logs/startup.log`

2. **数据库错误**
   - 确保data目录有写权限：`chmod 755 data`
   - 删除损坏的数据库：`rm data/slot-game.db`（会丢失数据）

3. **串口通信问题**
   - 检查串口设备：`ls /dev/ttyUSB*`
   - 添加用户到dialout组：`sudo usermod -a -G dialout $USER`

## 技术支持

- 项目地址：https://github.com/wfunc/slot-game
- 问题反馈：请提交Issue
EOF

# 设置脚本权限
chmod +x $RELEASE_DIR/*.sh

# 打包
echo -e "${GREEN}创建压缩包...${NC}"
cd release
tar -czf slot-game-arm64.tar.gz slot-game-arm64/
cd ..

# 计算文件大小
SIZE=$(du -h release/slot-game-arm64.tar.gz | cut -f1)

echo ""
echo "================================================"
echo -e "${GREEN}构建成功！${NC}"
echo "================================================"
echo "输出文件: release/slot-game-arm64.tar.gz"
echo "文件大小: $SIZE"
echo ""
echo "部署步骤:"
echo "1. 复制到目标机器: scp release/slot-game-arm64.tar.gz ztl@<目标IP>:~/"
echo "2. 登录目标机器: ssh ztl@<目标IP>"
echo "3. 解压文件: tar -xzf slot-game-arm64.tar.gz"
echo "4. 进入目录: cd slot-game-arm64"
echo "5. 启动服务: ./start.sh"
echo ""
echo -e "${YELLOW}提示: 首次运行会自动创建数据库和必要目录${NC}"