package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// SlotRepositoryTestSuite Slot游戏仓储测试套件
type SlotRepositoryTestSuite struct {
	suite.Suite
	db               *gorm.DB
	machineRepo      SlotMachineRepository
	spinRepo         SlotSpinRepository
	winLineRepo      SlotWinLineRepository
}

// SetupSuite 设置测试套件
func (suite *SlotRepositoryTestSuite) SetupSuite() {
	suite.db = SetupTestDB()
	suite.machineRepo = NewSlotMachineRepository(suite.db)
	suite.spinRepo = NewSlotSpinRepository(suite.db)
	suite.winLineRepo = NewSlotWinLineRepository(suite.db)
}

// TearDownSuite 清理测试套件
func (suite *SlotRepositoryTestSuite) TearDownSuite() {
	CleanupTestDB(suite.db)
}

// SetupTest 每个测试前执行
func (suite *SlotRepositoryTestSuite) SetupTest() {
	// 清理表数据
	suite.db.Exec("DELETE FROM slot_win_lines")
	suite.db.Exec("DELETE FROM slot_spins")
	suite.db.Exec("DELETE FROM slot_machines")
	suite.db.Exec("DELETE FROM game_sessions")
	suite.db.Exec("DELETE FROM games")
	suite.db.Exec("DELETE FROM users")
}

// TestSlotMachineRepository_Create 测试创建老虎机
func (suite *SlotRepositoryTestSuite) TestSlotMachineRepository_Create() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_001",
		Name:        "黄金老虎机",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		MinBet:      1,
		MaxBet:      100,
		RTP:         96.5,
		Volatility:  "medium",
		Status:      "active",
		Symbols:     models.JSONMap{"wild": "W", "scatter": "S"},
		PayTable:    models.JSONMap{"W": 1000, "S": 500},
		BonusConfig: models.JSONMap{"freeSpins": 10},
	}
	
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), machine.ID)
}

// TestSlotMachineRepository_FindByMachineID 测试按机器ID查找
func (suite *SlotRepositoryTestSuite) TestSlotMachineRepository_FindByMachineID() {
	ctx := context.Background()
	
	// 创建游戏和机器
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_002",
		Name:        "水果老虎机",
		Type:        "fruit",
		Reels:       3,
		Rows:        3,
		Paylines:    5,
		MinBet:      1,
		MaxBet:      50,
		RTP:         95.0,
		Volatility:  "low",
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 查找机器
	found, err := suite.machineRepo.FindByMachineID(ctx, "SLOT_002")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "SLOT_002", found.MachineID)
	assert.Equal(suite.T(), "水果老虎机", found.Name)
}

// TestSlotMachineRepository_FindByGameID 测试按游戏ID查找
func (suite *SlotRepositoryTestSuite) TestSlotMachineRepository_FindByGameID() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机游戏",
		Type:   "slot",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建多个机器
	machines := []*models.SlotMachine{
		{
			GameID:      game.ID,
			MachineID:   "SLOT_003",
			Name:        "机器1",
			Type:        "classic",
			Reels:       5,
			Rows:        3,
			Paylines:    20,
			Status:      "active",
			Symbols:     models.JSONMap{},
			PayTable:    models.JSONMap{},
			BonusConfig: models.JSONMap{},
		},
		{
			GameID:      game.ID,
			MachineID:   "SLOT_004",
			Name:        "机器2",
			Type:        "video",
			Reels:       5,
			Rows:        4,
			Paylines:    40,
			Status:      "active",
			Symbols:     models.JSONMap{},
			PayTable:    models.JSONMap{},
			BonusConfig: models.JSONMap{},
		},
	}
	
	for _, m := range machines {
		err = suite.machineRepo.Create(ctx, m)
		assert.NoError(suite.T(), err)
	}
	
	// 查找游戏的所有机器
	found, err := suite.machineRepo.FindByGameID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 2)
}

// TestSlotMachineRepository_UpdateStatus 测试更新机器状态
func (suite *SlotRepositoryTestSuite) TestSlotMachineRepository_UpdateStatus() {
	ctx := context.Background()
	
	// 创建游戏和机器
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err := suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_005",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 更新状态
	err = suite.machineRepo.UpdateStatus(ctx, machine.ID, "maintenance")
	assert.NoError(suite.T(), err)
	
	// 验证更新
	found, err := suite.machineRepo.FindByID(ctx, machine.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "maintenance", found.Status)
}

