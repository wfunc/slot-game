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

# 创建两个版本的服务文件

# 1. 专用用户版本 (sg用户 - 推荐)
cat > $RELEASE_DIR/slot-game-sg.service << 'EOF'
[Unit]
Description=Slot Game Server (Dedicated User)
After=network.target

[Service]
Type=simple
User=sg
Group=sg
WorkingDirectory=/home/sg/slot-game
ExecStart=/home/sg/slot-game/slot-game
Restart=on-failure
RestartSec=5
StandardOutput=append:/home/sg/slot-game/logs/service.log
StandardError=append:/home/sg/slot-game/logs/service-error.log

# 资源限制
LimitNOFILE=65535
LimitNPROC=4096

# 环境变量
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

# 2. 兼容版本 (ztl用户)
cat > $RELEASE_DIR/slot-game-ztl.service << 'EOF'
[Unit]
Description=Slot Game Server (ZTL User)
After=network.target

[Service]
Type=simple
User=ztl
Group=ztl
WorkingDirectory=/home/ztl/slot-game-arm64
ExecStart=/home/ztl/slot-game-arm64/slot-game
Restart=on-failure
RestartSec=5
StandardOutput=append:/home/ztl/slot-game-arm64/logs/service.log
StandardError=append:/home/ztl/slot-game-arm64/logs/service-error.log

# 资源限制
LimitNOFILE=65535
LimitNPROC=4096

# 环境变量
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

# 创建Chromium Kiosk服务文件（两个版本）

# 1. 专用用户版本 (sg用户 - 推荐)
cat > $RELEASE_DIR/chromium-kiosk-sg.service << 'EOF'
[Unit]
Description=Chromium Kiosk for Slot Game Web Interface (Dedicated User)
After=graphical-session.target slot-game.service
Wants=graphical-session.target
Requires=slot-game.service

[Service]
Type=simple
User=sg
Group=sg
Environment="DISPLAY=:0"
Environment="XDG_SESSION_TYPE=x11"
Environment="OZONE_PLATFORM=x11"
Environment="HOME=/home/sg"

# 等待slot-game服务完全启动（最多等待30秒）
# 使用多种方法检测，不依赖curl
ExecStartPre=/bin/bash -c 'timeout=30; while [ $timeout -gt 0 ]; do \
  if command -v curl >/dev/null 2>&1 && curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then \
    exit 0; \
  elif command -v wget >/dev/null 2>&1 && wget -q -O /dev/null http://127.0.0.1:8080 2>/dev/null; then \
    exit 0; \
  elif nc -z 127.0.0.1 8080 2>/dev/null; then \
    echo "Port 8080 is open, assuming service is ready"; \
    exit 0; \
  elif [ -f /proc/net/tcp ] && grep -q ":1F90" /proc/net/tcp; then \
    echo "Port 8080 (0x1F90) found in /proc/net/tcp"; \
    exit 0; \
  fi; \
  echo "Waiting for slot-game service... ($timeout seconds left)"; \
  sleep 2; \
  timeout=$((timeout-2)); \
done; \
echo "Error: slot-game service not responding on port 8080"; \
echo "Tip: Install curl or wget for better health checks"; \
exit 1'

# 启动Chromium Kiosk
ExecStart=/usr/bin/chromium \
  --user-data-dir=/tmp/chromium-kiosk \
  --kiosk --start-fullscreen \
  --new-window "http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo" \
  --use-gl=egl \
  --enable-gpu-rasterization \
  --ignore-gpu-blocklist \
  --disable-software-rasterizer \
  --canvas-oop-rasterization=disabled \
  --enable-accelerated-video-decode \
  --enable-features=VaapiVideoDecoder,VaapiVideoEncoder \
  --ozone-platform=x11 \
  --no-first-run --no-default-browser-check \
  --password-store=basic \
  --disable-password-manager-reauth \
  --disable-features=BackForwardCache,LowPriorityIframes \
  --disable-background-timer-throttling \
  --disable-renderer-backgrounding

Restart=always
RestartSec=5
StandardOutput=append:/home/sg/slot-game/logs/kiosk.log
StandardError=append:/home/sg/slot-game/logs/kiosk-error.log

[Install]
WantedBy=default.target
EOF

