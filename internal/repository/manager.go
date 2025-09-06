package repository

import (
	"context"
	"sync"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// Manager 仓储管理器，提供所有仓储的统一访问接口
type Manager struct {
	db *gorm.DB
	
	// 事务管理器
	txManager TransactionManager
	
	// 仓储实例（使用懒加载）
	gameSessionOnce    sync.Once
	gameSession        GameSessionRepository
	
	gameResultOnce     sync.Once
	gameResult         GameResultRepository
	
	deviceStatusOnce   sync.Once
	deviceStatus       DeviceStatusRepository
	
	systemConfigOnce   sync.Once
	systemConfig       SystemConfigRepository
	
	// 用户相关
	userOnce         sync.Once
	user             UserRepository
	
	userAuthOnce     sync.Once
	userAuth         UserAuthRepository
	
	userSessionOnce  sync.Once
	userSession      UserSessionRepository
	
	// 游戏相关
	gameOnce         sync.Once
	game             GameRepository
	
	gameRoomOnce     sync.Once
	gameRoom         GameRoomRepository
	
	// 交易相关
	walletOnce       sync.Once
	wallet           WalletRepository
	
	transactionOnce  sync.Once
	transaction      TransactionRepository
	
	// 日志相关
	systemLogOnce    sync.Once
	systemLog        SystemLogRepository
	
	errorLogOnce     sync.Once
	errorLog         ErrorLogRepository
	
	// Slot游戏相关
	slotMachineOnce  sync.Once
	slotMachine      SlotMachineRepository
	
	slotSpinOnce     sync.Once
	slotSpin         SlotSpinRepository
	
	slotWinLineOnce  sync.Once
	slotWinLine      SlotWinLineRepository
	
	// Pusher游戏相关
	pusherMachineOnce  sync.Once
	pusherMachine      PusherMachineRepository
	
	pusherSessionOnce  sync.Once
	pusherSession      PusherSessionRepository
	
	coinDropOnce       sync.Once
	coinDrop           CoinDropRepository
}

// NewManager 创建仓储管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:        db,
		txManager: NewTransactionManager(db),
	}
}

// GetDB 获取数据库实例
func (m *Manager) GetDB() *gorm.DB {
	return m.db
}

// Transaction 获取事务管理器
func (m *Manager) Transaction() TransactionManager {
	return m.txManager
}

// GameSession 获取游戏会话仓储
func (m *Manager) GameSession() GameSessionRepository {
	m.gameSessionOnce.Do(func() {
		m.gameSession = NewGameSessionRepository(m.db)
	})
	return m.gameSession
}

// GameResult 获取游戏结果仓储
func (m *Manager) GameResult() GameResultRepository {
	m.gameResultOnce.Do(func() {
		m.gameResult = NewGameResultRepository(m.db)
	})
	return m.gameResult
}

// DeviceStatus 获取设备状态仓储
func (m *Manager) DeviceStatus() DeviceStatusRepository {
	m.deviceStatusOnce.Do(func() {
		m.deviceStatus = NewDeviceStatusRepository(m.db)
	})
	return m.deviceStatus
}

// SystemConfig 获取系统配置仓储
func (m *Manager) SystemConfig() SystemConfigRepository {
	m.systemConfigOnce.Do(func() {
		m.systemConfig = NewSystemConfigRepository(m.db)
	})
	return m.systemConfig
}

// User 获取用户仓储
func (m *Manager) User() UserRepository {
	m.userOnce.Do(func() {
		m.user = NewUserRepository(m.db)
	})
	return m.user
}

// UserAuth 获取用户认证仓储
func (m *Manager) UserAuth() UserAuthRepository {
	m.userAuthOnce.Do(func() {
		m.userAuth = NewUserAuthRepository(m.db)
	})
	return m.userAuth
}

// UserSession 获取用户会话仓储
func (m *Manager) UserSession() UserSessionRepository {
	m.userSessionOnce.Do(func() {
		m.userSession = NewUserSessionRepository(m.db)
	})
	return m.userSession
}

// Game 获取游戏仓储
func (m *Manager) Game() GameRepository {
	m.gameOnce.Do(func() {
		m.game = NewGameRepository(m.db)
	})
	return m.game
}

// GameRoom 获取游戏房间仓储
func (m *Manager) GameRoom() GameRoomRepository {
	m.gameRoomOnce.Do(func() {
		m.gameRoom = NewGameRoomRepository(m.db)
	})
	return m.gameRoom
}

// Wallet 获取钱包仓储
func (m *Manager) Wallet() WalletRepository {
	m.walletOnce.Do(func() {
		m.wallet = NewWalletRepository(m.db)
	})
	return m.wallet
}

// Transaction 获取交易记录仓储
func (m *Manager) TransactionRepo() TransactionRepository {
	m.transactionOnce.Do(func() {
		m.transaction = NewTransactionRepository(m.db)
	})
	return m.transaction
}

// SystemLog 获取系统日志仓储
func (m *Manager) SystemLog() SystemLogRepository {
	m.systemLogOnce.Do(func() {
		m.systemLog = NewSystemLogRepository(m.db)
	})
	return m.systemLog
}

// ErrorLog 获取错误日志仓储
func (m *Manager) ErrorLog() ErrorLogRepository {
	m.errorLogOnce.Do(func() {
		m.errorLog = NewErrorLogRepository(m.db)
	})
	return m.errorLog
}

// SlotMachine 获取老虎机仓储
func (m *Manager) SlotMachine() SlotMachineRepository {
	m.slotMachineOnce.Do(func() {
		m.slotMachine = NewSlotMachineRepository(m.db)
	})
	return m.slotMachine
}

