# 代码验证检查清单

## 🎯 目标
确保生成的代码能够一次性运行成功，减少用户遇到的报错问题。

## 📋 前端项目验证清单

### Vue 3 项目
```bash
# 启动前检查
✅ Node版本 >= 16.0.0
✅ 项目初始化方式正确（npm create vue@latest）
✅ package.json 中所有依赖都已声明
✅ vite.config.js 配置正确
✅ 路径别名配置（@/）
✅ 环境变量使用 import.meta.env（不是process.env）

# 常见问题排查
❌ 问题：Cannot find module 'xxx'
✅ 解决：确保 npm install 并检查 package.json

❌ 问题：[vite] Internal server error
✅ 解决：检查 vite.config.js 和导入路径

❌ 问题：组件未注册
✅ 解决：确保组件正确导入和注册
```

### React 项目
```bash
# 启动前检查
✅ React版本与React DOM版本匹配
✅ React Router版本匹配（v5或v6语法不同）
✅ TypeScript配置（tsconfig.json）
✅ 所有JSX文件正确导入React

# 常见问题排查
❌ 问题：Module not found
✅ 解决：检查相对路径和别名配置

❌ 问题：Hook调用错误
✅ 解决：确保在函数组件顶层调用
```

### 通用前端检查
```javascript
// 依赖检查脚本
{
  "scripts": {
    "predev": "npm ls",  // 检查依赖树
    "dev": "vite",
    "type-check": "tsc --noEmit"  // TypeScript项目
  }
}
```

## 📋 后端项目验证清单

### Go 项目
```bash
# 启动前检查
✅ Go版本 >= 1.20
✅ go.mod 文件存在
✅ 所有import的包都在go.mod中
✅ main函数存在
✅ 端口配置正确

# 正确的启动流程
go mod init project-name    # 如果没有go.mod
go mod tidy                 # 整理依赖
go run main.go             # 运行

# 常见问题排查
❌ 问题：package xxx is not in GOROOT
✅ 解决：go mod tidy 或 go get xxx

❌ 问题：cannot find module providing package
✅ 解决：检查go.mod和import路径
```

### Python 项目
```bash
# 启动前检查
✅ Python版本 >= 3.8
✅ requirements.txt 文件完整
✅ 虚拟环境激活
✅ 所有import的包都在requirements.txt中

# 正确的启动流程
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
python main.py  # 或 uvicorn main:app --reload

# 常见问题排查
❌ 问题：ModuleNotFoundError
✅ 解决：pip install缺失的模块

❌ 问题：ImportError
✅ 解决：检查Python路径和模块结构
```

### Node.js 项目
```bash
# 启动前检查
✅ Node版本 >= 16
✅ package.json 完整
✅ 统一使用ES6或CommonJS（不混用）
✅ 环境变量配置（.env）

# 正确的启动流程
npm install
npm run dev  # 或 node server.js

# 常见问题排查
❌ 问题：Cannot use import statement
✅ 解决：package.json添加 "type": "module"

❌ 问题：CORS错误
✅ 解决：配置CORS中间件
```

## 🔍 通用验证步骤

### 1. 依赖验证
```bash
# 前端
npm ls          # 检查依赖树
npm audit       # 安全审计

# Python
pip list        # 查看已安装包
pip check       # 检查依赖完整性

# Go
go mod verify   # 验证依赖
go mod tidy     # 清理依赖
```

### 2. 配置文件验证
```yaml
# .env.example（所有项目都应该有）
# 前端
VITE_API_URL=http://localhost:8080/api
VITE_APP_TITLE=My App

# 后端
DATABASE_URL=postgresql://user:pass@localhost/db
REDIS_URL=redis://localhost:6379
PORT=8080
JWT_SECRET=change-this-secret
```

### 3. 健康检查端点
```javascript
// 所有后端项目都应该有
GET /api/health
Response: { "status": "healthy", "timestamp": "..." }
```

## 🚀 最佳实践

### 生成代码时必须包含
1. **完整的package.json/requirements.txt/go.mod**
2. **环境变量示例文件（.env.example）**
3. **README.md 包含启动步骤**
4. **错误处理和日志**
5. **CORS配置（后端）**

### 验证命令集合
```json
// package.json 示例
{
  "scripts": {
    "preinstall": "node -v",  // 检查Node版本
    "postinstall": "npm ls",  // 验证依赖安装
    "predev": "npm run lint", // 开发前检查
    "dev": "vite",
    "test": "vitest",
    "type-check": "tsc --noEmit"
  }
}
```

## ⚠️ 常见陷阱

### 前端陷阱
- Vue 2 vs Vue 3 API完全不同
- React 18的并发特性需要特殊处理
- Vite和Webpack配置不兼容
- TypeScript严格模式问题

### 后端陷阱
- Go的GOPATH vs Go Modules
- Python的全局包vs虚拟环境
- Node.js的ES6 vs CommonJS
- 数据库连接字符串格式
- CORS配置遗漏

## 📝 Agent自检承诺

作为专业的工程师Agent，我承诺：

1. ✅ 生成的代码包含所有必要的依赖声明
2. ✅ 提供清晰的启动步骤和命令
3. ✅ 包含错误处理和友好的错误提示
4. ✅ 配置文件示例完整且可用
5. ✅ 代码经过基本的语法和依赖验证
6. ✅ 提供常见问题的解决方案

## 🔧 调试技巧

### 快速定位问题
```bash
# 前端
npm run dev -- --debug      # Vite调试模式
npm ls packageName          # 检查特定包

# Python
python -m pip show package  # 查看包信息
python -c "import sys; print(sys.path)"  # Python路径

# Go
go env                      # Go环境变量
go list -m all             # 列出所有模块
```

### 版本兼容性检查
```javascript
// 检查脚本 check-versions.js
const pkg = require('./package.json');
console.log('Dependencies versions:');
Object.entries(pkg.dependencies).forEach(([name, version]) => {
  console.log(`${name}: ${version}`);
});
```

---

💡 **记住**：宁可多花时间验证，也不要让用户遇到运行错误！