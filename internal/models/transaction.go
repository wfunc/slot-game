package models

import (
	"time"
	
	"gorm.io/gorm"
)

// Wallet 用户钱包表
type Wallet struct {
	BaseModel
	UserID          uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Balance         int64     `gorm:"default:0" json:"balance"` // 余额（分）
	Coins           int64     `gorm:"default:0" json:"coins"`   // 游戏币
	Points          int64     `gorm:"default:0" json:"points"`  // 积分
	FrozenBalance   int64     `gorm:"default:0" json:"frozen_balance"`
	FrozenCoins     int64     `gorm:"default:0" json:"frozen_coins"`
	TotalDeposit    int64     `gorm:"default:0" json:"total_deposit"`
	TotalWithdraw   int64     `gorm:"default:0" json:"total_withdraw"`
	TotalBet        int64     `gorm:"default:0" json:"total_bet"`
	TotalWin        int64     `gorm:"default:0" json:"total_win"`
	TotalCoinsIn    int64     `gorm:"default:0" json:"total_coins_in"`  // 总投币数
	TotalCoinsOut   int64     `gorm:"default:0" json:"total_coins_out"` // 总落币数
	DailyBet        int64     `gorm:"default:0" json:"daily_bet"`
	DailyWin        int64     `gorm:"default:0" json:"daily_win"`
	LastResetAt     time.Time `json:"last_reset_at"`
	
	// 关联（注意：不直接嵌入 User，避免循环依赖）
	// 查询时使用 Preload("User") 来加载用户信息
}

// WalletTransaction 是 Transaction 的别名，用于兼容性
type WalletTransaction = Transaction

// Transaction 交易记录表
type Transaction struct {
	BaseModel
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	OrderNo         string    `gorm:"uniqueIndex;size:64;not null" json:"order_no"`
	Type            string    `gorm:"size:50;not null;index" json:"type"` // deposit, withdraw, bet, win, refund, bonus, transfer
	SubType         string    `gorm:"size:50" json:"sub_type"`
	Amount          int64     `gorm:"not null" json:"amount"`
	BeforeBalance   int64     `json:"before_balance"`
	AfterBalance    int64     `json:"after_balance"`
	Currency        string    `gorm:"size:10;default:'CNY'" json:"currency"`
	Status          string    `gorm:"size:20;default:'pending';index" json:"status"` // pending, processing, success, failed, cancelled
	RefID           string    `gorm:"size:100;index" json:"ref_id"` // 关联ID（游戏ID、充值ID等）
	RefType         string    `gorm:"size:50" json:"ref_type"`
	Description     string    `gorm:"size:500" json:"description"`
	Remark          string    `gorm:"size:500" json:"remark"`
	Metadata        JSONMap   `gorm:"type:json" json:"metadata"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	
	// 关联（注意：不直接嵌入 User，避免循环依赖）
}

// CoinPurchase 币购买记录表
type CoinPurchase struct {
	BaseModel
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	OrderNo         string    `gorm:"uniqueIndex;size:64;not null" json:"order_no"`
	TransactionID   uint      `gorm:"index" json:"transaction_id"`
	PackageID       uint      `json:"package_id"`
	PackageName     string    `gorm:"size:100" json:"package_name"`
	CoinAmount      int64     `gorm:"not null" json:"coin_amount"`
	BonusAmount     int64     `gorm:"default:0" json:"bonus_amount"`
	Price           int64     `gorm:"not null" json:"price"` // 分
	OriginalPrice   int64     `json:"original_price"`
	Discount        float64   `gorm:"default:1" json:"discount"`
	PayMethod       string    `gorm:"size:50" json:"pay_method"` // alipay, wechat, card, apple, google
	PayAccount      string    `gorm:"size:100" json:"pay_account"`
	PayTime         *time.Time `json:"pay_time,omitempty"`
	PayTransactionNo string   `gorm:"size:100" json:"pay_transaction_no"`
	Status          string    `gorm:"size:20;default:'pending'" json:"status"`
	ExpireAt        time.Time `json:"expire_at"`
	
	// 关联（注意：不直接嵌入 User，避免循环依赖）
	Transaction     *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

// Withdrawal 提现记录表
type Withdrawal struct {
	BaseModel
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	OrderNo         string    `gorm:"uniqueIndex;size:64;not null" json:"order_no"`
	TransactionID   uint      `gorm:"index" json:"transaction_id"`
	Amount          int64     `gorm:"not null" json:"amount"`
	Fee             int64     `gorm:"default:0" json:"fee"`
	ActualAmount    int64     `json:"actual_amount"`
	Method          string    `gorm:"size:50" json:"method"` // alipay, wechat, bank
	Account         string    `gorm:"size:100" json:"account"`
	AccountName     string    `gorm:"size:50" json:"account_name"`
	BankName        string    `gorm:"size:100" json:"bank_name"`
	BankBranch      string    `gorm:"size:200" json:"bank_branch"`
	Status          string    `gorm:"size:20;default:'pending';index" json:"status"`
	ReviewedBy      uint      `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewRemark    string    `gorm:"size:500" json:"review_remark"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	FailReason      string    `gorm:"size:500" json:"fail_reason"`
	
	// 关联（注意：不直接嵌入 User，避免循环依赖）
	Transaction     *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

// SystemConfig 系统配置表
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Type        string    `gorm:"size:20" json:"type"` // string, int, float, bool, json
	Description string    `gorm:"size:500" json:"description"`
	Group       string    `gorm:"size:50" json:"group"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SystemLog 系统日志表
type SystemLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Type        string    `gorm:"size:50;index" json:"type"` // login, logout, operation, system
	Action      string    `gorm:"size:100" json:"action"`
	Module      string    `gorm:"size:50" json:"module"`
	IP          string    `gorm:"size:50" json:"ip"`
	UserAgent   string    `gorm:"size:500" json:"user_agent"`
	Request     string    `gorm:"type:text" json:"request"`
	Response    string    `gorm:"type:text" json:"response"`
	Status      string    `gorm:"size:20" json:"status"`
	Duration    int       `json:"duration"` // 毫秒
	Extra       JSONMap   `gorm:"type:json" json:"extra"`
	CreatedAt   time.Time `json:"created_at"`
}

