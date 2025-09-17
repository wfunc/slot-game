package repository

import (
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// JackpotRepository JP奖池仓库
type JackpotRepository struct {
	db *gorm.DB
}

// NewJackpotRepository 创建JP奖池仓库
func NewJackpotRepository(db *gorm.DB) *JackpotRepository {
	return &JackpotRepository{db: db}
}

// InitializeJackpots 初始化JP奖池
func (r *JackpotRepository) InitializeJackpots(gameID uint) error {
	jpTypes := []struct {
		Type       string
		Percentage float64
		MinAmount  int64
		MaxAmount  int64
	}{
		{"JP1", 0.10, 1000, 1000000},    // JP1: 10%抽成
		{"JP2", 0.05, 5000, 5000000},    // JP2: 5%抽成
		{"JP3", 0.05, 10000, 10000000},  // JP3: 5%抽成
		{"JPALL", 0.05, 20000, 20000000}, // JPALL: 5%抽成
	}

	for _, jp := range jpTypes {
		var existing models.Jackpot
		result := r.db.Where("game_id = ? AND type = ?", gameID, jp.Type).First(&existing)
		
		if result.Error == gorm.ErrRecordNotFound {
			// 创建新的JP池
			jackpot := models.Jackpot{
				GameID:     gameID,
				Type:       jp.Type,
				Amount:     jp.MinAmount, // 初始化为最小金额
				MinAmount:  jp.MinAmount,
				MaxAmount:  jp.MaxAmount,
				Percentage: jp.Percentage,
				Status:     "active",
			}
			
			if err := r.db.Create(&jackpot).Error; err != nil {
				return fmt.Errorf("创建JP池 %s 失败: %w", jp.Type, err)
			}
		}
	}
	
	return nil
}

// GetJackpot 获取指定JP池
func (r *JackpotRepository) GetJackpot(gameID uint, jpType string) (*models.Jackpot, error) {
	var jackpot models.Jackpot
	err := r.db.Where("game_id = ? AND type = ?", gameID, jpType).First(&jackpot).Error
	if err != nil {
		return nil, err
	}
	return &jackpot, nil
}

// GetAllJackpots 获取所有JP池
func (r *JackpotRepository) GetAllJackpots(gameID uint) ([]models.Jackpot, error) {
	var jackpots []models.Jackpot
	err := r.db.Where("game_id = ?", gameID).Find(&jackpots).Error
	return jackpots, err
}

// AccumulateJackpot 累计JP奖池（事务处理）
func (r *JackpotRepository) AccumulateJackpot(tx *gorm.DB, gameID uint, betAmount int64) error {
	// 计算各JP池的累计金额
	jp1Amount := int64(float64(betAmount) * 0.10)  // JP1: 10%
	jp2Amount := int64(float64(betAmount) * 0.05)  // JP2: 5%
	jp3Amount := int64(float64(betAmount) * 0.05)  // JP3: 5%
	jpAllAmount := int64(float64(betAmount) * 0.05) // JPALL: 5%
	
	// 添加日志
	fmt.Printf("[JackpotRepo] 累计JP池 - 下注: %d, JP1: %d(10%%), JP2: %d(5%%), JP3: %d(5%%), JPALL: %d(5%%)\n", 
		betAmount, jp1Amount, jp2Amount, jp3Amount, jpAllAmount)

	updates := []struct {
		Type   string
		Amount int64
	}{
		{"JP1", jp1Amount},
		{"JP2", jp2Amount},
		{"JP3", jp3Amount},
		{"JPALL", jpAllAmount},
	}

	for _, update := range updates {
		// 更新JP池金额和统计 (使用SQLite兼容的CASE语法替代LEAST)
		result := tx.Model(&models.Jackpot{}).
			Where("game_id = ? AND type = ? AND status = ?", gameID, update.Type, "active").
			Updates(map[string]interface{}{
				"amount":   gorm.Expr("CASE WHEN amount + ? > max_amount THEN max_amount ELSE amount + ? END", update.Amount, update.Amount),
				"total_in": gorm.Expr("total_in + ?", update.Amount),
			})
		
		if result.Error != nil {
			return fmt.Errorf("更新JP池 %s 失败: %w", update.Type, result.Error)
		}
		
		if result.RowsAffected == 0 {
			// 如果没有找到，尝试初始化
			if err := r.InitializeJackpots(gameID); err != nil {
				return err
			}
			// 重试更新 (使用SQLite兼容的CASE语法替代LEAST)
			if err := tx.Model(&models.Jackpot{}).
				Where("game_id = ? AND type = ?", gameID, update.Type).
				Updates(map[string]interface{}{
					"amount":   gorm.Expr("CASE WHEN amount + ? > max_amount THEN max_amount ELSE amount + ? END", update.Amount, update.Amount),
					"total_in": gorm.Expr("total_in + ?", update.Amount),
				}).Error; err != nil {
				return fmt.Errorf("重试更新JP池 %s 失败: %w", update.Type, err)
			}
		}
	}
	
	return nil
}

// WinJackpot 中JP处理
func (r *JackpotRepository) WinJackpot(tx *gorm.DB, gameID uint, userID uint, sessionID uint, resultID uint, jpType string) (int64, error) {
	// 获取当前JP池
	var jackpot models.Jackpot
	if err := tx.Where("game_id = ? AND type = ? AND status = ?", gameID, jpType, "active").
		First(&jackpot).Error; err != nil {
		return 0, fmt.Errorf("获取JP池失败: %w", err)
	}
	
	// 记录中奖金额
	winAmount := jackpot.Amount
	poolBefore := jackpot.Amount
	
	// 重置JP池到最小金额
	now := time.Now()
	if err := tx.Model(&jackpot).Updates(map[string]interface{}{
		"amount":      jackpot.MinAmount,
		"last_won_at": now,
		"last_winner": userID,
		"win_count":   gorm.Expr("win_count + 1"),
		"total_out":   gorm.Expr("total_out + ?", winAmount),
	}).Error; err != nil {
		return 0, fmt.Errorf("更新JP池失败: %w", err)
	}
	
	// 记录中奖历史
	history := models.JackpotHistory{
		JackpotID:  jackpot.ID,
		UserID:     userID,
		SessionID:  sessionID,
		ResultID:   resultID,
		Amount:     winAmount,
		PoolBefore: poolBefore,
		PoolAfter:  jackpot.MinAmount,
		WonAt:      now,
	}
	
	if err := tx.Create(&history).Error; err != nil {
		return 0, fmt.Errorf("记录JP中奖历史失败: %w", err)
	}
	
	return winAmount, nil
}

// GetJackpotHistory 获取JP中奖历史
func (r *JackpotRepository) GetJackpotHistory(gameID uint, limit int) ([]models.JackpotHistory, error) {
	var history []models.JackpotHistory
	
	query := r.db.
		Joins("JOIN jackpots ON jackpots.id = jackpot_histories.jackpot_id").
		Where("jackpots.game_id = ?", gameID).
		Preload("Jackpot").
		Order("jackpot_histories.won_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&history).Error
	return history, err
}

// GetUserJackpotHistory 获取用户JP中奖历史
func (r *JackpotRepository) GetUserJackpotHistory(userID uint, limit int) ([]models.JackpotHistory, error) {
	var history []models.JackpotHistory
	
	query := r.db.
		Where("user_id = ?", userID).
		Preload("Jackpot").
		Order("won_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&history).Error
	return history, err
}

// GetJackpotStats 获取JP统计信息
func (r *JackpotRepository) GetJackpotStats(gameID uint) (map[string]interface{}, error) {
	var jackpots []models.Jackpot
	if err := r.db.Where("game_id = ?", gameID).Find(&jackpots).Error; err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	totalAmount := int64(0)
	totalIn := int64(0)
	totalOut := int64(0)
	totalWins := 0
	
	for _, jp := range jackpots {
		totalAmount += jp.Amount
		totalIn += jp.TotalIn
		totalOut += jp.TotalOut
		totalWins += jp.WinCount
		
		stats[jp.Type] = map[string]interface{}{
			"amount":     jp.Amount,
			"win_count":  jp.WinCount,
			"last_won":   jp.LastWonAt,
			"last_winner": jp.LastWinner,
		}
	}
	
	stats["total"] = map[string]interface{}{
		"amount":     totalAmount,
		"total_in":   totalIn,
		"total_out":  totalOut,
		"total_wins": totalWins,
	}
	
	return stats, nil
}