// TestSlotSpinRepository_Create 测试创建旋转记录
func (suite *SlotRepositoryTestSuite) TestSlotSpinRepository_Create() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Phone:    "13800138000",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	session := &models.GameSession{
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixNano()),
		UserID:    user.ID,
		GameID:    game.ID,
		RoomID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(session).Error
	assert.NoError(suite.T(), err)
	
	// 创建机器
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_006",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建旋转记录
	spin := &models.SlotSpin{
		SessionID:        session.ID,
		MachineID:        machine.ID,
		UserID:           user.ID,
		SpinID:           fmt.Sprintf("spin_%d", time.Now().UnixNano()),
		BetAmount:        10,
		BetLines:         20,
		TotalBet:         200,
		ReelPositions:    models.JSONMap{"reel1": []int{1, 2, 3}},
		WinAmount:        500,
		WinLines:         models.JSONMap{"line1": 100, "line2": 400},
		Multiplier:       2.5,
		IsBonus:          false,
		IsFreeSpini:      false,
		FreeSpinsAwarded: 0,
		BonusAwarded:     0,
		SpunAt:           time.Now(),
	}
	
	err = suite.spinRepo.Create(ctx, spin)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), spin.ID)
}

// TestSlotSpinRepository_FindBySessionID 测试按会话ID查找旋转记录
func (suite *SlotRepositoryTestSuite) TestSlotSpinRepository_FindBySessionID() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "testuser2",
		Email:    "test2@example.com",
		Phone:    "13800138001",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	session := &models.GameSession{
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixNano()),
		UserID:    user.ID,
		GameID:    game.ID,
		RoomID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(session).Error
	assert.NoError(suite.T(), err)
	
	// 创建机器
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_007",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建多个旋转记录
	for i := 0; i < 3; i++ {
		spin := &models.SlotSpin{
			SessionID:     session.ID,
			MachineID:     machine.ID,
			UserID:        user.ID,
			SpinID:        fmt.Sprintf("spin_%d_%d", i, time.Now().UnixNano()),
			BetAmount:     10,
			TotalBet:      10,
			ReelPositions: models.JSONMap{},
			WinAmount:     int64(i * 100),
			WinLines:      models.JSONMap{},
			SpunAt:        time.Now(),
		}
		err = suite.spinRepo.Create(ctx, spin)
		assert.NoError(suite.T(), err)
	}
	
	// 查找会话的旋转记录
	pagination := &Pagination{Page: 1, PageSize: 10}
	spins, err := suite.spinRepo.FindBySessionID(ctx, session.ID, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), spins, 3)
}

// TestSlotSpinRepository_GetUserStatistics 测试获取用户统计
func (suite *SlotRepositoryTestSuite) TestSlotSpinRepository_GetUserStatistics() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "testuser3",
		Email:    "test3@example.com",
		Phone:    "13800138002",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	session := &models.GameSession{
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixNano()),
		UserID:    user.ID,
		GameID:    game.ID,
		RoomID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(session).Error
	assert.NoError(suite.T(), err)
	
	// 创建机器
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_008",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建旋转记录
	spins := []*models.SlotSpin{
		{
			SessionID:        session.ID,
			MachineID:        machine.ID,
			UserID:           user.ID,
			SpinID:           fmt.Sprintf("spin_1_%d", time.Now().UnixNano()),
			BetAmount:        10,
			TotalBet:         10,
			ReelPositions:    models.JSONMap{},
			WinAmount:        50,
			WinLines:         models.JSONMap{},
			IsBonus:          true,
			FreeSpinsAwarded: 10,
			SpunAt:           time.Now(),
		},
		{
			SessionID:     session.ID,
			MachineID:     machine.ID,
			UserID:        user.ID,
			SpinID:        fmt.Sprintf("spin_2_%d", time.Now().UnixNano()),
			BetAmount:     20,
			TotalBet:      20,
			ReelPositions: models.JSONMap{},
			WinAmount:     100,
			WinLines:      models.JSONMap{},
			SpunAt:        time.Now(),
		},
	}
	
	for _, s := range spins {
		err = suite.spinRepo.Create(ctx, s)
		assert.NoError(suite.T(), err)
	}
	
	// 获取用户统计
	stats, err := suite.spinRepo.GetUserStatistics(ctx, user.ID, machine.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 2, stats.TotalSpins)
	assert.Equal(suite.T(), int64(30), stats.TotalBet)
	assert.Equal(suite.T(), int64(150), stats.TotalWin)
	assert.Equal(suite.T(), 1, stats.BonusCount)
	assert.Equal(suite.T(), 10, stats.FreeSpinsCount)
}

