package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
)

func TestTransactionManager_Begin(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// 开始事务
	tx, err := manager.Begin(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NotNil(t, tx.GetDB())

	// 提交事务
	err = tx.Commit()
	require.NoError(t, err)
}

func TestTransactionManager_BeginWithOptions(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// 使用选项开始事务
	opts := &TxOptions{
		ReadOnly: true,
		Timeout:  30,
	}
	
	tx, err := manager.BeginWithOptions(ctx, opts)
	require.NoError(t, err)
	assert.NotNil(t, tx)

	// 提交事务
	err = tx.Commit()
	require.NoError(t, err)
}

func TestTransactionManager_WithTransaction(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// 成功的事务
	var sessionID uint
	err := manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 创建游戏会话
		session := CreateTestGameSession(1, 1)
		err := tx.GameSession().Create(ctx, session)
		if err != nil {
			return err
		}
		sessionID = session.ID
		
		// 创建游戏结果
		result := CreateTestGameResult(session.ID, 1, 1, 100, 200)
		err = tx.GameResult().Create(ctx, result)
		if err != nil {
			return err
		}
		
		return nil
	})
	require.NoError(t, err)

	// 验证数据已创建
	sessionRepo := NewGameSessionRepository(db)
	session, err := sessionRepo.FindByID(ctx, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)
}

func TestTransactionManager_Rollback(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// 失败的事务（应该回滚）
	err := manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 创建游戏会话
		session := CreateTestGameSession(1, 1)
		session.SessionID = "rollback_test"
		err := tx.GameSession().Create(ctx, session)
		if err != nil {
			return err
		}
		
		// 故意返回错误以触发回滚
		return errors.New("故意的错误")
	})
	assert.Error(t, err)

	// 验证数据未创建（已回滚）
	sessionRepo := NewGameSessionRepository(db)
	session, err := sessionRepo.FindBySessionID(ctx, "rollback_test")
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestTransaction_CommitRollback(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// 测试手动提交
	tx1, err := manager.Begin(ctx)
	require.NoError(t, err)
	
	session1 := CreateTestGameSession(1, 1)
	session1.SessionID = "manual_commit"
	err = tx1.GameSession().Create(ctx, session1)
	require.NoError(t, err)
	
	err = tx1.Commit()
	require.NoError(t, err)
	
	// 验证重复提交错误
	err = tx1.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已提交")

	// 测试手动回滚
	tx2, err := manager.Begin(ctx)
	require.NoError(t, err)
	
	session2 := CreateTestGameSession(1, 1)
	session2.SessionID = "manual_rollback"
	err = tx2.GameSession().Create(ctx, session2)
	require.NoError(t, err)
	
	err = tx2.Rollback()
	require.NoError(t, err)
	
	// 验证重复回滚错误
	err = tx2.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已回滚")
	
	// 验证已回滚的事务不能提交
	err = tx2.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已回滚")
}

func TestTransaction_SavePoint(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	// SQLite支持SavePoint，但测试逻辑需要调整
	t.Run("SavePointBasics", func(t *testing.T) {
		tx, err := manager.Begin(ctx)
		require.NoError(t, err)
		
		// 创建设备1
		device1 := CreateTestDeviceStatus("sp_test_1", "测试设备1", "slot_machine", "online")
		err = tx.DeviceStatus().Create(ctx, device1)
		require.NoError(t, err)
		
		// 创建保存点
		err = tx.SavePoint("sp1")
		require.NoError(t, err)
		
		// 创建设备2
		device2 := CreateTestDeviceStatus("sp_test_2", "测试设备2", "slot_machine", "online")
		err = tx.DeviceStatus().Create(ctx, device2)
		require.NoError(t, err)
		
		// 回滚到保存点
		err = tx.RollbackToSavePoint("sp1")
		require.NoError(t, err)
		
		// 提交事务
		err = tx.Commit()
		require.NoError(t, err)
		
		// 验证结果
		deviceRepo := NewDeviceStatusRepository(db)
		
		// device1应该存在
		found1, err := deviceRepo.FindByDeviceID(ctx, "sp_test_1")
		require.NoError(t, err)
		assert.NotNil(t, found1)
		
		// device2不应该存在
		found2, err := deviceRepo.FindByDeviceID(ctx, "sp_test_2")
		assert.Error(t, err)
		assert.Nil(t, found2)
	})
}

