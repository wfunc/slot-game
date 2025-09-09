package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/game/slot"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SessionManager 游戏会话管理器
type SessionManager struct {
	mu              sync.RWMutex
	sessions        map[string]*GameSession
	logger          *zap.Logger
	persister       StatePersister
	gameResultRepo  repository.GameResultRepository
	walletRepo      repository.WalletRepository
	slotEngine      slot.Engine
	recoveryManager *RecoveryManager
	sessionTimeout  time.Duration
	maxSessions     int
}

// GameSession 游戏会话
type GameSession struct {
	SessionID    string
	UserID       uint
	StateMachine *StateMachine
	SlotEngine   slot.Engine
	StartTime    time.Time
	LastActivity time.Time
	SpinResult   *slot.SpinResult // 当前转动结果
	TotalBet     int64           // 总投注
	TotalWin     int64           // 总赢取
	SpinCount    int             // 转动次数
	mu           sync.RWMutex
}

// SessionConfig 会话管理器配置
type SessionConfig struct {
	Logger         *zap.Logger
	DB             *gorm.DB
	SessionTimeout time.Duration
	MaxSessions    int
}

// NewSessionManager 创建会话管理器
func NewSessionManager(config *SessionConfig) *SessionManager {
	persister := NewDatabaseStatePersister(config.DB)
	recoveryManager := NewRecoveryManager(config.Logger, persister, config.SessionTimeout)
	
	return &SessionManager{
		sessions:        make(map[string]*GameSession),
		logger:          config.Logger,
		persister:       persister,
		gameResultRepo:  repository.NewGameResultRepository(config.DB),
		walletRepo:      repository.NewWalletRepository(config.DB),
		slotEngine:      slot.NewEngine(slot.DefaultConfig()),
		recoveryManager: recoveryManager,
		sessionTimeout:  config.SessionTimeout,
		maxSessions:     config.MaxSessions,
	}
}

// CreateSession 创建新会话
func (sm *SessionManager) CreateSession(ctx context.Context, sessionID string, userID uint) (*GameSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// 检查会话数量限制
	if len(sm.sessions) >= sm.maxSessions {
		return nil, errors.New("会话数量已达上限")
	}
	
	// 检查会话是否已存在
	if _, exists := sm.sessions[sessionID]; exists {
		return nil, fmt.Errorf("会话已存在: %s", sessionID)
	}
	
	// 创建状态机
	stateMachine := NewStateMachine(sessionID, userID, sm.logger, sm.persister)
	
	// 设置状态变更回调
	stateMachine.OnStateChange(func(from, to GameState) {
		sm.logger.Info("游戏状态变更",
			zap.String("session_id", sessionID),
			zap.String("from", string(from)),
			zap.String("to", string(to)))
	})
	
	// 创建会话
	session := &GameSession{
		SessionID:    sessionID,
		UserID:       userID,
		StateMachine: stateMachine,
		SlotEngine:   sm.slotEngine,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}
	
	sm.sessions[sessionID] = session
	
	sm.logger.Info("创建游戏会话",
		zap.String("session_id", sessionID),
		zap.Uint("user_id", userID))
	
	return session, nil
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) (*GameSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", sessionID)
	}
	
	// 更新活动时间
	session.UpdateActivity()
	
	return session, nil
}

// RecoverOrCreateSession 恢复或创建会话
func (sm *SessionManager) RecoverOrCreateSession(ctx context.Context, sessionID string, userID uint) (*GameSession, error) {
	// 先尝试从内存获取
	if session, err := sm.GetSession(sessionID); err == nil {
		return session, nil
	}
	
	// 尝试恢复会话
	stateMachine, err := sm.recoveryManager.RecoverSession(ctx, sessionID)
	if err == nil {
		sm.mu.Lock()
		session := &GameSession{
			SessionID:    sessionID,
			UserID:       userID,
			StateMachine: stateMachine,
			SlotEngine:   sm.slotEngine,
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		}
		sm.sessions[sessionID] = session
		sm.mu.Unlock()
		
		sm.logger.Info("恢复游戏会话",
			zap.String("session_id", sessionID),
			zap.Uint("user_id", userID))
		
		return session, nil
	}
	
	// 创建新会话
	return sm.CreateSession(ctx, sessionID, userID)
}

// RemoveSession 移除会话
func (sm *SessionManager) RemoveSession(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}
	
	// 保存最终状态
	if err := sm.persister.Save(ctx, sessionID, session.StateMachine.toData()); err != nil {
		sm.logger.Error("保存会话状态失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}
	
	delete(sm.sessions, sessionID)
	
	sm.logger.Info("移除游戏会话",
		zap.String("session_id", sessionID),
		zap.Int("total_spins", session.SpinCount),
		zap.Int64("total_bet", session.TotalBet),
		zap.Int64("total_win", session.TotalWin))
	
	return nil
}

// CleanupInactiveSessions 清理不活跃的会话
func (sm *SessionManager) CleanupInactiveSessions(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	var toRemove []string
	
	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActivity) > sm.sessionTimeout {
			toRemove = append(toRemove, sessionID)
		}
	}
	
	for _, sessionID := range toRemove {
		session := sm.sessions[sessionID]
		
		// 保存状态
		if err := sm.persister.Save(ctx, sessionID, session.StateMachine.toData()); err != nil {
			sm.logger.Error("保存超时会话状态失败",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
		
		delete(sm.sessions, sessionID)
		
		sm.logger.Info("清理超时会话",
			zap.String("session_id", sessionID),
			zap.Duration("inactive", now.Sub(session.LastActivity)))
	}
}

// StartCleanupTask 启动清理任务
func (sm *SessionManager) StartCleanupTask(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				sm.logger.Info("停止会话清理任务")
				return
			case <-ticker.C:
				sm.CleanupInactiveSessions(ctx)
			}
		}
	}()
}

