package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
)

func TestSystemConfigRepository_Set(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 测试设置字符串配置
	err := repo.Set(ctx, "test_string", "test_value", "测试字符串配置")
	require.NoError(t, err)

	// 验证配置已创建
	config, err := repo.Get(ctx, "test_string")
	require.NoError(t, err)
	assert.Equal(t, "test_value", config.Value)
	assert.Equal(t, "string", config.Type)
	assert.Equal(t, "测试字符串配置", config.Description)

	// 测试更新现有配置
	err = repo.Set(ctx, "test_string", "new_value", "更新后的描述")
	require.NoError(t, err)

	// 验证配置已更新
	config, err = repo.Get(ctx, "test_string")
	require.NoError(t, err)
	assert.Equal(t, "new_value", config.Value)
	assert.Equal(t, "更新后的描述", config.Description)
}

func TestSystemConfigRepository_SetDifferentTypes(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	tests := []struct {
		key         string
		value       interface{}
		expectedVal string
		expectedType string
	}{
		{"test_int", 123, "123", "int"},
		{"test_int64", int64(456), "456", "int"},
		{"test_float", 78.9, "78.900000", "float"},
		{"test_bool", true, "true", "bool"},
		{"test_json", map[string]string{"key": "value"}, `{"key":"value"}`, "json"},
		{"test_array", []int{1, 2, 3}, "[1,2,3]", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := repo.Set(ctx, tt.key, tt.value, "测试配置")
			require.NoError(t, err)

			config, err := repo.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedVal, config.Value)
			assert.Equal(t, tt.expectedType, config.Type)
		})
	}
}

func TestSystemConfigRepository_GetString(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 设置配置
	err := repo.Set(ctx, "string_config", "hello", "")
	require.NoError(t, err)

	// 获取存在的配置
	val := repo.GetString(ctx, "string_config", "default")
	assert.Equal(t, "hello", val)

	// 获取不存在的配置，返回默认值
	val = repo.GetString(ctx, "non_existent", "default")
	assert.Equal(t, "default", val)
}

func TestSystemConfigRepository_GetInt(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 设置配置
	err := repo.Set(ctx, "int_config", 42, "")
	require.NoError(t, err)

	// 获取存在的配置
	val := repo.GetInt(ctx, "int_config", 0)
	assert.Equal(t, 42, val)

	// 获取不存在的配置，返回默认值
	val = repo.GetInt(ctx, "non_existent", 99)
	assert.Equal(t, 99, val)

	// 设置非数字配置
	err = repo.Set(ctx, "invalid_int", "not_a_number", "")
	require.NoError(t, err)
	
	// 获取无效的数字配置，返回默认值
	val = repo.GetInt(ctx, "invalid_int", 100)
	assert.Equal(t, 100, val)
}

func TestSystemConfigRepository_GetFloat(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 设置配置
	err := repo.Set(ctx, "float_config", 3.14159, "")
	require.NoError(t, err)

	// 获取存在的配置
	val := repo.GetFloat(ctx, "float_config", 0.0)
	assert.InDelta(t, 3.14159, val, 0.00001)

	// 获取不存在的配置，返回默认值
	val = repo.GetFloat(ctx, "non_existent", 2.71828)
	assert.Equal(t, 2.71828, val)
}

func TestSystemConfigRepository_GetBool(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 设置配置
	err := repo.Set(ctx, "bool_config", true, "")
	require.NoError(t, err)

	// 获取存在的配置
	val := repo.GetBool(ctx, "bool_config", false)
	assert.True(t, val)

	// 设置false配置
	err = repo.Set(ctx, "bool_false", false, "")
	require.NoError(t, err)
	val = repo.GetBool(ctx, "bool_false", true)
	assert.False(t, val)

	// 获取不存在的配置，返回默认值
	val = repo.GetBool(ctx, "non_existent", true)
	assert.True(t, val)
}

func TestSystemConfigRepository_GetJSON(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 定义测试结构
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// 设置JSON配置
	original := TestStruct{Name: "test", Value: 123}
	err := repo.Set(ctx, "json_config", original, "")
	require.NoError(t, err)

	// 获取JSON配置
	var result TestStruct
	err = repo.GetJSON(ctx, "json_config", &result)
	require.NoError(t, err)
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Value, result.Value)

	// 获取不存在的配置
	var notFound TestStruct
	err = repo.GetJSON(ctx, "non_existent", &notFound)
	assert.Error(t, err)
}

