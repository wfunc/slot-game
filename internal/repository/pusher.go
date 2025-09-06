package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// PusherMachineRepository 推币机仓储接口
type PusherMachineRepository interface {
	BaseRepository
	Create(ctx context.Context, machine *models.PusherMachine) error
	Update(ctx context.Context, machine *models.PusherMachine) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.PusherMachine, error)
	FindByMachineID(ctx context.Context, machineID string) (*models.PusherMachine, error)
	FindByGameID(ctx context.Context, gameID uint) ([]*models.PusherMachine, error)
	GetActive(ctx context.Context) ([]*models.PusherMachine, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateCoinPool(ctx context.Context, id uint, delta int) error
	UpdatePrizePool(ctx context.Context, id uint, delta int) error
}

// pusherMachineRepo 推币机仓储实现
type pusherMachineRepo struct {
	*BaseRepo
}

// NewPusherMachineRepository 创建推币机仓储
func NewPusherMachineRepository(db *gorm.DB) PusherMachineRepository {
	return &pusherMachineRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建推币机
func (r *pusherMachineRepo) Create(ctx context.Context, machine *models.PusherMachine) error {
	return r.db.WithContext(ctx).Create(machine).Error
}

// Update 更新推币机
func (r *pusherMachineRepo) Update(ctx context.Context, machine *models.PusherMachine) error {
	return r.db.WithContext(ctx).Save(machine).Error
}

// Delete 删除推币机
func (r *pusherMachineRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.PusherMachine{}, id).Error
}

// FindByID 根据ID查找
func (r *pusherMachineRepo) FindByID(ctx context.Context, id uint) (*models.PusherMachine, error) {
	var machine models.PusherMachine
	err := r.db.WithContext(ctx).First(&machine, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("推币机不存在")
		}
		return nil, err
	}
	return &machine, nil
}

// FindByMachineID 根据机器ID查找
func (r *pusherMachineRepo) FindByMachineID(ctx context.Context, machineID string) (*models.PusherMachine, error) {
	var machine models.PusherMachine
	err := r.db.WithContext(ctx).Where("machine_id = ?", machineID).First(&machine).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("推币机不存在")
		}
		return nil, err
	}
	return &machine, nil
}

// FindByGameID 根据游戏ID查找
func (r *pusherMachineRepo) FindByGameID(ctx context.Context, gameID uint) ([]*models.PusherMachine, error) {
	var machines []*models.PusherMachine
	err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Find(&machines).Error
	return machines, err
}

// GetActive 获取活跃的推币机
func (r *pusherMachineRepo) GetActive(ctx context.Context) ([]*models.PusherMachine, error) {
	var machines []*models.PusherMachine
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&machines).Error
	return machines, err
}