// GetActiveSessions 获取活跃会话数
func (sm *SessionManager) GetActiveSessions() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GetSessionStats 获取会话统计
func (sm *SessionManager) GetSessionStats(sessionID string) (map[string]interface{}, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	
	session.mu.RLock()
	defer session.mu.RUnlock()
	
	stats := map[string]interface{}{
		"session_id":   session.SessionID,
		"user_id":      session.UserID,
		"state":        session.StateMachine.GetState(),
		"start_time":   session.StartTime,
		"duration":     time.Since(session.StartTime).Seconds(),
		"spin_count":   session.SpinCount,
		"total_bet":    session.TotalBet,
		"total_win":    session.TotalWin,
		"rtp":          float64(session.TotalWin) / float64(session.TotalBet) * 100,
	}
	
	return stats, nil
}

// UpdateActivity 更新活动时间
func (gs *GameSession) UpdateActivity() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.LastActivity = time.Now()
}

// StartGame 开始游戏
func (gs *GameSession) StartGame(ctx context.Context, betAmount int64) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// 设置投注金额
	gs.StateMachine.SetBetAmount(betAmount)
	
	// 触发投币事件
	if err := gs.StateMachine.Trigger(ctx, "insert_coin"); err != nil {
		return fmt.Errorf("开始游戏失败: %w", err)
	}
	
	gs.LastActivity = time.Now()
	return nil
}

// Spin 执行转动
func (gs *GameSession) Spin(ctx context.Context) (*slot.SpinResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// 检查状态
	if gs.StateMachine.GetState() != StateReady {
		return nil, errors.New("当前状态不能执行转动")
	}
	
	// 触发开始转动
	if err := gs.StateMachine.Trigger(ctx, "start_spin"); err != nil {
		return nil, fmt.Errorf("开始转动失败: %w", err)
	}
	
	// 执行老虎机转动
	result, err := gs.SlotEngine.Spin(gs.UserID, gs.SessionID, gs.StateMachine.betAmount)
	if err != nil {
		return nil, fmt.Errorf("执行转动失败: %w", err)
	}
	gs.SpinResult = result
	
	// 更新统计
	gs.SpinCount++
	gs.TotalBet += gs.StateMachine.betAmount
	gs.TotalWin += result.TotalPayout
	
	// 设置中奖金额
	gs.StateMachine.SetWinAmount(result.TotalPayout)
	
	// 触发停止转动
	if err := gs.StateMachine.Trigger(ctx, "stop_spin"); err != nil {
		return nil, fmt.Errorf("停止转动失败: %w", err)
	}
	
	// 根据结果触发相应事件
	if result.TotalPayout > 0 {
		if err := gs.StateMachine.Trigger(ctx, "show_win"); err != nil {
			return nil, fmt.Errorf("展示中奖失败: %w", err)
		}
	} else {
		if err := gs.StateMachine.Trigger(ctx, "no_win"); err != nil {
			return nil, fmt.Errorf("处理未中奖失败: %w", err)
		}
	}
	
	gs.LastActivity = time.Now()
	return result, nil
}

// Settle 结算游戏
func (gs *GameSession) Settle(ctx context.Context) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	
	// 检查状态
	state := gs.StateMachine.GetState()
	if state != StateWinning && state != StateSettlement {
		return errors.New("当前状态不能结算")
	}
	
	// 如果在中奖展示状态，先触发结算
	if state == StateWinning {
		if err := gs.StateMachine.Trigger(ctx, "settle"); err != nil {
			return fmt.Errorf("触发结算失败: %w", err)
		}
	}
	
	// 完成游戏
	if err := gs.StateMachine.Trigger(ctx, "finish"); err != nil {
		return fmt.Errorf("完成游戏失败: %w", err)
	}
	
	gs.LastActivity = time.Now()
	return nil
}

// GetState 获取当前状态
func (gs *GameSession) GetState() GameState {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.StateMachine.GetState()
}

// GetLastResult 获取最后的转动结果
func (gs *GameSession) GetLastResult() *slot.SpinResult {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.SpinResult
}

// SaveGameRecord 保存游戏记录
func (sm *SessionManager) SaveGameRecord(ctx context.Context, session *GameSession) error {
	if session.SpinResult == nil {
		return errors.New("没有游戏结果")
	}
	
	record := &models.GameRecord{
		UserID:    session.UserID,
		RoundID:   fmt.Sprintf("%s-%d", session.SessionID, time.Now().Unix()),
		BetAmount: session.StateMachine.betAmount,
		WinAmount: session.SpinResult.TotalPayout,
		Result:    models.JSONMap(session.SpinResult.ToJSON()),
		PlayedAt:  time.Now(),
	}
	
	return sm.gameResultRepo.Create(ctx, record)
}