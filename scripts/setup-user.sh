#!/bin/bash

# 用户初始化脚本 - 创建专用的 sg 用户运行 slot-game 服务
# 这提供更好的权限隔离和数据保护

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "================================================"
echo -e "${GREEN}    Slot Game 用户管理脚本${NC}"
echo "================================================"
echo ""

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用sudo运行此脚本${NC}"
    echo "用法: sudo $0"
    exit 1
fi

# 配置变量
SG_USER="sg"
SG_GROUP="sg"
SG_HOME="/home/sg"
SG_APP_DIR="${SG_HOME}/slot-game"
CURRENT_USER="${SUDO_USER:-$USER}"

echo -e "${BLUE}当前配置：${NC}"
echo "  专用用户: ${SG_USER}"
echo "  用户主目录: ${SG_HOME}"
echo "  应用目录: ${SG_APP_DIR}"
echo "  数据目录: ${SG_APP_DIR}/data"
echo "  日志目录: ${SG_APP_DIR}/logs"
echo ""

# 选择操作
echo -e "${GREEN}请选择操作：${NC}"
echo "1) 创建新用户 ${SG_USER} 并初始化环境"
echo "2) 从 ztl 用户迁移到 ${SG_USER} 用户"
echo "3) 仅更新权限（如果用户已存在）"
echo "4) 删除 ${SG_USER} 用户和相关数据（慎用）"
echo "5) 查看当前状态"
read -p "请选择 [1-5]: " choice

case $choice in
    1)
        echo -e "\n${GREEN}创建新用户和环境...${NC}"
        
        # 1. 创建用户和组
        if id "$SG_USER" &>/dev/null; then
            echo -e "${YELLOW}用户 ${SG_USER} 已存在${NC}"
        else
            echo "创建用户 ${SG_USER}..."
            useradd -m -s /bin/bash -d ${SG_HOME} ${SG_USER}
            echo -e "${GREEN}✓ 用户创建成功${NC}"
        fi
        
        # 2. 创建目录结构
        echo "创建目录结构..."
        mkdir -p ${SG_APP_DIR}/{data,logs,config,static,backups}
        
        # 3. 设置权限
        echo "设置目录权限..."
        chown -R ${SG_USER}:${SG_GROUP} ${SG_HOME}
        chmod 755 ${SG_HOME}
        chmod 755 ${SG_APP_DIR}
        chmod 700 ${SG_APP_DIR}/data  # 数据目录仅用户可访问
        chmod 755 ${SG_APP_DIR}/logs
        chmod 755 ${SG_APP_DIR}/config
        chmod 755 ${SG_APP_DIR}/static
        chmod 700 ${SG_APP_DIR}/backups
        
        # 4. 添加到必要的组
        echo "配置用户组..."
        # 添加到 dialout 组（串口访问）
        usermod -a -G dialout ${SG_USER} 2>/dev/null || true
        # 添加到 video 组（GPU访问，Chromium需要）
        usermod -a -G video ${SG_USER} 2>/dev/null || true
        # 添加到 audio 组（音频访问）
        usermod -a -G audio ${SG_USER} 2>/dev/null || true
        
        # 5. 创建 sudoers 文件（仅允许特定命令）
        echo "配置 sudo 权限..."
        cat > /etc/sudoers.d/sg-user << EOF
# Allow sg user to manage slot-game services
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status slot-game
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl start chromium-kiosk
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl stop chromium-kiosk
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl restart chromium-kiosk
${SG_USER} ALL=(ALL) NOPASSWD: /bin/systemctl status chromium-kiosk
${SG_USER} ALL=(ALL) NOPASSWD: /bin/journalctl -u slot-game *
${SG_USER} ALL=(ALL) NOPASSWD: /bin/journalctl -u chromium-kiosk *
EOF
        chmod 440 /etc/sudoers.d/sg-user
        
        # 6. 创建管理脚本
        echo "创建管理脚本..."
        cat > ${SG_APP_DIR}/manage.sh << 'SCRIPT_EOF'
#!/bin/bash
# Slot Game 管理脚本

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

