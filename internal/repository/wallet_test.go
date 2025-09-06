package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// WalletRepositoryTestSuite 钱包仓储测试套件
type WalletRepositoryTestSuite struct {
	suite.Suite
	db           *gorm.DB
	walletRepo   WalletRepository
	transRepo    TransactionRepository
	userRepo     UserRepository
}

func (suite *WalletRepositoryTestSuite) SetupTest() {
	suite.db = SetupTestDB()
	suite.walletRepo = NewWalletRepository(suite.db)
	suite.transRepo = NewTransactionRepository(suite.db)
	suite.userRepo = NewUserRepository(suite.db)
}

func (suite *WalletRepositoryTestSuite) TearDownTest() {
	CleanupTestDB(suite.db)
}

// 创建测试用户
func (suite *WalletRepositoryTestSuite) createTestUser(username string) *models.User {
	user := &models.User{
		Username: username,
		Email:    username + "@example.com",
		Status:   "active",
	}
	err := suite.userRepo.Create(context.Background(), user)
	suite.Require().NoError(err)
	return user
}

// TestWalletRepository_Create 测试创建钱包
func (suite *WalletRepositoryTestSuite) TestWalletRepository_Create() {
	ctx := context.Background()
	user := suite.createTestUser("walletuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 10000,
	}
	
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), wallet.ID)
	
	// 验证数据
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(10000), found.Balance)
}

// TestWalletRepository_FindByUserID 测试根据用户ID查找钱包
func (suite *WalletRepositoryTestSuite) TestWalletRepository_FindByUserID() {
	ctx := context.Background()
	user := suite.createTestUser("findwalletuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 5000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), wallet.ID, found.ID)
	assert.Equal(suite.T(), int64(5000), found.Balance)
	
	// 测试不存在的钱包
	_, err = suite.walletRepo.FindByUserID(ctx, 99999)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "钱包不存在")
}

// TestWalletRepository_UpdateBalance 测试更新余额
func (suite *WalletRepositoryTestSuite) TestWalletRepository_UpdateBalance() {
	ctx := context.Background()
	user := suite.createTestUser("updatebalanceuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 更新余额
	err = suite.walletRepo.UpdateBalance(ctx, user.ID, 2000)
	assert.NoError(suite.T(), err)
	
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2000), found.Balance)
}

// TestWalletRepository_AddBalance 测试增加余额
func (suite *WalletRepositoryTestSuite) TestWalletRepository_AddBalance() {
	ctx := context.Background()
	user := suite.createTestUser("addbalanceuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 增加余额
	err = suite.walletRepo.AddBalance(ctx, user.ID, 500)
	assert.NoError(suite.T(), err)
	
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1500), found.Balance)
}

// TestWalletRepository_DeductBalance 测试扣减余额
func (suite *WalletRepositoryTestSuite) TestWalletRepository_DeductBalance() {
	ctx := context.Background()
	user := suite.createTestUser("deductbalanceuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 扣减余额（成功）
	err = suite.walletRepo.DeductBalance(ctx, user.ID, 300)
	assert.NoError(suite.T(), err)
	
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(700), found.Balance)
	
	// 扣减余额（余额不足）
	err = suite.walletRepo.DeductBalance(ctx, user.ID, 1000)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "余额不足")
	
	// 验证余额没有变化
	found, err = suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(700), found.Balance)
}

// TestWalletRepository_LockForUpdate 测试悲观锁
func (suite *WalletRepositoryTestSuite) TestWalletRepository_LockForUpdate() {
	ctx := context.Background()
	user := suite.createTestUser("lockwalletuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 在事务中锁定钱包
	tx := suite.db.Begin()
	defer tx.Rollback()
	
	txRepo := suite.walletRepo.WithTx(tx).(WalletRepository)
	locked, err := txRepo.LockForUpdate(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), wallet.ID, locked.ID)
}

