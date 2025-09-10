---
name: P0 - 恢复与持久化任务卡片
about: RedisStatePersister/退款/结算幂等/过期清理
title: "P0: 恢复与持久化 - [子任务名]"
labels: [P0, backend, recovery]
assignees: []
---

## 背景
来自 docs/development/backend-task-board.md A 区任务。

## 目标
- [ ] 按子任务拆解：
  - [ ] RedisStatePersister 实现（Save/Load/Delete/TTL）
  - [ ] Ready 状态超时退款
  - [ ] Settlement 幂等性验证
  - [ ] 批量清理过期会话（接口扩展）

## 验收标准
- [ ] 单测覆盖正常/过期/异常场景
- [ ] 退款产生正确交易，事务一致
- [ ] 重复结算无副作用
- [ ] 周期清理记录日志

## 参考路径
- internal/game/state_persistence.go
- internal/game/recovery.go

## 关联
- 文档：docs/development/backend-task-board.md

