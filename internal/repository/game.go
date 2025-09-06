package repository

import (
	"context"
	"errors"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// GameRepository 游戏仓储接口
type GameRepository interface {
	BaseRepository
	Create(ctx context.Context, game *models.Game) error
	Update(ctx context.Context, game *models.Game) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.Game, error)
	FindByName(ctx context.Context, name string) (*models.Game, error)
	FindByType(ctx context.Context, gameType string) ([]*models.Game, error)
	GetAll(ctx context.Context, pagination *Pagination) ([]*models.Game, error)
	GetActive(ctx context.Context) ([]*models.Game, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
}

// gameRepo 游戏仓储实现
type gameRepo struct {
	*BaseRepo
}

// NewGameRepository 创建游戏仓储
func NewGameRepository(db *gorm.DB) GameRepository {
	return &gameRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建游戏
func (r *gameRepo) Create(ctx context.Context, game *models.Game) error {
	return r.db.WithContext(ctx).Create(game).Error
}

// Update 更新游戏
func (r *gameRepo) Update(ctx context.Context, game *models.Game) error {
	return r.db.WithContext(ctx).Save(game).Error
}

// Delete 删除游戏
func (r *gameRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Game{}, id).Error
}

// FindByID 根据ID查找游戏
func (r *gameRepo) FindByID(ctx context.Context, id uint) (*models.Game, error) {
	var game models.Game
	err := r.db.WithContext(ctx).First(&game, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("游戏不存在")
		}
		return nil, err
	}
	return &game, nil
}

// FindByName 根据名称查找游戏
func (r *gameRepo) FindByName(ctx context.Context, name string) (*models.Game, error) {
	var game models.Game
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&game).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("游戏不存在")
		}
		return nil, err
	}
	return &game, nil
}

// FindByType 根据类型查找游戏
func (r *gameRepo) FindByType(ctx context.Context, gameType string) ([]*models.Game, error) {
	var games []*models.Game
	err := r.db.WithContext(ctx).Where("type = ?", gameType).Find(&games).Error
	return games, err
}

// GetAll 获取所有游戏（分页）
func (r *gameRepo) GetAll(ctx context.Context, pagination *Pagination) ([]*models.Game, error) {
	var games []*models.Game
	query := r.db.WithContext(ctx).Model(&models.Game{})
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&games).Error
	
	return games, err
}

// GetActive 获取所有活跃游戏
func (r *gameRepo) GetActive(ctx context.Context) ([]*models.Game, error) {
	var games []*models.Game
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&games).Error
	return games, err
}

// UpdateStatus 更新游戏状态
func (r *gameRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.Game{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// WithTx 使用事务
func (r *gameRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &gameRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// GameRoomRepository 游戏房间仓储接口
type GameRoomRepository interface {
	BaseRepository
	Create(ctx context.Context, room *models.GameRoom) error
	Update(ctx context.Context, room *models.GameRoom) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.GameRoom, error)
	FindByRoomNumber(ctx context.Context, roomNumber string) (*models.GameRoom, error)
	FindByGameID(ctx context.Context, gameID uint) ([]*models.GameRoom, error)
	GetActive(ctx context.Context) ([]*models.GameRoom, error)
	GetByType(ctx context.Context, roomType string) ([]*models.GameRoom, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdatePlayerCount(ctx context.Context, id uint, delta int) error
	GetRoomStatistics(ctx context.Context, roomID uint) (*RoomStatistics, error)
}

// RoomStatistics 房间统计信息
type RoomStatistics struct {
	TotalPlayers    int   `json:"total_players"`
	CurrentPlayers  int   `json:"current_players"`
	TotalBet        int64 `json:"total_bet"`
	TotalWin        int64 `json:"total_win"`
	TotalRounds     int   `json:"total_rounds"`
	AveragePlayTime int   `json:"average_play_time"`
}

// gameRoomRepo 游戏房间仓储实现
type gameRoomRepo struct {
	*BaseRepo
}

// NewGameRoomRepository 创建游戏房间仓储
func NewGameRoomRepository(db *gorm.DB) GameRoomRepository {
	return &gameRoomRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建房间
func (r *gameRoomRepo) Create(ctx context.Context, room *models.GameRoom) error {
	return r.db.WithContext(ctx).Create(room).Error
}

// Update 更新房间
func (r *gameRoomRepo) Update(ctx context.Context, room *models.GameRoom) error {
	return r.db.WithContext(ctx).Save(room).Error
}

// Delete 删除房间
func (r *gameRoomRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.GameRoom{}, id).Error
}

// FindByID 根据ID查找房间
func (r *gameRoomRepo) FindByID(ctx context.Context, id uint) (*models.GameRoom, error) {
	var room models.GameRoom
	err := r.db.WithContext(ctx).First(&room, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("房间不存在")
		}
		return nil, err
	}
	return &room, nil
}

// FindByRoomNumber 根据房间号查找
func (r *gameRoomRepo) FindByRoomNumber(ctx context.Context, roomNumber string) (*models.GameRoom, error) {
	var room models.GameRoom
	err := r.db.WithContext(ctx).Where("room_number = ?", roomNumber).First(&room).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("房间不存在")
		}
		return nil, err
	}
	return &room, nil
}

