package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// SlotMachineRepository 老虎机仓储接口
type SlotMachineRepository interface {
	BaseRepository
	Create(ctx context.Context, machine *models.SlotMachine) error
	Update(ctx context.Context, machine *models.SlotMachine) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.SlotMachine, error)
	FindByMachineID(ctx context.Context, machineID string) (*models.SlotMachine, error)
	FindByGameID(ctx context.Context, gameID uint) ([]*models.SlotMachine, error)
	GetActive(ctx context.Context) ([]*models.SlotMachine, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateLastSpin(ctx context.Context, id uint) error
}

// slotMachineRepo 老虎机仓储实现
type slotMachineRepo struct {
	*BaseRepo
}

// NewSlotMachineRepository 创建老虎机仓储
func NewSlotMachineRepository(db *gorm.DB) SlotMachineRepository {
	return &slotMachineRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建老虎机
func (r *slotMachineRepo) Create(ctx context.Context, machine *models.SlotMachine) error {
	return r.db.WithContext(ctx).Create(machine).Error
}

// Update 更新老虎机
func (r *slotMachineRepo) Update(ctx context.Context, machine *models.SlotMachine) error {
	return r.db.WithContext(ctx).Save(machine).Error
}

// Delete 删除老虎机
func (r *slotMachineRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.SlotMachine{}, id).Error
}

// FindByID 根据ID查找
func (r *slotMachineRepo) FindByID(ctx context.Context, id uint) (*models.SlotMachine, error) {
	var machine models.SlotMachine
	err := r.db.WithContext(ctx).First(&machine, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("老虎机不存在")
		}
		return nil, err
	}
	return &machine, nil
}

// FindByMachineID 根据机器ID查找
func (r *slotMachineRepo) FindByMachineID(ctx context.Context, machineID string) (*models.SlotMachine, error) {
	var machine models.SlotMachine
	err := r.db.WithContext(ctx).Where("machine_id = ?", machineID).First(&machine).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("老虎机不存在")
		}
		return nil, err
	}
	return &machine, nil
}

// FindByGameID 根据游戏ID查找
func (r *slotMachineRepo) FindByGameID(ctx context.Context, gameID uint) ([]*models.SlotMachine, error) {
	var machines []*models.SlotMachine
	err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Find(&machines).Error
	return machines, err
}

// GetActive 获取活跃的老虎机
func (r *slotMachineRepo) GetActive(ctx context.Context) ([]*models.SlotMachine, error) {
	var machines []*models.SlotMachine
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&machines).Error
	return machines, err
}

// UpdateStatus 更新状态
func (r *slotMachineRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.SlotMachine{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateLastSpin 更新最后旋转时间
func (r *slotMachineRepo) UpdateLastSpin(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.SlotMachine{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_spin_at":  time.Now(),
			"total_spins":   gorm.Expr("total_spins + 1"),
		}).Error
}

// WithTx 使用事务
func (r *slotMachineRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &slotMachineRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// SlotSpinRepository 老虎机旋转记录仓储接口
type SlotSpinRepository interface {
	BaseRepository
	Create(ctx context.Context, spin *models.SlotSpin) error
	BatchCreate(ctx context.Context, spins []*models.SlotSpin) error
	FindByID(ctx context.Context, id uint) (*models.SlotSpin, error)
	FindBySpinID(ctx context.Context, spinID string) (*models.SlotSpin, error)
	FindBySessionID(ctx context.Context, sessionID uint, pagination *Pagination) ([]*models.SlotSpin, error)
	FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.SlotSpin, error)
	FindByMachineID(ctx context.Context, machineID uint, pagination *Pagination) ([]*models.SlotSpin, error)
	GetStatistics(ctx context.Context, machineID uint, start, end time.Time) (*SpinStatistics, error)
}

// SpinStatistics 旋转统计
type SpinStatistics struct {
	TotalSpins   int   `json:"total_spins"`
	TotalBet     int64 `json:"total_bet"`
	TotalPayout  int64 `json:"total_payout"`
	TotalProfit  int64 `json:"total_profit"`
	WinCount     int   `json:"win_count"`
	LossCount    int   `json:"loss_count"`
	MaxWin       int64 `json:"max_win"`
	MinWin       int64 `json:"min_win"`
	AverageBet   int64 `json:"average_bet"`
	RTP          float64 `json:"rtp"` // Return to Player percentage
}

// slotSpinRepo 老虎机旋转记录仓储实现
type slotSpinRepo struct {
	*BaseRepo
}

// NewSlotSpinRepository 创建老虎机旋转记录仓储
func NewSlotSpinRepository(db *gorm.DB) SlotSpinRepository {
	return &slotSpinRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建旋转记录
func (r *slotSpinRepo) Create(ctx context.Context, spin *models.SlotSpin) error {
	return r.db.WithContext(ctx).Create(spin).Error
}

// BatchCreate 批量创建旋转记录
func (r *slotSpinRepo) BatchCreate(ctx context.Context, spins []*models.SlotSpin) error {
	if len(spins) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(spins, 100).Error
}

// FindByID 根据ID查找
func (r *slotSpinRepo) FindByID(ctx context.Context, id uint) (*models.SlotSpin, error) {
	var spin models.SlotSpin
	err := r.db.WithContext(ctx).First(&spin, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("旋转记录不存在")
		}
		return nil, err
	}
	return &spin, nil
}

// FindBySpinID 根据旋转ID查找
func (r *slotSpinRepo) FindBySpinID(ctx context.Context, spinID string) (*models.SlotSpin, error) {
	var spin models.SlotSpin
	err := r.db.WithContext(ctx).Where("spin_id = ?", spinID).First(&spin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("旋转记录不存在")
		}
		return nil, err
	}
	return &spin, nil
}

