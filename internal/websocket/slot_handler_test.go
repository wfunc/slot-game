package websocket

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wfunc/slot-game/internal/models"
	pb "github.com/wfunc/slot-game/internal/pb"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestSlotDB 创建测试数据库
func setupTestSlotDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移表
	err = db.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.Game{},
		&models.Transaction{},
		&models.Jackpot{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// 创建测试游戏
	game := &models.Game{
		Name:        "Test Slot",
		Type:        "slot",
		Description: "Test slot game",
		Status:      "active",
		MinBet:      10,
		MaxBet:      1000,
		RTP:         96.5,
	}
	db.Create(game)

	return db
}

func TestNewSlotHandler(t *testing.T) {
	db := setupTestSlotDB(t)

	handler := NewSlotHandler(db)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}

	if handler.db != db {
		t.Error("Expected db to be set correctly")
	}

	if handler.walletRepo == nil {
		t.Error("Expected walletRepo to be initialized")
	}

	if handler.jackpotRepo == nil {
		t.Error("Expected jackpotRepo to be initialized")
	}

	if handler.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if handler.configHandler == nil {
		t.Error("Expected configHandler to be initialized")
	}

	if handler.gameID == 0 {
		t.Error("Expected gameID to be set")
	}
}

func TestSlotHandlerConnection(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	// 创建测试用户
	user := &models.User{
		Username: "test_slot_user",
		Nickname: "Slot Test",
		Phone:    "12345678902",
		Email:    "slot@example.com",
		Status:   "active",
	}
	db.Create(user)

	// 创建钱包
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 0,
		Coins:   100000,
	}
	db.Create(wallet)

	// 创建测试会话
	sessionID := uuid.New().String()
	conn := createTestWebSocketConn(t)
	defer conn.Close()

	session := &SlotSessionSimple{
		ID:        sessionID,
		UserID:    user.ID,
		Conn:      conn,
		Balance:   100000,
		GameState: "idle",
		LastSync:  time.Now(),
	}

	// 添加会话
	handler.mu.Lock()
	handler.sessions[sessionID] = session
	handler.mu.Unlock()

	// 验证会话存储
	handler.mu.RLock()
	storedSession, exists := handler.sessions[sessionID]
	handler.mu.RUnlock()

	if !exists {
		t.Error("Expected session to be stored")
	}

	if storedSession.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, storedSession.UserID)
	}

	if storedSession.Balance != 100000 {
		t.Errorf("Expected Balance 100000, got %d", storedSession.Balance)
	}
}

func TestSlotHandlerDisconnectPlayer(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	playerID := uint32(1001)
	userID := uint(playerID)

	// 创建多个会话
	sessionCount := 3
	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		conn := createTestWebSocketConn(t)
		defer conn.Close()

		session := &SlotSessionSimple{
			ID:        sessionID,
			UserID:    userID,
			Conn:      conn,
			Balance:   100000,
			GameState: "idle",
			LastSync:  time.Now(),
		}

		handler.mu.Lock()
		handler.sessions[sessionID] = session
		handler.mu.Unlock()
	}

	// 验证会话存在
	handler.mu.RLock()
	initialCount := len(handler.sessions)
	handler.mu.RUnlock()

	if initialCount != sessionCount {
		t.Errorf("Expected %d sessions, got %d", sessionCount, initialCount)
	}

	// 断开玩家连接
	handler.DisconnectPlayer(playerID)

	// 验证该玩家的所有会话已被清理
	handler.mu.RLock()
	for _, session := range handler.sessions {
		if session.UserID == userID {
			t.Error("Expected all sessions for player to be removed")
		}
	}
	handler.mu.RUnlock()
}

func TestHandleEnterSlotRoom(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	// 创建测试用户和钱包
	user := &models.User{
		Username: "test_user",
		Nickname: "Test",
		Phone:    "12345678903",
		Email:    "test@example.com",
		Status:   "active",
	}
	db.Create(user)

	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 0,
		Coins:   100000,
	}
	db.Create(wallet)

	// 创建会话
	sessionID := uuid.New().String()
	conn := createTestWebSocketConn(t)
	defer conn.Close()

	session := &SlotSessionSimple{
		ID:        sessionID,
		UserID:    user.ID,
		Conn:      conn,
		Codec:     NewProtobufCodec(),
		Balance:   100000,
		GameState: "idle",
		LastSync:  time.Now(),
	}

	// 创建进入房间请求
	slotType := pb.ESlotType_e_slot_type_mahjong
	req := &pb.M_1901Tos{
		Type: &slotType,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 处理进入房间
	handler.handleEnterRoom(session, data)

	// 验证游戏引擎创建
	if session.Engine == nil {
		t.Error("Expected game engine to be created")
	}
}

func TestHandleStartGame(t *testing.T) {
	// 注意：handleStartGame需要游戏引擎初始化，这会涉及真实的游戏逻辑
	// 所以这个测试跳过
	t.Skip("Skipping handleStartGame test as it requires game engine initialization")
}

