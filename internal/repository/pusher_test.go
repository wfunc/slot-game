package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// PusherRepositoryTestSuite 推币机仓储测试套件
type PusherRepositoryTestSuite struct {
	suite.Suite
	db            *gorm.DB
	machineRepo   PusherMachineRepository
	sessionRepo   PusherSessionRepository
	coinDropRepo  CoinDropRepository
}

// SetupSuite 设置测试套件
func (suite *PusherRepositoryTestSuite) SetupSuite() {
	suite.db = SetupTestDB()
	suite.machineRepo = NewPusherMachineRepository(suite.db)
	suite.sessionRepo = NewPusherSessionRepository(suite.db)
	suite.coinDropRepo = NewCoinDropRepository(suite.db)
}

// TearDownSuite 清理测试套件
func (suite *PusherRepositoryTestSuite) TearDownSuite() {
	CleanupTestDB(suite.db)
}

// SetupTest 每个测试前执行
func (suite *PusherRepositoryTestSuite) SetupTest() {
	// 清理数据
	suite.db.Exec("DELETE FROM coin_drops")
	suite.db.Exec("DELETE FROM pusher_sessions")
	suite.db.Exec("DELETE FROM pusher_machines")
	suite.db.Exec("DELETE FROM games")
	suite.db.Exec("DELETE FROM users")
}

// TestPusherMachineRepository_Create 测试创建推币机
func (suite *PusherRepositoryTestSuite) TestPusherMachineRepository_Create() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:        "Super Pusher",
		Type:        "pusher",
		Description: "超级推币机",
		Status:      "active",
		MinBet:      1,
		MaxBet:      100,
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:        game.ID,
		MachineID:     "PUSHER001",
		Name:          "推币机1号",
		PlatformWidth: 100,
		PlatformDepth: 50,
		PusherForce:   10,
		PusherSpeed:   1.5,
		CoinValue:     0.1,
		SpecialItems:  models.JSONMap{"bonus": []string{"2x", "3x"}},
		CurrentCoins:  100,
		Status:        "active",
	}
	
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), machine.ID)
}

// TestPusherMachineRepository_FindByID 测试根据ID查找推币机
func (suite *PusherRepositoryTestSuite) TestPusherMachineRepository_FindByID() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER002",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 查找推币机
	found, err := suite.machineRepo.FindByID(ctx, machine.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), found)
	assert.Equal(suite.T(), machine.MachineID, found.MachineID)
	
	// 查找不存在的推币机
	notFound, err := suite.machineRepo.FindByID(ctx, 999999)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), notFound)
}

// TestPusherMachineRepository_Update 测试更新推币机
func (suite *PusherRepositoryTestSuite) TestPusherMachineRepository_Update() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:       game.ID,
		MachineID:    "PUSHER003",
		Name:         "更新测试",
		CurrentCoins: 100,
		Status:       "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 更新推币机
	machine.CurrentCoins = 200
	machine.Status = "maintenance"
	err = suite.machineRepo.Update(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 验证更新
	updated, err := suite.machineRepo.FindByID(ctx, machine.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, updated.CurrentCoins)
	assert.Equal(suite.T(), "maintenance", updated.Status)
}

// TestPusherSessionRepository_Create 测试创建推币机会话
func (suite *PusherRepositoryTestSuite) TestPusherSessionRepository_Create() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "pusher_user",
		Email:    "pusher@example.com",
		Phone:    "13900139000",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER004",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	gameSession := &models.GameSession{
		UserID:    user.ID,
		GameID:    game.ID,
		SessionID: "SESSION001",
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(gameSession).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机会话
	session := &models.PusherSession{
		SessionID:     gameSession.ID,
		MachineID:     machine.ID,
		CoinsInserted: 10,
		CoinsWon:      5,
		ItemsWon:      models.JSONMap{"bonus": 2},
		PushCount:     20,
		StartState:    models.JSONMap{"coins": 100},
		EndState:      models.JSONMap{"coins": 105},
	}
	
	err = suite.sessionRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), session.ID)
}

// TestPusherSessionRepository_FindBySessionID 测试根据会话ID查找
func (suite *PusherRepositoryTestSuite) TestPusherSessionRepository_FindBySessionID() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "pusher_user2",
		Email:    "pusher2@example.com",
		Phone:    "13900139001",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER005",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	gameSession := &models.GameSession{
		UserID:    user.ID,
		GameID:    game.ID,
		SessionID: "SESSION002",
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(gameSession).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机会话
	session := &models.PusherSession{
		SessionID:     gameSession.ID,
		MachineID:     machine.ID,
		CoinsInserted: 20,
		CoinsWon:      15,
	}
	err = suite.sessionRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	
	// 查找会话
	found, err := suite.sessionRepo.FindBySessionID(ctx, gameSession.SessionID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), found)
	assert.Equal(suite.T(), 20, found.CoinsInserted)
	assert.Equal(suite.T(), 15, found.CoinsWon)
}

