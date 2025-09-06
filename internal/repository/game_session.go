package repository

import (
	"context"
	"time"
	
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// GameSessionRepository 游戏会话仓储接口
type GameSessionRepository interface {
	BaseRepository
	Create(ctx context.Context, session *models.GameSession) error
	Update(ctx context.Context, session *models.GameSession) error
	UpdateBySessionID(ctx context.Context, sessionID string, updates map[string]interface{}) error
	FindByID(ctx context.Context, id uint) (*models.GameSession, error)
	FindBySessionID(ctx context.Context, sessionID string) (*models.GameSession, error)
	FindByUserID(ctx context.Context, userID uint, p *Pagination) ([]*models.GameSession, error)
	FindActiveByUserID(ctx context.Context, userID uint) (*models.GameSession, error)
	GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (*GameStatistics, error)
	EndSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context, expiredBefore time.Time) (int64, error)
}

// GameStatistics 游戏统计
type GameStatistics struct {
	TotalGames   int64   `json:"total_games"`
	TotalBet     int64   `json:"total_bet"`
	TotalWin     int64   `json:"total_win"`
	TotalProfit  int64   `json:"total_profit"`
	WinRate      float64 `json:"win_rate"`
	AverageBet   float64 `json:"average_bet"`
	MaxWin       int64   `json:"max_win"`
	TotalMinutes int64   `json:"total_minutes"`
}

// gameSessionRepo 游戏会话仓储实现
type gameSessionRepo struct {
	*BaseRepo
}

// NewGameSessionRepository 创建游戏会话仓储
func NewGameSessionRepository(db *gorm.DB) GameSessionRepository {
	return &gameSessionRepo{
		BaseRepo: NewBaseRepo(db),
	}
}

// Create 创建游戏会话
func (r *gameSessionRepo) Create(ctx context.Context, session *models.GameSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// Update 更新游戏会话
func (r *gameSessionRepo) Update(ctx context.Context, session *models.GameSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// UpdateBySessionID 根据会话ID更新
func (r *gameSessionRepo) UpdateBySessionID(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("session_id = ?", sessionID).
		Updates(updates).Error
}

// FindByID 根据ID查找
func (r *gameSessionRepo) FindByID(ctx context.Context, id uint) (*models.GameSession, error) {
	var session models.GameSession
	err := r.db.WithContext(ctx).
		Preload("Game").
		Preload("Room").
		First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindBySessionID 根据会话ID查找
func (r *gameSessionRepo) FindBySessionID(ctx context.Context, sessionID string) (*models.GameSession, error) {
	var session models.GameSession
	err := r.db.WithContext(ctx).
		Preload("Game").
		Preload("Room").
		Where("session_id = ?", sessionID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindByUserID 根据用户ID查找（分页）
func (r *gameSessionRepo) FindByUserID(ctx context.Context, userID uint, p *Pagination) ([]*models.GameSession, error) {
	var sessions []*models.GameSession
	
	// 查询总数
	r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("user_id = ?", userID).
		Count(&p.Total)
	
	// 查询数据
	err := r.db.WithContext(ctx).
		Preload("Game").
		Where("user_id = ?", userID).
		Order("created_at desc").
		Scopes(Paginate(p)).
		Find(&sessions).Error
	
	return sessions, err
}

// FindActiveByUserID 查找用户当前活跃的游戏会话
func (r *gameSessionRepo) FindActiveByUserID(ctx context.Context, userID uint) (*models.GameSession, error) {
	var session models.GameSession
	err := r.db.WithContext(ctx).
		Preload("Game").
		Preload("Room").
		Where("user_id = ? AND status = ?", userID, "playing").
		Order("created_at desc").
		First(&session).Error
	
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

// GetStatistics 获取统计数据
func (r *gameSessionRepo) GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (*GameStatistics, error) {
	var stats GameStatistics
	
	// 基础统计
	err := r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Select(
			"COUNT(*) as total_games",
			"COALESCE(SUM(total_bet), 0) as total_bet",
			"COALESCE(SUM(total_win), 0) as total_win",
			"COALESCE(SUM(total_win - total_bet), 0) as total_profit",
			"COALESCE(MAX(peak_win), 0) as max_win",
			"COALESCE(SUM(duration), 0) as total_minutes",
		).
		Row().Scan(
			&stats.TotalGames,
			&stats.TotalBet,
			&stats.TotalWin,
			&stats.TotalProfit,
			&stats.MaxWin,
			&stats.TotalMinutes,
		)
	
	if err != nil {
		return nil, err
	}
	
	// 计算胜率和平均下注
	if stats.TotalGames > 0 {
		// 查询获胜场次
		var winGames int64
		r.db.WithContext(ctx).
			Model(&models.GameSession{}).
			Where("user_id = ? AND total_win > total_bet AND created_at BETWEEN ? AND ?", 
				userID, startTime, endTime).
			Count(&winGames)
		
		stats.WinRate = float64(winGames) / float64(stats.TotalGames) * 100
		stats.AverageBet = float64(stats.TotalBet) / float64(stats.TotalGames)
	}
	
	// 转换为分钟
	stats.TotalMinutes = stats.TotalMinutes / 60
	
	return &stats, nil
}

// EndSession 结束会话
func (r *gameSessionRepo) EndSession(ctx context.Context, sessionID string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":   "ended",
		"ended_at": &now,
	}
	
	// 计算持续时间
	var session models.GameSession
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&session).Error; err == nil {
		duration := int(now.Sub(session.StartedAt).Seconds())
		updates["duration"] = duration
	}
	
	return r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("session_id = ?", sessionID).
		Updates(updates).Error
}

// CleanupExpiredSessions 清理过期会话
func (r *gameSessionRepo) CleanupExpiredSessions(ctx context.Context, expiredBefore time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("status = ? AND updated_at < ?", "playing", expiredBefore).
		Updates(map[string]interface{}{
			"status":   "expired",
			"ended_at": &expiredBefore,
		})
	
	return result.RowsAffected, result.Error
}

// WithTx 使用事务
func (r *gameSessionRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &gameSessionRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}