# 后端任务看板（执行版）

本看板与 docs/development/backend-todo.md 对齐，收敛为可执行任务清单，含优先级、验收标准与归档路径。

## 使用说明
- 状态：Todo | Doing | Blocked | Review | Done
- 优先级：P0（必须本周）/ P1（下周）/ P2（后续）
- 交付物：PR 链接或提交记录，必要时补充设计说明/测试报告

## A. 恢复与持久化（P0）
- [ ] P0 RedisStatePersister 实现（Save/Load/Delete/TTL）
  - 路径：internal/game/state_persistence.go
  - 验收：新增单测覆盖正常/过期/异常分支；GameService 在 Redis 缓存命中时可恢复
- [ ] P0 Ready 超时退款流程
  - 路径：internal/game/recovery.go（recoverReady）
  - 验收：超时触发退款交易，状态回 Idle，含事务一致性
- [ ] P0 Settlement 完成性确认
  - 路径：internal/game/recovery.go（recoverSettlement）
  - 验收：重复结算无副作用，确保幂等
- [ ] P1 清理过期会话（批量）
  - 路径：internal/game/recovery.go + persister 接口扩展
  - 验收：定时任务清理 N 分钟未更新会话，记录日志

## B. 认证与账号（P0）
- [ ] P0 重置密码：令牌存储/校验
  - 路径：internal/service/auth_service.go（TODO 段）
  - 验收：请求重置→生成令牌→验证→修改密码→撤销会话，含单测
- [ ] P0 重置密码：邮件发送
  - 交付：抽象 Mailer 接口 + 简易实现（日志/SMTP 可选）
- [ ] P1 邮箱验证流程
  - 验收：注册后可验证邮箱，支持重新发送
- [ ] P2 OAuth 登录/绑定（可选）

## C. 通信与设备（P0）
- [ ] P0 串口控制器纳入中奖链路
  - 路径：internal/hardware/serial.go + internal/game/*（触发推币）
  - 验收：中奖后推币（可配置力度/时长），异常自动回退；无串口时降级
- [ ] P1 命令队列/重试/降级策略
  - 验收：串口发送失败重试 N 次，队列可观测
- [ ] P1 设备状态 API
  - 新增：GET /api/v1/device/status，返回传感器/温度/错误计数
- [ ] P1 MQTT 客户端接入 + 主题约定 + 权限校验

## D. 管理与运营（P1）
- [ ] P1 管理用户：列表/封禁
- [ ] P1 系统统计与 RTP 趋势（最小闭环）
- [ ] P1 日志查询导出（按时间窗口）

## E. 观测与稳定（P0）
- [ ] P0 Prometheus 指标 + /metrics
  - 覆盖：会话活跃数、spin 耗时直方图、错误计数、RTP 指标、WS 在线数
- [ ] P0 限流（IP+账户+会话）与熔断/降级
- [ ] P1 健康检查增强（DB ping/外部依赖）

## F. 测试与交付（P0）
- [ ] P0 WebSocket E2E 测试（断线重连/批量转动通知）
- [ ] P0 API 集成测试（Start/Spin/Batch/Settle 正常边界与错误）
- [ ] P1 通信模拟测试（串口/MQTT mock）
- [ ] P1 负载/稳定性压测（清单与阈值）
- [ ] P0 OpenAPI 文档维护（本文件与接口一致）

## 里程碑对齐（建议）
- M3：恢复与持久化 + 指标/限流（P0 完成）
- M4：串口集成 + 管理最小闭环 + 集成测试（P0/P1 完成）
- M5：MQTT/观测完善 + 压测/部署手册