// TestWalletRepository_UpdateStatistics 测试更新统计信息
func (suite *WalletRepositoryTestSuite) TestWalletRepository_UpdateStatistics() {
	ctx := context.Background()
	user := suite.createTestUser("statswalletuser")
	
	wallet := &models.Wallet{
		UserID:       user.ID,
		Balance:      1000,
		TotalWin:     0,
		TotalBet:     0,
		TotalDeposit: 0,
		TotalWithdraw: 0,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 更新各种统计
	err = suite.walletRepo.UpdateStatistics(ctx, user.ID, "total_win", 500)
	assert.NoError(suite.T(), err)
	
	err = suite.walletRepo.UpdateStatistics(ctx, user.ID, "total_bet", 300)
	assert.NoError(suite.T(), err)
	
	err = suite.walletRepo.UpdateStatistics(ctx, user.ID, "total_deposit", 1000)
	assert.NoError(suite.T(), err)
	
	err = suite.walletRepo.UpdateStatistics(ctx, user.ID, "total_withdraw", 200)
	assert.NoError(suite.T(), err)
	
	// 验证统计数据
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(500), found.TotalWin)
	assert.Equal(suite.T(), int64(300), found.TotalBet)
	assert.Equal(suite.T(), int64(1000), found.TotalDeposit)
	assert.Equal(suite.T(), int64(200), found.TotalWithdraw)
	
	// 测试不允许的字段
	err = suite.walletRepo.UpdateStatistics(ctx, user.ID, "invalid_field", 100)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "不允许的字段")
}

// TestTransactionRepository_Create 测试创建交易记录
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_Create() {
	ctx := context.Background()
	user := suite.createTestUser("transuser")
	
	transaction := &models.Transaction{
		OrderNo:       "TX123456",
		UserID:       user.ID,
		Type:         "deposit",
		Amount:       1000,
		BeforeBalance: 0,
		AfterBalance:  1000,
		Status:       "success",
		Description:  "充值",
	}
	
	err := suite.transRepo.Create(ctx, transaction)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), transaction.ID)
}

// TestTransactionRepository_FindByTransactionID 测试根据交易ID查找
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_FindByTransactionID() {
	ctx := context.Background()
	user := suite.createTestUser("findtransuser")
	
	transaction := &models.Transaction{
		OrderNo:       "TX789012",
		UserID:       user.ID,
		Type:         "bet",
		Amount:       100,
		Status:       "success",
	}
	err := suite.transRepo.Create(ctx, transaction)
	assert.NoError(suite.T(), err)
	
	found, err := suite.transRepo.FindByTransactionID(ctx, "TX789012")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), transaction.ID, found.ID)
	assert.Equal(suite.T(), int64(100), found.Amount)
	
	// 测试不存在的交易
	_, err = suite.transRepo.FindByTransactionID(ctx, "NOTEXIST")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "交易记录不存在")
}

// TestTransactionRepository_FindByUserID 测试根据用户ID查找交易
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_FindByUserID() {
	ctx := context.Background()
	user := suite.createTestUser("usertransuser")
	
	// 创建多个交易记录
	for i := 0; i < 5; i++ {
		transaction := &models.Transaction{
			OrderNo:       fmt.Sprintf("TX_USER_%d", i),
			UserID:       user.ID,
			Type:         "bet",
			Amount:       int64(100 * (i + 1)),
			Status:       "success",
		}
		err := suite.transRepo.Create(ctx, transaction)
		assert.NoError(suite.T(), err)
	}
	
	// 测试分页
	pagination := &Pagination{
		Page:     1,
		PageSize: 3,
	}
	
	transactions, err := suite.transRepo.FindByUserID(ctx, user.ID, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), transactions, 3)
	assert.Equal(suite.T(), int64(5), pagination.Total)
}

// TestTransactionRepository_FindByType 测试根据类型查找交易
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_FindByType() {
	ctx := context.Background()
	user := suite.createTestUser("typetransuser")
	
	// 创建不同类型的交易
	types := []string{"bet", "win", "recharge", "withdraw"}
	for _, txType := range types {
		transaction := &models.Transaction{
			OrderNo:       fmt.Sprintf("TX_TYPE_%s", txType),
			UserID:       user.ID,
			Type:         txType,
			Amount:       1000,
			Status:       "success",
		}
		err := suite.transRepo.Create(ctx, transaction)
		assert.NoError(suite.T(), err)
	}
	
	// 查找特定类型
	pagination := &Pagination{
		Page:     1,
		PageSize: 10,
	}
	
	transactions, err := suite.transRepo.FindByType(ctx, "bet", pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), transactions, 1)
	assert.Equal(suite.T(), "bet", transactions[0].Type)
}

