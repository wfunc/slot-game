package repository

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// isCI 检查是否在CI环境中运行
func isCI() bool {
	// GitHub Actions 设置 CI=true
	// 其他CI系统也通常设置 CI 环境变量
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// SetupTestDB 为测试套件设置测试数据库
func SetupTestDB() *gorm.DB {
	// 使用内存数据库进行测试（更快，不需要文件系统，在所有环境中都能工作）
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	// 清理所有表数据（保留表结构）
	// 注意：清理顺序很重要，先清理有外键依赖的表
	tables := []interface{}{
		&models.SlotWinLine{},
		&models.SlotSpin{},
		&models.SlotMachine{},
		&models.CoinDrop{},
		&models.PusherSession{},
		&models.PusherMachine{},
		&models.GameResult{},
		&models.GameSession{},
		&models.Withdrawal{},
		&models.CoinPurchase{},
		&models.Transaction{},
		&models.Wallet{},
		&models.GameRoom{},
		&models.Game{},
		&models.UserSession{},
		&models.UserAuth{},
		&models.UserProfile{},
		&models.User{},
		&models.ErrorLog{},
		&models.SystemLog{},
		&models.SystemConfig{},
		&models.DeviceStatus{},
	}
	
	for _, table := range tables {
		db.Unscoped().Where("1 = 1").Delete(table)
	}

	// 自动迁移所有模型
	err = db.AutoMigrate(
		// 用户系统
		&models.User{},
		&models.UserProfile{},
		&models.UserAuth{},
		&models.UserSession{},

		// 游戏系统
		&models.Game{},
		&models.GameRoom{},
		&models.GameSession{},
		&models.GameResult{},

		// 交易系统
		&models.Wallet{},
		&models.Transaction{},
		&models.CoinPurchase{},
		&models.Withdrawal{},

		// 系统管理
		&models.SystemConfig{},
		&models.SystemLog{},
		&models.ErrorLog{},
		&models.DeviceStatus{},

		// Slot游戏
		&models.SlotMachine{},
		&models.SlotSpin{},
		&models.SlotWinLine{},

		// Pusher游戏
		&models.PusherMachine{},
		&models.PusherSession{},
		&models.CoinDrop{},
	)
	if err != nil {
		panic(err)
	}

	return db
}

// CleanupTestDB 清理测试数据库
func CleanupTestDB(db *gorm.DB) {
	// 关闭数据库连接
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

// TestDB 创建测试数据库
func TestDB(t *testing.T) *gorm.DB {
	// 使用实际的SQLite文件进行测试
	db, err := gorm.Open(sqlite.Open("../../data/slot-game-test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 清理所有表数据（保留表结构）
	// 注意：清理顺序很重要，先清理有外键依赖的表
	tables := []interface{}{
		&models.GameResult{},
		&models.GameSession{},
		&models.Withdrawal{},
		&models.CoinPurchase{},
		&models.Transaction{},
		&models.Wallet{},
		&models.GameRoom{},
		&models.Game{},
		&models.UserSession{},
		&models.UserAuth{},
		&models.UserProfile{},
		&models.User{},
		&models.ErrorLog{},
		&models.SystemLog{},
		&models.SystemConfig{},
		&models.DeviceStatus{},
	}
	
	for _, table := range tables {
		db.Unscoped().Where("1 = 1").Delete(table)
	}

	// 自动迁移所有模型
	err = db.AutoMigrate(
		// 用户系统
		&models.User{},
		&models.UserProfile{},
		&models.UserAuth{},
		&models.UserSession{},

		// 游戏系统
		&models.Game{},
		&models.GameRoom{},
		&models.GameSession{},
		&models.GameResult{},

		// 交易系统
		&models.Wallet{},
		&models.Transaction{},
		&models.CoinPurchase{},
		&models.Withdrawal{},

		// 系统管理
		&models.SystemConfig{},
		&models.SystemLog{},
		&models.ErrorLog{},
		&models.DeviceStatus{},
	)
	require.NoError(t, err)

	return db
}

// SeedTestData 创建测试数据
func SeedTestData(t *testing.T, db *gorm.DB) {
	// 创建测试用户
	users := []models.User{
		{
			Username: "testuser1",
			Phone:    "13800138001",
			Email:    "test1@example.com",
			Nickname: "测试用户1",
			Avatar:   "avatar1.png",
			Status:   "active",
		},
		{
			Username: "testuser2",
			Phone:    "13800138002",
			Email:    "test2@example.com",
			Nickname: "测试用户2",
			Avatar:   "avatar2.png",
			Status:   "active",
		},
	}
	err := db.Create(&users).Error
	require.NoError(t, err)

	// 创建测试游戏
	games := []models.Game{
		{
			Name:        "老虎机",
			Type:        "slot",
			Description: "经典老虎机游戏",
			Icon:        "slot.png",
			Status:      "active",
			MinBet:      1,
			MaxBet:      100,
			RTP:         96.5,
		},
		{
			Name:        "推币机",
			Type:        "pusher",
			Description: "推币机游戏",
			Icon:        "pusher.png",
			Status:      "active",
			MinBet:      5,
			MaxBet:      50,
			RTP:         95.0,
		},
	}
	err = db.Create(&games).Error
	require.NoError(t, err)

	// 创建测试游戏房间
	rooms := []models.GameRoom{
		{
			GameID:     games[0].ID,
			RoomNumber: "R001",
			Name:       "老虎机房间1",
			Type:       "normal",
			MinBet:     1,
			MaxBet:     100,
			Status:     "active",
		},
		{
			GameID:     games[1].ID,
			RoomNumber: "R002",
			Name:       "推币机房间1",
			Type:       "normal",
			MinBet:     5,
			MaxBet:     50,
			Status:     "active",
		},
	}
	err = db.Create(&rooms).Error
	require.NoError(t, err)

	// 创建测试钱包
	wallets := []models.Wallet{
		{
			UserID:        users[0].ID,
			Balance:       1000,
			TotalWithdraw: 0,
			TotalWin:      500,
			TotalBet:      300,
		},
		{
			UserID:        users[1].ID,
			Balance:       500,
			TotalWithdraw: 0,
			TotalWin:      100,
			TotalBet:      200,
		},
	}
	err = db.Create(&wallets).Error
	require.NoError(t, err)

	// 创建测试设备状态
	devices := []models.DeviceStatus{
		{
			DeviceID:   "device_001",
			DeviceName: "测试老虎机1",
			Type:       "slot_machine",
			Status:     "online",
			IP:         "192.168.1.100",
			Location:   "一楼大厅",
			Version:    "1.0.0",
			CPU:        45.5,
			Memory:     60.2,
			Disk:       30.5,
			LastPingAt: time.Now(),
		},
		{
			DeviceID:   "device_002",
			DeviceName: "测试推币机1",
			Type:       "pusher_machine",
			Status:     "offline",
			IP:         "192.168.1.101",
			Location:   "二楼VIP厅",
			Version:    "1.0.0",
			CPU:        0,
			Memory:     0,
			Disk:       0,
			LastPingAt: time.Now().Add(-10 * time.Minute),
		},
	}
	err = db.Create(&devices).Error
	require.NoError(t, err)

	// 创建测试系统配置
	configs := []models.SystemConfig{
		{
			Key:         "system_version",
			Value:       "1.0.0",
			Type:        "string",
			Group:       "system",
			Description: "系统版本",
			IsPublic:    true,
		},
		{
			Key:         "game.slot.min_bet",
			Value:       "1",
			Type:        "int",
			Group:       "game",
			Description: "老虎机最小下注",
			IsPublic:    false,
		},
		{
			Key:         "game.slot.max_bet",
			Value:       "100",
			Type:        "int",
			Group:       "game",
			Description: "老虎机最大下注",
			IsPublic:    false,
		},
		{
			Key:         "game.slot.rtp",
			Value:       "96.5",
			Type:        "float",
			Group:       "game",
			Description: "老虎机RTP",
			IsPublic:    false,
		},
		{
			Key:         "maintenance_mode",
			Value:       "false",
			Type:        "bool",
			Group:       "system",
			Description: "维护模式",
			IsPublic:    true,
		},
	}
	err = db.Create(&configs).Error
	require.NoError(t, err)
}

// AssertGameSession 验证游戏会话
func AssertGameSession(t *testing.T, expected, actual *models.GameSession) {
	assert.Equal(t, expected.SessionID, actual.SessionID)
	assert.Equal(t, expected.UserID, actual.UserID)
	assert.Equal(t, expected.GameID, actual.GameID)
	assert.Equal(t, expected.Status, actual.Status)
}

// AssertGameResult 验证游戏结果
func AssertGameResult(t *testing.T, expected, actual *models.GameResult) {
	assert.Equal(t, expected.RoundID, actual.RoundID)
	assert.Equal(t, expected.UserID, actual.UserID)
	assert.Equal(t, expected.GameID, actual.GameID)
	assert.Equal(t, expected.BetAmount, actual.BetAmount)
	assert.Equal(t, expected.WinAmount, actual.WinAmount)
}

// AssertDeviceStatus 验证设备状态
func AssertDeviceStatus(t *testing.T, expected, actual *models.DeviceStatus) {
	assert.Equal(t, expected.DeviceID, actual.DeviceID)
	assert.Equal(t, expected.DeviceName, actual.DeviceName)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Status, actual.Status)
}

// AssertSystemConfig 验证系统配置
func AssertSystemConfig(t *testing.T, expected, actual *models.SystemConfig) {
	assert.Equal(t, expected.Key, actual.Key)
	assert.Equal(t, expected.Value, actual.Value)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Group, actual.Group)
}

// CreateTestGameSession 创建测试游戏会话
func CreateTestGameSession(userID, gameID uint) *models.GameSession {
	return &models.GameSession{
		SessionID:   "test_session_" + time.Now().Format("20060102150405"),
		UserID:      userID,
		GameID:      gameID,
		RoomID:      1,
		Status:      "playing",
		TotalRounds: 0,
		TotalBet:    0,
		TotalWin:    0,
		PeakWin:     0,
		StartedAt:   time.Now(),
	}
}

// CreateTestGameResult 创建测试游戏结果
func CreateTestGameResult(sessionID, userID, gameID uint, betAmount, winAmount int64) *models.GameResult {
	multiplier := float64(0)
	if betAmount > 0 {
		multiplier = float64(winAmount) / float64(betAmount)
	}

	return &models.GameResult{
		SessionID:  sessionID,
		GameID:     gameID,
		UserID:     userID,
		RoundID:    "test_round_" + time.Now().Format("20060102150405"),
		BetAmount:  betAmount,
		WinAmount:  winAmount,
		Multiplier: multiplier,
		IsJackpot:  winAmount > betAmount*100,
		IsBonus:    winAmount > betAmount*10,
		PlayedAt:   time.Now(),
	}
}

// CreateTestDeviceStatus 创建测试设备状态
func CreateTestDeviceStatus(deviceID, deviceName, deviceType, status string) *models.DeviceStatus {
	return &models.DeviceStatus{
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Type:       deviceType,
		Status:     status,
		IP:         "192.168.1.100",
		Location:   "测试位置",
		Version:    "1.0.0",
		CPU:        50.0,
		Memory:     60.0,
		Disk:       70.0,
		LastPingAt: time.Now(),
	}
}

// CreateTestSystemConfig 创建测试系统配置
func CreateTestSystemConfig(key, value, configType, group string) *models.SystemConfig {
	return &models.SystemConfig{
		Key:         key,
		Value:       value,
		Type:        configType,
		Group:       group,
		Description: "测试配置: " + key,
		IsPublic:    false,
	}
}
