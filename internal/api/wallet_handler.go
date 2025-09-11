package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfunc/slot-game/internal/middleware"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WalletHandler 钱包处理器
type WalletHandler struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	db              *gorm.DB
	logger          *zap.Logger
}

// NewWalletHandler 创建钱包处理器
func NewWalletHandler(db *gorm.DB, logger *zap.Logger) *WalletHandler {
	return &WalletHandler{
		walletRepo:      repository.NewWalletRepository(db),
		transactionRepo: repository.NewTransactionRepository(db),
		db:              db,
		logger:          logger,
	}
}

// BalanceResponse 余额响应
type BalanceResponse struct {
	Balance      int64  `json:"balance"`
	FrozenAmount int64  `json:"frozen_amount"`
	Available    int64  `json:"available"`
	Currency     string `json:"currency"`
}

// DepositRequest 充值请求
type DepositRequest struct {
	Amount int64 `json:"amount" binding:"required,min=100,max=100000"`
}

// DepositResponse 充值响应
type DepositResponse struct {
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Balance       int64     `json:"balance"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// WithdrawRequest 提现请求
type WithdrawRequest struct {
	Amount int64 `json:"amount" binding:"required,min=100"`
}

// WithdrawResponse 提现响应
type WithdrawResponse struct {
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Balance       int64     `json:"balance"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// TransactionListResponse 交易列表响应
type TransactionListResponse struct {
	Transactions []TransactionInfo `json:"transactions"`
	Total        int64             `json:"total"`
	Page         int               `json:"page"`
	PageSize     int               `json:"page_size"`
}

// TransactionInfo 交易信息
type TransactionInfo struct {
	ID            uint      `json:"id"`
	OrderNo       string    `json:"order_no"`
	Type          string    `json:"type"`
	Amount        int64     `json:"amount"`
	BeforeBalance int64     `json:"before_balance"`
	AfterBalance  int64     `json:"after_balance"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// GetBalance 获取余额
// @Summary 获取余额
// @Description 获取当前用户钱包余额与可用余额
// @Tags Wallet
// @Security Bearer
// @Produce json
// @Success 200 {object} BalanceResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/wallet/balance [get]
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	// 获取钱包信息
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		// 如果钱包不存在，创建一个新钱包
		if err.Error() == "钱包不存在" {
			wallet = &models.Wallet{
				UserID:  userID,
				Balance: 10000, // 初始赠送10000金币
			}
			if err = h.walletRepo.Create(c.Request.Context(), wallet); err != nil {
				h.logger.Error("创建钱包失败", zap.Error(err))
				c.JSON(500, gin.H{"error": "创建钱包失败"})
				return
			}

			// 记录初始赠送交易
			transaction := &models.WalletTransaction{
				UserID:        userID,
				OrderNo:       fmt.Sprintf("INIT-%d-%d", userID, time.Now().Unix()),
				Type:          "deposit",
				Amount:        10000,
				BeforeBalance: 0,
				AfterBalance:  10000,
				RefType:       "system",
				RefID:         "initial",
				Description:   "新用户初始赠送",
				Status:        "success",
			}
			h.walletRepo.CreateTransaction(c.Request.Context(), transaction)
		} else {
			h.logger.Error("获取钱包失败", zap.Error(err))
			c.JSON(500, gin.H{"error": "获取余额失败"})
			return
		}
	}

	c.JSON(200, BalanceResponse{
		Balance:      wallet.Balance,
		FrozenAmount: wallet.FrozenBalance,
		Available:    wallet.Balance - wallet.FrozenBalance,
		Currency:     "COIN",
	})
}

// Deposit 充值（测试用）
// @Summary 充值（测试）
// @Description 增加账户余额，测试用接口
// @Tags Wallet
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body DepositRequest true "充值请求"
// @Success 200 {object} DepositResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/wallet/deposit [post]
func (h *WalletHandler) Deposit(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 获取钱包（加锁）
	wallet, err := h.walletRepo.WithTx(tx).(repository.WalletRepository).LockForUpdate(c.Request.Context(), userID)
	if err != nil {
		tx.Rollback()
		h.logger.Error("获取钱包失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "获取钱包失败"})
		return
	}

	// 增加余额
	beforeBalance := wallet.Balance
	if err := h.walletRepo.WithTx(tx).(repository.WalletRepository).AddBalance(c.Request.Context(), userID, req.Amount); err != nil {
		tx.Rollback()
		h.logger.Error("增加余额失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "充值失败"})
		return
	}

	// 创建交易记录
	transactionID := fmt.Sprintf("DEP-%d-%d", userID, time.Now().Unix())
	transaction := &models.WalletTransaction{
		UserID:        userID,
		OrderNo:       transactionID,
		Type:          "deposit",
		Amount:        req.Amount,
		BeforeBalance: beforeBalance,
		AfterBalance:  beforeBalance + req.Amount,
		RefType:       "manual",
		RefID:         "test",
		Description:   "测试充值",
		Status:        "success",
	}

	if err := h.walletRepo.WithTx(tx).(repository.WalletRepository).CreateTransaction(c.Request.Context(), transaction); err != nil {
		tx.Rollback()
		h.logger.Error("创建交易记录失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "记录交易失败"})
		return
	}

	// 更新统计
	h.walletRepo.WithTx(tx).(repository.WalletRepository).UpdateStatistics(c.Request.Context(), userID, "total_deposit", req.Amount)

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("提交事务失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "充值失败"})
		return
	}

	h.logger.Info("充值成功",
		zap.Uint("user_id", userID),
		zap.Int64("amount", req.Amount),
		zap.String("transaction_id", transactionID))

	c.JSON(200, DepositResponse{
		TransactionID: transactionID,
		Amount:        req.Amount,
		Balance:       beforeBalance + req.Amount,
		Status:        "success",
		CreatedAt:     time.Now(),
	})
}

// Withdraw 提现（模拟）
// @Summary 提现（模拟）
// @Description 扣减账户余额，模拟提现流程
// @Tags Wallet
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body WithdrawRequest true "提现请求"
// @Success 200 {object} WithdrawResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/wallet/withdraw [post]
func (h *WalletHandler) Withdraw(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 获取钱包（加锁）
	wallet, err := h.walletRepo.WithTx(tx).(repository.WalletRepository).LockForUpdate(c.Request.Context(), userID)
	if err != nil {
		tx.Rollback()
		h.logger.Error("获取钱包失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "获取钱包失败"})
		return
	}

	// 检查余额
	if wallet.Balance < req.Amount {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "余额不足"})
		return
	}

	// 扣减余额
	beforeBalance := wallet.Balance
	if err := h.walletRepo.WithTx(tx).(repository.WalletRepository).DeductBalance(c.Request.Context(), userID, req.Amount); err != nil {
		tx.Rollback()
		h.logger.Error("扣减余额失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "提现失败"})
		return
	}

	// 创建交易记录
	transactionID := fmt.Sprintf("WD-%d-%d", userID, time.Now().Unix())
	transaction := &models.WalletTransaction{
		UserID:        userID,
		OrderNo:       transactionID,
		Type:          "withdraw",
		Amount:        req.Amount,
		BeforeBalance: beforeBalance,
		AfterBalance:  beforeBalance - req.Amount,
		RefType:       "manual",
		RefID:         "test",
		Description:   "测试提现",
		Status:        "success",
	}

	if err := h.walletRepo.WithTx(tx).(repository.WalletRepository).CreateTransaction(c.Request.Context(), transaction); err != nil {
		tx.Rollback()
		h.logger.Error("创建交易记录失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "记录交易失败"})
		return
	}

	// 更新统计
	h.walletRepo.WithTx(tx).(repository.WalletRepository).UpdateStatistics(c.Request.Context(), userID, "total_withdraw", req.Amount)

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("提交事务失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "提现失败"})
		return
	}

	h.logger.Info("提现成功",
		zap.Uint("user_id", userID),
		zap.Int64("amount", req.Amount),
		zap.String("transaction_id", transactionID))

	c.JSON(200, WithdrawResponse{
		TransactionID: transactionID,
		Amount:        req.Amount,
		Balance:       beforeBalance - req.Amount,
		Status:        "success",
		CreatedAt:     time.Now(),
	})
}

// GetTransactions 获取交易记录
// @Summary 交易记录
// @Description 获取当前用户的交易记录（支持类型过滤与分页）
// @Tags Wallet
// @Security Bearer
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量（<=100）"
// @Param type query string false "交易类型（bet/win/deposit/withdraw）"
// @Success 200 {object} TransactionListResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/wallet/transactions [get]
func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	// 获取分页参数
	page := 1
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	// 限制页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	// 获取交易类型过滤
	txType := c.Query("type") // bet, win, deposit, withdraw

	// 创建分页对象
	pagination := &repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	// 查询交易记录
	var transactions []*models.Transaction
	var err error

	if txType != "" {
		transactions, err = h.transactionRepo.FindByType(c.Request.Context(), txType, pagination)
	} else {
		transactions, err = h.transactionRepo.FindByUserID(c.Request.Context(), userID, pagination)
	}

	if err != nil {
		h.logger.Error("获取交易记录失败",
			zap.Uint("user_id", userID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": "获取交易记录失败"})
		return
	}

	// 转换为响应格式
	var transactionInfos []TransactionInfo
	for _, tx := range transactions {
		transactionInfos = append(transactionInfos, TransactionInfo{
			ID:            tx.ID,
			OrderNo:       tx.OrderNo,
			Type:          tx.Type,
			Amount:        tx.Amount,
			BeforeBalance: tx.BeforeBalance,
			AfterBalance:  tx.AfterBalance,
			Description:   tx.Description,
			Status:        tx.Status,
			CreatedAt:     tx.CreatedAt,
		})
	}

	c.JSON(200, TransactionListResponse{
		Transactions: transactionInfos,
		Total:        pagination.Total,
		Page:         page,
		PageSize:     pageSize,
	})
}

// GetStatistics 获取钱包统计
// @Summary 钱包统计
// @Description 获取指定日期的钱包统计信息
// @Tags Wallet
// @Security Bearer
// @Produce json
// @Param date query string false "日期，格式YYYY-MM-DD"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/wallet/statistics [get]
func (h *WalletHandler) GetStatistics(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	// 获取日期参数（默认今天）
	date := time.Now()
	if d := c.Query("date"); d != "" {
		parsedDate, err := time.Parse("2006-01-02", d)
		if err == nil {
			date = parsedDate
		}
	}

	// 获取统计数据
	stats, err := h.transactionRepo.GetDailyStatistics(c.Request.Context(), userID, date)
	if err != nil {
		h.logger.Error("获取统计失败",
			zap.Uint("user_id", userID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": "获取统计失败"})
		return
	}

	// 获取钱包信息
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取钱包失败", zap.Error(err))
		wallet = &models.Wallet{Balance: 0}
	}

	c.JSON(200, gin.H{
		"balance":        wallet.Balance,
		"total_deposit":  wallet.TotalDeposit,
		"total_withdraw": wallet.TotalWithdraw,
		"total_bet":      wallet.TotalBet,
		"total_win":      wallet.TotalWin,
		"daily_stats":    stats,
		"date":           date.Format("2006-01-02"),
	})
}
