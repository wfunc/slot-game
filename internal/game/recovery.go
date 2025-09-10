package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RecoveryManager 游戏恢复管理器
type RecoveryManager struct {
	logger       *zap.Logger
	persister    StatePersister
	walletRepo   repository.WalletRepository
	db           *gorm.DB
	timeout      time.Duration // 会话超时时间
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager(logger *zap.Logger, persister StatePersister, db *gorm.DB, timeout time.Duration) *RecoveryManager {
	return &RecoveryManager{
		logger:       logger,
		persister:    persister,
		walletRepo:   repository.NewWalletRepository(db),
		db:           db,
		timeout:      timeout,
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
		rm.logger.Info("准备状态超时，执行退款",
			zap.String("session_id", sm.sessionID),
			zap.Int64("refund_amount", sm.betAmount))
		
		// 执行退款流程
		err := rm.db.Transaction(func(tx *gorm.DB) error {
			// 创建事务内的钱包仓储
			walletRepo := repository.NewWalletRepository(tx)
			
			// 退还投注金额到用户钱包
			if err := walletRepo.AddBalance(ctx, sm.userID, sm.betAmount); err != nil {
				return fmt.Errorf("退款失败: %w", err)
			}
			
			// 获取钱包当前余额
			wallet, err := walletRepo.FindByUserID(ctx, sm.userID)
			if err != nil {
				return fmt.Errorf("获取钱包信息失败: %w", err)
			}
			
			// 创建退款交易记录
			transaction := &models.Transaction{
				UserID:        sm.userID,
				OrderNo:       fmt.Sprintf("REFUND_%s_%d", sm.sessionID, time.Now().Unix()),
				Type:          "refund",
				SubType:       "game_timeout",
				Amount:        sm.betAmount,
				BeforeBalance: wallet.Balance - sm.betAmount,
				AfterBalance:  wallet.Balance,
				Currency:      "CNY",
				Status:        "success",
				RefID:         sm.sessionID,
				RefType:       "game_session",
				Description:   "游戏超时退款",
				Remark:        fmt.Sprintf("会话 %s 准备状态超时，退还投注金额", sm.sessionID),
			}
			processedAt := time.Now()
			transaction.ProcessedAt = &processedAt
			
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("创建退款记录失败: %w", err)
			}
			
			rm.logger.Info("退款成功",
				zap.String("session_id", sm.sessionID),
				zap.Uint("user_id", sm.userID),
				zap.Int64("amount", sm.betAmount),
				zap.String("order_no", transaction.OrderNo))
			
			return nil
		})
		
		if err != nil {
			rm.logger.Error("退款事务失败",
				zap.String("session_id", sm.sessionID),
				zap.Error(err))
			// 即使退款失败，也要触发超时以清理状态
		}
		
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
	rm.logger.Info("从结算状态恢复，检查结算完成性",
		zap.String("session_id", sm.sessionID),
		zap.Int64("win_amount", sm.winAmount))
	
	// 检查结算是否真的完成
	if sm.winAmount > 0 {
		// 查询是否已存在结算成功的交易记录
		var existingTransaction models.Transaction
		err := rm.db.WithContext(ctx).
			Where("ref_id = ? AND ref_type = ? AND type = ? AND status = ?", 
				sm.sessionID, "game_session", "win", "success").
			First(&existingTransaction).Error
		
		if err == gorm.ErrRecordNotFound {
			// 没有找到结算记录，需要重新执行结算
			rm.logger.Warn("未找到结算记录，重新执行结算",
				zap.String("session_id", sm.sessionID),
				zap.Int64("win_amount", sm.winAmount))
			
			// 执行结算
			err := rm.db.Transaction(func(tx *gorm.DB) error {
				walletRepo := repository.NewWalletRepository(tx)
				
				// 添加中奖金额到钱包
				if err := walletRepo.AddBalance(ctx, sm.userID, sm.winAmount); err != nil {
					return fmt.Errorf("添加中奖金额失败: %w", err)
				}
				
				// 获取钱包信息
				wallet, err := walletRepo.FindByUserID(ctx, sm.userID)
				if err != nil {
					return fmt.Errorf("获取钱包信息失败: %w", err)
				}
				
				// 创建中奖交易记录
				transaction := &models.Transaction{
					UserID:        sm.userID,
					OrderNo:       fmt.Sprintf("WIN_%s_%d", sm.sessionID, time.Now().Unix()),
					Type:          "win",
					SubType:       "game_recovery",
					Amount:        sm.winAmount,
					BeforeBalance: wallet.Balance - sm.winAmount,
					AfterBalance:  wallet.Balance,
					Currency:      "CNY",
					Status:        "success",
					RefID:         sm.sessionID,
					RefType:       "game_session",
					Description:   "游戏中奖（恢复结算）",
					Remark:        fmt.Sprintf("会话 %s 恢复时完成结算", sm.sessionID),
				}
				processedAt := time.Now()
				transaction.ProcessedAt = &processedAt
				
				if err := tx.Create(transaction).Error; err != nil {
					return fmt.Errorf("创建中奖记录失败: %w", err)
				}
				
				// 更新钱包统计
				if err := walletRepo.UpdateStatistics(ctx, sm.userID, "total_win", sm.winAmount); err != nil {
					rm.logger.Warn("更新钱包统计失败", 
						zap.Uint("user_id", sm.userID),
						zap.Error(err))
				}
				
				rm.logger.Info("恢复结算成功",
					zap.String("session_id", sm.sessionID),
					zap.Uint("user_id", sm.userID),
					zap.Int64("amount", sm.winAmount),
					zap.String("order_no", transaction.OrderNo))
				
				return nil
			})
			
			if err != nil {
				rm.logger.Error("恢复结算失败",
					zap.String("session_id", sm.sessionID),
					zap.Error(err))
				// 结算失败，保持在结算状态等待手动处理
				return fmt.Errorf("恢复结算失败: %w", err)
			}
		} else if err == nil {
			// 找到了结算记录，说明已经结算过了
			rm.logger.Info("结算已完成，直接结束游戏",
				zap.String("session_id", sm.sessionID),
				zap.String("transaction_order", existingTransaction.OrderNo),
				zap.Int64("amount", existingTransaction.Amount))
		} else {
			// 查询出错
			return fmt.Errorf("查询结算记录失败: %w", err)
		}
	}
	
	// 结束游戏
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
	rm.logger.Info("开始清理过期会话",
		zap.Duration("timeout", rm.timeout))
	
	// 计算过期时间点
	expiredBefore := time.Now().Add(-rm.timeout)
	
	// 查询所有过期的游戏状态
	var expiredStates []models.GameState
	err := rm.db.WithContext(ctx).
		Where("updated_at < ? AND current_state != ?", expiredBefore, "idle").
		Find(&expiredStates).Error
	
	if err != nil {
		rm.logger.Error("查询过期会话失败", zap.Error(err))
		return fmt.Errorf("查询过期会话失败: %w", err)
	}
	
	if len(expiredStates) == 0 {
		rm.logger.Info("没有需要清理的过期会话")
		return nil
	}
	
	rm.logger.Info("找到过期会话",
		zap.Int("count", len(expiredStates)))
	
	// 批量处理过期会话
	successCount := 0
	failCount := 0
	refundCount := 0
	
	for _, state := range expiredStates {
		// 反序列化状态数据
		var stateData StateMachineData
		if err := json.Unmarshal([]byte(state.StateData), &stateData); err != nil {
			rm.logger.Error("反序列化状态数据失败",
				zap.String("session_id", state.SessionID),
				zap.Error(err))
			failCount++
			continue
		}
		
		// 根据状态决定清理策略
		switch stateData.CurrentState {
		case StateReady:
			// 准备状态需要退款
			if stateData.BetAmount > 0 {
				rm.logger.Info("清理准备状态会话，执行退款",
					zap.String("session_id", state.SessionID),
					zap.Uint("user_id", state.UserID),
					zap.Int64("bet_amount", stateData.BetAmount))
				
				// 执行退款
				err := rm.performRefund(ctx, state.SessionID, state.UserID, stateData.BetAmount, "expired_cleanup")
				if err != nil {
					rm.logger.Error("清理时退款失败",
						zap.String("session_id", state.SessionID),
						zap.Error(err))
					failCount++
					continue
				}
				refundCount++
			}
			
		case StateSettlement:
			// 结算状态需要检查是否已完成结算
			if stateData.WinAmount > 0 {
				// 检查是否已有结算记录
				var existingTransaction models.Transaction
				err := rm.db.WithContext(ctx).
					Where("ref_id = ? AND ref_type = ? AND type = ? AND status = ?",
						state.SessionID, "game_session", "win", "success").
					First(&existingTransaction).Error
				
				if err == gorm.ErrRecordNotFound {
					// 未结算，执行结算
					rm.logger.Info("清理结算状态会话，执行结算",
						zap.String("session_id", state.SessionID),
						zap.Uint("user_id", state.UserID),
						zap.Int64("win_amount", stateData.WinAmount))
					
					err := rm.performSettlement(ctx, state.SessionID, state.UserID, stateData.WinAmount)
					if err != nil {
						rm.logger.Error("清理时结算失败",
							zap.String("session_id", state.SessionID),
							zap.Error(err))
						failCount++
						continue
					}
				}
			}
		}
		
		// 删除过期会话
		if err := rm.persister.Delete(ctx, state.SessionID); err != nil {
			rm.logger.Error("删除会话失败",
				zap.String("session_id", state.SessionID),
				zap.Error(err))
			failCount++
			continue
		}
		
		// 从数据库删除状态记录
		if err := rm.db.WithContext(ctx).Delete(&state).Error; err != nil {
			rm.logger.Error("删除数据库记录失败",
				zap.String("session_id", state.SessionID),
				zap.Error(err))
			// 不计入失败，因为内存已清理
		}
		
		successCount++
	}
	
	rm.logger.Info("清理过期会话完成",
		zap.Int("total", len(expiredStates)),
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.Int("refunded", refundCount))
	
	return nil
}

// performRefund 执行退款操作（辅助方法）
func (rm *RecoveryManager) performRefund(ctx context.Context, sessionID string, userID uint, amount int64, reason string) error {
	return rm.db.Transaction(func(tx *gorm.DB) error {
		walletRepo := repository.NewWalletRepository(tx)
		
		// 退还金额
		if err := walletRepo.AddBalance(ctx, userID, amount); err != nil {
			return fmt.Errorf("退款失败: %w", err)
		}
		
		// 获取钱包信息
		wallet, err := walletRepo.FindByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("获取钱包信息失败: %w", err)
		}
		
		// 创建退款记录
		transaction := &models.Transaction{
			UserID:        userID,
			OrderNo:       fmt.Sprintf("REFUND_%s_%d", sessionID, time.Now().Unix()),
			Type:          "refund",
			SubType:       reason,
			Amount:        amount,
			BeforeBalance: wallet.Balance - amount,
			AfterBalance:  wallet.Balance,
			Currency:      "CNY",
			Status:        "success",
			RefID:         sessionID,
			RefType:       "game_session",
			Description:   "游戏会话过期退款",
			Remark:        fmt.Sprintf("会话 %s 过期清理时退款", sessionID),
		}
		processedAt := time.Now()
		transaction.ProcessedAt = &processedAt
		
		return tx.Create(transaction).Error
	})
}