// FindByGameID 根据游戏ID查找房间
func (r *gameRoomRepo) FindByGameID(ctx context.Context, gameID uint) ([]*models.GameRoom, error) {
	var rooms []*models.GameRoom
	err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Find(&rooms).Error
	return rooms, err
}

// GetActive 获取所有活跃房间
func (r *gameRoomRepo) GetActive(ctx context.Context) ([]*models.GameRoom, error) {
	var rooms []*models.GameRoom
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&rooms).Error
	return rooms, err
}

// GetByType 根据类型获取房间
func (r *gameRoomRepo) GetByType(ctx context.Context, roomType string) ([]*models.GameRoom, error) {
	var rooms []*models.GameRoom
	err := r.db.WithContext(ctx).Where("type = ?", roomType).Find(&rooms).Error
	return rooms, err
}

// UpdateStatus 更新房间状态
func (r *gameRoomRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.GameRoom{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdatePlayerCount 更新玩家数量
func (r *gameRoomRepo) UpdatePlayerCount(ctx context.Context, id uint, delta int) error {
	result := r.db.WithContext(ctx).
		Model(&models.GameRoom{}).
		Where("id = ?", id).
		Update("current_players", gorm.Expr("current_players + ?", delta))
	
	if result.Error != nil {
		return result.Error
	}
	
	// 确保玩家数量不会小于0
	r.db.WithContext(ctx).
		Model(&models.GameRoom{}).
		Where("id = ? AND current_players < 0", id).
		Update("current_players", 0)
	
	return nil
}

// GetRoomStatistics 获取房间统计信息
func (r *gameRoomRepo) GetRoomStatistics(ctx context.Context, roomID uint) (*RoomStatistics, error) {
	stats := &RoomStatistics{}
	
	// 获取当前房间信息
	var room models.GameRoom
	err := r.db.WithContext(ctx).First(&room, roomID).Error
	if err != nil {
		return nil, err
	}
	stats.CurrentPlayers = room.CurrentPlayers
	
	// 获取游戏会话统计
	var sessionStats struct {
		TotalPlayers int
		TotalBet     int64
		TotalWin     int64
		TotalRounds  int
		AvgDuration  int
	}
	
	r.db.WithContext(ctx).
		Model(&models.GameSession{}).
		Where("room_id = ?", roomID).
		Select(`
			COUNT(DISTINCT user_id) as total_players,
			COALESCE(SUM(total_bet), 0) as total_bet,
			COALESCE(SUM(total_win), 0) as total_win,
			COALESCE(SUM(total_rounds), 0) as total_rounds,
			COALESCE(AVG(duration), 0) as avg_duration
		`).
		Scan(&sessionStats)
	
	stats.TotalPlayers = sessionStats.TotalPlayers
	stats.TotalBet = sessionStats.TotalBet
	stats.TotalWin = sessionStats.TotalWin
	stats.TotalRounds = sessionStats.TotalRounds
	stats.AveragePlayTime = sessionStats.AvgDuration
	
	return stats, nil
}

// WithTx 使用事务
func (r *gameRoomRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &gameRoomRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}