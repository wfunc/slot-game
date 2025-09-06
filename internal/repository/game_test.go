package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// GameRepositoryTestSuite 游戏仓储测试套件
type GameRepositoryTestSuite struct {
	suite.Suite
	db           *gorm.DB
	gameRepo     GameRepository
	roomRepo     GameRoomRepository
	sessionRepo  GameSessionRepository
}

func (suite *GameRepositoryTestSuite) SetupTest() {
	suite.db = SetupTestDB()
	suite.gameRepo = NewGameRepository(suite.db)
	suite.roomRepo = NewGameRoomRepository(suite.db)
	suite.sessionRepo = NewGameSessionRepository(suite.db)
}

func (suite *GameRepositoryTestSuite) TearDownTest() {
	CleanupTestDB(suite.db)
}

// TestGameRepository_Create 测试创建游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_Create() {
	ctx := context.Background()
	
	game := &models.Game{
		Name:        "Test Slot",
		Type:        "slot",
		Description: "Test slot game",
		Status:      "active",
		MinBet:      10,
		MaxBet:      10000,
		RTP:         96.5,
	}
	
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), game.ID)
	
	// 验证数据
	found, err := suite.gameRepo.FindByID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), game.Name, found.Name)
	assert.Equal(suite.T(), game.Type, found.Type)
}

// TestGameRepository_FindByName 测试根据名称查找游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_FindByName() {
	ctx := context.Background()
	
	game := &models.Game{
		Name:   "Unique Game",
		Type:   "slot",
		Status: "active",
	}
	
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	found, err := suite.gameRepo.FindByName(ctx, "Unique Game")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), game.ID, found.ID)
	
	// 测试不存在的游戏
	_, err = suite.gameRepo.FindByName(ctx, "Not Exist")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "游戏不存在")
}

// TestGameRepository_FindByType 测试根据类型查找游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_FindByType() {
	ctx := context.Background()
	
	// 创建不同类型的游戏
	slotGame := &models.Game{
		Name:   "Slot Game 1",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, slotGame)
	assert.NoError(suite.T(), err)
	
	pusherGame := &models.Game{
		Name:   "Pusher Game 1",
		Type:   "pusher",
		Status: "active",
	}
	err = suite.gameRepo.Create(ctx, pusherGame)
	assert.NoError(suite.T(), err)
	
	// 查找slot类型游戏
	slotGames, err := suite.gameRepo.FindByType(ctx, "slot")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), slotGames, 1)
	assert.Equal(suite.T(), "slot", slotGames[0].Type)
	
	// 查找pusher类型游戏
	pusherGames, err := suite.gameRepo.FindByType(ctx, "pusher")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), pusherGames, 1)
	assert.Equal(suite.T(), "pusher", pusherGames[0].Type)
}

// TestGameRepository_Update 测试更新游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_Update() {
	ctx := context.Background()
	
	game := &models.Game{
		Name:   "Update Game",
		Type:   "slot",
		Status: "active",
		MinBet: 10,
		MaxBet: 1000,
	}
	
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 更新游戏信息
	game.Description = "Updated description"
	game.MinBet = 20
	game.MaxBet = 2000
	err = suite.gameRepo.Update(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 验证更新
	found, err := suite.gameRepo.FindByID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated description", found.Description)
	assert.Equal(suite.T(), 20, found.MinBet)
	assert.Equal(suite.T(), 2000, found.MaxBet)
}

// TestGameRepository_UpdateStatus 测试更新游戏状态
func (suite *GameRepositoryTestSuite) TestGameRepository_UpdateStatus() {
	ctx := context.Background()
	
	game := &models.Game{
		Name:   "Status Game",
		Type:   "slot",
		Status: "active",
	}
	
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 更新状态
	err = suite.gameRepo.UpdateStatus(ctx, game.ID, "maintenance")
	assert.NoError(suite.T(), err)
	
	found, err := suite.gameRepo.FindByID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "maintenance", found.Status)
}

// TestGameRepository_GetActive 测试获取活跃游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_GetActive() {
	ctx := context.Background()
	
	// 创建活跃和非活跃游戏
	activeGame := &models.Game{
		Name:   "Active Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, activeGame)
	assert.NoError(suite.T(), err)
	
	inactiveGame := &models.Game{
		Name:   "Inactive Game",
		Type:   "slot",
		Status: "inactive",
	}
	err = suite.gameRepo.Create(ctx, inactiveGame)
	assert.NoError(suite.T(), err)
	
	// 获取活跃游戏
	games, err := suite.gameRepo.GetActive(ctx)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), games, 1)
	assert.Equal(suite.T(), "active", games[0].Status)
}