// performSettlement 执行结算操作（辅助方法）
func (rm *RecoveryManager) performSettlement(ctx context.Context, sessionID string, userID uint, winAmount int64) error {
	return rm.db.Transaction(func(tx *gorm.DB) error {
		walletRepo := repository.NewWalletRepository(tx)
		
		// 添加中奖金额
		if err := walletRepo.AddBalance(ctx, userID, winAmount); err != nil {
			return fmt.Errorf("添加中奖金额失败: %w", err)
		}
		
		// 获取钱包信息
		wallet, err := walletRepo.FindByUserID(ctx, userID)
		if err != nil {
			return fmt.Errorf("获取钱包信息失败: %w", err)
		}
		
		// 创建中奖记录
		transaction := &models.Transaction{
			UserID:        userID,
			OrderNo:       fmt.Sprintf("WIN_%s_%d", sessionID, time.Now().Unix()),
			Type:          "win",
			SubType:       "expired_settlement",
			Amount:        winAmount,
			BeforeBalance: wallet.Balance - winAmount,
			AfterBalance:  wallet.Balance,
			Currency:      "CNY",
			Status:        "success",
			RefID:         sessionID,
			RefType:       "game_session",
			Description:   "游戏中奖（过期结算）",
			Remark:        fmt.Sprintf("会话 %s 过期清理时结算", sessionID),
		}
		processedAt := time.Now()
		transaction.ProcessedAt = &processedAt
		
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("创建中奖记录失败: %w", err)
		}
		
		// 更新钱包统计
		if err := walletRepo.UpdateStatistics(ctx, userID, "total_win", winAmount); err != nil {
			rm.logger.Warn("更新钱包统计失败",
				zap.Uint("user_id", userID),
				zap.Error(err))
		}
		
		return nil
	})
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