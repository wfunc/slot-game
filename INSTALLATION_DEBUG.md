# 安装脚本调试指南

## 问题：安装脚本卡住

如果安装脚本在执行过程中卡住，可以按照以下步骤调试：

### 1. 启用调试模式

运行安装脚本时设置DEBUG环境变量：

```bash
sudo DEBUG=1 ./install_v2.sh
```

这将显示详细的调试信息，帮助定位脚本卡在哪一步。

### 2. 常见卡住位置及解决方案

#### A. 卡在 "[✓] slot-game 服务启动成功" 后

**可能原因**：
- chromium-kiosk 服务启动超时
- systemctl 命令无响应
- 数据库优化命令卡住

**解决方案**：

1. 检查服务状态：
```bash
systemctl status slot-game
systemctl status chromium-kiosk
```

2. 查看服务日志：
```bash
journalctl -u slot-game -n 50
journalctl -u chromium-kiosk -n 50
```

3. 手动停止服务并重新运行安装：
```bash
sudo systemctl stop slot-game chromium-kiosk
sudo ./install_v2.sh
```

#### B. 数据库迁移超时

**症状**：
- 日志显示 "Migrating serial_logs" 后长时间无响应
- 数据库文件被锁定

**解决方案**：

1. 运行数据库修复脚本：
```bash
sudo ./scripts/fix_db_lock.sh /home/sg/slot-game/data/game.db
```

2. 手动优化数据库：
```bash
sudo systemctl stop slot-game
sqlite3 /home/sg/slot-game/data/game.db <<EOF
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 30000;
VACUUM;
EOF
sudo systemctl start slot-game
```

### 3. 脚本执行顺序

install_v2.sh 的执行顺序：

1. **检测安装类型** - 判断是新安装还是更新
2. **备份数据库**（仅更新时）
3. **修复数据库锁**（仅更新时）
4. **设置用户和目录**
5. **部署文件**
6. **配置服务**（创建systemd服务文件）
7. **优化数据库**（在服务启动前）
8. **启动服务**
   - 启动 slot-game
   - 启动 chromium-kiosk（如果有图形界面）
9. **显示安装摘要**

### 4. 超时保护

脚本中已添加以下超时保护：

- `systemctl` 命令：10秒超时
- `sqlite3` 优化命令：30秒超时
- `sqlite3` 查询命令：5秒超时
- `systemctl is-active` 检查：2秒超时

### 5. 手动调试步骤

如果脚本仍然卡住，可以手动执行每个步骤：

```bash
# 1. 停止所有服务
sudo systemctl stop slot-game chromium-kiosk

# 2. 清理数据库锁
rm -f /home/sg/slot-game/data/game.db-wal
rm -f /home/sg/slot-game/data/game.db-shm

# 3. 优化数据库
sqlite3 /home/sg/slot-game/data/game.db "PRAGMA journal_mode = WAL;"

# 4. 手动启动服务
sudo systemctl start slot-game
sleep 5
sudo systemctl start chromium-kiosk

# 5. 检查服务状态
systemctl status slot-game
systemctl status chromium-kiosk
```

### 6. 日志文件位置

- 安装脚本输出：标准输出
- slot-game 服务日志：`journalctl -u slot-game`
- chromium-kiosk 日志：`journalctl -u chromium-kiosk`
- 数据库文件：`/home/sg/slot-game/data/game.db`

### 7. 清理并重新安装

如果问题持续，可以完全清理并重新安装：

```bash
# 停止服务
sudo systemctl stop slot-game chromium-kiosk
sudo systemctl disable slot-game chromium-kiosk

# 备份数据
sudo cp -r /home/sg/slot-game/data /tmp/slot-game-backup

# 删除安装
sudo rm -rf /home/sg/slot-game
sudo rm -rf /home/ztl/slot-game-arm64
sudo rm -f /etc/systemd/system/slot-game.service
sudo rm -f /etc/systemd/system/chromium-kiosk.service

# 重新安装
sudo ./install_v2.sh

# 恢复数据（如果需要）
sudo cp /tmp/slot-game-backup/game.db /home/sg/slot-game/data/
sudo chown sg:sg /home/sg/slot-game/data/game.db
```

## 性能优化建议

### serial_logs 表优化

如果 serial_logs 表数据过多（超过10万条），建议定期清理：

```bash
# 保留最近30天的数据
sqlite3 /home/sg/slot-game/data/game.db "DELETE FROM serial_logs WHERE created_at < date('now','-30 day');"

# 执行VACUUM回收空间
sqlite3 /home/sg/slot-game/data/game.db "VACUUM;"
```

### 自动清理脚本

创建定时任务自动清理：

```bash
# 创建清理脚本
cat > /home/sg/slot-game/cleanup.sh <<'EOF'
#!/bin/bash
DB_PATH="/home/sg/slot-game/data/game.db"
sqlite3 "$DB_PATH" "DELETE FROM serial_logs WHERE created_at < date('now','-30 day');"
sqlite3 "$DB_PATH" "VACUUM;"
EOF

chmod +x /home/sg/slot-game/cleanup.sh

# 添加到crontab（每周日凌晨3点执行）
echo "0 3 * * 0 /home/sg/slot-game/cleanup.sh" | crontab -u sg -
```

## 联系支持

如果问题仍无法解决，请提供以下信息：

1. 安装脚本的DEBUG输出：`sudo DEBUG=1 ./install_v2.sh 2>&1 | tee install.log`
2. 服务日志：`journalctl -u slot-game -n 100 > slot-game.log`
3. 数据库信息：`sqlite3 game.db "SELECT COUNT(*) FROM serial_logs;"`
4. 系统信息：`uname -a` 和 `lsb_release -a`