package database

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/wfunc/slot-game/internal/logger"
	"github.com/wfunc/slot-game/internal/models"
	"go.uber.org/zap"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 清理过期锁文件
	CleanupStaleLocks()

	// 获取迁移锁，避免多个进程同时迁移
	dbPath := getDBPath()
	if dbPath != "" {
		lockFile, err := acquireMigrationLock(dbPath)
		if err != nil {
			logger.Error("无法获取迁移锁", zap.Error(err))
			return fmt.Errorf("获取迁移锁失败: %w", err)
		}
		defer releaseMigrationLock(lockFile)
	}

	// 注意：SQLite性能优化已在database.go的Init函数中完成

	// 定义需要迁移的模型（排除 Jackpot 相关表，单独处理）
	migrationModels := []interface{}{
		// 用户相关
		&models.User{},
		&models.UserProfile{},
		&models.UserAuth{},
		&models.UserSession{},

		// 游戏相关
		&models.Game{},
		&models.GameRoom{},
		&models.GameSession{},
		&models.GameResult{},
		&models.GameState{},  // 添加游戏状态表

		// 老虎机相关
		&models.SlotMachine{},
		&models.SlotSpin{},
		&models.SlotWinLine{},

		// JP奖池相关 - 已移除，单独处理
		// &models.Jackpot{},
		// &models.JackpotHistory{},

		// 推币机相关
		&models.PusherMachine{},
		&models.PusherSession{},
		&models.CoinDrop{},

		// 交易相关
		&models.Transaction{},
		&models.CoinPurchase{},
		&models.Withdrawal{},
		&models.Wallet{},

		// 系统相关
		&models.SystemLog{},
		&models.ErrorLog{},
		&models.DeviceStatus{},
		&models.SystemConfig{},

		// 串口日志相关
		&models.SerialLog{},
	}

	// 执行迁移
	logger.Info("开始数据库迁移...")

	// 设置 SQLite 专用配置，避免锁定问题
	if DB.Dialector.Name() == "sqlite" {
		// 禁用外键约束，避免重建表时的问题
		DB.Exec("PRAGMA foreign_keys = OFF")
		defer DB.Exec("PRAGMA foreign_keys = ON")
	}

	for _, model := range migrationModels {
		tableName := getTableName(model)

		// 检查表是否存在且有数据
		if shouldSkipMigration(tableName) {
			logger.Info("跳过大型表的迁移", zap.String("table", tableName))
			continue
		}

		// 永远不要让GORM尝试迁移jackpot相关表
		// 这些表由createJackpotTables单独处理
		if tableName == "jackpots" || tableName == "jackpot_histories" {
			logger.Debug("跳过jackpot表迁移，由专用函数处理", zap.String("table", tableName))
			continue
		}

		if err := DB.AutoMigrate(model); err != nil {
			// 如果错误涉及jackpot表，忽略它（可能是其他表引用了jackpot）
			if strings.Contains(strings.ToLower(err.Error()), "jackpot") {
				logger.Warn("忽略与jackpot相关的迁移错误", zap.Error(err))
				continue
			}
			logger.Error("迁移失败",
				zap.String("model", fmt.Sprintf("%T", model)),
				zap.Error(err),
			)
			return err
		}
		logger.Debug("迁移成功", zap.String("model", fmt.Sprintf("%T", model)))
	}

	// 手动创建 Jackpot 相关表（避免 GORM 重建表导致锁定）
	if err := createJackpotTables(); err != nil {
		logger.Warn("创建Jackpot表失败，可能已存在", zap.Error(err))
	}

	// 创建索引
	if err := createIndexes(); err != nil {
		return err
	}

	// 初始化默认数据
	if err := initDefaultData(); err != nil {
		return err
	}

	logger.Info("数据库迁移完成")
	return nil
}