// SlotSpin 获取老虎机旋转记录仓储
func (m *Manager) SlotSpin() SlotSpinRepository {
	m.slotSpinOnce.Do(func() {
		m.slotSpin = NewSlotSpinRepository(m.db)
	})
	return m.slotSpin
}

// SlotWinLine 获取老虎机中奖线仓储
func (m *Manager) SlotWinLine() SlotWinLineRepository {
	m.slotWinLineOnce.Do(func() {
		m.slotWinLine = NewSlotWinLineRepository(m.db)
	})
	return m.slotWinLine
}

// PusherMachine 获取推币机仓储
func (m *Manager) PusherMachine() PusherMachineRepository {
	m.pusherMachineOnce.Do(func() {
		m.pusherMachine = NewPusherMachineRepository(m.db)
	})
	return m.pusherMachine
}

// PusherSession 获取推币机会话仓储
func (m *Manager) PusherSession() PusherSessionRepository {
	m.pusherSessionOnce.Do(func() {
		m.pusherSession = NewPusherSessionRepository(m.db)
	})
	return m.pusherSession
}

// CoinDrop 获取推币掉落记录仓储
func (m *Manager) CoinDrop() CoinDropRepository {
	m.coinDropOnce.Do(func() {
		m.coinDrop = NewCoinDropRepository(m.db)
	})
	return m.coinDrop
}

// WithTransaction 在事务中执行操作
func (m *Manager) WithTransaction(ctx context.Context, fn func(tx *Transaction) error) error {
	return m.txManager.WithTransaction(ctx, fn)
}

// WithReadOnlyTransaction 在只读事务中执行操作
func (m *Manager) WithReadOnlyTransaction(ctx context.Context, fn func(tx *Transaction) error) error {
	opts := &TxOptions{
		ReadOnly: true,
	}
	return m.txManager.WithTransactionOptions(ctx, opts, fn)
}

// RepositoryProvider 仓储提供者接口，用于依赖注入
type RepositoryProvider interface {
	GetManager() *Manager
	GameSession() GameSessionRepository
	GameResult() GameResultRepository
	DeviceStatus() DeviceStatusRepository
	SystemConfig() SystemConfigRepository
}

// provider 仓储提供者实现
type provider struct {
	manager *Manager
}

// NewProvider 创建仓储提供者
func NewProvider(db *gorm.DB) RepositoryProvider {
	return &provider{
		manager: NewManager(db),
	}
}

// GetManager 获取仓储管理器
func (p *provider) GetManager() *Manager {
	return p.manager
}

// GameSession 获取游戏会话仓储
func (p *provider) GameSession() GameSessionRepository {
	return p.manager.GameSession()
}

// GameResult 获取游戏结果仓储
func (p *provider) GameResult() GameResultRepository {
	return p.manager.GameResult()
}

// DeviceStatus 获取设备状态仓储
func (p *provider) DeviceStatus() DeviceStatusRepository {
	return p.manager.DeviceStatus()
}

// SystemConfig 获取系统配置仓储
func (p *provider) SystemConfig() SystemConfigRepository {
	return p.manager.SystemConfig()
}

// UnitOfWork 工作单元模式实现
type UnitOfWork struct {
	tx         *Transaction
	manager    *Manager
	operations []func(*Transaction) error
}

// NewUnitOfWork 创建工作单元
func NewUnitOfWork(manager *Manager) *UnitOfWork {
	return &UnitOfWork{
		manager:    manager,
		operations: make([]func(*Transaction) error, 0),
	}
}

// Register 注册操作
func (u *UnitOfWork) Register(op func(*Transaction) error) {
	u.operations = append(u.operations, op)
}

// Commit 提交所有操作
func (u *UnitOfWork) Commit(ctx context.Context) error {
	return u.manager.WithTransaction(ctx, func(tx *Transaction) error {
		for _, op := range u.operations {
			if err := op(tx); err != nil {
				return err
			}
		}
		return nil
	})
}

// Clear 清除所有操作
func (u *UnitOfWork) Clear() {
	u.operations = u.operations[:0]
}

// BatchOperator 批量操作器
type BatchOperator struct {
	manager *Manager
}

// NewBatchOperator 创建批量操作器
func NewBatchOperator(manager *Manager) *BatchOperator {
	return &BatchOperator{manager: manager}
}

// CreateGameSessionWithResults 创建游戏会话并批量创建游戏结果
func (b *BatchOperator) CreateGameSessionWithResults(
	ctx context.Context,
	session *models.GameSession,
	results []*models.GameResult,
) error {
	return b.manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 创建游戏会话
		if err := tx.GameSession().Create(ctx, session); err != nil {
			return err
		}
		
		// 设置结果的会话ID
		for _, result := range results {
			result.SessionID = session.ID
		}
		
		// 批量创建游戏结果
		if err := tx.GameResult().BatchCreate(ctx, results); err != nil {
			return err
		}
		
		return nil
	})
}

// UpdateDeviceStatusBatch 批量更新设备状态
func (b *BatchOperator) UpdateDeviceStatusBatch(
	ctx context.Context,
	devices []*models.DeviceStatus,
) error {
	return b.manager.WithTransaction(ctx, func(tx *Transaction) error {
		for _, device := range devices {
			if err := tx.DeviceStatus().Update(ctx, device); err != nil {
				return err
			}
		}
		return nil
	})
}

// RefreshSystemConfig 刷新系统配置（事务中）
func (b *BatchOperator) RefreshSystemConfig(ctx context.Context) error {
	return b.manager.WithTransaction(ctx, func(tx *Transaction) error {
		return tx.SystemConfig().RefreshCache(ctx)
	})
}