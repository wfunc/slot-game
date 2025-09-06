package repository

import (
	"context"
	"time"
	
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// GameResultRepository 游戏结果仓储接口
type GameResultRepository interface {
	BaseRepository
	Create(ctx context.Context, result *models.GameResult) error
	BatchCreate(ctx context.Context, results []*models.GameResult) error
	FindByID(ctx context.Context, id uint) (*models.GameResult, error)
	FindByRoundID(ctx context.Context, roundID string) (*models.GameResult, error)
	FindBySessionID(ctx context.Context, sessionID uint, p *Pagination) ([]*models.GameResult, error)
	FindByGameID(ctx context.Context, gameID uint, p *Pagination) ([]*models.GameResult, error)
	FindWinsByUserID(ctx context.Context, userID uint, p *Pagination) ([]*models.GameResult, error)
	GetWinStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (*WinStatistics, error)
	GetJackpotRecords(ctx context.Context, limit int) ([]*models.GameResult, error)
	GetBigWins(ctx context.Context, minAmount int64, limit int) ([]*models.GameResult, error)
}

// WinStatistics 中奖统计
type WinStatistics struct {
	TotalRounds     int64   `json:"total_rounds"`
	WinRounds       int64   `json:"win_rounds"`
	WinRate         float64 `json:"win_rate"`
	TotalWinAmount  int64   `json:"total_win_amount"`
	MaxWinAmount    int64   `json:"max_win_amount"`
	MaxMultiplier   float64 `json:"max_multiplier"`
	AverageWin      float64 `json:"average_win"`
	JackpotCount    int64   `json:"jackpot_count"`
	BonusCount      int64   `json:"bonus_count"`
	BigWinCount     int64   `json:"big_win_count"`    // 大奖次数（赢取>100倍投注）
	MegaWinCount    int64   `json:"mega_win_count"`   // 巨奖次数（赢取>250倍投注）
	SuperWinCount   int64   `json:"super_win_count"`  // 超级大奖（赢取>500倍投注）
}

// gameResultRepo 游戏结果仓储实现
type gameResultRepo struct {
	*BaseRepo
}

// NewGameResultRepository 创建游戏结果仓储
func NewGameResultRepository(db *gorm.DB) GameResultRepository {
	return &gameResultRepo{
		BaseRepo: NewBaseRepo(db),
	}
}

// Create 创建游戏结果
func (r *gameResultRepo) Create(ctx context.Context, result *models.GameResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

// BatchCreate 批量创建游戏结果
func (r *gameResultRepo) BatchCreate(ctx context.Context, results []*models.GameResult) error {
	if len(results) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(results, 100).Error
}

// FindByID 根据ID查找
func (r *gameResultRepo) FindByID(ctx context.Context, id uint) (*models.GameResult, error) {
	var result models.GameResult
	err := r.db.WithContext(ctx).
		Preload("Game").
		Preload("Session").
		First(&result, id).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FindByRoundID 根据回合ID查找
func (r *gameResultRepo) FindByRoundID(ctx context.Context, roundID string) (*models.GameResult, error) {
	var result models.GameResult
	err := r.db.WithContext(ctx).
		Preload("Game").
		Preload("Session").
		Where("round_id = ?", roundID).
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FindBySessionID 根据会话ID查找
func (r *gameResultRepo) FindBySessionID(ctx context.Context, sessionID uint, p *Pagination) ([]*models.GameResult, error) {
	var results []*models.GameResult
	
	// 查询总数
	r.db.WithContext(ctx).
		Model(&models.GameResult{}).
		Where("session_id = ?", sessionID).
		Count(&p.Total)
	
	// 查询数据
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("played_at desc").
		Scopes(Paginate(p)).
		Find(&results).Error
	
	return results, err
}

// FindByGameID 根据游戏ID查找
func (r *gameResultRepo) FindByGameID(ctx context.Context, gameID uint, p *Pagination) ([]*models.GameResult, error) {
	var results []*models.GameResult
	
	// 查询总数
	r.db.WithContext(ctx).
		Model(&models.GameResult{}).
		Where("game_id = ?", gameID).
		Count(&p.Total)
	
	// 查询数据
	err := r.db.WithContext(ctx).
		Where("game_id = ?", gameID).
		Order("played_at desc").
		Scopes(Paginate(p)).
		Find(&results).Error
	
	return results, err
}

// FindWinsByUserID 查找用户的中奖记录
func (r *gameResultRepo) FindWinsByUserID(ctx context.Context, userID uint, p *Pagination) ([]*models.GameResult, error) {
	var results []*models.GameResult
	
	// 查询总数
	r.db.WithContext(ctx).
		Model(&models.GameResult{}).
		Where("user_id = ? AND win_amount > 0", userID).
		Count(&p.Total)
	
	// 查询数据
	err := r.db.WithContext(ctx).
		Preload("Game").
		Where("user_id = ? AND win_amount > 0", userID).
		Order("win_amount desc, played_at desc").
		Scopes(Paginate(p)).
		Find(&results).Error
	
	return results, err
}

// GetWinStatistics 获取中奖统计
func (r *gameResultRepo) GetWinStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (*WinStatistics, error) {
	var stats WinStatistics
	
	// 基础统计
	err := r.db.WithContext(ctx).
		Model(&models.GameResult{}).
		Where("user_id = ? AND played_at BETWEEN ? AND ?", userID, startTime, endTime).
		Select(
			"COUNT(*) as total_rounds",
			"COUNT(CASE WHEN win_amount > 0 THEN 1 END) as win_rounds",
			"COALESCE(SUM(win_amount), 0) as total_win_amount",
			"COALESCE(MAX(win_amount), 0) as max_win_amount",
			"COALESCE(MAX(multiplier), 0) as max_multiplier",
			"COUNT(CASE WHEN is_jackpot THEN 1 END) as jackpot_count",
			"COUNT(CASE WHEN is_bonus THEN 1 END) as bonus_count",
		).
		Row().Scan(
			&stats.TotalRounds,
			&stats.WinRounds,
			&stats.TotalWinAmount,
			&stats.MaxWinAmount,
			&stats.MaxMultiplier,
			&stats.JackpotCount,
			&stats.BonusCount,
		)
	
	if err != nil {
		return nil, err
	}
	
	// 计算胜率和平均赢取
	if stats.TotalRounds > 0 {
		stats.WinRate = float64(stats.WinRounds) / float64(stats.TotalRounds) * 100
	}
	if stats.WinRounds > 0 {
		stats.AverageWin = float64(stats.TotalWinAmount) / float64(stats.WinRounds)
	}
	
	// 统计大奖次数（根据倍数）
	type WinLevel struct {
		BigWin   int64
		MegaWin  int64
		SuperWin int64
	}
	
	var winLevel WinLevel
	r.db.WithContext(ctx).
		Model(&models.GameResult{}).
		Where("user_id = ? AND played_at BETWEEN ? AND ? AND win_amount > 0", userID, startTime, endTime).
		Select(
			"COUNT(CASE WHEN multiplier >= 100 THEN 1 END) as big_win",
			"COUNT(CASE WHEN multiplier >= 250 THEN 1 END) as mega_win",
			"COUNT(CASE WHEN multiplier >= 500 THEN 1 END) as super_win",
		).
		Row().Scan(&winLevel.BigWin, &winLevel.MegaWin, &winLevel.SuperWin)
	
	stats.BigWinCount = winLevel.BigWin
	stats.MegaWinCount = winLevel.MegaWin
	stats.SuperWinCount = winLevel.SuperWin
	
	return &stats, nil
}

// GetJackpotRecords 获取大奖记录
func (r *gameResultRepo) GetJackpotRecords(ctx context.Context, limit int) ([]*models.GameResult, error) {
	var results []*models.GameResult
	
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	
	err := r.db.WithContext(ctx).
		Preload("Game").
		Where("is_jackpot = ?", true).
		Order("played_at desc").
		Limit(limit).
		Find(&results).Error
	
	return results, err
}

// GetBigWins 获取大额中奖记录
func (r *gameResultRepo) GetBigWins(ctx context.Context, minAmount int64, limit int) ([]*models.GameResult, error) {
	var results []*models.GameResult
	
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	
	err := r.db.WithContext(ctx).
		Preload("Game").
		Where("win_amount >= ?", minAmount).
		Order("win_amount desc, played_at desc").
		Limit(limit).
		Find(&results).Error
	
	return results, err
}

// WithTx 使用事务
func (r *gameResultRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &gameResultRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}