// TestTransactionRepository_UpdateStatus 测试更新交易状态
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_UpdateStatus() {
	ctx := context.Background()
	user := suite.createTestUser("updatestatususer")
	
	transaction := &models.Transaction{
		OrderNo:       "TX_STATUS",
		UserID:       user.ID,
		Type:         "deposit",
		Amount:       1000,
		Status:       "pending",
	}
	err := suite.transRepo.Create(ctx, transaction)
	assert.NoError(suite.T(), err)
	
	// 更新状态
	err = suite.transRepo.UpdateStatus(ctx, "TX_STATUS", "success")
	assert.NoError(suite.T(), err)
	
	found, err := suite.transRepo.FindByTransactionID(ctx, "TX_STATUS")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "success", found.Status)
}

// TestTransactionRepository_GetDailyStatistics 测试获取日统计
func (suite *WalletRepositoryTestSuite) TestTransactionRepository_GetDailyStatistics() {
	ctx := context.Background()
	user := suite.createTestUser("dailystatsuser")
	
	// 创建今天的交易记录
	today := time.Now()
	transactions := []struct {
		Type   string
		Amount int64
	}{
		{"bet", 100},
		{"bet", 200},
		{"win", 500},
		{"deposit", 1000},
		{"withdraw", 300},
	}
	
	for i, tx := range transactions {
		transaction := &models.Transaction{
			OrderNo:       fmt.Sprintf("TX_DAILY_%d", i),
			UserID:       user.ID,
			Type:         tx.Type,
			Amount:       tx.Amount,
			Status:       "success",
		}
		err := suite.transRepo.Create(ctx, transaction)
		assert.NoError(suite.T(), err)
	}
	
	// 获取统计
	stats, err := suite.transRepo.GetDailyStatistics(ctx, user.ID, today)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1500), stats.TotalIn) // win + deposit
	assert.Equal(suite.T(), int64(600), stats.TotalOut) // bet + withdraw
	assert.Equal(suite.T(), int64(900), stats.NetAmount)
	assert.Equal(suite.T(), 5, stats.TransCount)
	assert.Equal(suite.T(), 1, stats.WinCount)
	assert.Equal(suite.T(), 2, stats.BetCount)
	assert.Equal(suite.T(), int64(1000), stats.RechargeSum)
	assert.Equal(suite.T(), int64(300), stats.WithdrawSum)
}

// TestWalletRepository_WithTx 测试事务支持
func (suite *WalletRepositoryTestSuite) TestWalletRepository_WithTx() {
	ctx := context.Background()
	user := suite.createTestUser("txwalletuser")
	
	// 开始事务
	tx := suite.db.Begin()
	defer tx.Rollback()
	
	txWalletRepo := suite.walletRepo.WithTx(tx).(WalletRepository)
	txTransRepo := suite.transRepo.WithTx(tx).(TransactionRepository)
	
	// 在事务中创建钱包
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := txWalletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 在事务中创建交易
	transaction := &models.Transaction{
		OrderNo:       "TX_IN_TX",
		UserID:       user.ID,
		Type:         "deposit",
		Amount:       1000,
		Status:       "success",
	}
	err = txTransRepo.Create(ctx, transaction)
	assert.NoError(suite.T(), err)
	
	// 事务内可以查到
	found, err := txWalletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1000), found.Balance)
	
	// 回滚后查不到
	tx.Rollback()
	
	_, err = suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.Error(suite.T(), err)
}

// TestWalletRepository_ConcurrentBalance 测试并发余额操作
func (suite *WalletRepositoryTestSuite) TestWalletRepository_ConcurrentBalance() {
	ctx := context.Background()
	user := suite.createTestUser("concurrentuser")
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 1000,
	}
	err := suite.walletRepo.Create(ctx, wallet)
	assert.NoError(suite.T(), err)
	
	// 模拟并发增加余额
	done := make(chan bool, 2)
	
	go func() {
		err := suite.walletRepo.AddBalance(ctx, user.ID, 100)
		assert.NoError(suite.T(), err)
		done <- true
	}()
	
	go func() {
		err := suite.walletRepo.AddBalance(ctx, user.ID, 200)
		assert.NoError(suite.T(), err)
		done <- true
	}()
	
	// 等待完成
	<-done
	<-done
	
	// 验证最终余额
	found, err := suite.walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1300), found.Balance)
}

func TestWalletRepositorySuite(t *testing.T) {
	suite.Run(t, new(WalletRepositoryTestSuite))
}