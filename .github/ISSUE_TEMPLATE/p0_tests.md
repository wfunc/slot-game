---
name: P0 - 集成与E2E测试
about: WebSocket E2E、API 集成测试、通信模拟
title: "P0: 测试与交付 - [子任务名]"
labels: [P0, backend, testing]
assignees: []
---

## 背景
来自 docs/development/backend-task-board.md F 区任务。

## 目标
- [ ] WebSocket E2E（断线重连、批量结果推送）
- [ ] API 集成测试（Start/Spin/Batch/Settle 正常+异常）
- [ ] 通信模拟（串口/MQTT mock）
- [ ] OpenAPI 校验（openapi.yaml 与实现一致）

## 验收标准
- [ ] CI 中可执行，失败可定位
- [ ] 覆盖率达项目约定阈值

## 参考路径
- test/test_features.sh（可扩展）
- docs/api/openapi.yaml