// TestCoinDropRepository_Create 测试创建掉币记录
func (suite *PusherRepositoryTestSuite) TestCoinDropRepository_Create() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "pusher_user3",
		Email:    "pusher3@example.com",
		Phone:    "13900139002",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER006",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	gameSession := &models.GameSession{
		UserID:    user.ID,
		GameID:    game.ID,
		SessionID: "SESSION003",
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(gameSession).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机会话
	pusherSession := &models.PusherSession{
		SessionID: gameSession.ID,
		MachineID: machine.ID,
	}
	err = suite.sessionRepo.Create(ctx, pusherSession)
	assert.NoError(suite.T(), err)
	
	// 创建掉币记录
	coinDrop := &models.CoinDrop{
		SessionID: pusherSession.ID,
		DropTime:  time.Now(),
		CoinCount: 5,
		ItemType:  "coin",
		ItemValue: 50,
		Position:  models.JSONMap{"x": 10, "y": 20},
	}
	
	err = suite.coinDropRepo.Create(ctx, coinDrop)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), coinDrop.ID)
}

// TestCoinDropRepository_BatchCreate 测试批量创建掉币记录
func (suite *PusherRepositoryTestSuite) TestCoinDropRepository_BatchCreate() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "pusher_user4",
		Email:    "pusher4@example.com",
		Phone:    "13900139003",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER007",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	gameSession := &models.GameSession{
		UserID:    user.ID,
		GameID:    game.ID,
		SessionID: "SESSION004",
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(gameSession).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机会话
	pusherSession := &models.PusherSession{
		SessionID: gameSession.ID,
		MachineID: machine.ID,
	}
	err = suite.sessionRepo.Create(ctx, pusherSession)
	assert.NoError(suite.T(), err)
	
	// 批量创建掉币记录
	var drops []*models.CoinDrop
	for i := 0; i < 5; i++ {
		drops = append(drops, &models.CoinDrop{
			SessionID: pusherSession.ID,
			DropTime:  time.Now().Add(time.Duration(i) * time.Second),
			CoinCount: i + 1,
			ItemType:  "coin",
			ItemValue: int64((i + 1) * 10),
			Position:  models.JSONMap{"x": i * 10, "y": i * 20},
		})
	}
	
	err = suite.coinDropRepo.BatchCreate(ctx, drops)
	assert.NoError(suite.T(), err)
	
	// 验证创建结果
	found, err := suite.coinDropRepo.FindBySessionID(ctx, pusherSession.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 5)
}

// TestCoinDropRepository_GetStatistics 测试获取掉币统计
func (suite *PusherRepositoryTestSuite) TestCoinDropRepository_GetStatistics() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "pusher_user5",
		Email:    "pusher5@example.com",
		Phone:    "13900139004",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "Test Pusher",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机
	machine := &models.PusherMachine{
		GameID:    game.ID,
		MachineID: "PUSHER008",
		Name:      "测试推币机",
		Status:    "active",
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	gameSession := &models.GameSession{
		UserID:    user.ID,
		GameID:    game.ID,
		SessionID: "SESSION005",
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(gameSession).Error
	assert.NoError(suite.T(), err)
	
	// 创建推币机会话
	pusherSession := &models.PusherSession{
		SessionID: gameSession.ID,
		MachineID: machine.ID,
	}
	err = suite.sessionRepo.Create(ctx, pusherSession)
	assert.NoError(suite.T(), err)
	
	// 创建多个掉币记录
	drops := []*models.CoinDrop{
		{
			SessionID: pusherSession.ID,
			DropTime:  time.Now(),
			CoinCount: 10,
			ItemType:  "coin",
			ItemValue: 100,
		},
		{
			SessionID: pusherSession.ID,
			DropTime:  time.Now(),
			CoinCount: 5,
			ItemType:  "bonus",
			ItemValue: 200,
		},
		{
			SessionID: pusherSession.ID,
			DropTime:  time.Now(),
			CoinCount: 3,
			ItemType:  "special",
			ItemValue: 500,
		},
	}
	
	for _, drop := range drops {
		err = suite.coinDropRepo.Create(ctx, drop)
		assert.NoError(suite.T(), err)
	}
	
	// 获取掉落记录
	foundDrops, err := suite.coinDropRepo.FindBySessionID(ctx, pusherSession.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), foundDrops, 3)
	
	// 验证掉落数据
	totalCoins := 0
	var totalValue int64
	for _, drop := range foundDrops {
		totalCoins += drop.CoinCount
		totalValue += drop.ItemValue
	}
	assert.Equal(suite.T(), 18, totalCoins)
	assert.Equal(suite.T(), int64(800), totalValue)
}

// TestRunPusherRepositoryTestSuite 运行推币机仓储测试套件
func TestRunPusherRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PusherRepositoryTestSuite))
}