func TestSystemConfigRepository_SetBatch(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 批量设置配置
	configs := map[string]interface{}{
		"batch_1": "value1",
		"batch_2": 123,
		"batch_3": true,
		"batch_4": 45.67,
	}

	err := repo.SetBatch(ctx, configs)
	require.NoError(t, err)

	// 验证所有配置已设置
	assert.Equal(t, "value1", repo.GetString(ctx, "batch_1", ""))
	assert.Equal(t, 123, repo.GetInt(ctx, "batch_2", 0))
	assert.True(t, repo.GetBool(ctx, "batch_3", false))
	assert.InDelta(t, 45.67, repo.GetFloat(ctx, "batch_4", 0.0), 0.01)
}

func TestSystemConfigRepository_Update(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 先创建配置
	err := repo.Set(ctx, "update_test", "initial", "初始描述")
	require.NoError(t, err)

	// 更新配置值
	err = repo.Update(ctx, "update_test", "updated")
	require.NoError(t, err)

	// 验证值已更新，但描述保持不变
	config, err := repo.Get(ctx, "update_test")
	require.NoError(t, err)
	assert.Equal(t, "updated", config.Value)
	assert.Equal(t, "初始描述", config.Description)
}

func TestSystemConfigRepository_Delete(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 创建配置
	err := repo.Set(ctx, "delete_test", "value", "")
	require.NoError(t, err)

	// 验证配置存在
	config, err := repo.Get(ctx, "delete_test")
	require.NoError(t, err)
	assert.NotNil(t, config)

	// 删除配置
	err = repo.Delete(ctx, "delete_test")
	require.NoError(t, err)

	// 验证配置已删除
	config, err = repo.Get(ctx, "delete_test")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestSystemConfigRepository_GetAll(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 获取所有配置
	configs, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, configs)

	// 验证配置按组和键排序
	for i := 1; i < len(configs); i++ {
		prev := configs[i-1]
		curr := configs[i]
		
		if prev.Group == curr.Group {
			assert.LessOrEqual(t, prev.Key, curr.Key)
		} else {
			assert.LessOrEqual(t, prev.Group, curr.Group)
		}
	}
}

func TestSystemConfigRepository_GetByGroup(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 创建特定组的配置
	groupConfigs := []struct {
		key   string
		value string
		group string
	}{
		{"group_test_1", "value1", "test_group"},
		{"group_test_2", "value2", "test_group"},
		{"group_test_3", "value3", "other_group"},
	}

	for _, gc := range groupConfigs {
		config := &models.SystemConfig{
			Key:         gc.key,
			Value:       gc.value,
			Type:        "string",
			Group:       gc.group,
			Description: "测试配置",
		}
		err := db.Create(config).Error
		require.NoError(t, err)
	}

	// 刷新缓存
	err := repo.RefreshCache(ctx)
	require.NoError(t, err)

	// 获取特定组的配置
	configs, err := repo.GetByGroup(ctx, "test_group")
	require.NoError(t, err)
	assert.Len(t, configs, 2)
	
	// 验证都是同一组
	for _, config := range configs {
		assert.Equal(t, "test_group", config.Group)
	}
}

func TestSystemConfigRepository_GetPublic(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 创建公开和私有配置
	publicConfig := &models.SystemConfig{
		Key:         "public_test",
		Value:       "public_value",
		Type:        "string",
		Group:       "public",
		Description: "公开配置",
		IsPublic:    true,
	}
	privateConfig := &models.SystemConfig{
		Key:         "private_test",
		Value:       "private_value",
		Type:        "string",
		Group:       "private",
		Description: "私有配置",
		IsPublic:    false,
	}

	err := db.Create(publicConfig).Error
	require.NoError(t, err)
	err = db.Create(privateConfig).Error
	require.NoError(t, err)

	// 刷新缓存
	err = repo.RefreshCache(ctx)
	require.NoError(t, err)

	// 获取公开配置
	configs, err := repo.GetPublic(ctx)
	require.NoError(t, err)

	// 验证只返回公开配置
	hasPublic := false
	hasPrivate := false
	for _, config := range configs {
		assert.True(t, config.IsPublic)
		if config.Key == "public_test" {
			hasPublic = true
		}
		if config.Key == "private_test" {
			hasPrivate = true
		}
	}
	assert.True(t, hasPublic)
	assert.False(t, hasPrivate)
}