// createIndexes 创建数据库索引
func createIndexes() error {
	// 用户表索引
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_users_username"), zap.Error(err))
	}
	
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_users_phone"), zap.Error(err))
	}

	// 游戏会话索引
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_game_sessions_user_id ON game_sessions(user_id)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_game_sessions_user_id"), zap.Error(err))
	}
	
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_game_sessions_status ON game_sessions(status)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_game_sessions_status"), zap.Error(err))
	}

	// 交易表索引
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_transactions_user_id"), zap.Error(err))
	}
	
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_transactions_type"), zap.Error(err))
	}
	
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_transactions_created_at"), zap.Error(err))
	}

	// 串口日志表索引
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_serial_logs_device_type ON serial_logs(device_type)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_serial_logs_device_type"), zap.Error(err))
	}

	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_serial_logs_direction ON serial_logs(direction)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_serial_logs_direction"), zap.Error(err))
	}

	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_serial_logs_function ON serial_logs(function)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_serial_logs_function"), zap.Error(err))
	}

	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_serial_logs_request_id ON serial_logs(request_id)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_serial_logs_request_id"), zap.Error(err))
	}

	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_serial_logs_created_at ON serial_logs(created_at)").Error; err != nil {
		logger.Warn("创建索引失败", zap.String("index", "idx_serial_logs_created_at"), zap.Error(err))
	}

	logger.Info("数据库索引创建完成")
	return nil
}

// initDefaultData 初始化默认数据
func initDefaultData() error {
	// 检查是否已有数据
	var count int64
	DB.Model(&models.SystemConfig{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 创建默认系统配置
	defaultConfigs := []models.SystemConfig{
		{
			Key:         "system.version",
			Value:       "1.0.0",
			Description: "系统版本",
			Type:        "string",
		},
		{
			Key:         "game.min_bet",
			Value:       "1",
			Description: "最小下注金额",
			Type:        "int",
		},
		{
			Key:         "game.max_bet",
			Value:       "100",
			Description: "最大下注金额",
			Type:        "int",
		},
		{
			Key:         "slot.jackpot_rate",
			Value:       "0.01",
			Description: "老虎机大奖概率",
			Type:        "float",
		},
		{
			Key:         "pusher.coin_value",
			Value:       "0.1",
			Description: "推币机单个币价值",
			Type:        "float",
		},
		{
			Key:         "wallet.initial_coins",
			Value:       "100",
			Description: "新用户初始币数",
			Type:        "int",
		},
		{
			Key:         "wallet.daily_bonus",
			Value:       "50",
			Description: "每日签到奖励币数",
			Type:        "int",
		},
	}

	for _, config := range defaultConfigs {
		if err := DB.Create(&config).Error; err != nil {
			logger.Error("创建默认配置失败", 
				zap.String("key", config.Key),
				zap.Error(err),
			)
		}
	}

	// 创建默认游戏类型
	defaultGames := []models.Game{
		{
			Name:        "经典老虎机",
			Type:        "slot",
			Description: "3x3经典老虎机游戏",
			Status:      "active",
			MinBet:      1,
			MaxBet:      100,
		},
		{
			Name:        "推币机",
			Type:        "pusher",
			Description: "经典推币机游戏",
			Status:      "active",
			MinBet:      1,
			MaxBet:      50,
		},
	}

	for _, game := range defaultGames {
		if err := DB.Create(&game).Error; err != nil {
			logger.Error("创建默认游戏失败",
				zap.String("name", game.Name),
				zap.Error(err),
			)
		}
	}

	logger.Info("默认数据初始化完成")
	return nil
}


// getTableName 获取模型对应的表名
func getTableName(model interface{}) string {
	// 使用反射获取类型
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 尝试调用TableName方法
	if tabler, ok := model.(interface{ TableName() string }); ok {
		return tabler.TableName()
	}

	// 否则使用GORM默认的表名规则
	modelName := t.Name()
	// 转换为蛇形命名并复数化
	tableName := toSnakeCase(modelName) + "s"
	return tableName
}

// toSnakeCase 将驼峰命名转换为蛇形命名
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// shouldSkipMigration 检查是否应该跳过迁移
func shouldSkipMigration(tableName string) bool {
	// 对于serial_logs这种大表，检查是否已存在且有大量数据
	if tableName == "serial_logs" {
		var count int64
		var exists bool

		// 检查表是否存在
		err := DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists).Error
		if err != nil || !exists {
			return false
		}

		// 检查表中的数据量
		DB.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)

		// 如果表存在且数据量超过10000条，跳过迁移
		if count > 10000 {
			logger.Info("表中数据量较大，跳过AutoMigrate",
				zap.String("table", tableName),
				zap.Int64("count", count))

			// 仅添加新的索引，不修改表结构
			ensureIndexesForLargeTable(tableName)
			return true
		}
	}
	return false
}

