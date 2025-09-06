package database

import (
	"fmt"
	
	"github.com/wfunc/slot-game/internal/logger"
	"github.com/wfunc/slot-game/internal/models"
	"go.uber.org/zap"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 定义需要迁移的模型
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
		
		// 老虎机相关
		&models.SlotMachine{},
		&models.SlotSpin{},
		&models.SlotWinLine{},
		
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
	}

	// 执行迁移
	logger.Info("开始数据库迁移...")
	
	for _, model := range migrationModels {
		if err := DB.AutoMigrate(model); err != nil {
			logger.Error("迁移失败", 
				zap.String("model", fmt.Sprintf("%T", model)),
				zap.Error(err),
			)
			return err
		}
		logger.Debug("迁移成功", zap.String("model", fmt.Sprintf("%T", model)))
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