// TestGameRepository_GetAll 测试获取所有游戏（分页）
func (suite *GameRepositoryTestSuite) TestGameRepository_GetAll() {
	ctx := context.Background()
	
	// 创建多个游戏
	for i := 0; i < 5; i++ {
		game := &models.Game{
			Name:   fmt.Sprintf("Game %d", i),
			Type:   "slot",
			Status: "active",
		}
		err := suite.gameRepo.Create(ctx, game)
		assert.NoError(suite.T(), err)
	}
	
	// 测试分页
	pagination := &Pagination{
		Page:     1,
		PageSize: 3,
	}
	
	games, err := suite.gameRepo.GetAll(ctx, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), games, 3)
	assert.Equal(suite.T(), int64(5), pagination.Total)
}

// TestGameRepository_Delete 测试删除游戏
func (suite *GameRepositoryTestSuite) TestGameRepository_Delete() {
	ctx := context.Background()
	
	game := &models.Game{
		Name:   "Delete Game",
		Type:   "slot",
		Status: "active",
	}
	
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	err = suite.gameRepo.Delete(ctx, game.ID)
	assert.NoError(suite.T(), err)
	
	// 删除后应该找不到
	_, err = suite.gameRepo.FindByID(ctx, game.ID)
	assert.Error(suite.T(), err)
}

// TestGameRoomRepository_Create 测试创建游戏房间
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_Create() {
	ctx := context.Background()
	
	// 先创建游戏
	game := &models.Game{
		Name:   "Room Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 创建房间
	room := &models.GameRoom{
		GameID:      game.ID,
		RoomNumber:  "ROOM001",
		Name:        "Test Room",
		Type:        "normal",
		MinBet:      10,
		MaxBet:      1000,
		MaxPlayers:  100,
		CurrentPlayers: 0,
		Status:      "active",
	}
	
	err = suite.roomRepo.Create(ctx, room)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), room.ID)
}

// TestGameRoomRepository_FindByRoomNumber 测试根据房间号查找
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_FindByRoomNumber() {
	ctx := context.Background()
	
	// 创建游戏和房间
	game := &models.Game{
		Name:   "Find Room Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	room := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "ROOM002",
		Name:       "Find Room",
		Type:       "normal",
		Status:     "active",
	}
	err = suite.roomRepo.Create(ctx, room)
	assert.NoError(suite.T(), err)
	
	found, err := suite.roomRepo.FindByRoomNumber(ctx, "ROOM002")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), room.ID, found.ID)
	
	// 测试不存在的房间
	_, err = suite.roomRepo.FindByRoomNumber(ctx, "NOTEXIST")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "房间不存在")
}

// TestGameRoomRepository_FindByGameID 测试根据游戏ID查找房间
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_FindByGameID() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "Multi Room Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 创建多个房间
	for i := 0; i < 3; i++ {
		room := &models.GameRoom{
			GameID:     game.ID,
			RoomNumber: fmt.Sprintf("ROOM%03d", i+10),
			Name:       fmt.Sprintf("Room %d", i),
			Type:       "normal",
			Status:     "active",
		}
		err = suite.roomRepo.Create(ctx, room)
		assert.NoError(suite.T(), err)
	}
	
	rooms, err := suite.roomRepo.FindByGameID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), rooms, 3)
}

// TestGameRoomRepository_UpdatePlayerCount 测试更新玩家数量
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_UpdatePlayerCount() {
	ctx := context.Background()
	
	// 创建游戏和房间
	game := &models.Game{
		Name:   "Player Count Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	room := &models.GameRoom{
		GameID:         game.ID,
		RoomNumber:     "ROOM003",
		Name:           "Player Room",
		Type:           "normal",
		CurrentPlayers: 10,
		Status:         "active",
	}
	err = suite.roomRepo.Create(ctx, room)
	assert.NoError(suite.T(), err)
	
	// 增加玩家
	err = suite.roomRepo.UpdatePlayerCount(ctx, room.ID, 5)
	assert.NoError(suite.T(), err)
	
	found, err := suite.roomRepo.FindByID(ctx, room.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 15, found.CurrentPlayers)
	
	// 减少玩家
	err = suite.roomRepo.UpdatePlayerCount(ctx, room.ID, -3)
	assert.NoError(suite.T(), err)
	
	found, err = suite.roomRepo.FindByID(ctx, room.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 12, found.CurrentPlayers)
	
	// 测试负数保护（减少到负数应该变为0）
	err = suite.roomRepo.UpdatePlayerCount(ctx, room.ID, -20)
	assert.NoError(suite.T(), err)
	
	found, err = suite.roomRepo.FindByID(ctx, room.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, found.CurrentPlayers)
}

