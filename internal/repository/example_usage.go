package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
)

// ExampleUsage 展示如何使用仓储和事务
type ExampleUsage struct {
	manager *Manager
}

// NewExampleUsage 创建示例
func NewExampleUsage(manager *Manager) *ExampleUsage {
	return &ExampleUsage{manager: manager}
}

// Example1_SimpleTransaction 简单事务示例
func (e *ExampleUsage) Example1_SimpleTransaction(ctx context.Context, userID uint, betAmount int64) error {
	// 在事务中执行多个操作
	return e.manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 1. 创建游戏会话
		session := &models.GameSession{
			SessionID: fmt.Sprintf("session_%d", time.Now().Unix()),
			UserID:    userID,
			GameID:    1,
			Status:    "playing",
			StartedAt: time.Now(),
		}
		
		if err := tx.GameSession().Create(ctx, session); err != nil {
			return fmt.Errorf("创建游戏会话失败: %w", err)
		}
		
		// 2. 记录游戏结果
		result := &models.GameResult{
			SessionID:  session.ID,
			GameID:     session.GameID,
			UserID:     session.UserID,
			RoundID:    fmt.Sprintf("round_%d", time.Now().Unix()),
			BetAmount:  betAmount,
			WinAmount:  betAmount * 2, // 假设赢了2倍
			Multiplier: 2.0,
			PlayedAt:   time.Now(),
		}
		
		if err := tx.GameResult().Create(ctx, result); err != nil {
			return fmt.Errorf("记录游戏结果失败: %w", err)
		}
		
		// 3. 更新会话统计
		updates := map[string]interface{}{
			"total_rounds": 1,
			"total_bet":    betAmount,
			"total_win":    result.WinAmount,
		}
		
		if err := tx.GameSession().UpdateBySessionID(ctx, session.SessionID, updates); err != nil {
			return fmt.Errorf("更新会话统计失败: %w", err)
		}
		
		// 所有操作成功，事务将自动提交
		return nil
	})
}

// Example2_ReadOnlyTransaction 只读事务示例
func (e *ExampleUsage) Example2_ReadOnlyTransaction(ctx context.Context, userID uint) (*GameStatistics, error) {
	var stats *GameStatistics
	
	// 使用只读事务提高性能
	err := e.manager.WithReadOnlyTransaction(ctx, func(tx *Transaction) error {
		var err error
		
		// 获取用户的游戏统计
		startTime := time.Now().AddDate(0, -1, 0) // 最近一个月
		endTime := time.Now()
		
		stats, err = tx.GameSession().GetStatistics(ctx, userID, startTime, endTime)
		if err != nil {
			return fmt.Errorf("获取游戏统计失败: %w", err)
		}
		
		return nil
	})
	
	return stats, err
}

// Example3_SavePointUsage 使用保存点的示例
func (e *ExampleUsage) Example3_SavePointUsage(ctx context.Context) error {
	return e.manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 第一步操作
		device1 := &models.DeviceStatus{
			DeviceID:   "device_001",
			DeviceName: "老虎机1号",
			Type:       "slot_machine",
			Status:     "online",
			LastPingAt: time.Now(),
		}
		
		if err := tx.DeviceStatus().Create(ctx, device1); err != nil {
			return err
		}
		
		// 创建保存点
		if err := tx.SavePoint("after_device1"); err != nil {
			return err
		}
		
		// 第二步操作（可能失败）
		device2 := &models.DeviceStatus{
			DeviceID:   "device_002",
			DeviceName: "老虎机2号",
			Type:       "slot_machine",
			Status:     "online",
			LastPingAt: time.Now(),
		}
		
		if err := tx.DeviceStatus().Create(ctx, device2); err != nil {
			// 回滚到保存点，device1仍然会被保存
			tx.RollbackToSavePoint("after_device1")
			// 可以选择继续其他操作或返回错误
		}
		
		// 第三步操作
		config := &models.SystemConfig{
			Key:   "maintenance_mode",
			Value: "false",
			Type:  "bool",
		}
		
		if err := tx.SystemConfig().Set(ctx, config.Key, config.Value, "系统维护模式"); err != nil {
			return err
		}
		
		return nil
	})
}

// Example4_UnitOfWork 工作单元模式示例
func (e *ExampleUsage) Example4_UnitOfWork(ctx context.Context) error {
	// 创建工作单元
	uow := NewUnitOfWork(e.manager)
	
	// 注册操作1：创建游戏会话
	uow.Register(func(tx *Transaction) error {
		session := &models.GameSession{
			SessionID: "uow_session_001",
			UserID:    1,
			GameID:    1,
			Status:    "playing",
			StartedAt: time.Now(),
		}
		return tx.GameSession().Create(ctx, session)
	})
	
	// 注册操作2：更新设备状态
	uow.Register(func(tx *Transaction) error {
		return tx.DeviceStatus().UpdateStatus(ctx, "device_001", "maintenance", nil)
	})
	
	// 注册操作3：记录系统配置
	uow.Register(func(tx *Transaction) error {
		return tx.SystemConfig().Set(ctx, "last_maintenance", time.Now().Format(time.RFC3339), "最后维护时间")
	})
	
	// 提交所有操作（在一个事务中）
	if err := uow.Commit(ctx); err != nil {
		return fmt.Errorf("工作单元提交失败: %w", err)
	}
	
	// 清除已完成的操作
	uow.Clear()
	
	return nil
}

