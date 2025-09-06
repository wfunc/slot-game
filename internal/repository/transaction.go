package repository

import (
	"context"
	"fmt"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// TransactionManager 事务管理器接口
type TransactionManager interface {
	// Begin 开始事务
	Begin(ctx context.Context) (*Transaction, error)
	// BeginWithOptions 使用选项开始事务
	BeginWithOptions(ctx context.Context, opts *TxOptions) (*Transaction, error)
	// WithTransaction 在事务中执行函数
	WithTransaction(ctx context.Context, fn func(tx *Transaction) error) error
	// WithTransactionOptions 使用选项在事务中执行函数
	WithTransactionOptions(ctx context.Context, opts *TxOptions, fn func(tx *Transaction) error) error
}

// TxOptions 事务选项
type TxOptions struct {
	// IsolationLevel 事务隔离级别
	IsolationLevel string
	// ReadOnly 是否只读事务
	ReadOnly bool
	// Timeout 事务超时时间（秒）
	Timeout int
}

// Transaction 事务包装器
type Transaction struct {
	tx         *gorm.DB
	ctx        context.Context
	committed  bool
	rolledback bool
	
	// 事务中的仓储实例
	gameSession    GameSessionRepository
	gameResult     GameResultRepository
	deviceStatus   DeviceStatusRepository
	systemConfig   SystemConfigRepository
	
	// 用户相关
	user           UserRepository
	userAuth       UserAuthRepository
	userSession    UserSessionRepository
	
	// 游戏相关
	game           GameRepository
	gameRoom       GameRoomRepository
	
	// 交易相关
	wallet         WalletRepository
	transaction    TransactionRepository
	
	// 日志相关
	systemLog      SystemLogRepository
	errorLog       ErrorLogRepository
	
	// Slot游戏相关
	slotMachine    SlotMachineRepository
	slotSpin       SlotSpinRepository
	slotWinLine    SlotWinLineRepository
	
	// Pusher游戏相关
	pusherMachine  PusherMachineRepository
	pusherSession  PusherSessionRepository
	coinDrop       CoinDropRepository
}

// txManager 事务管理器实现
type txManager struct {
	db *gorm.DB
}

// NewTransactionManager 创建事务管理器
func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &txManager{db: db}
}

// Begin 开始事务
func (m *txManager) Begin(ctx context.Context) (*Transaction, error) {
	return m.BeginWithOptions(ctx, nil)
}

// BeginWithOptions 使用选项开始事务
func (m *txManager) BeginWithOptions(ctx context.Context, opts *TxOptions) (*Transaction, error) {
	tx := m.db.WithContext(ctx)
	
	// 开始事务
	tx = tx.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	
	// 应用事务选项（SQLite不支持SET TRANSACTION，所以我们只记录选项）
	// 在实际的MySQL/PostgreSQL中，这里会设置隔离级别和只读选项
	// if opts != nil {
	//     if opts.IsolationLevel != "" {
	//         sql := fmt.Sprintf("SET TRANSACTION ISOLATION LEVEL %s", opts.IsolationLevel)
	//         tx.Exec(sql)
	//     }
	//     if opts.ReadOnly {
	//         tx.Exec("SET TRANSACTION READ ONLY")
	//     }
	// }
	
	return &Transaction{
		tx:  tx,
		ctx: ctx,
	}, nil
}

// WithTransaction 在事务中执行函数
func (m *txManager) WithTransaction(ctx context.Context, fn func(tx *Transaction) error) error {
	return m.WithTransactionOptions(ctx, nil, fn)
}

