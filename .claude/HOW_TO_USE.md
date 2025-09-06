# 🎯 Agent命令系统使用指南

## ⚠️ 重要说明
`/pm`、`/ui`、`/fe`、`/be` 等**不是**Claude Code的内置斜杠命令！
这些是我们设计的Agent调用约定，需要通过以下方式使用：

## ✅ 正确的使用方法

### 方法1：直接请求Agent角色
```
请基于 @agents/.claude/prompts/pm.md，作为产品经理帮我分析需求
```

### 方法2：简化请求（推荐）
```
作为产品经理（pm.md），帮我分析电商平台需求
```

### 方法3：使用我们的约定格式
当你想调用特定Agent时，直接说：
```
[PM] 分析用户管理系统需求
[UI] 设计登录页面
[FE] 实现用户认证组件
[BE] 设计用户API接口
```

## 📋 Agent角色对照表

| 角色标记 | 对应文件 | Agent角色 | 职责范围 |
|---------|---------|-----------|----------|
| [PM] | prompts/pm.md | 产品经理 | 需求分析、PRD、用户故事 |
| [UI] | prompts/designer.md | UI设计师 | 界面设计、用户体验 |
| [FE] | prompts/frontend.md | 前端工程师 | Web开发、组件实现 |
| [BE] | prompts/backend.md | 后端工程师 | API开发、数据库设计 |

## 💬 实际对话示例

### 示例1：产品需求分析
```
你：[PM] 帮我分析一个待办事项应用的需求

Claude：基于产品经理角色定义（@agents/.claude/prompts/pm.md），我来帮您分析待办事项应用的需求：

## 需求分析
### 目标用户
- 个人用户：需要管理日常任务
- 团队用户：需要协作完成项目

### 核心功能
1. 任务管理（创建、编辑、删除）
2. 分类和标签系统
3. 优先级设置
...
[按照PM角色规范输出PRD]
```

### 示例2：UI设计
```
你：[UI] 基于上述需求设计任务列表界面

Claude：基于UI设计师角色定义（@agents/.claude/prompts/designer.md），我来设计任务列表界面：

## 设计方案
### 视觉层级
- 卡片式布局展示任务
- 颜色编码表示优先级
...
[按照Designer角色规范输出设计]
```

### 示例3：前端开发
```
你：[FE] 用React实现任务卡片组件

Claude：基于前端工程师角色定义（@agents/.claude/prompts/frontend.md），我来实现任务卡片组件：

```typescript
import React from 'react';
// ... 组件实现
```
[按照Frontend角色规范输出代码]
```

## 🔄 完整工作流示例

```
1. 你：[PM] 分析在线教育平台需求
   Claude：[输出PRD文档]

2. 你：[UI] 基于PRD设计课程列表页
   Claude：[输出设计方案]

3. 你：[FE] 实现课程列表组件
   Claude：[输出React代码]

4. 你：[BE] 设计课程API
   Claude：[输出API规范]
```

## 🎭 角色切换说明

1. **明确指定**：使用[PM]、[UI]、[FE]、[BE]标记
2. **上下文保持**：Claude会记住当前角色直到切换
3. **协作模式**：可以在对话中切换不同角色

## 🚀 快速开始模板

```
# 产品经理任务
[PM] 帮我规划[项目名称]的MVP功能

# UI设计任务  
[UI] 设计[页面/组件名称]

# 前端开发任务
[FE] 用[React/Vue]实现[组件名称]

# 后端开发任务
[BE] 设计[功能模块]的RESTful API
```

## 📝 最佳实践

### ✅ 推荐做法
- 使用清晰的角色标记 [PM]、[UI] 等
- 按照需求→设计→开发的顺序
- 保持角色连贯性
- 引用前一个角色的输出

### ❌ 避免做法
- 不要直接输入 /pm（这不是内置命令）
- 不要在一个请求中混合多个角色
- 不要跳过必要的前置步骤

## 🆘 常见问题

**Q: 为什么 /pm 命令不工作？**
A: 这不是Claude Code的内置命令，请使用 [PM] 标记或直接请求角色。

**Q: 如何让Claude记住角色？**
A: 在对话开始时明确指定角色，Claude会保持该角色直到你切换。

**Q: 可以自定义新的Agent吗？**
A: 可以！在 prompts/ 目录下创建新的角色定义文件即可。

## 📚 参考资源

- 产品经理规范：`@agents/.claude/prompts/pm.md`
- UI设计师规范：`@agents/.claude/prompts/designer.md`
- 前端工程师规范：`@agents/.claude/prompts/frontend.md`
- 后端工程师规范：`@agents/.claude/prompts/backend.md`
- 协作流程：`@agents/workflows/collaboration.md`