// ErrorLog 错误日志表
type ErrorLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Level       string    `gorm:"size:20;index" json:"level"` // debug, info, warn, error, fatal
	Module      string    `gorm:"size:50" json:"module"`
	Function    string    `gorm:"size:100" json:"function"`
	Message     string    `gorm:"type:text" json:"message"`
	Stack       string    `gorm:"type:text" json:"stack"`
	File        string    `gorm:"size:255" json:"file"`
	Line        int       `json:"line"`
	Context     JSONMap   `gorm:"type:json" json:"context"`
	IP          string    `gorm:"size:50" json:"ip"`
	UserAgent   string    `gorm:"size:500" json:"user_agent"`
	IsResolved  bool      `gorm:"default:false" json:"is_resolved"`
	ResolvedBy  uint      `json:"resolved_by"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DeviceStatus 设备状态表
type DeviceStatus struct {
	BaseModel
	DeviceID    string    `gorm:"uniqueIndex;size:100;not null" json:"device_id"`
	DeviceName  string    `gorm:"size:100" json:"device_name"`
	Type        string    `gorm:"size:50" json:"type"` // slot, pusher, server, client
	Status      string    `gorm:"size:20;default:'online'" json:"status"` // online, offline, error, maintenance
	IP          string    `gorm:"size:50" json:"ip"`
	Location    string    `gorm:"size:200" json:"location"`
	Version     string    `gorm:"size:20" json:"version"`
	LastPingAt  time.Time `json:"last_ping_at"`
	CPU         float64   `json:"cpu"`
	Memory      float64   `json:"memory"`
	Disk        float64   `json:"disk"`
	Network     JSONMap   `gorm:"type:json" json:"network"`
	Extra       JSONMap   `gorm:"type:json" json:"extra"`
}

// BeforeCreate 钱包创建前的钩子
func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	w.LastResetAt = time.Now()
	return nil
}

// UpdateBalance 更新余额
func (w *Wallet) UpdateBalance(amount int64) {
	w.Balance += amount
	if amount > 0 {
		w.TotalDeposit += amount
	} else {
		w.TotalWithdraw += (-amount)
	}
}

// UpdateGameStats 更新游戏统计
func (w *Wallet) UpdateGameStats(bet, win int64) {
	w.TotalBet += bet
	w.TotalWin += win
	w.DailyBet += bet
	w.DailyWin += win
}

// ResetDailyStats 重置每日统计
func (w *Wallet) ResetDailyStats() {
	now := time.Now()
	if now.Day() != w.LastResetAt.Day() {
		w.DailyBet = 0
		w.DailyWin = 0
		w.LastResetAt = now
	}
}

// CanWithdraw 检查是否可以提现
func (w *Wallet) CanWithdraw(amount int64) bool {
	return w.Balance-w.FrozenBalance >= amount
}

// CanBet 检查是否可以下注
func (w *Wallet) CanBet(amount int64) bool {
	return w.Coins-w.FrozenCoins >= amount
}