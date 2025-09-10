---
name: P0 - 串口控制器集成
about: 将硬件串口控制纳入中奖链路，含降级与重试
title: "P0: 串口控制器集成 - [子任务名]"
labels: [P0, backend, hardware]
assignees: []
---

## 背景
来自 docs/development/backend-task-board.md C 区任务。

## 目标
- [ ] 中奖后触发推币（力度/时长可配）
- [ ] 命令队列与重试机制
- [ ] 串口异常的降级策略（mock/跳过）
- [ ] 设备状态 API（传感器、温度、错误计数）

## 验收标准
- [ ] 无串口环境下业务不阻塞
- [ ] 推币动作可观测（日志/指标）
- [ ] 错误回退与幂等保障

## 参考路径
- internal/hardware/serial.go
- internal/game/*（触发点）
- internal/api/*（新增设备状态端点）