case "$1" in
    start)
        echo -e "${GREEN}启动服务...${NC}"
        sudo systemctl start slot-game
        sudo systemctl start chromium-kiosk
        ;;
    stop)
        echo -e "${YELLOW}停止服务...${NC}"
        sudo systemctl stop chromium-kiosk
        sudo systemctl stop slot-game
        ;;
    restart)
        echo -e "${YELLOW}重启服务...${NC}"
        sudo systemctl restart slot-game
        sleep 2
        sudo systemctl restart chromium-kiosk
        ;;
    status)
        echo -e "${GREEN}服务状态：${NC}"
        sudo systemctl status slot-game --no-pager
        echo ""
        sudo systemctl status chromium-kiosk --no-pager
        ;;
    logs)
        echo -e "${GREEN}查看日志（Ctrl+C退出）：${NC}"
        sudo journalctl -f -u slot-game -u chromium-kiosk
        ;;
    backup)
        echo -e "${GREEN}备份数据...${NC}"
        BACKUP_FILE="/home/sg/slot-game/backups/backup-$(date +%Y%m%d-%H%M%S).tar.gz"
        tar -czf $BACKUP_FILE /home/sg/slot-game/data/
        echo -e "${GREEN}备份完成: $BACKUP_FILE${NC}"
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs|backup}"
        exit 1
        ;;
