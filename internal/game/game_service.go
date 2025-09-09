package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GameService 游戏服务（业务逻辑层）
type GameService struct {
	sessionManager *SessionManager
	userRepo       repository.UserRepository
	walletRepo     repository.WalletRepository
	gameResultRepo repository.GameResultRepository
	logger         *zap.Logger
	db             *gorm.DB
}

// GameServiceConfig 游戏服务配置
type GameServiceConfig struct {
	DB             *gorm.DB
	Logger         *zap.Logger
	SessionTimeout time.Duration
	MaxSessions    int
}

// NewGameService 创建游戏服务
func NewGameService(config *GameServiceConfig) *GameService {
	sessionConfig := &SessionConfig{
		Logger:         config.Logger,
		DB:             config.DB,
		SessionTimeout: config.SessionTimeout,
		MaxSessions:    config.MaxSessions,
	}
	
	return &GameService{
		sessionManager: NewSessionManager(sessionConfig),
		userRepo:       repository.NewUserRepository(config.DB),
		walletRepo:     repository.NewWalletRepository(config.DB),
		gameResultRepo: repository.NewGameResultRepository(config.DB),
		logger:         config.Logger,
		db:             config.DB,
	}
}

// StartGame 开始游戏
func (s *GameService) StartGame(ctx context.Context, userID uint, sessionID string, betAmount int64) error {
	// 验证用户
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}
	
	if !user.IsActive() {
		return errors.New("用户账户已被禁用")
	}
	
	// 检查余额
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取钱包失败: %w", err)
	}
	
	if wallet.Balance < betAmount {
		return errors.New("余额不足")
	}
	
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	
	// 扣除投注金额
	if err := s.walletRepo.WithTx(tx).(repository.WalletRepository).DeductBalance(ctx, userID, betAmount); err != nil {
		tx.Rollback()
		return fmt.Errorf("扣除投注失败: %w", err)
	}
	
	// 记录交易
	transaction := &models.WalletTransaction{
		UserID:        userID,
		OrderNo:       fmt.Sprintf("BET-%s-%d", sessionID, time.Now().Unix()),
		Type:          "bet",
		Amount:        betAmount,
		BeforeBalance: wallet.Balance,
		AfterBalance:  wallet.Balance - betAmount,
		RefType:       "game",
		RefID:         sessionID,
		Description:   "游戏投注",
		Status:        "success",
	}
	
	if err := s.walletRepo.WithTx(tx).(repository.WalletRepository).CreateTransaction(ctx, transaction); err != nil {
		tx.Rollback()
		return fmt.Errorf("记录交易失败: %w", err)
	}
	
	// 创建或恢复会话
	session, err := s.sessionManager.RecoverOrCreateSession(ctx, sessionID, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("创建会话失败: %w", err)
	}
	
	// 开始游戏
	if err := session.StartGame(ctx, betAmount); err != nil {
		tx.Rollback()
		return fmt.Errorf("开始游戏失败: %w", err)
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	s.logger.Info("游戏开始",
		zap.Uint("user_id", userID),
		zap.String("session_id", sessionID),
		zap.Int64("bet_amount", betAmount))
	
	return nil
}