// createJackpotTables 手动创建 Jackpot 相关表
func createJackpotTables() error {
	// 只在 SQLite 中需要特殊处理
	if DB.Dialector.Name() != "sqlite" {
		// 其他数据库使用正常的 AutoMigrate
		if err := DB.AutoMigrate(&models.Jackpot{}); err != nil {
			return err
		}
		if err := DB.AutoMigrate(&models.JackpotHistory{}); err != nil {
			return err
		}
		return nil
	}

	// SQLite: 手动创建表（如果不存在）
	createJackpotsSQL := `
	CREATE TABLE IF NOT EXISTS jackpots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		game_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		amount INTEGER DEFAULT 0,
		min_amount INTEGER DEFAULT 0,
		max_amount INTEGER DEFAULT 0,
		percentage REAL DEFAULT 0,
		last_won_at DATETIME,
		last_winner INTEGER,
		win_count INTEGER DEFAULT 0,
		total_in INTEGER DEFAULT 0,
		total_out INTEGER DEFAULT 0,
		status TEXT DEFAULT 'active',
		UNIQUE(game_id, type)
	);
	`
	if err := DB.Exec(createJackpotsSQL).Error; err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}

	createJackpotHistoriesSQL := `
	CREATE TABLE IF NOT EXISTS jackpot_histories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		jackpot_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		amount INTEGER NOT NULL,
		won_at DATETIME NOT NULL,
		machine_id TEXT,
		bet_amount INTEGER
	);
	`
	if err := DB.Exec(createJackpotHistoriesSQL).Error; err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}

	// 创建索引
	ensureIndexesForTable("jackpots")
	ensureIndexesForTable("jackpot_histories")

	logger.Info("Jackpot表创建/验证完成")
	return nil
}

// ensureIndexesForTable 为表确保索引存在（通用版本）
func ensureIndexesForTable(tableName string) {
	if tableName == "jackpots" {
		indexes := []string{
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_game_type ON jackpots(game_id, type)",
		}
		for _, idx := range indexes {
			if err := DB.Exec(idx).Error; err != nil {
				if !strings.Contains(err.Error(), "already exists") {
					logger.Warn("创建索引失败", zap.String("index", idx), zap.Error(err))
				}
			}
		}
	}

	if tableName == "jackpot_histories" {
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_jackpot_histories_jackpot_id ON jackpot_histories(jackpot_id)",
			"CREATE INDEX IF NOT EXISTS idx_jackpot_histories_user_id ON jackpot_histories(user_id)",
			"CREATE INDEX IF NOT EXISTS idx_jackpot_histories_won_at ON jackpot_histories(won_at)",
		}
		for _, idx := range indexes {
			if err := DB.Exec(idx).Error; err != nil {
				if !strings.Contains(err.Error(), "already exists") {
					logger.Warn("创建索引失败", zap.String("index", idx), zap.Error(err))
				}
			}
		}
	}
}

// ensureIndexesForLargeTable 为大表确保索引存在
func ensureIndexesForLargeTable(tableName string) {
	if tableName == "serial_logs" {
		// 仅创建不存在的索引，避免重建表
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_device_type ON serial_logs(device_type)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_direction ON serial_logs(direction)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_function ON serial_logs(function)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_request_id ON serial_logs(request_id)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_created_at ON serial_logs(created_at)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_bet ON serial_logs(bet)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_prize ON serial_logs(prize)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_win ON serial_logs(win)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_response_code ON serial_logs(response_code)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_session_id ON serial_logs(session_id)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_ident ON serial_logs(ident)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_timestamp ON serial_logs(timestamp)",
			"CREATE INDEX IF NOT EXISTS idx_serial_logs_command ON serial_logs(command)",
		}

		for _, idx := range indexes {
			if err := DB.Exec(idx).Error; err != nil {
				// 忽略索引已存在的错误
				if !strings.Contains(err.Error(), "already exists") {
					logger.Warn("创建索引失败", zap.String("index", idx), zap.Error(err))
				}
			}
		}
	}
}

// DropAllTables 删除所有表（仅用于测试环境）
func DropAllTables() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 获取所有表名
	var tables []string
	if err := DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables).Error; err != nil {
		return err
	}

	// 删除所有表
	for _, table := range tables {
		if err := DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Error; err != nil {
			logger.Error("删除表失败", zap.String("table", table), zap.Error(err))
			return err
		}
	}

	logger.Info("所有表已删除")
	return nil
}