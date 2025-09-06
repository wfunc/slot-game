# Multi-Agent协作系统使用指南

## ⚠️ 重要提示
`/pm`、`/ui` 等**不是**Claude Code内置命令！请使用 **[PM]**、**[UI]** 标记来调用Agent角色。
详细说明请查看：`@agents/.claude/HOW_TO_USE.md`

## 🎯 系统概述

这是一个基于Claude的多Agent协作开发系统，通过模拟不同角色（产品经理、UI设计师、前端工程师、后端工程师）的专业Agent，实现从需求分析到产品交付的完整开发流程。

📖 **查看完整团队介绍**：[TEAM_OVERVIEW.md](./TEAM_OVERVIEW.md)

## 📂 目录结构

```
agents/
├── CLAUDE.md           # 全局配置文件（核心）
├── prompts/            # 各Agent的专用prompt
│   ├── pm.md          # 产品经理
│   ├── designer.md    # UI设计师
│   ├── frontend.md    # 前端工程师
│   └── backend.md     # 后端工程师
├── workflows/          # 工作流程定义
│   └── collaboration.md # 协作流程
└── README.md          # 使用指南（本文件）
```

## 🚀 快速开始

### 1. 基本使用方法

#### 方法一：直接引用prompt
在与Claude对话时，直接要求其扮演特定角色：
```
请你作为产品经理，基于agents/prompts/pm.md的角色定义，帮我分析以下需求...
```

#### 方法二：加载完整配置
```
请加载agents/CLAUDE.md配置，我需要启动一个完整的开发项目
```

#### 方法三：分阶段协作
```
# 需求阶段
基于agents/prompts/pm.md，作为产品经理分析需求

# 设计阶段
基于agents/prompts/designer.md，作为UI设计师设计界面

# 开发阶段
基于agents/prompts/frontend.md和backend.md，实现功能
```

## 💡 实际使用案例

### 案例1：开发一个待办事项应用

```markdown
步骤1 - 需求分析：
"作为产品经理（参考agents/prompts/pm.md），帮我分析一个待办事项应用的需求"

步骤2 - UI设计：
"作为UI设计师（参考agents/prompts/designer.md），基于上述需求设计界面"

步骤3 - 前端开发：
"作为前端工程师（参考agents/prompts/frontend.md），实现待办事项的React组件"

步骤4 - 后端开发：
"作为后端工程师（参考agents/prompts/backend.md），设计待办事项的API"
```

### 案例2：团队协作模式

```markdown
# 启动完整团队
"加载agents/CLAUDE.md，启动多Agent协作模式，我要开发一个在线商城"

# Claude会自动：
1. PM分析需求，输出PRD
2. Designer设计UI，输出设计稿
3. Backend设计API，输出接口文档
4. Frontend实现界面，输出前端代码
```

## 📋 Agent能力矩阵

| Agent | 核心能力 | 主要输出 | 协作对象 |
|-------|---------|---------|----------|
| **产品经理** | 需求分析、用户研究、项目规划 | PRD、用户故事、路线图 | 全员 |
| **UI设计师** | 界面设计、交互设计、视觉设计 | 设计稿、设计规范、组件库 | PM、前端 |
| **前端工程师** | Web开发、组件开发、性能优化 | 前端代码、组件、文档 | 设计师、后端 |
| **后端工程师** | API开发、数据库设计、系统架构 | 后端代码、API文档、数据库 | 前端、运维 |

## 🔧 高级用法

### 1. 自定义Agent角色

创建新的Agent prompt：
```markdown
# agents/prompts/devops.md
你是一位DevOps工程师...
[定义角色能力和规范]
```

### 2. 修改协作流程

编辑 `workflows/collaboration.md` 来调整团队协作方式：
- 修改阶段定义
- 调整任务分配
- 自定义交付标准

### 3. 集成到项目

将此系统集成到实际项目：
```bash
# 1. 复制agents目录到项目根目录
cp -r agents /your-project/

# 2. 在项目中引用
"基于./agents/CLAUDE.md配置开始开发"
```

## 📝 使用建议

### Do's ✅
1. **明确角色**：每次对话明确当前Agent角色
2. **遵循流程**：按照需求→设计→开发→测试的流程
3. **保持连贯**：在同一对话中保持角色和上下文连贯
4. **文档输出**：要求输出符合各角色规范的文档

### Don'ts ❌
1. **角色混淆**：避免在同一任务中频繁切换角色
2. **跳过阶段**：不要跳过需求分析直接开发
3. **忽视协作**：不要忽视Agent间的协作关系

## 🎓 学习路径

1. **入门**：先熟悉单个Agent的使用
2. **进阶**：尝试两个Agent的协作（如PM+Designer）
3. **高级**：运行完整的团队协作流程
4. **专家**：自定义和扩展Agent系统

## 🤔 常见问题

### Q1: 如何让Claude记住角色设定？
**A**: 在对话开始时明确加载特定的prompt文件，或使用"请继续作为[角色]"来保持角色。

### Q2: 可以同时使用多个Agent吗？
**A**: 可以，但建议在不同的任务阶段切换Agent，而不是同时扮演多个角色。

### Q3: 如何评估Agent的输出质量？
**A**: 参考各Agent prompt中定义的输出规范和质量标准。

### Q4: 可以修改Agent的行为吗？
**A**: 可以，直接编辑对应的prompt文件来调整Agent的行为和输出。

## 📊 效果评估

使用此系统后，你应该能够：
- ✅ 获得专业的需求分析文档
- ✅ 得到符合规范的设计方案
- ✅ 生成高质量的前后端代码
- ✅ 实现高效的团队协作流程

## 🔄 持续优化

1. **收集反馈**：记录使用过程中的问题
2. **优化prompt**：根据实际效果调整prompt
3. **扩展能力**：添加新的Agent角色
4. **分享经验**：与团队分享最佳实践

## 📚 扩展资源

- [Claude官方文档](https://docs.anthropic.com/claude/docs)
- [Prompt工程最佳实践](https://www.promptingguide.ai/)
- [敏捷开发方法论](https://agilemanifesto.org/)
- [设计系统指南](https://designsystems.com/)

---

💬 **提示**：开始使用时，可以从简单的单Agent任务开始，逐步尝试更复杂的协作场景。记住，这个系统的核心价值在于模拟真实的开发团队协作，让AI能够从不同专业角度提供帮助。

🚦 **快速启动命令**：
```
"加载agents/CLAUDE.md，我要开发[你的项目名称]"
```