// WithTransactionOptions 使用选项在事务中执行函数
func (m *txManager) WithTransactionOptions(ctx context.Context, opts *TxOptions, fn func(tx *Transaction) error) error {
	tx, err := m.BeginWithOptions(ctx, opts)
	if err != nil {
		return err
	}
	
	// 确保事务被处理
	defer func() {
		if !tx.committed && !tx.rolledback {
			tx.Rollback()
		}
	}()
	
	// 执行业务逻辑
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// Commit 提交事务
func (t *Transaction) Commit() error {
	if t.committed {
		return fmt.Errorf("事务已提交")
	}
	if t.rolledback {
		return fmt.Errorf("事务已回滚")
	}
	
	if err := t.tx.Commit().Error; err != nil {
		return err
	}
	
	t.committed = true
	return nil
}

// Rollback 回滚事务
func (t *Transaction) Rollback() error {
	if t.committed {
		return fmt.Errorf("事务已提交，无法回滚")
	}
	if t.rolledback {
		return fmt.Errorf("事务已回滚")
	}
	
	if err := t.tx.Rollback().Error; err != nil {
		return err
	}
	
	t.rolledback = true
	return nil
}

// GetDB 获取事务中的数据库实例
func (t *Transaction) GetDB() *gorm.DB {
	return t.tx
}

// GameSession 获取事务中的游戏会话仓储
func (t *Transaction) GameSession() GameSessionRepository {
	if t.gameSession == nil {
		t.gameSession = &gameSessionRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.gameSession
}

// GameResult 获取事务中的游戏结果仓储
func (t *Transaction) GameResult() GameResultRepository {
	if t.gameResult == nil {
		t.gameResult = &gameResultRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.gameResult
}

// DeviceStatus 获取事务中的设备状态仓储
func (t *Transaction) DeviceStatus() DeviceStatusRepository {
	if t.deviceStatus == nil {
		t.deviceStatus = &deviceStatusRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.deviceStatus
}

// SystemConfig 获取事务中的系统配置仓储
func (t *Transaction) SystemConfig() SystemConfigRepository {
	if t.systemConfig == nil {
		// 系统配置需要特殊处理，因为它有缓存
		// 在事务中，我们创建一个新的实例，但共享缓存
		baseRepo := &BaseRepo{db: t.tx}
		// 注意：这里假设有一个全局的配置缓存
		t.systemConfig = &systemConfigRepo{
			BaseRepo: baseRepo,
			cache:    make(map[string]*models.SystemConfig), // 事务中使用独立缓存
		}
	}
	return t.systemConfig
}

// User 获取事务中的用户仓储
func (t *Transaction) User() UserRepository {
	if t.user == nil {
		t.user = &userRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.user
}

// UserAuth 获取事务中的用户认证仓储
func (t *Transaction) UserAuth() UserAuthRepository {
	if t.userAuth == nil {
		t.userAuth = &userAuthRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.userAuth
}

// UserSession 获取事务中的用户会话仓储
func (t *Transaction) UserSession() UserSessionRepository {
	if t.userSession == nil {
		t.userSession = &userSessionRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.userSession
}

// Game 获取事务中的游戏仓储
func (t *Transaction) Game() GameRepository {
	if t.game == nil {
		t.game = &gameRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.game
}

// GameRoom 获取事务中的游戏房间仓储
func (t *Transaction) GameRoom() GameRoomRepository {
	if t.gameRoom == nil {
		t.gameRoom = &gameRoomRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.gameRoom
}

// Wallet 获取事务中的钱包仓储
func (t *Transaction) Wallet() WalletRepository {
	if t.wallet == nil {
		t.wallet = &walletRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.wallet
}

// TransactionRepo 获取事务中的交易记录仓储
func (t *Transaction) TransactionRepo() TransactionRepository {
	if t.transaction == nil {
		t.transaction = &transactionRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.transaction
}

// SystemLog 获取事务中的系统日志仓储
func (t *Transaction) SystemLog() SystemLogRepository {
	if t.systemLog == nil {
		t.systemLog = &systemLogRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.systemLog
}

// ErrorLog 获取事务中的错误日志仓储
func (t *Transaction) ErrorLog() ErrorLogRepository {
	if t.errorLog == nil {
		t.errorLog = &errorLogRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.errorLog
}

// SlotMachine 获取事务中的老虎机仓储
func (t *Transaction) SlotMachine() SlotMachineRepository {
	if t.slotMachine == nil {
		t.slotMachine = &slotMachineRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.slotMachine
}

// SlotSpin 获取事务中的老虎机旋转记录仓储
func (t *Transaction) SlotSpin() SlotSpinRepository {
	if t.slotSpin == nil {
		t.slotSpin = &slotSpinRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.slotSpin
}

// SlotWinLine 获取事务中的老虎机中奖线仓储
func (t *Transaction) SlotWinLine() SlotWinLineRepository {
	if t.slotWinLine == nil {
		t.slotWinLine = &slotWinLineRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.slotWinLine
}

// PusherMachine 获取事务中的推币机仓储
func (t *Transaction) PusherMachine() PusherMachineRepository {
	if t.pusherMachine == nil {
		t.pusherMachine = &pusherMachineRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.pusherMachine
}

// PusherSession 获取事务中的推币机会话仓储
func (t *Transaction) PusherSession() PusherSessionRepository {
	if t.pusherSession == nil {
		t.pusherSession = &pusherSessionRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.pusherSession
}

// CoinDrop 获取事务中的推币掉落记录仓储
func (t *Transaction) CoinDrop() CoinDropRepository {
	if t.coinDrop == nil {
		t.coinDrop = &coinDropRepo{
			BaseRepo: &BaseRepo{db: t.tx},
		}
	}
	return t.coinDrop
}

// SavePoint 创建保存点
func (t *Transaction) SavePoint(name string) error {
	return t.tx.SavePoint(name).Error
}

// RollbackToSavePoint 回滚到保存点
func (t *Transaction) RollbackToSavePoint(name string) error {
	return t.tx.RollbackTo(name).Error
}

// TransactionHelper 事务辅助函数
type TransactionHelper struct {
	manager TransactionManager
}

// NewTransactionHelper 创建事务辅助器
func NewTransactionHelper(manager TransactionManager) *TransactionHelper {
	return &TransactionHelper{manager: manager}
}

// ExecuteInTransaction 在事务中执行多个操作
func (h *TransactionHelper) ExecuteInTransaction(ctx context.Context, operations ...func(tx *Transaction) error) error {
	return h.manager.WithTransaction(ctx, func(tx *Transaction) error {
		for i, op := range operations {
			// 创建保存点
			savePoint := fmt.Sprintf("sp_%d", i)
			if err := tx.SavePoint(savePoint); err != nil {
				return err
			}
			
			// 执行操作
			if err := op(tx); err != nil {
				// 回滚到保存点
				tx.RollbackToSavePoint(savePoint)
				return err
			}
		}
		return nil
	})
}

// RunInReadOnlyTransaction 在只读事务中执行
func (h *TransactionHelper) RunInReadOnlyTransaction(ctx context.Context, fn func(tx *Transaction) error) error {
	opts := &TxOptions{
		ReadOnly: true,
	}
	return h.manager.WithTransactionOptions(ctx, opts, fn)
}

// RunWithRetry 带重试的事务执行
func (h *TransactionHelper) RunWithRetry(ctx context.Context, maxRetries int, fn func(tx *Transaction) error) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		err := h.manager.WithTransaction(ctx, fn)
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// 检查是否是可重试的错误（如死锁）
		if !isRetryableError(err) {
			return err
		}
		
		// 等待一段时间后重试（指数退避）
		// time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Millisecond * 100)
	}
	
	return fmt.Errorf("事务执行失败，已重试%d次: %w", maxRetries, lastErr)
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	// 这里可以根据具体的数据库错误类型来判断
	// 例如：死锁、连接超时等
	errStr := err.Error()
	
	// MySQL死锁错误
	if contains(errStr, "Deadlock") {
		return true
	}
	
	// PostgreSQL死锁错误
	if contains(errStr, "deadlock detected") {
		return true
	}
	
	// 连接错误
	if contains(errStr, "connection") && contains(errStr, "timeout") {
		return true
	}
	
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}

// 事务隔离级别常量
const (
	// IsolationLevelReadUncommitted 读未提交
	IsolationLevelReadUncommitted = "READ UNCOMMITTED"
	// IsolationLevelReadCommitted 读已提交
	IsolationLevelReadCommitted = "READ COMMITTED"
	// IsolationLevelRepeatableRead 可重复读
	IsolationLevelRepeatableRead = "REPEATABLE READ"
	// IsolationLevelSerializable 串行化
	IsolationLevelSerializable = "SERIALIZABLE"
)