// TestSlotWinLineRepository_Create 测试创建中奖线记录
func (suite *SlotRepositoryTestSuite) TestSlotWinLineRepository_Create() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "testuser4",
		Email:    "test4@example.com",
		Phone:    "13800138003",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	session := &models.GameSession{
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixNano()),
		UserID:    user.ID,
		GameID:    game.ID,
		RoomID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(session).Error
	assert.NoError(suite.T(), err)
	
	// 创建机器
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_009",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建旋转记录
	spin := &models.SlotSpin{
		SessionID:     session.ID,
		MachineID:     machine.ID,
		UserID:        user.ID,
		SpinID:        fmt.Sprintf("spin_%d", time.Now().UnixNano()),
		BetAmount:     10,
		TotalBet:      200,
		ReelPositions: models.JSONMap{},
		WinAmount:     500,
		WinLines:      models.JSONMap{},
		SpunAt:        time.Now(),
	}
	err = suite.spinRepo.Create(ctx, spin)
	assert.NoError(suite.T(), err)
	
	// 创建中奖线记录
	winLine := &models.SlotWinLine{
		SpinID:       spin.ID,
		LineNumber:   1,
		Symbol:       "W",
		SymbolCount:  5,
		WinAmount:    100,
		Multiplier:   1,
		Positions:    models.JSONMap{"reel1": 1, "reel2": 1, "reel3": 1, "reel4": 1, "reel5": 1},
	}
	
	err = suite.winLineRepo.Create(ctx, winLine)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), winLine.ID)
}

// TestSlotWinLineRepository_FindBySpinID 测试按旋转ID查找中奖线
func (suite *SlotRepositoryTestSuite) TestSlotWinLineRepository_FindBySpinID() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "testuser5",
		Email:    "test5@example.com",
		Phone:    "13800138004",
		Status:   "active",
	}
	err := suite.db.Create(user).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏
	game := &models.Game{
		Name:   "测试老虎机",
		Type:   "slot",
		Status: "active",
	}
	err = suite.db.Create(game).Error
	assert.NoError(suite.T(), err)
	
	// 创建游戏会话
	session := &models.GameSession{
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixNano()),
		UserID:    user.ID,
		GameID:    game.ID,
		RoomID:    1,
		Status:    "playing",
		StartedAt: time.Now(),
	}
	err = suite.db.Create(session).Error
	assert.NoError(suite.T(), err)
	
	// 创建机器
	machine := &models.SlotMachine{
		GameID:      game.ID,
		MachineID:   "SLOT_010",
		Name:        "测试机器",
		Type:        "classic",
		Reels:       5,
		Rows:        3,
		Paylines:    20,
		Status:      "active",
		Symbols:     models.JSONMap{},
		PayTable:    models.JSONMap{},
		BonusConfig: models.JSONMap{},
	}
	err = suite.machineRepo.Create(ctx, machine)
	assert.NoError(suite.T(), err)
	
	// 创建旋转记录
	spin := &models.SlotSpin{
		SessionID:     session.ID,
		MachineID:     machine.ID,
		UserID:        user.ID,
		SpinID:        fmt.Sprintf("spin_%d", time.Now().UnixNano()),
		BetAmount:     10,
		TotalBet:      200,
		ReelPositions: models.JSONMap{},
		WinAmount:     500,
		WinLines:      models.JSONMap{},
		SpunAt:        time.Now(),
	}
	err = suite.spinRepo.Create(ctx, spin)
	assert.NoError(suite.T(), err)
	
	// 创建多条中奖线
	winLines := []*models.SlotWinLine{
		{
			SpinID:       spin.ID,
			LineNumber:   1,
			Symbol:       "W",
			SymbolCount:  5,
			WinAmount:    100,
			Multiplier:   1,
			Positions:    models.JSONMap{},
		},
		{
			SpinID:       spin.ID,
			LineNumber:   2,
			Symbol:       "S",
			SymbolCount:  3,
			WinAmount:    50,
			Multiplier:   1,
			Positions:    models.JSONMap{},
		},
	}
	
	for _, wl := range winLines {
		err = suite.winLineRepo.Create(ctx, wl)
		assert.NoError(suite.T(), err)
	}
	
	// 查找旋转的中奖线
	found, err := suite.winLineRepo.FindBySpinID(ctx, spin.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), found, 2)
}

// TestSlotRepositorySuite 运行Slot游戏仓储测试套件
func TestSlotRepositorySuite(t *testing.T) {
	suite.Run(t, new(SlotRepositoryTestSuite))
}