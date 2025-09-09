# 🎰 老虎机游戏API快速开始指南

## ✅ API实现完成！

恭喜！老虎机游戏的核心API已经全部实现并可以运行了。

### 已完成功能
- ✅ **老虎机游戏API** - 开始、转动、结算、历史记录
- ✅ **钱包系统API** - 余额查询、充值、提现、交易记录
- ✅ **用户认证** - 注册、登录、JWT令牌
- ✅ **测试脚本** - 自动化API测试

## 🚀 快速启动

### 方式1：自动启动并测试
```bash
# 自动构建、启动服务器并运行测试
./scripts/start_and_test.sh
```

### 方式2：手动启动
```bash
# 1. 构建
make build

# 2. 启动服务器
./bin/server

# 3. 在新终端运行测试
./scripts/test_api.sh
```

### 方式3：使用Make命令
```bash
# 启动服务器（开发模式）
make run

# 运行测试
make test
```

## 📝 API端点列表

### 认证接口
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| POST | `/api/v1/auth/refresh` | 刷新令牌 |
| POST | `/api/v1/auth/logout` | 用户登出 |

### 老虎机游戏接口
| 方法 | 端点 | 描述 | 需要认证 |
|------|------|------|---------|
| POST | `/api/v1/slot/start` | 开始游戏 | ✅ |
| POST | `/api/v1/slot/spin` | 执行转动 | ✅ |
| POST | `/api/v1/slot/settle` | 结算游戏 | ✅ |
| GET | `/api/v1/slot/history` | 游戏历史 | ✅ |
| GET | `/api/v1/slot/session/:id` | 会话信息 | ✅ |
| GET | `/api/v1/slot/stats` | 用户统计 | ✅ |

### 钱包接口
| 方法 | 端点 | 描述 | 需要认证 |
|------|------|------|---------|
| GET | `/api/v1/wallet/balance` | 查询余额 | ✅ |
| POST | `/api/v1/wallet/deposit` | 充值（测试） | ✅ |
| POST | `/api/v1/wallet/withdraw` | 提现（模拟） | ✅ |
| GET | `/api/v1/wallet/transactions` | 交易记录 | ✅ |
| GET | `/api/v1/wallet/statistics` | 钱包统计 | ✅ |

## 🎮 游戏流程示例

### 1. 注册新用户
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "player001",
    "email": "player001@example.com",
    "password": "Test123456!"
  }'
```

### 2. 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "player001",
    "password": "Test123456!"
  }'

# 响应示例
{
  "code": "SUCCESS",
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_in": 3600
  }
}
```

### 3. 开始游戏
```bash
TOKEN="你的JWT令牌"

curl -X POST http://localhost:8080/api/v1/slot/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bet_amount": 100}'

# 响应示例
{
  "session_id": "slot_abc123_1234567890",
  "balance": 9900,
  "message": "游戏已开始，请执行转动"
}
```

### 4. 执行转动
```bash
SESSION_ID="slot_abc123_1234567890"

curl -X POST http://localhost:8080/api/v1/slot/spin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"$SESSION_ID\"}"

# 响应示例
{
  "result": {
    "symbols": [["🍒","🍋","🍊"], ["🍇","🍒","🍋"], ["🍊","🍇","🍒"]],
    "win_lines": [{"line": 1, "payout": 200}],
    "total_payout": 200
  },
  "balance": 10100,
  "state": "winning"
}
```

### 5. 结算游戏
```bash
curl -X POST http://localhost:8080/api/v1/slot/settle \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\": \"$SESSION_ID\"}"
```

## 🧪 Postman测试

1. **导入测试集合**
   - 打开Postman
   - 点击 Import
   - 选择文件：`docs/postman/slot-game-api.json`

2. **设置环境变量**
   - BASE_URL: `http://localhost:8080`
   - TOKEN: 登录后自动设置
   - SESSION_ID: 开始游戏后自动设置

3. **运行测试流程**
   - 先运行"注册"或"登录"获取令牌
   - 然后可以测试其他需要认证的接口

## 📊 数据库查看

```bash
# 使用SQLite命令行查看数据
sqlite3 data/slot_game.db

# 查看所有表
.tables

# 查看用户
SELECT * FROM users;

# 查看钱包
SELECT * FROM wallets;

# 查看游戏记录
SELECT * FROM game_results;

# 查看交易记录
SELECT * FROM transactions;
```

## 🔧 配置说明

配置文件：`config/config.yaml`

```yaml
server:
  host: 0.0.0.0
  port: 8080

database:
  driver: sqlite
  dsn: data/slot_game.db

game:
  session_timeout: 30m
  max_sessions: 1000
  initial_balance: 10000  # 新用户初始金币
```

## 🚨 注意事项

1. **测试环境**
   - 充值和提现功能仅为测试用途
   - 新用户自动获得10000金币初始余额

2. **安全性**
   - JWT令牌有效期为1小时
   - 所有游戏API需要认证
   - 密码要求：至少8位，包含大小写字母、数字和特殊字符

3. **性能限制**
   - 单用户最多1000个会话
   - 会话超时时间30分钟
   - API请求频率限制（待实现）

## 📈 下一步开发计划

### 短期（本周）
- [ ] WebSocket实时推送
- [ ] 游戏动画数据生成
- [ ] 管理后台API

### 中期（下周）
- [ ] 排行榜功能
- [ ] 活动系统
- [ ] VIP等级系统

### 长期（月度）
- [ ] 多种老虎机主题
- [ ] 社交功能
- [ ] 数据分析面板

## 🐛 问题排查

### 服务器无法启动
```bash
# 检查端口占用
lsof -i :8080

# 查看日志
tail -f logs/server.log
```

### 数据库连接失败
```bash
# 检查数据库文件
ls -la data/slot_game.db

# 重新初始化数据库
rm data/slot_game.db
./bin/server  # 会自动创建新数据库
```

### API认证失败
- 检查Token是否过期
- 确认Authorization头格式：`Bearer YOUR_TOKEN`
- 查看服务器日志中的具体错误

## 📞 联系支持

遇到问题？查看：
- 详细实现指南：`docs/development/API_IMPLEMENTATION_GUIDE.md`
- 后端TODO列表：`docs/development/backend-todo.md`
- 项目README：`README.md`

---

🎉 **恭喜！你的老虎机游戏API已经可以运行了！**