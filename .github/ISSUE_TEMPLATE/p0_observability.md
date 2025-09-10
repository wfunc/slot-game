---
name: P0 - 观测与稳定任务卡片
about: Prometheus 指标、限流/熔断、健康检查增强
title: "P0: 观测与稳定 - [子任务名]"
labels: [P0, backend, observability]
assignees: []
---

## 背景
来自 docs/development/backend-task-board.md E 区任务。

## 目标
- [ ] 暴露 /metrics（会话、spin 耗时、错误计数、WS 在线等）
- [ ] 接口限流（IP+账户+会话）
- [ ] 依赖熔断/降级（DB/串口/MQTT）
- [ ] 健康检查增强（DB ping/外部依赖）

## 验收标准
- [ ] Grafana Dashboard 示例可用
- [ ] 压测下限流与熔断有效
- [ ] 健康检查返回子系统状态

## 参考路径
- cmd/server/main.go（健康检查）
- internal/websocket/*（在线数）
- internal/game/*（会话与spin指标）