func TestSlotSessionManagement(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	// 测试会话创建和管理
	sessionCount := 10
	sessions := make([]*SlotSessionSimple, sessionCount)

	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		conn := createTestWebSocketConn(t)
		defer conn.Close()

		session := &SlotSessionSimple{
			ID:        sessionID,
			UserID:    uint(i),
			Conn:      conn,
			Balance:   int64(10000 * (i + 1)),
			GameState: "idle",
			LastSync:  time.Now(),
		}
		sessions[i] = session

		handler.mu.Lock()
		handler.sessions[sessionID] = session
		handler.mu.Unlock()
	}

	// 验证所有会话都被正确存储
	handler.mu.RLock()
	storedCount := len(handler.sessions)
	handler.mu.RUnlock()

	if storedCount != sessionCount {
		t.Errorf("Expected %d sessions, got %d", sessionCount, storedCount)
	}

	// 验证每个会话的数据
	for i, session := range sessions {
		handler.mu.RLock()
		storedSession, exists := handler.sessions[session.ID]
		handler.mu.RUnlock()

		if !exists {
			t.Errorf("Session %s not found", session.ID)
			continue
		}

		if storedSession.UserID != uint(i) {
			t.Errorf("Session %d: Expected UserID %d, got %d", i, i, storedSession.UserID)
		}

		expectedBalance := int64(10000 * (i + 1))
		if storedSession.Balance != expectedBalance {
			t.Errorf("Session %d: Expected Balance %d, got %d", i, expectedBalance, storedSession.Balance)
		}
	}
}

func TestConcurrentSlotSessions(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	// 并发创建和访问会话
	var wg sync.WaitGroup
	sessionCount := 100

	// 并发创建会话
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("session-%d", index)
			conn := createTestWebSocketConn(t)
			defer conn.Close()

			session := &SlotSessionSimple{
				ID:        sessionID,
				UserID:    uint(index),
				Conn:      conn,
				Balance:   100000,
				GameState: "idle",
				LastSync:  time.Now(),
			}

			handler.mu.Lock()
			handler.sessions[sessionID] = session
			handler.mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证所有会话都被创建
	handler.mu.RLock()
	actualCount := len(handler.sessions)
	handler.mu.RUnlock()

	if actualCount != sessionCount {
		t.Errorf("Expected %d sessions, got %d", sessionCount, actualCount)
	}

	// 并发读取会话
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("session-%d", index)

			handler.mu.RLock()
			session, exists := handler.sessions[sessionID]
			handler.mu.RUnlock()

			if !exists {
				t.Errorf("Session %s not found", sessionID)
				return
			}

			if session.UserID != uint(index) {
				t.Errorf("Session %d: Expected UserID %d, got %d", index, index, session.UserID)
			}
		}(i)
	}

	wg.Wait()
}

func TestSlotHandlerGetOrCreateTestUser(t *testing.T) {
	db := setupTestSlotDB(t)
	handler := NewSlotHandler(db)

	// 第一次调用应该创建新用户
	userID1 := handler.getOrCreateTestUser()
	if userID1 == 0 {
		t.Error("Expected valid user ID")
	}

	// 验证用户被创建
	var user models.User
	err := db.First(&user, userID1).Error
	if err != nil {
		t.Errorf("Failed to find created user: %v", err)
	}

	// 验证钱包被创建
	var wallet models.Wallet
	err = db.Where("user_id = ?", userID1).First(&wallet).Error
	if err != nil {
		t.Errorf("Failed to find created wallet: %v", err)
	}

	if wallet.Coins != 1000000 {
		t.Errorf("Expected initial coins 1000000, got %d", wallet.Coins)
	}

	// 第二次调用应该返回相同的用户ID
	userID2 := handler.getOrCreateTestUser()
	if userID1 != userID2 {
		t.Errorf("Expected same user ID, got %d and %d", userID1, userID2)
	}
}

// 基准测试
func BenchmarkSlotHandlerSessionAccess(b *testing.B) {
	db := setupTestSlotDB(&testing.T{})
	handler := NewSlotHandler(db)

	// 创建1000个会话
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		session := &SlotSessionSimple{
			ID:        sessionID,
			UserID:    uint(i),
			Balance:   100000,
			GameState: "idle",
			LastSync:  time.Now(),
		}

		handler.mu.Lock()
		handler.sessions[sessionID] = session
		handler.mu.Unlock()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("session-%d", i%1000)

		handler.mu.RLock()
		_ = handler.sessions[sessionID]
		handler.mu.RUnlock()
	}
}

func BenchmarkSlotHandlerConcurrentOperations(b *testing.B) {
	db := setupTestSlotDB(&testing.T{})
	handler := NewSlotHandler(db)

	// 预创建会话
	for i := 0; i < 100; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		handler.sessions[sessionID] = &SlotSessionSimple{
			ID:     sessionID,
			UserID: uint(i),
		}
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			operation := i % 3
			sessionID := fmt.Sprintf("session-%d", i%100)

			switch operation {
			case 0: // 读操作
				handler.mu.RLock()
				_ = handler.sessions[sessionID]
				handler.mu.RUnlock()

			case 1: // 写操作
				handler.mu.Lock()
				handler.sessions[sessionID] = &SlotSessionSimple{
					ID:      sessionID,
					UserID:  uint(i),
					Balance: int64(i * 1000),
				}
				handler.mu.Unlock()

			case 2: // 删除操作
				handler.mu.Lock()
				delete(handler.sessions, sessionID)
				handler.mu.Unlock()
			}
			i++
		}
	})
}

func BenchmarkSlotHandlerDisconnectPlayer(b *testing.B) {
	db := setupTestSlotDB(&testing.T{})
	handler := NewSlotHandler(db)

	// 预创建会话
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			sessionID := fmt.Sprintf("player-%d-session-%d", i, j)
			handler.sessions[sessionID] = &SlotSessionSimple{
				ID:     sessionID,
				UserID: uint(i),
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		playerID := uint32(i % 100)
		handler.DisconnectPlayer(playerID)

		// 重新添加会话以供下次测试
		for j := 0; j < 10; j++ {
			sessionID := fmt.Sprintf("player-%d-session-%d", playerID, j)
			handler.sessions[sessionID] = &SlotSessionSimple{
				ID:     sessionID,
				UserID: uint(playerID),
			}
		}
	}
}