esac
SCRIPT_EOF
        chmod +x ${SG_APP_DIR}/manage.sh
        chown ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}/manage.sh
        
        echo -e "\n${GREEN}✅ 用户环境创建完成！${NC}"
        echo ""
        echo "下一步："
        echo "1. 解压程序包到 ${SG_APP_DIR}/"
        echo "   tar -xzf slot-game-arm64.tar.gz -C ${SG_APP_DIR}/ --strip-components=1"
        echo "2. 设置文件权限："
        echo "   chown -R ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}"
        echo "3. 更新服务文件使用 ${SG_USER} 用户"
        echo "4. 重启服务"
        ;;
        
    2)
        echo -e "\n${GREEN}从 ztl 迁移到 ${SG_USER}...${NC}"
        
        # 检查sg用户是否存在
        if ! id "$SG_USER" &>/dev/null; then
            echo -e "${RED}用户 ${SG_USER} 不存在，请先选择选项1创建用户${NC}"
            exit 1
        fi
        
        # 停止服务
        echo "停止运行中的服务..."
        systemctl stop chromium-kiosk 2>/dev/null || true
        systemctl stop slot-game 2>/dev/null || true
        
        # 备份旧数据
        OLD_DIR="/home/ztl/slot-game-arm64"
        if [ -d "$OLD_DIR" ]; then
            echo "备份 ztl 用户的数据..."
            BACKUP_DIR="${SG_APP_DIR}/backups/migration-$(date +%Y%m%d-%H%M%S)"
            mkdir -p $BACKUP_DIR
            
            # 复制数据文件
            if [ -d "$OLD_DIR/data" ]; then
                echo "迁移数据库..."
                cp -a $OLD_DIR/data/* ${SG_APP_DIR}/data/ 2>/dev/null || true
            fi
            
            # 复制配置文件
            if [ -d "$OLD_DIR/config" ]; then
                echo "迁移配置..."
                cp -a $OLD_DIR/config/* ${SG_APP_DIR}/config/ 2>/dev/null || true
            fi
            
            # 复制程序文件
            echo "复制程序文件..."
            cp -a $OLD_DIR/slot-game ${SG_APP_DIR}/ 2>/dev/null || true
            cp -a $OLD_DIR/*.sh ${SG_APP_DIR}/ 2>/dev/null || true
            cp -a $OLD_DIR/static/* ${SG_APP_DIR}/static/ 2>/dev/null || true
            
            # 设置权限
            echo "更新权限..."
            chown -R ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}
            chmod +x ${SG_APP_DIR}/slot-game
            chmod +x ${SG_APP_DIR}/*.sh
            
            echo -e "${GREEN}✓ 数据迁移完成${NC}"
            
            # 保留旧数据的备份
            echo "创建旧数据备份..."
            tar -czf ${BACKUP_DIR}/ztl-backup.tar.gz -C /home/ztl slot-game-arm64 2>/dev/null || true
            echo "备份保存在: ${BACKUP_DIR}/ztl-backup.tar.gz"
        else
            echo -e "${YELLOW}未找到 ztl 用户的旧数据${NC}"
        fi
        
        echo -e "\n${GREEN}✅ 迁移完成！${NC}"
        echo "请更新服务文件中的用户和路径配置"
        ;;
        
    3)
        echo -e "\n${GREEN}更新权限...${NC}"
        
        if ! id "$SG_USER" &>/dev/null; then
            echo -e "${RED}用户 ${SG_USER} 不存在${NC}"
            exit 1
        fi
        
        chown -R ${SG_USER}:${SG_GROUP} ${SG_APP_DIR}
        chmod 755 ${SG_APP_DIR}
        chmod 700 ${SG_APP_DIR}/data
        chmod 755 ${SG_APP_DIR}/logs
        chmod +x ${SG_APP_DIR}/slot-game 2>/dev/null || true
        chmod +x ${SG_APP_DIR}/*.sh 2>/dev/null || true
        
        echo -e "${GREEN}✓ 权限更新完成${NC}"
        ;;
        
    4)
        echo -e "\n${RED}⚠️  警告：这将删除用户和所有数据！${NC}"
        read -p "确认删除用户 ${SG_USER}？输入 'DELETE' 确认: " confirm
        
        if [ "$confirm" = "DELETE" ]; then
            echo "停止服务..."
            systemctl stop chromium-kiosk 2>/dev/null || true
            systemctl stop slot-game 2>/dev/null || true
            
            echo "创建最终备份..."
            if [ -d "${SG_APP_DIR}" ]; then
                tar -czf /tmp/sg-final-backup-$(date +%Y%m%d-%H%M%S).tar.gz ${SG_APP_DIR}
                echo "备份保存在: /tmp/sg-final-backup-*.tar.gz"
            fi
            
            echo "删除用户..."
            userdel -r ${SG_USER} 2>/dev/null || true
            rm -f /etc/sudoers.d/sg-user
            
            echo -e "${GREEN}✓ 用户已删除${NC}"
        else
            echo -e "${YELLOW}操作已取消${NC}"
        fi
        ;;
        
    5)
        echo -e "\n${GREEN}当前状态：${NC}"
        
        # 检查用户
        if id "$SG_USER" &>/dev/null; then
            echo -e "用户 ${SG_USER}: ${GREEN}✓ 存在${NC}"
            echo "  用户ID: $(id -u ${SG_USER})"
            echo "  组: $(groups ${SG_USER})"
        else
            echo -e "用户 ${SG_USER}: ${RED}✗ 不存在${NC}"
        fi
        
        # 检查目录
        if [ -d "${SG_APP_DIR}" ]; then
            echo -e "应用目录: ${GREEN}✓ 存在${NC}"
            echo "  路径: ${SG_APP_DIR}"
            echo "  权限: $(stat -c %a ${SG_APP_DIR})"
            echo "  所有者: $(stat -c %U:%G ${SG_APP_DIR})"
            
            # 检查子目录
            for dir in data logs config static backups; do
                if [ -d "${SG_APP_DIR}/$dir" ]; then
                    echo -e "  $dir/: ${GREEN}✓${NC} $(stat -c %a ${SG_APP_DIR}/$dir)"
                else
                    echo -e "  $dir/: ${RED}✗${NC}"
                fi
            done
        else
            echo -e "应用目录: ${RED}✗ 不存在${NC}"
        fi
        
        # 检查服务状态
        echo -e "\n服务状态："
        if systemctl is-active slot-game >/dev/null 2>&1; then
            echo -e "  slot-game: ${GREEN}运行中${NC}"
        else
            echo -e "  slot-game: ${YELLOW}未运行${NC}"
        fi
        
        if systemctl is-active chromium-kiosk >/dev/null 2>&1; then
            echo -e "  chromium-kiosk: ${GREEN}运行中${NC}"
        else
            echo -e "  chromium-kiosk: ${YELLOW}未运行${NC}"
        fi
        ;;
        
    *)
        echo -e "${RED}无效选择${NC}"
        exit 1
        ;;
esac

echo ""
echo "================================================"
echo -e "${GREEN}操作完成${NC}"
echo "================================================"