func TestTransaction_RepositoryAccess(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	ctx := context.Background()

	err := manager.WithTransaction(ctx, func(tx *Transaction) error {
		// 测试获取各种仓储
		gameSession := tx.GameSession()
		assert.NotNil(t, gameSession)
		
		gameResult := tx.GameResult()
		assert.NotNil(t, gameResult)
		
		deviceStatus := tx.DeviceStatus()
		assert.NotNil(t, deviceStatus)
		
		systemConfig := tx.SystemConfig()
		assert.NotNil(t, systemConfig)
		
		// 验证重复获取返回相同实例
		gameSession2 := tx.GameSession()
		assert.Equal(t, gameSession, gameSession2)
		
		return nil
	})
	require.NoError(t, err)
}

func TestTransactionHelper_ExecuteInTransaction(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	helper := NewTransactionHelper(manager)
	ctx := context.Background()

	var sessionID, resultID uint
	
	// 定义操作
	op1 := func(tx *Transaction) error {
		session := CreateTestGameSession(1, 1)
		session.SessionID = "helper_test"
		err := tx.GameSession().Create(ctx, session)
		sessionID = session.ID
		return err
	}
	
	op2 := func(tx *Transaction) error {
		result := CreateTestGameResult(sessionID, 1, 1, 100, 200)
		err := tx.GameResult().Create(ctx, result)
		resultID = result.ID
		return err
	}
	
	// 执行操作
	err := helper.ExecuteInTransaction(ctx, op1, op2)
	require.NoError(t, err)
	
	// 验证数据已创建
	sessionRepo := NewGameSessionRepository(db)
	session, err := sessionRepo.FindBySessionID(ctx, "helper_test")
	require.NoError(t, err)
	assert.NotNil(t, session)
	
	resultRepo := NewGameResultRepository(db)
	result, err := resultRepo.FindByID(ctx, resultID)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransactionHelper_RunInReadOnlyTransaction(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewTransactionManager(db)
	helper := NewTransactionHelper(manager)
	ctx := context.Background()

	var sessionCount int64
	
	err := helper.RunInReadOnlyTransaction(ctx, func(tx *Transaction) error {
		// 在只读事务中查询数据
		var sessions []models.GameSession
		err := tx.GetDB().Find(&sessions).Error
		if err != nil {
			return err
		}
		sessionCount = int64(len(sessions))
		return nil
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sessionCount, int64(0))
}

func TestManager_Integration(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewManager(db)
	ctx := context.Background()

	// 测试获取各种仓储
	sessionRepo := manager.GameSession()
	assert.NotNil(t, sessionRepo)
	
	resultRepo := manager.GameResult()
	assert.NotNil(t, resultRepo)
	
	deviceRepo := manager.DeviceStatus()
	assert.NotNil(t, deviceRepo)
	
	configRepo := manager.SystemConfig()
	assert.NotNil(t, configRepo)
	
	// 测试事务管理器
	txManager := manager.Transaction()
	assert.NotNil(t, txManager)
	
	// 测试在事务中执行
	err := manager.WithTransaction(ctx, func(tx *Transaction) error {
		session := CreateTestGameSession(1, 1)
		session.SessionID = "manager_test"
		return tx.GameSession().Create(ctx, session)
	})
	require.NoError(t, err)
	
	// 验证数据已创建
	session, err := sessionRepo.FindBySessionID(ctx, "manager_test")
	require.NoError(t, err)
	assert.NotNil(t, session)
}

func TestBatchOperator_CreateGameSessionWithResults(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewManager(db)
	batch := NewBatchOperator(manager)
	ctx := context.Background()

	// 创建会话和结果
	session := CreateTestGameSession(1, 1)
	session.SessionID = "batch_session"
	
	// 为每个结果设置唯一的RoundID
	results := []*models.GameResult{
		CreateTestGameResult(0, 1, 1, 100, 200),
		CreateTestGameResult(0, 1, 1, 100, 0),
		CreateTestGameResult(0, 1, 1, 100, 500),
	}
	for i, result := range results {
		result.RoundID = fmt.Sprintf("batch_round_%d", i)
	}
	
	// 批量创建
	err := batch.CreateGameSessionWithResults(ctx, session, results)
	require.NoError(t, err)
	
	// 验证会话已创建
	sessionRepo := manager.GameSession()
	foundSession, err := sessionRepo.FindBySessionID(ctx, "batch_session")
	require.NoError(t, err)
	assert.NotNil(t, foundSession)
	
	// 验证结果已创建并关联到会话
	resultRepo := manager.GameResult()
	pagination := NewPagination(1, 10)
	foundResults, err := resultRepo.FindBySessionID(ctx, foundSession.ID, pagination)
	require.NoError(t, err)
	assert.Len(t, foundResults, 3)
	
	// 验证所有结果都关联到正确的会话
	for _, result := range foundResults {
		assert.Equal(t, foundSession.ID, result.SessionID)
	}
}

func TestBatchOperator_UpdateDeviceStatusBatch(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewManager(db)
	batch := NewBatchOperator(manager)
	ctx := context.Background()

	// 创建测试设备
	devices := make([]*models.DeviceStatus, 3)
	for i := 0; i < 3; i++ {
		device := CreateTestDeviceStatus(
			"batch_device_"+string(rune('0'+i)),
			"批量设备"+string(rune('0'+i)),
			"slot_machine",
			"online",
		)
		err := manager.DeviceStatus().Create(ctx, device)
		require.NoError(t, err)
		devices[i] = device
	}
	
	// 批量更新状态
	for _, device := range devices {
		device.Status = "maintenance"
		device.CPU = 99.9
	}
	
	err := batch.UpdateDeviceStatusBatch(ctx, devices)
	require.NoError(t, err)
	
	// 验证所有设备已更新
	for i, _ := range devices {
		found, err := manager.DeviceStatus().FindByDeviceID(ctx, "batch_device_"+string(rune('0'+i)))
		require.NoError(t, err)
		assert.Equal(t, "maintenance", found.Status)
		assert.Equal(t, 99.9, found.CPU)
	}
}

func TestUnitOfWork(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	manager := NewManager(db)
	ctx := context.Background()
	
	// 创建工作单元
	uow := NewUnitOfWork(manager)
	
	// var sessionID uint
	var configKey = "uow_test_config"
	
	// 注册操作1：创建游戏会话
	uow.Register(func(tx *Transaction) error {
		session := CreateTestGameSession(1, 1)
		session.SessionID = "uow_session"
		err := tx.GameSession().Create(ctx, session)
		// sessionID = session.ID
		return err
	})
	
	// 注册操作2：创建系统配置
	uow.Register(func(tx *Transaction) error {
		return tx.SystemConfig().Set(ctx, configKey, "uow_value", "UOW测试配置")
	})
	
	// 提交工作单元
	err := uow.Commit(ctx)
	require.NoError(t, err)
	
	// 验证操作已执行
	session, err := manager.GameSession().FindBySessionID(ctx, "uow_session")
	require.NoError(t, err)
	assert.NotNil(t, session)
	
	config, err := manager.SystemConfig().Get(ctx, configKey)
	require.NoError(t, err)
	assert.Equal(t, "uow_value", config.Value)
	
	// 清除工作单元
	uow.Clear()
	
	// 再次提交应该什么都不做
	err = uow.Commit(ctx)
	require.NoError(t, err)
}

func TestProvider(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	provider := NewProvider(db)
	
	// 测试获取管理器
	manager := provider.GetManager()
	assert.NotNil(t, manager)
	
	// 测试获取各种仓储
	assert.NotNil(t, provider.GameSession())
	assert.NotNil(t, provider.GameResult())
	assert.NotNil(t, provider.DeviceStatus())
	assert.NotNil(t, provider.SystemConfig())
}