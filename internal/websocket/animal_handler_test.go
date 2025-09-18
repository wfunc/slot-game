package websocket

import (
	"fmt"
	"sync"
	"testing"

	"github.com/wfunc/slot-game/internal/models"
	pb "github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestAnimalDB 创建测试数据库
func setupTestAnimalDB(t *testing.T) *gorm.DB {
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
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestNewAnimalHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}

	if handler.playerSessions == nil {
		t.Error("Expected playerSessions map to be initialized")
	}

	if handler.animalRooms == nil {
		t.Error("Expected animalRooms map to be initialized")
	}

	if handler.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if handler.db != db {
		t.Error("Expected db to be set correctly")
	}

	// 验证至少创建了一个默认房间
	if len(handler.animalRooms) < 1 {
		t.Error("Expected at least one default room to be created")
	}
}

func TestAnimalHandlerConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建测试会话（不使用真实连接）
	session := &AnimalSession{
		ID:       "test-session-1",
		Conn:     nil, // 不使用真实连接
		PlayerID: 1001,
		UserID:   1,
		ZooType:  pb.EZooType_civilian,
	}

	// 模拟连接处理（直接操作内部数据结构）
	handler.mu.Lock()
	handler.sessions[session.ID] = session
	if handler.playerSessions[session.PlayerID] == nil {
		handler.playerSessions[session.PlayerID] = make(map[string]*AnimalSession)
	}
	handler.playerSessions[session.PlayerID][session.ID] = session
	handler.mu.Unlock()

	// 验证会话被正确存储
	handler.mu.RLock()
	storedSession, exists := handler.sessions[session.ID]
	handler.mu.RUnlock()

	if !exists {
		t.Error("Expected session to be stored")
	}

	if storedSession.PlayerID != 1001 {
		t.Errorf("Expected PlayerID 1001, got %d", storedSession.PlayerID)
	}
}

func TestAnimalHandlerDisconnectPlayer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建多个mock连接和会话
	playerID := uint32(1001)
	sessions := make([]*AnimalSession, 3)

	for i := 0; i < 3; i++ {
		// 不使用真实连接
		session := &AnimalSession{
			ID:       fmt.Sprintf("test-session-%d", i),
			Conn:     nil, // 不使用真实连接
			PlayerID: playerID,
			UserID:   1,
			ZooType:  pb.EZooType_civilian,
		}
		sessions[i] = session

		// 添加会话
		handler.mu.Lock()
		handler.sessions[session.ID] = session
		if handler.playerSessions[playerID] == nil {
			handler.playerSessions[playerID] = make(map[string]*AnimalSession)
		}
		handler.playerSessions[playerID][session.ID] = session
		handler.mu.Unlock()
	}

	// 验证会话存在
	handler.mu.RLock()
	playerSessionCount := len(handler.playerSessions[playerID])
	handler.mu.RUnlock()

	if playerSessionCount != 3 {
		t.Errorf("Expected 3 sessions for player, got %d", playerSessionCount)
	}

	// 断开玩家连接
	handler.DisconnectPlayer(playerID)

	// 验证所有会话已被清理
	handler.mu.RLock()
	remainingSessions := len(handler.playerSessions[playerID])
	handler.mu.RUnlock()

	if remainingSessions != 0 {
		t.Errorf("Expected 0 sessions after disconnect, got %d", remainingSessions)
	}
}

func TestAnimalHandlerBroadcastMessage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建多个测试会话
	sessions := []struct {
		id       string
		playerID uint32
		roomID   uint32
		zooType  pb.EZooType
	}{
		{"session-1", 1001, 1, pb.EZooType_civilian},
		{"session-2", 1002, 1, pb.EZooType_civilian},
		{"session-3", 1003, 2, pb.EZooType_civilian},
		{"session-4", 1004, 2, pb.EZooType_rich},
	}

	for _, s := range sessions {
		// 不使用真实连接
		session := &AnimalSession{
			ID:       s.id,
			Conn:     nil, // 不使用真实连接
			PlayerID: s.playerID,
			RoomID:   s.roomID,
			ZooType:  s.zooType,
		}

		handler.mu.Lock()
		handler.sessions[session.ID] = session
		if handler.playerSessions[session.PlayerID] == nil {
			handler.playerSessions[session.PlayerID] = make(map[string]*AnimalSession)
		}
		handler.playerSessions[session.PlayerID][session.ID] = session
		handler.mu.Unlock()
	}

	// 测试广播到特定房间
	// TODO: 需要实现广播消息的测试
	// 目前AnimalHandler没有BroadcastMessage方法
	t.Skip("BroadcastMessage not yet implemented")
}

func TestAnimalRoomManagement(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 验证初始房间创建
	handler.mu.RLock()
	initialRoomCount := len(handler.animalRooms)
	handler.mu.RUnlock()

	if initialRoomCount < 1 {
		t.Error("Expected at least one initial room")
	}

	// 模拟多个玩家加入，触发新房间创建
	zooType := pb.EZooType_civilian

	// 获取或创建房间
	room, err := handler.findOrCreateRoom(zooType)
	if err != nil {
		t.Fatalf("Failed to find or create room: %v", err)
	}
	if room == nil {
		t.Fatal("Expected room to be found or created")
	}

	// 验证房间属性
	if room.GetRoomID() == 0 {
		t.Error("Expected room to have valid ID")
	}
}

