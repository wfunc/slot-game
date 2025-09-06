package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
)

func TestGameResultRepository_Create(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 创建游戏结果
	result := CreateTestGameResult(1, 1, 1, 100, 200)
	err := repo.Create(ctx, result)
	require.NoError(t, err)
	assert.NotZero(t, result.ID)

	// 验证结果已创建
	found, err := repo.FindByRoundID(ctx, result.RoundID)
	require.NoError(t, err)
	AssertGameResult(t, result, found)
}

func TestGameResultRepository_BatchCreate(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 批量创建游戏结果
	results := make([]*models.GameResult, 5)
	for i := 0; i < 5; i++ {
		results[i] = CreateTestGameResult(1, 1, 1, int64((i+1)*100), int64((i+1)*150))
		results[i].RoundID = results[i].RoundID + "_" + string(rune('0'+i))
	}

	err := repo.BatchCreate(ctx, results)
	require.NoError(t, err)

	// 验证所有结果已创建
	for _, result := range results {
		found, err := repo.FindByRoundID(ctx, result.RoundID)
		require.NoError(t, err)
		assert.NotNil(t, found)
	}
}

func TestGameResultRepository_FindBySessionID(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 创建多个游戏结果
	sessionID := uint(1)
	for i := 0; i < 5; i++ {
		result := CreateTestGameResult(sessionID, 1, 1, 100, int64(i*100))
		result.RoundID = result.RoundID + "_" + string(rune('0'+i))
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 查询会话的所有结果
	pagination := NewPagination(1, 10)
	results, err := repo.FindBySessionID(ctx, sessionID, pagination)
	require.NoError(t, err)
	assert.Len(t, results, 5)
	assert.Equal(t, int64(5), pagination.Total)
}

func TestGameResultRepository_FindWinsByUserID(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	userID := uint(1)
	
	// 创建有赢取的结果
	for i := 0; i < 3; i++ {
		result := CreateTestGameResult(1, userID, 1, 100, int64((i+1)*200))
		result.RoundID = "win_" + string(rune('0'+i))
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 创建未赢取的结果
	for i := 0; i < 2; i++ {
		result := CreateTestGameResult(1, userID, 1, 100, 0)
		result.RoundID = "lose_" + string(rune('0'+i))
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 查询中奖记录
	pagination := NewPagination(1, 10)
	wins, err := repo.FindWinsByUserID(ctx, userID, pagination)
	require.NoError(t, err)
	assert.Len(t, wins, 3) // 只返回有赢取的记录
	assert.Equal(t, int64(3), pagination.Total)
	
	// 验证都是赢取记录
	for _, win := range wins {
		assert.Greater(t, win.WinAmount, int64(0))
	}
}

func TestGameResultRepository_GetWinStatistics(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	userID := uint(1)
	now := time.Now()
	
	// 创建各种类型的游戏结果
	testData := []struct {
		betAmount  int64
		winAmount  int64
		multiplier float64
		isJackpot  bool
		isBonus    bool
	}{
		{100, 0, 0, false, false},         // 输
		{100, 200, 2, false, false},       // 小赢
		{100, 1000, 10, false, true},      // 奖励
		{100, 10000, 100, false, false},   // 大奖
		{100, 25000, 250, false, false},   // 巨奖
		{100, 50000, 500, true, false},    // 大奖池
		{100, 100000, 1000, true, true},   // 超级大奖
	}

	for i, td := range testData {
		result := CreateTestGameResult(1, userID, 1, td.betAmount, td.winAmount)
		result.RoundID = "stat_" + string(rune('0'+i))
		result.Multiplier = td.multiplier
		result.IsJackpot = td.isJackpot
		result.IsBonus = td.isBonus
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 获取统计
	startTime := now.Add(-24 * time.Hour)
	endTime := now.Add(24 * time.Hour)
	stats, err := repo.GetWinStatistics(ctx, userID, startTime, endTime)
	require.NoError(t, err)

	assert.Equal(t, int64(7), stats.TotalRounds)
	assert.Equal(t, int64(6), stats.WinRounds) // 6个赢的回合
	assert.Equal(t, int64(186200), stats.TotalWinAmount) // 200+1000+10000+25000+50000+100000
	assert.Equal(t, int64(100000), stats.MaxWinAmount)
	assert.Equal(t, float64(1000), stats.MaxMultiplier)
	assert.Equal(t, int64(2), stats.JackpotCount)
	assert.Equal(t, int64(2), stats.BonusCount)
	assert.Equal(t, int64(4), stats.BigWinCount)   // >=100倍
	assert.Equal(t, int64(3), stats.MegaWinCount)  // >=250倍
	assert.Equal(t, int64(2), stats.SuperWinCount) // >=500倍
	assert.Greater(t, stats.WinRate, float64(80))  // 赢率>80%
}

func TestGameResultRepository_GetJackpotRecords(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 创建大奖记录
	for i := 0; i < 5; i++ {
		result := CreateTestGameResult(1, 1, 1, 100, int64((i+1)*10000))
		result.RoundID = "jackpot_" + string(rune('0'+i))
		result.IsJackpot = true
		err := repo.Create(ctx, result)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // 确保时间顺序
	}

	// 创建普通记录
	normalResult := CreateTestGameResult(1, 1, 1, 100, 200)
	normalResult.RoundID = "normal_1"
	normalResult.IsJackpot = false
	err := repo.Create(ctx, normalResult)
	require.NoError(t, err)

	// 获取大奖记录
	jackpots, err := repo.GetJackpotRecords(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, jackpots, 3)
	
	// 验证都是大奖记录，且按时间倒序（最新的在前）
	for i, jackpot := range jackpots {
		assert.True(t, jackpot.IsJackpot)
		if i > 0 {
			// 后面的记录应该更早或相同时间
			assert.True(t, jackpot.PlayedAt.Before(jackpots[i-1].PlayedAt) || 
				jackpot.PlayedAt.Equal(jackpots[i-1].PlayedAt))
		}
	}
}

func TestGameResultRepository_GetBigWins(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 创建不同金额的记录
	amounts := []int64{100, 500, 1000, 5000, 10000}
	for i, amount := range amounts {
		result := CreateTestGameResult(1, 1, 1, 100, amount)
		result.RoundID = "bigwin_" + string(rune('0'+i))
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 获取大额中奖（>=1000）
	bigWins, err := repo.GetBigWins(ctx, 1000, 10)
	require.NoError(t, err)
	assert.Len(t, bigWins, 3) // 1000, 5000, 10000
	
	// 验证都是大额中奖，且按金额倒序
	for i, win := range bigWins {
		assert.GreaterOrEqual(t, win.WinAmount, int64(1000))
		if i > 0 {
			assert.LessOrEqual(t, win.WinAmount, bigWins[i-1].WinAmount)
		}
	}
	
	// 验证最大金额在最前
	assert.Equal(t, int64(10000), bigWins[0].WinAmount)
}

func TestGameResultRepository_Pagination(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 创建20个游戏结果
	for i := 0; i < 20; i++ {
		result := CreateTestGameResult(1, 1, 1, 100, int64(i*100))
		result.RoundID = "page_" + string(rune('0'+i))
		err := repo.Create(ctx, result)
		require.NoError(t, err)
	}

	// 测试第一页
	page1 := NewPagination(1, 5)
	results1, err := repo.FindByGameID(ctx, 1, page1)
	require.NoError(t, err)
	assert.Len(t, results1, 5)
	assert.Equal(t, int64(20), page1.Total)

	// 测试第二页
	page2 := NewPagination(2, 5)
	results2, err := repo.FindByGameID(ctx, 1, page2)
	require.NoError(t, err)
	assert.Len(t, results2, 5)
	
	// 确保两页数据不同
	assert.NotEqual(t, results1[0].ID, results2[0].ID)

	// 测试最后一页
	page5 := NewPagination(5, 5)
	results5, err := repo.FindByGameID(ctx, 1, page5)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results5), 5)
}

func TestGameResultRepository_WithTx(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewGameResultRepository(db)
	ctx := context.Background()

	// 开始事务
	tx := db.Begin()

	// 在事务中创建结果
	txRepo := repo.WithTx(tx).(*gameResultRepo)
	result := CreateTestGameResult(1, 1, 1, 100, 200)
	err := txRepo.Create(ctx, result)
	require.NoError(t, err)

	// 回滚事务
	tx.Rollback()

	// 验证结果未被创建
	found, err := repo.FindByRoundID(ctx, result.RoundID)
	assert.Error(t, err)
	assert.Nil(t, found)
}