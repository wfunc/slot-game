package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
)

func TestGameSessionRepository_Create(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 测试创建游戏会话
	session := CreateTestGameSession(1, 1)
	err := repo.Create(ctx, session)
	require.NoError(t, err)
	assert.NotZero(t, session.ID)

	// 验证会话已创建
	found, err := repo.FindBySessionID(ctx, session.SessionID)
	require.NoError(t, err)
	AssertGameSession(t, session, found)
}

func TestGameSessionRepository_Update(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建会话
	session := CreateTestGameSession(1, 1)
	err := repo.Create(ctx, session)
	require.NoError(t, err)

	// 更新会话
	session.TotalBet = 100
	session.TotalWin = 200
	session.Status = "ended"
	err = repo.Update(ctx, session)
	require.NoError(t, err)

	// 验证更新
	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), found.TotalBet)
	assert.Equal(t, int64(200), found.TotalWin)
	assert.Equal(t, "ended", found.Status)
}

func TestGameSessionRepository_UpdateBySessionID(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建会话
	session := CreateTestGameSession(1, 1)
	err := repo.Create(ctx, session)
	require.NoError(t, err)

	// 使用SessionID更新
	updates := map[string]interface{}{
		"total_rounds": 5,
		"total_bet":    500,
		"total_win":    600,
		"peak_win":     300,
	}
	err = repo.UpdateBySessionID(ctx, session.SessionID, updates)
	require.NoError(t, err)

	// 验证更新
	found, err := repo.FindBySessionID(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, 5, found.TotalRounds)
	assert.Equal(t, int64(500), found.TotalBet)
	assert.Equal(t, int64(600), found.TotalWin)
	assert.Equal(t, int64(300), found.PeakWin)
}

func TestGameSessionRepository_FindByUserID(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 为用户1创建多个会话
	for i := 0; i < 5; i++ {
		session := CreateTestGameSession(1, 1)
		session.SessionID = session.SessionID + "_" + string(rune('0'+i))
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	// 查询用户1的会话
	pagination := NewPagination(1, 10)
	sessions, err := repo.FindByUserID(ctx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, sessions, 5)
	assert.Equal(t, int64(5), pagination.Total)
}

func TestGameSessionRepository_FindActiveByUserID(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建已结束的会话
	endedSession := CreateTestGameSession(1, 1)
	endedSession.Status = "ended"
	err := repo.Create(ctx, endedSession)
	require.NoError(t, err)

	// 创建活跃会话
	activeSession := CreateTestGameSession(1, 1)
	activeSession.Status = "playing"
	activeSession.SessionID = "active_session"
	err = repo.Create(ctx, activeSession)
	require.NoError(t, err)

	// 查找活跃会话
	found, err := repo.FindActiveByUserID(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "playing", found.Status)
	assert.Equal(t, "active_session", found.SessionID)
}

func TestGameSessionRepository_GetStatistics(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建多个游戏会话
	now := time.Now()
	for i := 0; i < 3; i++ {
		session := CreateTestGameSession(1, 1)
		session.SessionID = session.SessionID + "_" + string(rune('0'+i))
		session.TotalBet = int64((i + 1) * 100)
		session.TotalWin = int64((i + 1) * 150)
		session.PeakWin = int64((i + 1) * 50)
		session.Duration = 3600 // 1小时
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	// 获取统计
	startTime := now.Add(-24 * time.Hour)
	endTime := now.Add(24 * time.Hour)
	stats, err := repo.GetStatistics(ctx, 1, startTime, endTime)
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.TotalGames)
	assert.Equal(t, int64(600), stats.TotalBet)  // 100+200+300
	assert.Equal(t, int64(900), stats.TotalWin)  // 150+300+450
	assert.Equal(t, int64(300), stats.TotalProfit) // 900-600
	assert.Equal(t, int64(150), stats.MaxWin)     // max peak win
	assert.Equal(t, int64(180), stats.TotalMinutes) // 3*3600/60 = 180
}

func TestGameSessionRepository_EndSession(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建活跃会话
	session := CreateTestGameSession(1, 1)
	session.Status = "playing"
	err := repo.Create(ctx, session)
	require.NoError(t, err)

	// 等待一秒以产生持续时间
	time.Sleep(1 * time.Second)

	// 结束会话
	err = repo.EndSession(ctx, session.SessionID)
	require.NoError(t, err)

	// 验证会话已结束
	found, err := repo.FindBySessionID(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, "ended", found.Status)
	assert.NotNil(t, found.EndedAt)
	assert.Greater(t, found.Duration, 0)
}

func TestGameSessionRepository_CleanupExpiredSessions(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 创建过期会话
	oldSession := CreateTestGameSession(1, 1)
	oldSession.Status = "playing"
	err := repo.Create(ctx, oldSession)
	require.NoError(t, err)

	// 手动设置更新时间为过去
	db.Model(&models.GameSession{}).
		Where("id = ?", oldSession.ID).
		Update("updated_at", time.Now().Add(-2*time.Hour))

	// 创建新会话
	newSession := CreateTestGameSession(1, 1)
	newSession.Status = "playing"
	newSession.SessionID = "new_session"
	err = repo.Create(ctx, newSession)
	require.NoError(t, err)

	// 清理过期会话
	expiredBefore := time.Now().Add(-1 * time.Hour)
	affected, err := repo.CleanupExpiredSessions(ctx, expiredBefore)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	// 验证过期会话状态
	found, err := repo.FindByID(ctx, oldSession.ID)
	require.NoError(t, err)
	assert.Equal(t, "expired", found.Status)

	// 验证新会话未受影响
	found, err = repo.FindBySessionID(ctx, "new_session")
	require.NoError(t, err)
	assert.Equal(t, "playing", found.Status)
}

func TestGameSessionRepository_WithTx(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameSessionRepository(db)
	ctx := context.Background()

	// 开始事务
	tx := db.Begin()

	// 在事务中创建会话
	txRepo := repo.WithTx(tx).(*gameSessionRepo)
	session := CreateTestGameSession(1, 1)
	err := txRepo.Create(ctx, session)
	require.NoError(t, err)

	// 回滚事务
	tx.Rollback()

	// 验证会话未被创建
	found, err := repo.FindBySessionID(ctx, session.SessionID)
	assert.Error(t, err)
	assert.Nil(t, found)
}