# 2. 兼容版本 (ztl用户)
cat > $RELEASE_DIR/chromium-kiosk-ztl.service << 'EOF'
[Unit]
Description=Chromium Kiosk for Slot Game Web Interface (ZTL User)
After=graphical-session.target slot-game.service
Wants=graphical-session.target
Requires=slot-game.service

[Service]
Type=simple
User=ztl
Group=ztl
Environment="DISPLAY=:0"
Environment="XDG_SESSION_TYPE=x11"
Environment="OZONE_PLATFORM=x11"
Environment="HOME=/home/ztl"

# 等待slot-game服务完全启动（最多等待30秒）
# 使用多种方法检测，不依赖curl
ExecStartPre=/bin/bash -c 'timeout=30; while [ $timeout -gt 0 ]; do \
  if command -v curl >/dev/null 2>&1 && curl -f http://127.0.0.1:8080 >/dev/null 2>&1; then \
    exit 0; \
  elif command -v wget >/dev/null 2>&1 && wget -q -O /dev/null http://127.0.0.1:8080 2>/dev/null; then \
    exit 0; \
  elif nc -z 127.0.0.1 8080 2>/dev/null; then \
    echo "Port 8080 is open, assuming service is ready"; \
    exit 0; \
  elif [ -f /proc/net/tcp ] && grep -q ":1F90" /proc/net/tcp; then \
    echo "Port 8080 (0x1F90) found in /proc/net/tcp"; \
    exit 0; \
  fi; \
  echo "Waiting for slot-game service... ($timeout seconds left)"; \
  sleep 2; \
  timeout=$((timeout-2)); \
done; \
echo "Error: slot-game service not responding on port 8080"; \
echo "Tip: Install curl or wget for better health checks"; \
exit 1'

# 启动Chromium Kiosk
ExecStart=/usr/bin/chromium \
  --user-data-dir=/tmp/chromium-kiosk \
  --kiosk --start-fullscreen \
  --new-window "http://127.0.0.1:8080/static/web-mobile/?token=68bf99c4aedf1c000b000434&type=zoo" \
  --use-gl=egl \
  --enable-gpu-rasterization \
  --ignore-gpu-blocklist \
  --disable-software-rasterizer \
  --canvas-oop-rasterization=disabled \
  --enable-accelerated-video-decode \
  --enable-features=VaapiVideoDecoder,VaapiVideoEncoder \
  --ozone-platform=x11 \
  --no-first-run --no-default-browser-check \
  --password-store=basic \
  --disable-password-manager-reauth \
  --disable-features=BackForwardCache,LowPriorityIframes \
  --disable-background-timer-throttling \
  --disable-renderer-backgrounding

Restart=always
RestartSec=5
StandardOutput=append:/home/ztl/slot-game-arm64/logs/kiosk.log
StandardError=append:/home/ztl/slot-game-arm64/logs/kiosk-error.log

[Install]
WantedBy=default.target
EOF

# 复制自动化安装脚本
echo -e "${GREEN}复制自动化安装脚本...${NC}"
if [ -f "scripts/install.sh" ]; then
    cp scripts/install.sh $RELEASE_DIR/install.sh
    chmod +x $RELEASE_DIR/install.sh
    echo -e "${GREEN}✓ 已复制自动化安装脚本${NC}"
else
    echo -e "${YELLOW}⚠ 未找到自动化安装脚本，使用内置版本${NC}"
    # 创建备用安装脚本（简化版）
    cat > $RELEASE_DIR/install.sh << 'EOF'
#!/bin/bash
echo "请使用最新的自动化安装脚本"
echo "从项目仓库获取: scripts/install.sh"
exit 1
EOF
fi

# 创建README
cat > $RELEASE_DIR/README.md << 'EOF'
# 老虎机游戏服务部署说明

## 一键自动安装（推荐）

```bash
# 解压并自动安装
tar -xzf slot-game-arm64.tar.gz
cd slot-game-arm64
sudo ./install.sh

# 安装完成后重启系统
sudo reboot
```

**安装脚本会自动：**
- 创建专用的 sg 用户运行 slot-game 服务
- 配置 ztl 用户运行 chromium-kiosk 服务
- 设置所有必要的权限和目录
- 配置服务开机自启动
- 无需任何用户交互，完全自动化

