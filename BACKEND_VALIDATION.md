# 后端功能验证清单

## 🎯 核心目标
确保后端API能够真正运行，提供完整的数据服务，而不是返回假数据或占位符。

## ✅ 必须实现的功能清单

### 1. 健康检查端点
```bash
# 必须有健康检查端点
GET /api/health
Response: 
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 2. CORS配置（重要！）
```bash
# 必须配置CORS，否则前端无法调用
Access-Control-Allow-Origin: * 或 http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

### 3. 完整的CRUD操作
每个资源都必须实现：
- **C**reate - POST /api/v1/resources
- **R**ead - GET /api/v1/resources 和 GET /api/v1/resources/:id
- **U**pdate - PUT /api/v1/resources/:id
- **D**elete - DELETE /api/v1/resources/:id

### 4. 列表接口功能
GET接口必须支持：
- **分页**: ?page=1&pageSize=10
- **搜索**: ?search=keyword
- **筛选**: ?status=active&category=electronics
- **排序**: ?sortBy=createdAt&order=desc

### 5. 数据持久化
- 至少使用SQLite或内存数据库
- 数据重启后不丢失（SQLite文件）
- 有初始测试数据

### 6. 错误处理
```json
// 400 Bad Request
{
  "error": "请求参数错误",
  "details": "金额必须大于0"
}

// 404 Not Found
{
  "error": "资源不存在"
}

// 500 Internal Server Error
{
  "error": "服务器内部错误"
}
```

## 📋 各语言启动验证

### Go项目验证
```bash
# 1. 检查go.mod是否存在
ls go.mod

# 2. 初始化（如果没有go.mod）
go mod init project-name

# 3. 安装依赖
go mod tidy

# 4. 运行项目
go run main.go

# 5. 测试健康检查
curl http://localhost:8080/api/health

# 6. 测试CORS（应该返回正确的headers）
curl -I -X OPTIONS http://localhost:8080/api/v1/orders
```

### Python项目验证
```bash
# 1. 创建虚拟环境
python -m venv venv

# 2. 激活虚拟环境
source venv/bin/activate  # Linux/Mac
venv\Scripts\activate      # Windows

# 3. 安装依赖
pip install -r requirements.txt

# 4. 运行项目
# FastAPI
uvicorn main:app --reload --host 0.0.0.0 --port 8080
# Flask
python app.py
# Django
python manage.py runserver 0.0.0.0:8080

# 5. 测试健康检查
curl http://localhost:8080/api/health
```

### Node.js项目验证
```bash
# 1. 安装依赖
npm install

# 2. 运行项目
npm run dev  # 或 npm start

# 3. 测试健康检查
curl http://localhost:8080/api/health

# 4. 测试API
curl http://localhost:8080/api/v1/orders
```

## 🔍 功能验证脚本

创建一个测试脚本 `test_api.sh`：

```bash
#!/bin/bash

API_URL="http://localhost:8080"

echo "===== API功能验证开始 ====="

# 1. 健康检查
echo "1. 测试健康检查..."
curl -s $API_URL/api/health | jq .
echo ""

# 2. 测试CORS
echo "2. 测试CORS配置..."
curl -I -X OPTIONS $API_URL/api/v1/orders 2>/dev/null | grep -i "access-control"
echo ""

# 3. 获取列表
echo "3. 测试获取列表..."
curl -s $API_URL/api/v1/orders | jq .
echo ""

# 4. 创建数据
echo "4. 测试创建数据..."
curl -s -X POST $API_URL/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "orderNo": "TEST001",
    "merchantName": "测试商户",
    "amount": 100.50
  }' | jq .
echo ""

# 5. 测试搜索
echo "5. 测试搜索功能..."
curl -s "$API_URL/api/v1/orders?search=TEST" | jq .
echo ""

# 6. 测试分页
echo "6. 测试分页功能..."
curl -s "$API_URL/api/v1/orders?page=1&pageSize=5" | jq .
echo ""

echo "===== API功能验证完成 ====="
```