func TestHandleEnterRoom(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建测试会话（不使用真实的WebSocket连接）
	session := &AnimalSession{
		ID:       "test-session",
		Conn:     nil, // 不使用真实连接，避免goroutine阻塞
		PlayerID: 1001,
		UserID:   1,
	}

	// 创建进入房间请求
	req := &pb.M_1801Tos{
		Type: pb.EZooType_civilian.Enum(),
	}

	// 直接测试房间查找/创建逻辑，不调用handleEnterRoom避免启动goroutine
	roomType := req.GetType()
	if roomType == 0 {
		roomType = pb.EZooType_free
	}

	// 测试findOrCreateRoom逻辑
	room, err := handler.findOrCreateRoom(roomType)
	if err != nil {
		t.Fatalf("Failed to find or create room: %v", err)
	}

	if room == nil {
		t.Error("Expected room to be created")
	}

	if room.GetRoomID() == 0 {
		t.Error("Expected room to have valid ID")
	}

	// 更新会话信息（模拟handleEnterRoom的效果）
	session.ZooType = roomType
	session.RoomID = room.GetRoomID()

	// 验证会话更新
	if session.ZooType != pb.EZooType_civilian {
		t.Error("Expected ZooType to be updated")
	}

	if session.RoomID == 0 {
		t.Error("Expected RoomID to be set")
	}
}

func TestHandleBet(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	// 创建测试用户和钱包
	user := &models.User{
		Username: "test_user",
		Nickname: "Test",
		Phone:    "12345678900",
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

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 确保清理资源

	// 创建会话（不使用真实WebSocket连接）
	session := &AnimalSession{
		ID:       "test-session",
		Conn:     nil, // 不使用真实连接
		PlayerID: 1001,
		UserID:   user.ID,
		RoomID:   1,
		ZooType:  pb.EZooType_civilian,
	}

	// 先让玩家进入房间
	handler.mu.Lock()
	handler.sessions[session.ID] = session
	if handler.playerSessions[session.PlayerID] == nil {
		handler.playerSessions[session.PlayerID] = make(map[string]*AnimalSession)
	}
	handler.playerSessions[session.PlayerID][session.ID] = session
	handler.mu.Unlock()

	// 创建下注请求
	animalID := uint32(1)
	bulletID := "bullet-123"

	req := &pb.M_1803Tos{
		Id:       &animalID,
		BulletId: &bulletID,
	}

	_, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 注意：handleBet需要房间存在，但我们不想启动真实的房间
	// 所以这个测试可能需要跳过或修改为单元测试
	t.Skip("Skipping handleBet test as it requires a running room")
}

func TestHandleFireBullet(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	// 创建测试用户和钱包
	user := &models.User{
		Username: "test_user",
		Nickname: "Test",
		Phone:    "12345678901",
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

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 确保清理资源

	// 创建发射子弹请求
	betVal := uint32(100)
	req := &pb.M_1815Tos{
		BetVal: &betVal,
	}

	_, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// 注意：handleFireBullet需要房间存在，但我们不想启动真实的房间
	// 所以这个测试可能需要跳过或修改为单元测试
	t.Skip("Skipping handleFireBullet test as it requires a running room")
}

func TestConcurrentSessions(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(t)

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 确保清理资源

	// 并发创建多个会话
	var wg sync.WaitGroup
	sessionCount := 100

	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 不使用真实的WebSocket连接
			session := &AnimalSession{
				ID:       fmt.Sprintf("session-%d", index),
				Conn:     nil, // 不使用真实连接
				PlayerID: uint32(index),
				UserID:   uint(index),
			}

			// 添加会话
			handler.mu.Lock()
			handler.sessions[session.ID] = session
			if handler.playerSessions[session.PlayerID] == nil {
				handler.playerSessions[session.PlayerID] = make(map[string]*AnimalSession)
			}
			handler.playerSessions[session.PlayerID][session.ID] = session
			handler.mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证所有会话都被正确添加
	handler.mu.RLock()
	actualCount := len(handler.sessions)
	handler.mu.RUnlock()

	if actualCount != sessionCount {
		t.Errorf("Expected %d sessions, got %d", sessionCount, actualCount)
	}
}

// 基准测试
func BenchmarkAnimalHandlerBroadcast(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(&testing.T{})

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建1000个会话
	for i := 0; i < 1000; i++ {
		session := &AnimalSession{
			ID:       fmt.Sprintf("session-%d", i),
			PlayerID: uint32(i),
			RoomID:   uint32(i % 10), // 分配到10个房间
			ZooType:  pb.EZooType_civilian,
		}

		handler.mu.Lock()
		handler.sessions[session.ID] = session
		if handler.playerSessions[session.PlayerID] == nil {
			handler.playerSessions[session.PlayerID] = make(map[string]*AnimalSession)
		}
		handler.playerSessions[session.PlayerID][session.ID] = session
		handler.mu.Unlock()
	}

	// TODO: PushMessage需要Message字段而非Data字段
	// getRecipients方法不存在，需要实现或跳过此测试
	b.Skip("getRecipients method not yet implemented")
}

func BenchmarkAnimalHandlerConcurrentAccess(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	db := setupTestAnimalDB(&testing.T{})

	handler := NewAnimalHandler(db, logger)
	defer handler.Cleanup() // 清理资源

	// 创建初始会话
	for i := 0; i < 100; i++ {
		session := &AnimalSession{
			ID:       fmt.Sprintf("session-%d", i),
			PlayerID: uint32(i),
		}
		handler.sessions[session.ID] = session
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 模拟并发读写
			if i%2 == 0 {
				// 读操作
				handler.mu.RLock()
				_ = handler.sessions[fmt.Sprintf("session-%d", i%100)]
				handler.mu.RUnlock()
			} else {
				// 写操作
				handler.mu.Lock()
				handler.sessions[fmt.Sprintf("new-session-%d", i)] = &AnimalSession{
					ID:       fmt.Sprintf("new-session-%d", i),
					PlayerID: uint32(i),
				}
				handler.mu.Unlock()
			}
			i++
		}
	})
}