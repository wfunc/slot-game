package game

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	
	// 迁移模型
	err = db.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.WalletTransaction{},
		&models.Transaction{},
		&models.GameState{},
		&models.GameRecord{},
	)
	require.NoError(t, err)
	
	return db
}

// createTestUser 创建测试用户和钱包
func createTestUser(t *testing.T, db *gorm.DB) uint {
	// 使用时间戳创建唯一的用户名
	timestamp := time.Now().UnixNano()
	user := &models.User{
		Username: fmt.Sprintf("testuser_%d", timestamp),
		Email:    fmt.Sprintf("test_%d@example.com", timestamp),
		Phone:    fmt.Sprintf("138%08d", timestamp%100000000),
		Status:   "active",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 10000, // 100元
	}
	err = db.Create(wallet).Error
	require.NoError(t, err)
	
	return user.ID
}

func TestRecoveryManager_RecoverReady_Timeout(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	persister := NewMemoryStatePersister()
	userID := createTestUser(t, db)
	
	// 创建恢复管理器
	rm := NewRecoveryManager(logger, persister, db, 30*time.Minute)
	
	// 创建一个准备状态的会话数据
	sessionID := "test-session-1"
	stateData := &StateMachineData{
		SessionID:    sessionID,
		UserID:       userID,
		CurrentState: StateReady,
		BetAmount:    100, // 1元投注
		LastUpdate:   time.Now().Add(-10 * time.Minute), // 10分钟前
	}
	
	// 保存状态
	ctx := context.Background()
	err := persister.Save(ctx, sessionID, stateData)
	require.NoError(t, err)
	
	// 创建状态机并恢复
	sm := NewStateMachine(sessionID, userID, logger, persister)
	sm.LoadFromData(stateData)
	
	// 执行恢复（应该触发退款）
	err = rm.recoverReady(ctx, sm)
	assert.NoError(t, err)
	
	// 验证退款是否成功
	walletRepo := repository.NewWalletRepository(db)
	wallet, err := walletRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(10100), wallet.Balance) // 原100元 + 1元退款
	
	// 验证交易记录
	var transaction models.Transaction
	err = db.Where("user_id = ? AND type = ? AND sub_type = ?", 
		userID, "refund", "game_timeout").First(&transaction).Error
	require.NoError(t, err)
	assert.Equal(t, int64(100), transaction.Amount)
	assert.Equal(t, "success", transaction.Status)
}

func TestRecoveryManager_RecoverSettlement_Incomplete(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	persister := NewMemoryStatePersister()
	userID := createTestUser(t, db)
	
	// 创建恢复管理器
	rm := NewRecoveryManager(logger, persister, db, 30*time.Minute)
	
	// 创建一个结算状态的会话数据（有中奖金额但未结算）
	sessionID := "test-session-2"
	stateData := &StateMachineData{
		SessionID:    sessionID,
		UserID:       userID,
		CurrentState: StateSettlement,
		BetAmount:    100,
		WinAmount:    500, // 赢了5元
		LastUpdate:   time.Now(),
	}
	
	// 保存状态
	ctx := context.Background()
	err := persister.Save(ctx, sessionID, stateData)
	require.NoError(t, err)
	
	// 创建状态机并恢复
	sm := NewStateMachine(sessionID, userID, logger, persister)
	sm.LoadFromData(stateData)
	
	// 执行恢复（应该完成结算）
	err = rm.recoverSettlement(ctx, sm)
	assert.NoError(t, err)
	
	// 验证结算是否成功
	walletRepo := repository.NewWalletRepository(db)
	wallet, err := walletRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(10500), wallet.Balance) // 原100元 + 5元中奖
	
	// 验证交易记录
	var transaction models.Transaction
	err = db.Where("user_id = ? AND type = ? AND sub_type = ?", 
		userID, "win", "game_recovery").First(&transaction).Error
	require.NoError(t, err)
	assert.Equal(t, int64(500), transaction.Amount)
	assert.Equal(t, "success", transaction.Status)
}