func TestSystemConfigRepository_Cache(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db).(*systemConfigRepo)
	ctx := context.Background()

	// 设置配置
	err := repo.Set(ctx, "cache_test", "cached_value", "")
	require.NoError(t, err)

	// 第一次获取（从数据库）
	config1, err := repo.Get(ctx, "cache_test")
	require.NoError(t, err)
	assert.Equal(t, "cached_value", config1.Value)

	// 验证缓存中有数据
	assert.Contains(t, repo.cache, "cache_test")

	// 直接修改数据库（绕过缓存）
	db.Model(&models.SystemConfig{}).
		Where("`key` = ?", "cache_test").
		Update("value", "direct_update")

	// 第二次获取（从缓存）
	config2, err := repo.Get(ctx, "cache_test")
	require.NoError(t, err)
	assert.Equal(t, "cached_value", config2.Value) // 仍然是缓存的值

	// 刷新缓存
	err = repo.RefreshCache(ctx)
	require.NoError(t, err)

	// 第三次获取（缓存已刷新）
	config3, err := repo.Get(ctx, "cache_test")
	require.NoError(t, err)
	assert.Equal(t, "direct_update", config3.Value)
}

func TestConfigHelper_GetGameConfig(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	helper := NewConfigHelper(repo)
	ctx := context.Background()

	// 设置游戏配置
	gameConfigs := map[string]interface{}{
		"game.slot.min_bet":       10,
		"game.slot.max_bet":       1000,
		"game.slot.jackpot_rate":  0.02,
		"game.slot.rtp":           97.5,
		"game.pusher.coin_value":  0.5,
		"game.pusher.min_force":   10,
		"game.pusher.max_force":   30,
		"game.pusher.push_interval": 2000,
		"wallet.initial_coins":    200,
		"wallet.daily_bonus":      100,
		"wallet.max_coins":        999999,
	}

	for key, value := range gameConfigs {
		err := repo.Set(ctx, key, value, "")
		require.NoError(t, err)
	}

	// 获取游戏配置
	config, err := helper.GetGameConfig(ctx)
	require.NoError(t, err)

	// 验证老虎机配置
	assert.Equal(t, 10, config.Slot.MinBet)
	assert.Equal(t, 1000, config.Slot.MaxBet)
	assert.InDelta(t, 0.02, config.Slot.JackpotRate, 0.001)
	assert.InDelta(t, 97.5, config.Slot.RTP, 0.1)

	// 验证推币机配置
	assert.InDelta(t, 0.5, config.Pusher.CoinValue, 0.01)
	assert.Equal(t, 10, config.Pusher.MinForce)
	assert.Equal(t, 30, config.Pusher.MaxForce)
	assert.Equal(t, 2000, config.Pusher.PushInterval)

	// 验证钱包配置
	assert.Equal(t, 200, config.Wallet.InitialCoins)
	assert.Equal(t, 100, config.Wallet.DailyBonus)
	assert.Equal(t, 999999, config.Wallet.MaxCoins)
}

func TestSystemConfigRepository_WithTx(t *testing.T) {
	db := TestDB(t)
	SeedTestData(t, db)
	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// 测试事务场景
	t.Run("Rollback", func(t *testing.T) {
		// 开始事务
		tx := db.Begin()
		require.NotNil(t, tx)

		// 在事务中设置配置
		txRepo := &systemConfigRepo{
			BaseRepo: &BaseRepo{db: tx},
			cache:    make(map[string]*models.SystemConfig),
		}
		err := txRepo.Set(ctx, "tx_rollback_config", "tx_value", "事务测试")
		require.NoError(t, err)

		// 回滚事务
		tx.Rollback()

		// 验证配置未被创建
		config, err := repo.Get(ctx, "tx_rollback_config")
		assert.Error(t, err)
		assert.Nil(t, config)
	})
	
	t.Run("Commit", func(t *testing.T) {
		// 开始事务
		tx := db.Begin()
		
		// 在事务中设置配置
		txRepo := &systemConfigRepo{
			BaseRepo: &BaseRepo{db: tx},
			cache:    make(map[string]*models.SystemConfig),
		}
		err := txRepo.Set(ctx, "tx_commit_config", "tx_value", "事务测试")
		require.NoError(t, err)
		
		// 提交事务
		tx.Commit()
		
		// 验证配置已创建
		config, err := repo.Get(ctx, "tx_commit_config")
		require.NoError(t, err)
		assert.Equal(t, "tx_value", config.Value)
	})
}