## ⚠️ 常见错误及解决方案

### 1. CORS错误
**错误信息**: `Access to fetch at 'http://localhost:8080' from origin 'http://localhost:3000' has been blocked by CORS policy`

**解决方案**:
```go
// Go - Gin
import "github.com/gin-contrib/cors"
router.Use(cors.Default())
```

```python
# Python - FastAPI
from fastapi.middleware.cors import CORSMiddleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)
```

```javascript
// Node.js - Express
const cors = require('cors');
app.use(cors());
```

### 2. 数据库连接错误
**错误信息**: `dial tcp 127.0.0.1:5432: connect: connection refused`

**解决方案**:
- 使用SQLite代替PostgreSQL/MySQL（开发阶段）
- 确保数据库服务已启动
- 检查连接字符串配置

### 3. 依赖缺失错误
**Go错误**: `cannot find package`
```bash
go mod tidy  # 自动下载依赖
```

**Python错误**: `ModuleNotFoundError`
```bash
pip install -r requirements.txt
```

**Node.js错误**: `Cannot find module`
```bash
npm install
```

### 4. 端口占用错误
**错误信息**: `bind: address already in use`

**解决方案**:
```bash
# 查找占用端口的进程
lsof -i :8080  # Mac/Linux
netstat -ano | findstr :8080  # Windows

# 结束进程或更换端口
```

## 📊 API完整性评分标准

| 检查项 | 分值 | 说明 |
|--------|------|------|
| 健康检查端点 | 10分 | /api/health 返回200 |
| CORS配置正确 | 15分 | 前端能够访问 |
| CRUD完整性 | 30分 | 增删改查都能用 |
| 数据持久化 | 15分 | 重启后数据不丢 |
| 搜索功能 | 10分 | 支持关键词搜索 |
| 分页功能 | 10分 | 支持分页参数 |
| 错误处理 | 5分 | 返回有意义的错误信息 |
| 初始数据 | 5分 | 有测试数据 |

**评分等级**：
- A级（90-100分）：生产就绪
- B级（70-89分）：功能完整
- C级（50-69分）：基本可用
- D级（<50分）：需要改进

## 🚀 快速启动模板

### Go项目模板
```bash
# 创建项目
mkdir myapi && cd myapi

# 初始化模块
go mod init myapi

# 创建main.go（使用上面的完整示例）

# 安装依赖
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
go get gorm.io/gorm
go get gorm.io/driver/sqlite

# 运行
go run main.go
```

### Python项目模板
```bash
# 创建项目
mkdir myapi && cd myapi

# 创建虚拟环境
python -m venv venv
source venv/bin/activate

# 创建requirements.txt
cat > requirements.txt << EOF
fastapi==0.104.1
uvicorn[standard]==0.24.0
sqlalchemy==2.0.23
python-dotenv==1.0.0
EOF

# 安装依赖
pip install -r requirements.txt

# 创建main.py（使用上面的完整示例）

# 运行
uvicorn main:app --reload --host 0.0.0.0 --port 8080
```

## 💡 最佳实践建议

1. **先实现健康检查** - 这是最基本的端点
2. **使用SQLite开发** - 避免数据库配置问题
3. **初始化测试数据** - 方便前端开发调试
4. **必须配置CORS** - 否则前端无法调用
5. **使用统一的响应格式** - 前端更容易处理
6. **实现真实的业务逻辑** - 不要返回假数据
7. **添加请求日志** - 方便调试问题

## 🎯 验证成功标准

后端服务启动后，应该能够：
1. ✅ 访问 http://localhost:8080/api/health 返回健康状态
2. ✅ 前端能够调用API（无CORS错误）
3. ✅ 能创建新数据并保存
4. ✅ 能查询到创建的数据
5. ✅ 能更新和删除数据
6. ✅ 搜索和分页功能正常
7. ✅ 重启服务后数据还在

**记住：宁可功能简单但完整，不要功能复杂但不能用！**