## 快速开始（手动模式）

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

- `install.sh` - 一键自动安装脚本（无需交互，推荐使用）
- `start.sh` - 手动启动服务
- `stop.sh` - 手动停止服务
- `status.sh` - 查看服务状态
- `setup-user.sh` - 用户管理脚本（install.sh会自动调用）
- `fix-chromium-path.sh` - 修复Chromium路径问题（如遇到浏览器路径错误）

## 服务介绍

### 1. Slot Game服务
主游戏服务器，提供HTTP和WebSocket接口：
- 运行用户：sg（专用用户，数据隔离）
- HTTP端口：8080
- WebSocket路径：/ws/game
- 数据库：SQLite（位于 `/home/sg/slot-game/data/`）
- 工作目录：`/home/sg/slot-game`
- 日志：通过 journalctl -u slot-game 查看

### 2. Chromium Kiosk服务
全屏浏览器模式，自动打开游戏界面：
- 运行用户：ztl（图形界面用户）
- 依赖：需要slot-game服务先启动
- 特性：自动等待服务就绪后启动
- URL：自动加载游戏界面
- 日志：通过 journalctl -u chromium-kiosk 查看

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
- 图形环境：Chromium Kiosk需要X11或Wayland

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

4. **Chromium Kiosk问题**
   - 检查图形环境：`echo $DISPLAY`（应该显示:0）
   - 确认slot-game服务已启动：`systemctl status slot-game`
   - 检查Chromium是否安装：`which chromium` 或 `which chromium-browser`
   - 查看Kiosk日志：`journalctl -u chromium-kiosk -f`
   - 手动测试连接：`curl http://127.0.0.1:8080`
   
   **常见错误："/usr/bin/chromium-browser: No such file or directory"**
   - 原因：不同Ubuntu版本的Chromium路径不同
   - 解决：运行 `sudo ./fix-chromium-path.sh` 自动修复路径
   - 或手动安装：`sudo apt install chromium-browser` 或 `sudo apt install chromium`

## 技术支持

- 项目地址：https://github.com/wfunc/slot-game
- 问题反馈：请提交Issue
EOF

# 创建服务检查脚本
cat > $RELEASE_DIR/check-services.sh << 'EOF'
#!/bin/bash

# 服务健康检查脚本

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo "================================================"
echo "          服务健康检查"
echo "================================================"

# 1. 检查slot-game服务状态
echo -e "\n${GREEN}1. 检查 slot-game 服务状态${NC}"
if systemctl is-active slot-game >/dev/null 2>&1; then
    echo -e "   ${GREEN}✓ slot-game 服务正在运行${NC}"
    systemctl status slot-game --no-pager | head -10
else
    echo -e "   ${RED}✗ slot-game 服务未运行${NC}"
    echo -e "   ${YELLOW}提示：请先启动 slot-game 服务${NC}"
    echo -e "   ${YELLOW}命令：sudo systemctl start slot-game${NC}"
fi

# 2. 检查端口监听
echo -e "\n${GREEN}2. 检查端口监听状态${NC}"
if ss -tlnp | grep -q ":8080"; then
    echo -e "   ${GREEN}✓ 8080端口正在监听${NC}"
else
    echo -e "   ${RED}✗ 8080端口未监听${NC}"
    echo -e "   ${YELLOW}提示：检查slot-game配置文件${NC}"
fi

# 3. 测试HTTP响应
echo -e "\n${GREEN}3. 测试HTTP响应${NC}"
if curl -f -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080 | grep -q "200\|301\|302"; then
    echo -e "   ${GREEN}✓ HTTP服务响应正常${NC}"
    echo -e "   响应代码：$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080)"
else
    echo -e "   ${RED}✗ HTTP服务无响应${NC}"
    echo -e "   ${YELLOW}提示：检查服务日志${NC}"
    echo -e "   ${YELLOW}命令：sudo journalctl -u slot-game -n 50${NC}"
fi

# 4. 检查图形环境
echo -e "\n${GREEN}4. 检查图形环境${NC}"
if [ -n "$DISPLAY" ]; then
    echo -e "   ${GREEN}✓ DISPLAY环境变量已设置：$DISPLAY${NC}"
else
    echo -e "   ${YELLOW}⚠ DISPLAY环境变量未设置${NC}"
    echo -e "   ${YELLOW}提示：Kiosk服务需要图形环境${NC}"