// Example5_BatchOperator 批量操作示例
func (e *ExampleUsage) Example5_BatchOperator(ctx context.Context) error {
	batch := NewBatchOperator(e.manager)
	
	// 创建游戏会话和多个游戏结果
	session := &models.GameSession{
		SessionID: "batch_session_001",
		UserID:    1,
		GameID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	
	results := []*models.GameResult{
		{
			GameID:     1,
			UserID:     1,
			RoundID:    "round_001",
			BetAmount:  100,
			WinAmount:  200,
			Multiplier: 2.0,
			PlayedAt:   time.Now(),
		},
		{
			GameID:     1,
			UserID:     1,
			RoundID:    "round_002",
			BetAmount:  100,
			WinAmount:  0,
			Multiplier: 0,
			PlayedAt:   time.Now().Add(1 * time.Minute),
		},
		{
			GameID:     1,
			UserID:     1,
			RoundID:    "round_003",
			BetAmount:  100,
			WinAmount:  500,
			Multiplier: 5.0,
			PlayedAt:   time.Now().Add(2 * time.Minute),
		},
	}
	
	// 批量创建（在一个事务中）
	if err := batch.CreateGameSessionWithResults(ctx, session, results); err != nil {
		return fmt.Errorf("批量创建失败: %w", err)
	}
	
	return nil
}

// Example6_ComplexTransaction 复杂事务示例：游戏结算
func (e *ExampleUsage) Example6_ComplexTransaction(ctx context.Context, sessionID string) error {
	helper := NewTransactionHelper(e.manager.Transaction())
	
	// 使用带重试的事务（处理死锁等情况）
	return helper.RunWithRetry(ctx, 3, func(tx *Transaction) error {
		// 1. 获取游戏会话
		session, err := tx.GameSession().FindBySessionID(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("获取游戏会话失败: %w", err)
		}
		
		// 2. 获取该会话的所有游戏结果
		pagination := NewPagination(1, 100)
		results, err := tx.GameResult().FindBySessionID(ctx, session.ID, pagination)
		if err != nil {
			return fmt.Errorf("获取游戏结果失败: %w", err)
		}
		
		// 3. 计算统计数据
		var totalBet, totalWin int64
		var peakWin int64
		
		for _, result := range results {
			totalBet += result.BetAmount
			totalWin += result.WinAmount
			if result.WinAmount > peakWin {
				peakWin = result.WinAmount
			}
		}
		
		// 4. 更新会话统计
		now := time.Now()
		duration := int(now.Sub(session.StartedAt).Seconds())
		
		updates := map[string]interface{}{
			"status":       "ended",
			"ended_at":     &now,
			"total_rounds": len(results),
			"total_bet":    totalBet,
			"total_win":    totalWin,
			"peak_win":     peakWin,
			"duration":     duration,
		}
		
		if err := tx.GameSession().UpdateBySessionID(ctx, sessionID, updates); err != nil {
			return fmt.Errorf("更新会话失败: %w", err)
		}
		
		// 5. 记录系统日志（假设有日志仓储）
		logEntry := fmt.Sprintf("会话 %s 结算完成: 总投注=%d, 总赢取=%d, 净利润=%d",
			sessionID, totalBet, totalWin, totalWin-totalBet)
		
		// 这里可以调用日志仓储记录日志
		_ = logEntry
		
		// 6. 如果是大奖，更新系统配置
		if peakWin > 10000 { // 假设超过10000是大奖
			jackpotKey := fmt.Sprintf("last_jackpot_%d", session.UserID)
			if err := tx.SystemConfig().Set(ctx, jackpotKey, now.Format(time.RFC3339), "最后大奖时间"); err != nil {
				return fmt.Errorf("更新大奖记录失败: %w", err)
			}
		}
		
		return nil
	})
}

// Example7_HealthCheck 健康检查示例（使用多个仓储）
func (e *ExampleUsage) Example7_HealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceStatus),
	}
	
	// 检查设备状态
	deviceReport, err := e.manager.DeviceStatus().GetHealthReport(ctx)
	if err != nil {
		result.Services["devices"] = ServiceStatus{
			Name:    "设备监控",
			Status:  "error",
			Message: err.Error(),
		}
	} else {
		status := "healthy"
		if deviceReport.OfflineDevices > 0 {
			status = "degraded"
		}
		result.Services["devices"] = ServiceStatus{
			Name:    "设备监控",
			Status:  status,
			Message: fmt.Sprintf("在线: %d, 离线: %d", deviceReport.OnlineDevices, deviceReport.OfflineDevices),
			Details: deviceReport,
		}
	}
	
	// 检查游戏会话
	activeSessions := 0
	// 这里可以添加查询活跃会话的逻辑
	
	result.Services["games"] = ServiceStatus{
		Name:    "游戏服务",
		Status:  "healthy",
		Message: fmt.Sprintf("活跃会话: %d", activeSessions),
	}
	
	// 检查系统配置
	_, err = e.manager.SystemConfig().Get(ctx, "system_version")
	if err != nil {
		result.Services["config"] = ServiceStatus{
			Name:    "配置服务",
			Status:  "error",
			Message: "无法读取系统配置",
		}
	} else {
		result.Services["config"] = ServiceStatus{
			Name:    "配置服务",
			Status:  "healthy",
			Message: "配置正常",
		}
	}
	
	// 计算整体状态
	result.OverallStatus = "healthy"
	for _, service := range result.Services {
		if service.Status == "error" {
			result.OverallStatus = "error"
			break
		} else if service.Status == "degraded" && result.OverallStatus != "error" {
			result.OverallStatus = "degraded"
		}
	}
	
	return result, nil
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Timestamp     time.Time                 `json:"timestamp"`
	OverallStatus string                    `json:"overall_status"`
	Services      map[string]ServiceStatus  `json:"services"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name    string      `json:"name"`
	Status  string      `json:"status"` // healthy, degraded, error
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}