package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletRepository 钱包仓储接口
type WalletRepository interface {
	BaseRepository
	Create(ctx context.Context, wallet *models.Wallet) error
	FindByUserID(ctx context.Context, userID uint) (*models.Wallet, error)
	GetByUserID(ctx context.Context, userID uint) (*models.Wallet, error) // 别名，兼容性
	UpdateBalance(ctx context.Context, userID uint, amount int64) error
	AddBalance(ctx context.Context, userID uint, amount int64) error
	DeductBalance(ctx context.Context, userID uint, amount int64) error
	LockForUpdate(ctx context.Context, userID uint) (*models.Wallet, error)
	UpdateStatistics(ctx context.Context, userID uint, field string, amount int64) error
	UpdateGameStatsTx(tx *gorm.DB, userID uint, betAmount, winAmount, coinsIn, coinsOut int64) error
	CreateTransaction(ctx context.Context, transaction *models.WalletTransaction) error
}

// walletRepo 钱包仓储实现
type walletRepo struct {
	*BaseRepo
}

// NewWalletRepository 创建钱包仓储
func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建钱包
func (r *walletRepo) Create(ctx context.Context, wallet *models.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

// FindByUserID 根据用户ID查找钱包
func (r *walletRepo) FindByUserID(ctx context.Context, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("钱包不存在")
		}
		return nil, err
	}
	return &wallet, nil
}

// GetByUserID 根据用户ID查找钱包（兼容性别名）
func (r *walletRepo) GetByUserID(ctx context.Context, userID uint) (*models.Wallet, error) {
	return r.FindByUserID(ctx, userID)
}

// UpdateBalance 更新余额
func (r *walletRepo) UpdateBalance(ctx context.Context, userID uint, amount int64) error {
	return r.db.WithContext(ctx).
		Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Update("balance", amount).Error
}

// AddBalance 增加余额
func (r *walletRepo) AddBalance(ctx context.Context, userID uint, amount int64) error {
	return r.db.WithContext(ctx).
		Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// DeductBalance 扣减余额
func (r *walletRepo) DeductBalance(ctx context.Context, userID uint, amount int64) error {
	result := r.db.WithContext(ctx).
		Model(&models.Wallet{}).
		Where("user_id = ? AND balance >= ?", userID, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("余额不足")
	}
	
	return nil
}

// LockForUpdate 锁定钱包用于更新（悲观锁）
func (r *walletRepo) LockForUpdate(ctx context.Context, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("钱包不存在")
		}
		return nil, err
	}
	return &wallet, nil
}

// UpdateStatistics 更新统计信息
func (r *walletRepo) UpdateStatistics(ctx context.Context, userID uint, field string, amount int64) error {
	allowedFields := map[string]bool{
		"total_withdraw": true,
		"total_win":      true,
		"total_bet":      true,
		"total_deposit":  true,
		"total_coins_in": true,
		"total_coins_out": true,
	}
	
	if !allowedFields[field] {
		return errors.New("不允许的字段")
	}
	
	return r.db.WithContext(ctx).
		Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Update(field, gorm.Expr(field+" + ?", amount)).Error
}

// UpdateGameStatsTx 在事务中更新游戏统计（包括投币/落币）
func (r *walletRepo) UpdateGameStatsTx(tx *gorm.DB, userID uint, betAmount, winAmount, coinsIn, coinsOut int64) error {
	updates := map[string]interface{}{
		"total_bet":       gorm.Expr("total_bet + ?", betAmount),
		"total_win":       gorm.Expr("total_win + ?", winAmount),
		"total_coins_in":  gorm.Expr("total_coins_in + ?", coinsIn),
		"total_coins_out": gorm.Expr("total_coins_out + ?", coinsOut),
		"daily_bet":       gorm.Expr("daily_bet + ?", betAmount),
		"daily_win":       gorm.Expr("daily_win + ?", winAmount),
		"coins":          gorm.Expr("coins - ? + ?", betAmount, winAmount), // 更新游戏币余额
	}
	
	result := tx.Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Updates(updates)
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("钱包不存在")
	}
	
	return nil
}

