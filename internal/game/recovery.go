package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RecoveryManager 游戏恢复管理器
type RecoveryManager struct {
	logger    *zap.Logger
	persister StatePersister
	timeout   time.Duration // 会话超时时间
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager(logger *zap.Logger, persister StatePersister, timeout time.Duration) *RecoveryManager {
	return &RecoveryManager{
		logger:    logger,
		persister: persister,
		timeout:   timeout,
	}
}

// RecoverSession 恢复游戏会话
func (rm *RecoveryManager) RecoverSession(ctx context.Context, sessionID string) (*StateMachine, error) {
	// 从持久化存储加载状态
	stateData, err := rm.persister.Load(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("加载会话状态失败: %w", err)
	}
	
	// 检查会话是否超时
	if time.Since(stateData.LastUpdate) > rm.timeout {
		rm.logger.Warn("会话已超时",
			zap.String("session_id", sessionID),
			zap.Time("last_update", stateData.LastUpdate),
			zap.Duration("timeout", rm.timeout))
		
		// 删除超时的会话
		if err := rm.persister.Delete(ctx, sessionID); err != nil {
			rm.logger.Error("删除超时会话失败", zap.Error(err))
		}
		
		return nil, errors.New("会话已超时")
	}
	
	// 创建新的状态机并恢复状态
	sm := NewStateMachine(sessionID, stateData.UserID, rm.logger, rm.persister)
	sm.LoadFromData(stateData)
	
	// 根据当前状态决定恢复策略
	recoveryStrategy := rm.getRecoveryStrategy(stateData.CurrentState)
	if err := recoveryStrategy(ctx, sm); err != nil {
		return nil, fmt.Errorf("执行恢复策略失败: %w", err)
	}
	
	rm.logger.Info("会话恢复成功",
		zap.String("session_id", sessionID),
		zap.String("state", string(stateData.CurrentState)))
	
	return sm, nil
}

// getRecoveryStrategy 根据状态获取恢复策略
func (rm *RecoveryManager) getRecoveryStrategy(state GameState) func(context.Context, *StateMachine) error {
	strategies := map[GameState]func(context.Context, *StateMachine) error{
		StateIdle:        rm.recoverIdle,
		StateReady:       rm.recoverReady,
		StateSpinning:    rm.recoverSpinning,
		StateCalculating: rm.recoverCalculating,
		StateWinning:     rm.recoverWinning,
		StateSettlement:  rm.recoverSettlement,
		StateError:       rm.recoverError,
	}
	
	if strategy, exists := strategies[state]; exists {
		return strategy
	}
	
	// 默认策略：重置到待机状态
	return rm.recoverToIdle
}

// recoverIdle 恢复待机状态
func (rm *RecoveryManager) recoverIdle(ctx context.Context, sm *StateMachine) error {
	// 待机状态无需特殊处理
	return nil
}

// recoverReady 恢复准备状态
func (rm *RecoveryManager) recoverReady(ctx context.Context, sm *StateMachine) error {
	// 检查投注金额是否有效
	if sm.betAmount <= 0 {
		rm.logger.Warn("准备状态但投注金额无效，重置到待机",
			zap.String("session_id", sm.sessionID))
		return sm.Trigger(ctx, "cancel")
	}
	
	// 检查是否超过最大等待时间（例如5分钟）
	if time.Since(sm.lastUpdate) > 5*time.Minute {
		rm.logger.Info("准备状态超时，退还投注",
			zap.String("session_id", sm.sessionID))
		// TODO: 触发退款流程
		return sm.Trigger(ctx, "timeout")
	}
	
	return nil
}

// recoverSpinning 恢复转动状态
func (rm *RecoveryManager) recoverSpinning(ctx context.Context, sm *StateMachine) error {
	rm.logger.Info("从转动状态恢复，继续到计算阶段",
		zap.String("session_id", sm.sessionID))
	
	// 转动中断，直接进入计算阶段
	// 这里应该根据之前的随机数种子重新生成结果
	return sm.Trigger(ctx, "stop_spin")
}

// recoverCalculating 恢复计算状态
func (rm *RecoveryManager) recoverCalculating(ctx context.Context, sm *StateMachine) error {
	rm.logger.Info("从计算状态恢复",
		zap.String("session_id", sm.sessionID))
	
	// 重新计算结果
	if sm.winAmount > 0 {
		return sm.Trigger(ctx, "show_win")
	}
	return sm.Trigger(ctx, "no_win")
}

// recoverWinning 恢复中奖展示状态
func (rm *RecoveryManager) recoverWinning(ctx context.Context, sm *StateMachine) error {
	rm.logger.Info("从中奖展示状态恢复，进入结算",
		zap.String("session_id", sm.sessionID),
		zap.Int64("win_amount", sm.winAmount))
	
	// 直接进入结算
	return sm.Trigger(ctx, "settle")
}

// recoverSettlement 恢复结算状态
func (rm *RecoveryManager) recoverSettlement(ctx context.Context, sm *StateMachine) error {
	rm.logger.Info("从结算状态恢复，完成游戏",
		zap.String("session_id", sm.sessionID))
	
	// 确保结算完成，然后结束游戏
	// TODO: 检查结算是否真的完成
	return sm.Trigger(ctx, "finish")
}

// recoverError 恢复错误状态
func (rm *RecoveryManager) recoverError(ctx context.Context, sm *StateMachine) error {
	rm.logger.Warn("从错误状态恢复",
		zap.String("session_id", sm.sessionID),
		zap.String("error", sm.errorMsg))
	
	// 尝试恢复到待机状态
	return sm.Trigger(ctx, "recover")
}

// recoverToIdle 默认恢复策略：重置到待机
func (rm *RecoveryManager) recoverToIdle(ctx context.Context, sm *StateMachine) error {
	rm.logger.Warn("使用默认恢复策略，重置到待机状态",
		zap.String("session_id", sm.sessionID),
		zap.String("from_state", string(sm.GetState())))
	
	sm.Reset()
	return nil
}

// CleanupExpiredSessions 清理过期会话（定期任务）
func (rm *RecoveryManager) CleanupExpiredSessions(ctx context.Context) error {
	// TODO: 实现批量清理过期会话
	// 这需要在持久化层添加批量查询和删除的接口
	rm.logger.Info("开始清理过期会话")
	
	// 示例实现（需要扩展持久化接口）
	// sessions, err := rm.persister.LoadExpired(ctx, rm.timeout)
	// if err != nil {
	//     return err
	// }
	// 
	// for _, session := range sessions {
	//     if err := rm.persister.Delete(ctx, session.SessionID); err != nil {
	//         rm.logger.Error("删除过期会话失败", 
	//             zap.String("session_id", session.SessionID),
	//             zap.Error(err))
	//     }
	// }
	
	return nil
}

// RecoveryMiddleware 恢复中间件（用于API层）
type RecoveryMiddleware struct {
	manager *RecoveryManager
}

// NewRecoveryMiddleware 创建恢复中间件
func NewRecoveryMiddleware(manager *RecoveryManager) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		manager: manager,
	}
}

// RecoverOrCreate 恢复或创建会话
func (m *RecoveryMiddleware) RecoverOrCreate(ctx context.Context, sessionID string, userID uint) (*StateMachine, error) {
	// 尝试恢复现有会话
	sm, err := m.manager.RecoverSession(ctx, sessionID)
	if err == nil {
		return sm, nil
	}
	
	// 恢复失败，创建新会话
	m.manager.logger.Info("创建新会话",
		zap.String("session_id", sessionID),
		zap.Uint("user_id", userID))
	
	sm = NewStateMachine(sessionID, userID, m.manager.logger, m.manager.persister)
	
	// 保存初始状态
	if err := m.manager.persister.Save(ctx, sessionID, sm.toData()); err != nil {
		return nil, fmt.Errorf("保存初始状态失败: %w", err)
	}
	
	return sm, nil
}