fi

# 5. 检查Chromium安装
echo -e "\n${GREEN}5. 检查Chromium浏览器${NC}"
if which chromium >/dev/null 2>&1; then
    echo -e "   ${GREEN}✓ Chromium已安装${NC}"
    chromium --version 2>/dev/null || echo "   版本信息不可用"
else
    echo -e "   ${RED}✗ Chromium未安装${NC}"
    echo -e "   ${YELLOW}提示：安装Chromium${NC}"
    echo -e "   ${YELLOW}命令：sudo apt install chromium${NC}"
fi

# 6. 检查curl安装
echo -e "\n${GREEN}6. 检查curl工具${NC}"
if which curl >/dev/null 2>&1; then
    echo -e "   ${GREEN}✓ curl已安装${NC}"
else
    echo -e "   ${RED}✗ curl未安装${NC}"
    echo -e "   ${YELLOW}提示：安装curl${NC}"
    echo -e "   ${YELLOW}命令：sudo apt install curl${NC}"
fi

# 7. 检查chromium-kiosk服务
echo -e "\n${GREEN}7. 检查 chromium-kiosk 服务状态${NC}"
if [ -f /etc/systemd/system/chromium-kiosk.service ]; then
    echo -e "   ${GREEN}✓ chromium-kiosk.service 已安装${NC}"
    if systemctl is-active chromium-kiosk >/dev/null 2>&1; then
        echo -e "   ${GREEN}✓ chromium-kiosk 服务正在运行${NC}"
    else
        echo -e "   ${YELLOW}⚠ chromium-kiosk 服务未运行${NC}"
        # 显示最近的错误日志
        echo -e "\n   最近的日志："
        journalctl -u chromium-kiosk -n 5 --no-pager 2>/dev/null
    fi
else
    echo -e "   ${YELLOW}⚠ chromium-kiosk.service 未安装${NC}"
fi

# 汇总
echo -e "\n================================================"
echo -e "${GREEN}检查完成${NC}"
echo ""
echo "如果chromium-kiosk启动失败，常见原因："
echo "1. slot-game服务未启动或端口错误"
echo "2. 缺少图形环境（DISPLAY未设置）"
echo "3. Chromium未安装或路径错误"
echo "4. curl工具未安装"
echo ""
echo "建议按顺序执行："
echo "1. sudo systemctl start slot-game"
echo "2. sudo systemctl status slot-game"
echo "3. curl http://127.0.0.1:8080"
echo "4. sudo systemctl start chromium-kiosk"
echo "================================================"
EOF

# 复制用户管理脚本
cp scripts/setup-user.sh $RELEASE_DIR/

# 复制修复脚本
if [ -f "scripts/fix-chromium-path.sh" ]; then
    cp scripts/fix-chromium-path.sh $RELEASE_DIR/
    echo -e "${GREEN}✓ 已复制 Chromium 路径修复脚本${NC}"
fi

# 复制优化脚本
if [ -f "scripts/chromium-optimized.sh" ]; then
    cp scripts/chromium-optimized.sh $RELEASE_DIR/
    echo -e "${GREEN}✓ 已复制 Chromium 优化配置脚本${NC}"
fi



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
echo "部署步骤（一键自动安装）:"
echo "1. 复制到目标机器: scp release/slot-game-arm64.tar.gz ztl@<目标IP>:~/"
echo "2. 登录目标机器: ssh ztl@<目标IP>"
echo "3. 解压文件: tar -xzf slot-game-arm64.tar.gz"
echo "4. 进入目录: cd slot-game-arm64"
echo "5. 自动安装: sudo ./install.sh"
echo "6. 重启系统: sudo reboot"
echo ""
echo -e "${GREEN}✅ 安装脚本会自动完成所有配置，无需用户交互${NC}"
echo -e "${GREEN}✅ 重启后服务会自动启动（slot-game在sg用户，chromium在ztl用户）${NC}"
echo ""
echo "手动启动（可选）:"
echo "- 启动服务: ./start.sh"
echo "- 查看状态: slot-game-manage status"
echo ""
echo -e "${YELLOW}提示: 安装脚本会自动创建sg用户、配置权限、设置开机自启${NC}"