// Spin 执行转动
func (s *GameService) Spin(ctx context.Context, sessionID string) (*SpinResponse, error) {
	// 获取会话
	session, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %w", err)
	}
	
	// 执行转动
	result, err := session.Spin(ctx)
	if err != nil {
		return nil, fmt.Errorf("转动失败: %w", err)
	}
	
	// 如果有中奖，增加余额
	if result.TotalPayout > 0 {
		tx := s.db.Begin()
		
		// 增加余额
		if err := s.walletRepo.WithTx(tx).(repository.WalletRepository).AddBalance(ctx, session.UserID, result.TotalPayout); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("增加余额失败: %w", err)
		}
		
		// 获取当前余额
		wallet, _ := s.walletRepo.WithTx(tx).(repository.WalletRepository).GetByUserID(ctx, session.UserID)
		
		// 记录中奖交易
		transaction := &models.WalletTransaction{
			UserID:        session.UserID,
			OrderNo:       fmt.Sprintf("WIN-%s-%d", sessionID, time.Now().Unix()),
			Type:          "win",
			Amount:        result.TotalPayout,
			BeforeBalance: wallet.Balance - result.TotalPayout,
			AfterBalance:  wallet.Balance,
			RefType:       "game",
			RefID:         sessionID,
			Description:   fmt.Sprintf("游戏中奖 - %s", result.GetWinDescription()),
			Status:        "success",
		}
		
		if err := s.walletRepo.WithTx(tx).(repository.WalletRepository).CreateTransaction(ctx, transaction); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("记录中奖交易失败: %w", err)
		}
		
		tx.Commit()
	}
	
	// 保存游戏记录
	if err := s.sessionManager.SaveGameRecord(ctx, session); err != nil {
		s.logger.Error("保存游戏记录失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}
	
	// 构建响应
	response := &SpinResponse{
		SessionID:   sessionID,
		Result:      result,
		State:       string(session.GetState()),
		TotalBet:    session.TotalBet,
		TotalWin:    session.TotalWin,
		SpinCount:   session.SpinCount,
	}
	
	return response, nil
}

// Settle 结算游戏
func (s *GameService) Settle(ctx context.Context, sessionID string) error {
	// 获取会话
	session, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	
	// 执行结算
	if err := session.Settle(ctx); err != nil {
		return fmt.Errorf("结算失败: %w", err)
	}
	
	s.logger.Info("游戏结算完成",
		zap.String("session_id", sessionID),
		zap.Int64("total_bet", session.TotalBet),
		zap.Int64("total_win", session.TotalWin))
	
	return nil
}

// GetSessionInfo 获取会话信息
func (s *GameService) GetSessionInfo(ctx context.Context, sessionID string) (*SessionInfo, error) {
	stats, err := s.sessionManager.GetSessionStats(sessionID)
	if err != nil {
		return nil, err
	}
	
	session, _ := s.sessionManager.GetSession(sessionID)
	
	info := &SessionInfo{
		SessionID:    sessionID,
		UserID:       stats["user_id"].(uint),
		State:        stats["state"].(GameState),
		StartTime:    stats["start_time"].(time.Time),
		Duration:     stats["duration"].(float64),
		SpinCount:    stats["spin_count"].(int),
		TotalBet:     stats["total_bet"].(int64),
		TotalWin:     stats["total_win"].(int64),
		RTP:          stats["rtp"].(float64),
		LastResult:   session.GetLastResult(),
		ValidEvents:  session.StateMachine.GetValidEvents(),
	}
	
	return info, nil
}

// EndSession 结束会话
func (s *GameService) EndSession(ctx context.Context, sessionID string) error {
	// 获取会话
	session, err := s.sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	
	// 确保游戏已结算
	state := session.GetState()
	if state != StateIdle && state != StateError {
		if err := session.Settle(ctx); err != nil {
			s.logger.Error("结算失败",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}
	
	// 移除会话
	if err := s.sessionManager.RemoveSession(ctx, sessionID); err != nil {
		return fmt.Errorf("移除会话失败: %w", err)
	}
	
	return nil
}

// GetUserGameHistory 获取用户游戏历史
func (s *GameService) GetUserGameHistory(ctx context.Context, userID uint, limit int) ([]*models.GameRecord, error) {
	pagination := &repository.Pagination{
		Page:     1,
		PageSize: limit,
	}
	return s.gameResultRepo.FindWinsByUserID(ctx, userID, pagination)
}

// GetUserStats 获取用户统计
func (s *GameService) GetUserStats(ctx context.Context, userID uint) (*UserGameStats, error) {
	pagination := &repository.Pagination{
		Page:     1,
		PageSize: 1000,
	}
	records, err := s.gameResultRepo.FindWinsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, err
	}
	
	stats := &UserGameStats{
		UserID:      userID,
		TotalGames:  len(records),
		TotalBet:    0,
		TotalWin:    0,
		BiggestWin:  0,
		LastPlayed:  time.Time{},
	}
	
	for _, record := range records {
		stats.TotalBet += record.BetAmount
		stats.TotalWin += record.WinAmount
		if record.WinAmount > stats.BiggestWin {
			stats.BiggestWin = record.WinAmount
		}
		if record.PlayedAt.After(stats.LastPlayed) {
			stats.LastPlayed = record.PlayedAt
		}
	}
	
	if stats.TotalBet > 0 {
		stats.RTP = float64(stats.TotalWin) / float64(stats.TotalBet) * 100
	}
	
	return stats, nil
}

// Start 启动游戏服务
func (s *GameService) Start(ctx context.Context) {
	// 启动会话清理任务
	s.sessionManager.StartCleanupTask(ctx, 5*time.Minute)
	
	s.logger.Info("游戏服务已启动")
}

// Stop 停止游戏服务
func (s *GameService) Stop(ctx context.Context) {
	// 保存所有活跃会话
	s.sessionManager.mu.Lock()
	defer s.sessionManager.mu.Unlock()
	
	for sessionID, session := range s.sessionManager.sessions {
		if err := s.sessionManager.persister.Save(ctx, sessionID, session.StateMachine.toData()); err != nil {
			s.logger.Error("保存会话失败",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}
	
	s.logger.Info("游戏服务已停止")
}