func TestRecoveryManager_RecoverSettlement_AlreadyCompleted(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	persister := NewMemoryStatePersister()
	userID := createTestUser(t, db)
	
	// 创建恢复管理器
	rm := NewRecoveryManager(logger, persister, db, 30*time.Minute)
	
	// 创建一个已经结算的交易记录
	sessionID := "test-session-3"
	existingTx := &models.Transaction{
		UserID:   userID,
		OrderNo:  "WIN_test-session-3_123456",
		Type:     "win",
		Amount:   500,
		Status:   "success",
		RefID:    sessionID,
		RefType:  "game_session",
	}
	err := db.Create(existingTx).Error
	require.NoError(t, err)
	
	// 创建结算状态的会话数据
	stateData := &StateMachineData{
		SessionID:    sessionID,
		UserID:       userID,
		CurrentState: StateSettlement,
		BetAmount:    100,
		WinAmount:    500,
		LastUpdate:   time.Now(),
	}
	
	// 保存状态
	ctx := context.Background()
	err = persister.Save(ctx, sessionID, stateData)
	require.NoError(t, err)
	
	// 创建状态机并恢复
	sm := NewStateMachine(sessionID, userID, logger, persister)
	sm.LoadFromData(stateData)
	
	// 执行恢复（不应该重复结算）
	err = rm.recoverSettlement(ctx, sm)
	assert.NoError(t, err)
	
	// 验证余额没有增加
	walletRepo := repository.NewWalletRepository(db)
	wallet, err := walletRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), wallet.Balance) // 余额保持不变
	
	// 验证只有一条交易记录
	var count int64
	err = db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND ref_id = ?", 
			userID, "win", sessionID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRecoveryManager_CleanupExpiredSessions(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	persister := NewMemoryStatePersister()
	
	// 创建恢复管理器（超时时间设为1秒用于测试）
	rm := NewRecoveryManager(logger, persister, db, 1*time.Second)
	
	// 创建多个测试用户和会话
	ctx := context.Background()
	
	// 先创建用户，添加小延迟避免时间戳冲突
	user1ID := createTestUser(t, db)
	time.Sleep(10 * time.Millisecond)
	user2ID := createTestUser(t, db)
	time.Sleep(10 * time.Millisecond)
	user3ID := createTestUser(t, db)
	
	// 用户1：有一个过期的Ready状态会话（需要退款）
	expiredTime := time.Now().Add(-5 * time.Second)
	session1 := &models.GameState{
		SessionID:    "expired-session-1",
		UserID:       user1ID,
		CurrentState: "ready",
		StateData:    fmt.Sprintf(`{"session_id":"expired-session-1","user_id":%d,"current_state":"ready","bet_amount":200,"win_amount":0}`, user1ID),
		UpdatedAt:    expiredTime,
	}
	err := db.Create(session1).Error
	require.NoError(t, err)
	
	// 用户2：有一个过期的Settlement状态会话（需要结算）
	session2 := &models.GameState{
		SessionID:    "expired-session-2",
		UserID:       user2ID,
		CurrentState: "settlement",
		StateData:    fmt.Sprintf(`{"session_id":"expired-session-2","user_id":%d,"current_state":"settlement","bet_amount":100,"win_amount":300}`, user2ID),
		UpdatedAt:    expiredTime,
	}
	err = db.Create(session2).Error
	require.NoError(t, err)
	
	// 创建一个未过期的会话（不应该被清理）
	session3 := &models.GameState{
		SessionID:    "active-session",
		UserID:       user3ID,
		CurrentState: "spinning",
		StateData:    fmt.Sprintf(`{"session_id":"active-session","user_id":%d,"current_state":"spinning","bet_amount":100,"win_amount":0}`, user3ID),
		UpdatedAt:    time.Now(),
	}
	err = db.Create(session3).Error
	require.NoError(t, err)
	
	// 执行清理
	err = rm.CleanupExpiredSessions(ctx)
	assert.NoError(t, err)
	
	// 验证用户1的退款
	walletRepo := repository.NewWalletRepository(db)
	wallet1, err := walletRepo.FindByUserID(ctx, user1ID)
	require.NoError(t, err)
	assert.Equal(t, int64(10200), wallet1.Balance) // 原100元 + 2元退款
	
	// 验证用户2的结算
	wallet2, err := walletRepo.FindByUserID(ctx, user2ID)
	require.NoError(t, err)
	assert.Equal(t, int64(10300), wallet2.Balance) // 原100元 + 3元中奖
	
	// 验证过期会话已被删除
	var count int64
	err = db.Model(&models.GameState{}).
		Where("session_id IN ?", []string{"expired-session-1", "expired-session-2"}).
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	
	// 验证未过期会话仍然存在
	err = db.Model(&models.GameState{}).
		Where("session_id = ?", "active-session").
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRecoveryManager_RecoverSession(t *testing.T) {
	// 设置测试环境
	db := setupTestDB(t)
	logger := zap.NewNop()
	persister := NewDatabaseStatePersister(db)
	userID := createTestUser(t, db)
	
	// 创建恢复管理器
	rm := NewRecoveryManager(logger, persister, db, 30*time.Minute)
	
	// 创建并保存一个会话状态
	sessionID := "test-recover-session"
	stateData := &StateMachineData{
		SessionID:    sessionID,
		UserID:       userID,
		CurrentState: StateSpinning,
		BetAmount:    100,
		LastUpdate:   time.Now(),
	}
	
	ctx := context.Background()
	err := persister.Save(ctx, sessionID, stateData)
	require.NoError(t, err)
	
	// 恢复会话
	sm, err := rm.RecoverSession(ctx, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, sm)
	assert.Equal(t, sessionID, sm.sessionID)
	assert.Equal(t, userID, sm.userID)
	
	// 测试超时的会话
	expiredData := &StateMachineData{
		SessionID:    "expired-session",
		UserID:       userID,
		CurrentState: StateSpinning,
		BetAmount:    100,
		LastUpdate:   time.Now().Add(-40 * time.Minute),
	}
	
	err = persister.Save(ctx, "expired-session", expiredData)
	require.NoError(t, err)
	
	// 尝试恢复超时会话应该失败
	sm, err = rm.RecoverSession(ctx, "expired-session")
	assert.Error(t, err)
	assert.Nil(t, sm)
	assert.Contains(t, err.Error(), "会话已超时")
}