// UpdateStatus 更新状态
func (r *pusherMachineRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.PusherMachine{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateCoinPool 更新币池
func (r *pusherMachineRepo) UpdateCoinPool(ctx context.Context, id uint, delta int) error {
	result := r.db.WithContext(ctx).
		Model(&models.PusherMachine{}).
		Where("id = ?", id).
		Update("coin_pool", gorm.Expr("coin_pool + ?", delta))
	
	if result.Error != nil {
		return result.Error
	}
	
	// 确保币池不会小于0
	r.db.WithContext(ctx).
		Model(&models.PusherMachine{}).
		Where("id = ? AND coin_pool < 0", id).
		Update("coin_pool", 0)
	
	return nil
}

// UpdatePrizePool 更新奖品池
func (r *pusherMachineRepo) UpdatePrizePool(ctx context.Context, id uint, delta int) error {
	result := r.db.WithContext(ctx).
		Model(&models.PusherMachine{}).
		Where("id = ?", id).
		Update("prize_pool", gorm.Expr("prize_pool + ?", delta))
	
	if result.Error != nil {
		return result.Error
	}
	
	// 确保奖品池不会小于0
	r.db.WithContext(ctx).
		Model(&models.PusherMachine{}).
		Where("id = ? AND prize_pool < 0", id).
		Update("prize_pool", 0)
	
	return nil
}

// WithTx 使用事务
func (r *pusherMachineRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &pusherMachineRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// PusherSessionRepository 推币机会话仓储接口
type PusherSessionRepository interface {
	BaseRepository
	Create(ctx context.Context, session *models.PusherSession) error
	Update(ctx context.Context, session *models.PusherSession) error
	FindByID(ctx context.Context, id uint) (*models.PusherSession, error)
	FindBySessionID(ctx context.Context, sessionID string) (*models.PusherSession, error)
	FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.PusherSession, error)
	FindByMachineID(ctx context.Context, machineID uint, pagination *Pagination) ([]*models.PusherSession, error)
	FindActive(ctx context.Context) ([]*models.PusherSession, error)
	EndSession(ctx context.Context, sessionID string) error
	GetStatistics(ctx context.Context, machineID uint, start, end time.Time) (*PusherStatistics, error)
}

// PusherStatistics 推币机统计
type PusherStatistics struct {
	TotalSessions    int   `json:"total_sessions"`
	TotalCoinsUsed   int   `json:"total_coins_used"`
	TotalCoinsWon    int   `json:"total_coins_won"`
	TotalPrizesWon   int   `json:"total_prizes_won"`
	TotalPrizeValue  int64 `json:"total_prize_value"`
	AverageCoinsUsed int   `json:"average_coins_used"`
	AverageCoinsWon  int   `json:"average_coins_won"`
	AveragePlayTime  int   `json:"average_play_time"`
}

// pusherSessionRepo 推币机会话仓储实现
type pusherSessionRepo struct {
	*BaseRepo
}

// NewPusherSessionRepository 创建推币机会话仓储
func NewPusherSessionRepository(db *gorm.DB) PusherSessionRepository {
	return &pusherSessionRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建会话
func (r *pusherSessionRepo) Create(ctx context.Context, session *models.PusherSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// Update 更新会话
func (r *pusherSessionRepo) Update(ctx context.Context, session *models.PusherSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// FindByID 根据ID查找
func (r *pusherSessionRepo) FindByID(ctx context.Context, id uint) (*models.PusherSession, error) {
	var session models.PusherSession
	err := r.db.WithContext(ctx).First(&session, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, err
	}
	return &session, nil
}

// FindBySessionID 根据会话ID查找
func (r *pusherSessionRepo) FindBySessionID(ctx context.Context, sessionID string) (*models.PusherSession, error) {
	var session models.PusherSession
	// First try to find by GameSession.SessionID string
	err := r.db.WithContext(ctx).Model(&models.PusherSession{}).
		Joins("JOIN game_sessions ON pusher_sessions.session_id = game_sessions.id").
		Where("game_sessions.session_id = ?", sessionID).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, err
	}
	return &session, nil
}

// FindByUserID 根据用户ID查找
func (r *pusherSessionRepo) FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.PusherSession, error) {
	var sessions []*models.PusherSession
	query := r.db.WithContext(ctx).Model(&models.PusherSession{}).Where("user_id = ?", userID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&sessions).Error
	
	return sessions, err
}

// FindByMachineID 根据机器ID查找
func (r *pusherSessionRepo) FindByMachineID(ctx context.Context, machineID uint, pagination *Pagination) ([]*models.PusherSession, error) {
	var sessions []*models.PusherSession
	query := r.db.WithContext(ctx).Model(&models.PusherSession{}).Where("machine_id = ?", machineID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&sessions).Error
	
	return sessions, err
}

// FindActive 查找活跃会话
func (r *pusherSessionRepo) FindActive(ctx context.Context) ([]*models.PusherSession, error) {
	var sessions []*models.PusherSession
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&sessions).Error
	return sessions, err
}

// EndSession 结束会话
func (r *pusherSessionRepo) EndSession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).
		Model(&models.PusherSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":   "ended",
			"end_time": time.Now(),
		}).Error
}