// TestGameRoomRepository_GetRoomStatistics 测试获取房间统计
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_GetRoomStatistics() {
	ctx := context.Background()
	
	// 创建游戏和房间
	game := &models.Game{
		Name:   "Stats Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	room := &models.GameRoom{
		GameID:         game.ID,
		RoomNumber:     "ROOM004",
		Name:           "Stats Room",
		Type:           "normal",
		CurrentPlayers: 5,
		Status:         "active",
	}
	err = suite.roomRepo.Create(ctx, room)
	assert.NoError(suite.T(), err)
	
	// 创建一些游戏会话数据
	for i := 0; i < 3; i++ {
		session := &models.GameSession{
			UserID:      uint(i + 1),
			RoomID:      room.ID,
			SessionID:   fmt.Sprintf("SESS%03d", i),
			TotalBet:    int64(100 * (i + 1)),
			TotalWin:    int64(50 * (i + 1)),
			TotalRounds: i + 1,
			Duration:    3600 * (i + 1), // 1, 2, 3 hours
			Status:      "completed",
		}
		err = suite.sessionRepo.Create(ctx, session)
		assert.NoError(suite.T(), err)
	}
	
	// 获取统计
	stats, err := suite.roomRepo.GetRoomStatistics(ctx, room.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 5, stats.CurrentPlayers)
	assert.Equal(suite.T(), 3, stats.TotalPlayers)
	assert.Equal(suite.T(), int64(600), stats.TotalBet) // 100 + 200 + 300
	assert.Equal(suite.T(), int64(300), stats.TotalWin) // 50 + 100 + 150
	assert.Equal(suite.T(), 6, stats.TotalRounds) // 1 + 2 + 3
	assert.Equal(suite.T(), 7200, stats.AveragePlayTime) // (3600 + 7200 + 10800) / 3
}

// TestGameRoomRepository_GetActive 测试获取活跃房间
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_GetActive() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "Active Room Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 创建活跃和非活跃房间
	activeRoom := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "ACTIVE001",
		Name:       "Active Room",
		Type:       "normal",
		Status:     "active",
	}
	err = suite.roomRepo.Create(ctx, activeRoom)
	assert.NoError(suite.T(), err)
	
	inactiveRoom := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "INACTIVE001",
		Name:       "Inactive Room",
		Type:       "normal",
		Status:     "maintenance",
	}
	err = suite.roomRepo.Create(ctx, inactiveRoom)
	assert.NoError(suite.T(), err)
	
	// 获取活跃房间
	rooms, err := suite.roomRepo.GetActive(ctx)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), rooms, 1)
	assert.Equal(suite.T(), "active", rooms[0].Status)
}

// TestGameRoomRepository_GetByType 测试根据类型获取房间
func (suite *GameRepositoryTestSuite) TestGameRoomRepository_GetByType() {
	ctx := context.Background()
	
	// 创建游戏
	game := &models.Game{
		Name:   "Type Room Game",
		Type:   "slot",
		Status: "active",
	}
	err := suite.gameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 创建不同类型的房间
	normalRoom := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "NORMAL001",
		Name:       "Normal Room",
		Type:       "normal",
		Status:     "active",
	}
	err = suite.roomRepo.Create(ctx, normalRoom)
	assert.NoError(suite.T(), err)
	
	vipRoom := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "VIP001",
		Name:       "VIP Room",
		Type:       "vip",
		Status:     "active",
	}
	err = suite.roomRepo.Create(ctx, vipRoom)
	assert.NoError(suite.T(), err)
	
	// 获取normal类型房间
	normalRooms, err := suite.roomRepo.GetByType(ctx, "normal")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), normalRooms, 1)
	assert.Equal(suite.T(), "normal", normalRooms[0].Type)
	
	// 获取vip类型房间
	vipRooms, err := suite.roomRepo.GetByType(ctx, "vip")
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), vipRooms, 1)
	assert.Equal(suite.T(), "vip", vipRooms[0].Type)
}

// TestGameRepository_WithTx 测试事务支持
func (suite *GameRepositoryTestSuite) TestGameRepository_WithTx() {
	ctx := context.Background()
	
	// 开始事务
	tx := suite.db.Begin()
	defer tx.Rollback()
	
	txGameRepo := suite.gameRepo.WithTx(tx).(GameRepository)
	txRoomRepo := suite.roomRepo.WithTx(tx).(GameRoomRepository)
	
	// 在事务中创建游戏
	game := &models.Game{
		Name:   "TX Game",
		Type:   "slot",
		Status: "active",
	}
	err := txGameRepo.Create(ctx, game)
	assert.NoError(suite.T(), err)
	
	// 在事务中创建房间
	room := &models.GameRoom{
		GameID:     game.ID,
		RoomNumber: "TX001",
		Name:       "TX Room",
		Type:       "normal",
		Status:     "active",
	}
	err = txRoomRepo.Create(ctx, room)
	assert.NoError(suite.T(), err)
	
	// 事务内可以查到
	foundGame, err := txGameRepo.FindByID(ctx, game.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), game.Name, foundGame.Name)
	
	foundRoom, err := txRoomRepo.FindByID(ctx, room.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), room.RoomNumber, foundRoom.RoomNumber)
	
	// 回滚后查不到
	tx.Rollback()
	
	_, err = suite.gameRepo.FindByID(ctx, game.ID)
	assert.Error(suite.T(), err)
	
	_, err = suite.roomRepo.FindByID(ctx, room.ID)
	assert.Error(suite.T(), err)
}

func TestGameRepositorySuite(t *testing.T) {
	suite.Run(t, new(GameRepositoryTestSuite))
}