// CreateTransaction 创建交易记录
func (r *walletRepo) CreateTransaction(ctx context.Context, transaction *models.WalletTransaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// WithTx 使用事务
func (r *walletRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &walletRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// TransactionRepository 交易记录仓储接口
type TransactionRepository interface {
	BaseRepository
	Create(ctx context.Context, transaction *models.Transaction) error
	FindByID(ctx context.Context, id uint) (*models.Transaction, error)
	FindByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error)
	FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.Transaction, error)
	FindByType(ctx context.Context, txType string, pagination *Pagination) ([]*models.Transaction, error)
	UpdateStatus(ctx context.Context, transactionID string, status string) error
	GetDailyStatistics(ctx context.Context, userID uint, date time.Time) (*TransactionStats, error)
}

// TransactionStats 交易统计
type TransactionStats struct {
	TotalIn      int64 `json:"total_in"`
	TotalOut     int64 `json:"total_out"`
	NetAmount    int64 `json:"net_amount"`
	TransCount   int   `json:"trans_count"`
	WinCount     int   `json:"win_count"`
	BetCount     int   `json:"bet_count"`
	RechargeSum  int64 `json:"recharge_sum"`
	WithdrawSum  int64 `json:"withdraw_sum"`
}

// transactionRepo 交易记录仓储实现
type transactionRepo struct {
	*BaseRepo
}

// NewTransactionRepository 创建交易记录仓储
func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建交易记录
func (r *transactionRepo) Create(ctx context.Context, transaction *models.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// FindByID 根据ID查找交易
func (r *transactionRepo) FindByID(ctx context.Context, id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.WithContext(ctx).First(&transaction, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("交易记录不存在")
		}
		return nil, err
	}
	return &transaction, nil
}

// FindByTransactionID 根据交易ID查找
func (r *transactionRepo) FindByTransactionID(ctx context.Context, transactionID string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.WithContext(ctx).Where("order_no = ?", transactionID).First(&transaction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("交易记录不存在")
		}
		return nil, err
	}
	return &transaction, nil
}

// FindByUserID 查找用户的交易记录
func (r *transactionRepo) FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	query := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("user_id = ?", userID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&transactions).Error
	
	return transactions, err
}

// FindByType 根据类型查找交易
func (r *transactionRepo) FindByType(ctx context.Context, txType string, pagination *Pagination) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	query := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("type = ?", txType)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&transactions).Error
	
	return transactions, err
}

// UpdateStatus 更新交易状态
func (r *transactionRepo) UpdateStatus(ctx context.Context, transactionID string, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("order_no = ?", transactionID).
		Update("status", status).Error
}

// GetDailyStatistics 获取日统计
func (r *transactionRepo) GetDailyStatistics(ctx context.Context, userID uint, date time.Time) (*TransactionStats, error) {
	stats := &TransactionStats{}
	
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	// 统计收入
	var totalIn int64
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type IN (?) AND created_at BETWEEN ? AND ?", 
			userID, []string{"win", "deposit", "refund"}, startOfDay, endOfDay).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalIn)
	stats.TotalIn = totalIn
	
	// 统计支出
	var totalOut int64
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type IN (?) AND created_at BETWEEN ? AND ?", 
			userID, []string{"bet", "withdraw"}, startOfDay, endOfDay).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalOut)
	stats.TotalOut = totalOut
	
	stats.NetAmount = totalIn - totalOut
	
	// 统计交易次数
	var transCount int64
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startOfDay, endOfDay).
		Count(&transCount)
	stats.TransCount = int(transCount)
	
	// 分类统计
	var winCount int64
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND created_at BETWEEN ? AND ?", 
			userID, "win", startOfDay, endOfDay).
		Count(&winCount)
	stats.WinCount = int(winCount)
	
	var betCount int64
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND created_at BETWEEN ? AND ?", 
			userID, "bet", startOfDay, endOfDay).
		Count(&betCount)
	stats.BetCount = int(betCount)
	
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND created_at BETWEEN ? AND ?", 
			userID, "deposit", startOfDay, endOfDay).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.RechargeSum)
	
	r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND created_at BETWEEN ? AND ?", 
			userID, "withdraw", startOfDay, endOfDay).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.WithdrawSum)
	
	return stats, nil
}

// WithTx 使用事务
func (r *transactionRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &transactionRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}