// GetStatistics 获取统计数据
func (r *pusherSessionRepo) GetStatistics(ctx context.Context, machineID uint, start, end time.Time) (*PusherStatistics, error) {
	stats := &PusherStatistics{}
	
	query := r.db.WithContext(ctx).Model(&models.PusherSession{})
	if machineID > 0 {
		query = query.Where("machine_id = ?", machineID)
	}
	if !start.IsZero() && !end.IsZero() {
		query = query.Where("created_at BETWEEN ? AND ?", start, end)
	}
	
	// 获取基本统计
	var result struct {
		TotalSessions   int
		TotalCoinsUsed  int
		TotalCoinsWon   int
		TotalPrizesWon  int
		TotalPrizeValue int64
		AvgPlayTime     int
	}
	
	query.Select(`
		COUNT(*) as total_sessions,
		COALESCE(SUM(coins_used), 0) as total_coins_used,
		COALESCE(SUM(coins_won), 0) as total_coins_won,
		COALESCE(SUM(prizes_won), 0) as total_prizes_won,
		COALESCE(SUM(prize_value), 0) as total_prize_value,
		COALESCE(AVG(play_time), 0) as avg_play_time
	`).Scan(&result)
	
	stats.TotalSessions = result.TotalSessions
	stats.TotalCoinsUsed = result.TotalCoinsUsed
	stats.TotalCoinsWon = result.TotalCoinsWon
	stats.TotalPrizesWon = result.TotalPrizesWon
	stats.TotalPrizeValue = result.TotalPrizeValue
	stats.AveragePlayTime = result.AvgPlayTime
	
	if stats.TotalSessions > 0 {
		stats.AverageCoinsUsed = stats.TotalCoinsUsed / stats.TotalSessions
		stats.AverageCoinsWon = stats.TotalCoinsWon / stats.TotalSessions
	}
	
	return stats, nil
}

// WithTx 使用事务
func (r *pusherSessionRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &pusherSessionRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// CoinDropRepository 推币掉落记录仓储接口
type CoinDropRepository interface {
	BaseRepository
	Create(ctx context.Context, drop *models.CoinDrop) error
	BatchCreate(ctx context.Context, drops []*models.CoinDrop) error
	FindBySessionID(ctx context.Context, sessionID uint) ([]*models.CoinDrop, error)
	FindByDropType(ctx context.Context, dropType string) ([]*models.CoinDrop, error)
	GetTopDrops(ctx context.Context, limit int) ([]*models.CoinDrop, error)
	GetDailyDrops(ctx context.Context, date time.Time) ([]*models.CoinDrop, error)
}

// coinDropRepo 推币掉落记录仓储实现
type coinDropRepo struct {
	*BaseRepo
}

// NewCoinDropRepository 创建推币掉落记录仓储
func NewCoinDropRepository(db *gorm.DB) CoinDropRepository {
	return &coinDropRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建掉落记录
func (r *coinDropRepo) Create(ctx context.Context, drop *models.CoinDrop) error {
	return r.db.WithContext(ctx).Create(drop).Error
}

// BatchCreate 批量创建掉落记录
func (r *coinDropRepo) BatchCreate(ctx context.Context, drops []*models.CoinDrop) error {
	if len(drops) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(drops, 100).Error
}

// FindBySessionID 根据会话ID查找
func (r *coinDropRepo) FindBySessionID(ctx context.Context, sessionID uint) ([]*models.CoinDrop, error) {
	var drops []*models.CoinDrop
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&drops).Error
	return drops, err
}

// FindByDropType 根据掉落类型查找
func (r *coinDropRepo) FindByDropType(ctx context.Context, dropType string) ([]*models.CoinDrop, error) {
	var drops []*models.CoinDrop
	err := r.db.WithContext(ctx).Where("drop_type = ?", dropType).Find(&drops).Error
	return drops, err
}

// GetTopDrops 获取最高掉落记录
func (r *coinDropRepo) GetTopDrops(ctx context.Context, limit int) ([]*models.CoinDrop, error) {
	var drops []*models.CoinDrop
	err := r.db.WithContext(ctx).
		Order("value DESC").
		Limit(limit).
		Find(&drops).Error
	return drops, err
}

// GetDailyDrops 获取每日掉落记录
func (r *coinDropRepo) GetDailyDrops(ctx context.Context, date time.Time) ([]*models.CoinDrop, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	var drops []*models.CoinDrop
	err := r.db.WithContext(ctx).
		Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Find(&drops).Error
	return drops, err
}

// WithTx 使用事务
func (r *coinDropRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &coinDropRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}