// FindBySessionID 根据会话ID查找
func (r *slotSpinRepo) FindBySessionID(ctx context.Context, sessionID uint, pagination *Pagination) ([]*models.SlotSpin, error) {
	var spins []*models.SlotSpin
	query := r.db.WithContext(ctx).Model(&models.SlotSpin{}).Where("session_id = ?", sessionID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&spins).Error
	
	return spins, err
}

// FindByUserID 根据用户ID查找
func (r *slotSpinRepo) FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.SlotSpin, error) {
	var spins []*models.SlotSpin
	query := r.db.WithContext(ctx).Model(&models.SlotSpin{}).Where("user_id = ?", userID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&spins).Error
	
	return spins, err
}

// FindByMachineID 根据机器ID查找
func (r *slotSpinRepo) FindByMachineID(ctx context.Context, machineID uint, pagination *Pagination) ([]*models.SlotSpin, error) {
	var spins []*models.SlotSpin
	query := r.db.WithContext(ctx).Model(&models.SlotSpin{}).Where("machine_id = ?", machineID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&spins).Error
	
	return spins, err
}

// GetStatistics 获取统计数据
func (r *slotSpinRepo) GetStatistics(ctx context.Context, machineID uint, start, end time.Time) (*SpinStatistics, error) {
	stats := &SpinStatistics{}
	
	query := r.db.WithContext(ctx).Model(&models.SlotSpin{})
	if machineID > 0 {
		query = query.Where("machine_id = ?", machineID)
	}
	if !start.IsZero() && !end.IsZero() {
		query = query.Where("created_at BETWEEN ? AND ?", start, end)
	}
	
	// 获取基本统计
	var result struct {
		TotalSpins  int
		TotalBet    int64
		TotalPayout int64
		MaxWin      int64
		MinWin      int64
	}
	
	query.Select(`
		COUNT(*) as total_spins,
		COALESCE(SUM(bet_amount), 0) as total_bet,
		COALESCE(SUM(payout), 0) as total_payout,
		COALESCE(MAX(payout), 0) as max_win,
		COALESCE(MIN(CASE WHEN payout > 0 THEN payout END), 0) as min_win
	`).Scan(&result)
	
	stats.TotalSpins = result.TotalSpins
	stats.TotalBet = result.TotalBet
	stats.TotalPayout = result.TotalPayout
	stats.MaxWin = result.MaxWin
	stats.MinWin = result.MinWin
	stats.TotalProfit = stats.TotalBet - stats.TotalPayout
	
	if stats.TotalSpins > 0 {
		stats.AverageBet = stats.TotalBet / int64(stats.TotalSpins)
	}
	
	if stats.TotalBet > 0 {
		stats.RTP = float64(stats.TotalPayout) / float64(stats.TotalBet) * 100
	}
	
	// 统计输赢次数
	var winCount int64
	query.Where("payout > 0").Count(&winCount)
	stats.WinCount = int(winCount)
	stats.LossCount = stats.TotalSpins - stats.WinCount
	
	return stats, nil
}

// WithTx 使用事务
func (r *slotSpinRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &slotSpinRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// SlotWinLineRepository 老虎机中奖线仓储接口
type SlotWinLineRepository interface {
	BaseRepository
	Create(ctx context.Context, winLine *models.SlotWinLine) error
	BatchCreate(ctx context.Context, winLines []*models.SlotWinLine) error
	FindBySpinID(ctx context.Context, spinID uint) ([]*models.SlotWinLine, error)
	FindBySymbol(ctx context.Context, symbol string) ([]*models.SlotWinLine, error)
	GetTopWins(ctx context.Context, limit int) ([]*models.SlotWinLine, error)
}

// slotWinLineRepo 老虎机中奖线仓储实现
type slotWinLineRepo struct {
	*BaseRepo
}

// NewSlotWinLineRepository 创建老虎机中奖线仓储
func NewSlotWinLineRepository(db *gorm.DB) SlotWinLineRepository {
	return &slotWinLineRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建中奖线
func (r *slotWinLineRepo) Create(ctx context.Context, winLine *models.SlotWinLine) error {
	return r.db.WithContext(ctx).Create(winLine).Error
}

// BatchCreate 批量创建中奖线
func (r *slotWinLineRepo) BatchCreate(ctx context.Context, winLines []*models.SlotWinLine) error {
	if len(winLines) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(winLines, 100).Error
}

// FindBySpinID 根据旋转ID查找
func (r *slotWinLineRepo) FindBySpinID(ctx context.Context, spinID uint) ([]*models.SlotWinLine, error) {
	var winLines []*models.SlotWinLine
	err := r.db.WithContext(ctx).Where("spin_id = ?", spinID).Find(&winLines).Error
	return winLines, err
}

// FindBySymbol 根据符号查找
func (r *slotWinLineRepo) FindBySymbol(ctx context.Context, symbol string) ([]*models.SlotWinLine, error) {
	var winLines []*models.SlotWinLine
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&winLines).Error
	return winLines, err
}

// GetTopWins 获取最高赢奖记录
func (r *slotWinLineRepo) GetTopWins(ctx context.Context, limit int) ([]*models.SlotWinLine, error) {
	var winLines []*models.SlotWinLine
	err := r.db.WithContext(ctx).
		Order("win_amount DESC").
		Limit(limit).
		Find(&winLines).Error
	return winLines, err
}

// WithTx 使用事务
func (r *slotWinLineRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &slotWinLineRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}