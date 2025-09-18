package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pb "github.com/wfunc/slot-game/internal/pb"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockWebSocketConn 模拟WebSocket连接
type mockWebSocketConn struct {
	*websocket.Conn
	messages [][]byte
	closed   bool
}

func (m *mockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.messages = append(m.messages, data)
	return nil
}

func (m *mockWebSocketConn) Close() error {
	m.closed = true
	return nil
}

// 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	return db
}

// 创建测试WebSocket连接
func createTestWebSocketConn(t *testing.T) *websocket.Conn {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// 保持连接打开
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// 连接到测试服务器
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}

	return conn
}

func TestNewBridgeHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}

	if handler.gameHandlers == nil {
		t.Error("Expected gameHandlers map to be initialized")
	}

	if handler.logger != logger {
		t.Error("Expected logger to be set correctly")
	}

	if handler.db != db {
		t.Error("Expected db to be set correctly")
	}
}

func TestRegisterGameHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建mock游戏handler
	animalHandler := &AnimalHandler{}
	slotHandler := &SlotHandler{}

	// 注册handler
	handler.RegisterGameHandler("animal", animalHandler)
	handler.RegisterGameHandler("slot", slotHandler)

	// 验证注册
	if _, ok := handler.gameHandlers["animal"]; !ok {
		t.Error("Expected animal handler to be registered")
	}

	if _, ok := handler.gameHandlers["slot"]; !ok {
		t.Error("Expected slot handler to be registered")
	}

	if handler.animalHandler != animalHandler {
		t.Error("Expected animalHandler to be set")
	}

	if handler.slotHandler != slotHandler {
		t.Error("Expected slotHandler to be set")
	}
}

func TestHandleConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建测试连接
	conn := createTestWebSocketConn(t)
	defer conn.Close()

	playerID := uint32(12345)

	// 处理连接
	handler.HandleConnection(conn, playerID)

	// 验证会话创建
	session, exists := handler.GetPlayerSession(playerID)
	if !exists {
		t.Error("Expected session to be created")
	}

	if session.PlayerID != playerID {
		t.Errorf("Expected PlayerID %d, got %d", playerID, session.PlayerID)
	}

	if session.Conn != conn {
		t.Error("Expected connection to be set correctly")
	}
}

func TestDisconnectPlayer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建测试连接
	conn := createTestWebSocketConn(t)
	defer conn.Close()

	playerID := uint32(12345)

	// 添加连接
	handler.HandleConnection(conn, playerID)

	// 验证连接存在
	if _, exists := handler.GetPlayerSession(playerID); !exists {
		t.Error("Expected session to exist before disconnect")
	}

	// 断开连接
	handler.DisconnectPlayer(playerID)

	// 验证连接已移除
	if _, exists := handler.GetPlayerSession(playerID); exists {
		t.Error("Expected session to be removed after disconnect")
	}
}

func TestTriggerGameSwitch(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建mock连接
	mockConn := &mockWebSocketConn{
		messages: make([][]byte, 0),
	}

	playerID := uint32(12345)

	// 手动添加会话
	session := &BridgeSession{
		Conn:        mockConn.Conn,
		PlayerID:    playerID,
		CurrentGame: "slot",
		LastActive:  time.Now(),
	}
	handler.sessions[playerID] = session

	// 创建桥接数据
	freeRounds := uint32(10)
	multiplier := float32(2.0)
	bonusPool := uint64(100000)
	triggerType := "bonus"
	triggerPos := []uint32{1, 2, 3}

	bridgeData := &pb.PBridgeData{
		FreeRounds:  &freeRounds,
		Multiplier:  &multiplier,
		BonusPool:   &bonusPool,
		TriggerType: &triggerType,
		TriggerPos:  triggerPos,
	}

	// 触发游戏切换
	err := handler.TriggerGameSwitch(playerID, "slot", "animal",
		pb.ESwitchType_switch_after_round, bridgeData)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 验证会话状态更新
	if session.CurrentGame != "animal" {
		t.Errorf("Expected CurrentGame to be 'animal', got '%s'", session.CurrentGame)
	}

	if session.PreviousGame != "slot" {
		t.Errorf("Expected PreviousGame to be 'slot', got '%s'", session.PreviousGame)
	}
}

func TestNotifyGameReturn(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建mock连接
	mockConn := &mockWebSocketConn{
		messages: make([][]byte, 0),
	}

	playerID := uint32(12345)

	// 手动添加会话
	session := &BridgeSession{
		Conn:         mockConn.Conn,
		PlayerID:     playerID,
		CurrentGame:  "animal",
		PreviousGame: "slot",
		LastActive:   time.Now(),
	}
	handler.sessions[playerID] = session

	// 创建返回结果
	totalWin := uint64(50000)
	achievements := uint32(3)
	extraData := `{"level": 5, "bonus": true}`

	result := &pb.PBridgeResult{
		TotalWin:     &totalWin,
		Achievements: &achievements,
		ExtraData:    &extraData,
	}

	// 通知游戏返回
	err := handler.NotifyGameReturn(playerID, "animal", 50000, result)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 验证会话状态
	if session.CurrentGame != "slot" {
		t.Errorf("Expected CurrentGame to be 'slot', got '%s'", session.CurrentGame)
	}

	if session.PreviousGame != "animal" {
		t.Errorf("Expected PreviousGame to be 'animal', got '%s'", session.PreviousGame)
	}
}

func TestNotifyAnimalTrigger(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 创建mock连接
	mockConn := &mockWebSocketConn{
		messages: make([][]byte, 0),
	}

	playerID := uint32(12345)

	// 手动添加会话
	session := &BridgeSession{
		Conn:        mockConn.Conn,
		PlayerID:    playerID,
		CurrentGame: "slot",
		LastActive:  time.Now(),
	}
	handler.sessions[playerID] = session

	// 创建桥接数据
	freeRounds := uint32(5)
	multiplier := float32(1.5)

	bridgeData := &pb.PBridgeData{
		FreeRounds: &freeRounds,
		Multiplier: &multiplier,
	}

	// 通知Animal触发
	err := handler.NotifyAnimalTrigger(playerID, true, "scatter_bonus", bridgeData)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetPlayersInGame(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 添加多个会话
	sessions := []struct {
		playerID uint32
		game     string
	}{
		{1001, "slot"},
		{1002, "animal"},
		{1003, "slot"},
		{1004, "animal"},
		{1005, "slot"},
	}

	for _, s := range sessions {
		handler.sessions[s.playerID] = &BridgeSession{
			PlayerID:    s.playerID,
			CurrentGame: s.game,
		}
	}

	// 获取slot游戏的玩家
	slotPlayers := handler.GetPlayersInGame("slot")
	if len(slotPlayers) != 3 {
		t.Errorf("Expected 3 players in slot, got %d", len(slotPlayers))
	}

	// 获取animal游戏的玩家
	animalPlayers := handler.GetPlayersInGame("animal")
	if len(animalPlayers) != 2 {
		t.Errorf("Expected 2 players in animal, got %d", len(animalPlayers))
	}
}

func TestUpdateBridgeData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	playerID := uint32(12345)

	// 添加会话
	session := &BridgeSession{
		PlayerID:   playerID,
		BridgeData: nil,
	}
	handler.sessions[playerID] = session

	// 更新桥接数据
	freeRounds := uint32(15)
	multiplier := float32(3.0)

	newData := &pb.PBridgeData{
		FreeRounds: &freeRounds,
		Multiplier: &multiplier,
	}

	err := handler.UpdateBridgeData(playerID, newData)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// 验证数据更新
	if session.BridgeData == nil {
		t.Error("Expected BridgeData to be updated")
	}

	if *session.BridgeData.FreeRounds != 15 {
		t.Errorf("Expected FreeRounds to be 15, got %d", *session.BridgeData.FreeRounds)
	}

	if *session.BridgeData.Multiplier != 3.0 {
		t.Errorf("Expected Multiplier to be 3.0, got %f", *session.BridgeData.Multiplier)
	}
}

func TestGetStatistics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(t)

	handler := NewBridgeHandler(logger, db)

	// 注册游戏handler
	handler.RegisterGameHandler("slot", &SlotHandler{})
	handler.RegisterGameHandler("animal", &AnimalHandler{})

	// 添加测试会话
	handler.sessions[1001] = &BridgeSession{PlayerID: 1001, CurrentGame: "slot"}
	handler.sessions[1002] = &BridgeSession{PlayerID: 1002, CurrentGame: "animal"}
	handler.sessions[1003] = &BridgeSession{PlayerID: 1003, CurrentGame: "slot"}
	handler.sessions[1004] = &BridgeSession{PlayerID: 1004, CurrentGame: ""}

	// 获取统计信息
	stats := handler.GetStatistics()

	// 验证总会话数
	if totalSessions, ok := stats["total_sessions"].(int); !ok || totalSessions != 4 {
		t.Errorf("Expected total_sessions to be 4, got %v", stats["total_sessions"])
	}

	// 验证handler数量
	if handlers, ok := stats["handlers"].(int); !ok || handlers != 2 {
		t.Errorf("Expected handlers to be 2, got %v", stats["handlers"])
	}

	// 验证游戏分布
	if games, ok := stats["games"].(map[string]int); ok {
		if games["slot"] != 2 {
			t.Errorf("Expected 2 players in slot, got %d", games["slot"])
		}
		if games["animal"] != 1 {
			t.Errorf("Expected 1 player in animal, got %d", games["animal"])
		}
	} else {
		t.Error("Expected games statistics to be present")
	}
}

// 基准测试
func BenchmarkTriggerGameSwitch(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(&testing.T{})

	handler := NewBridgeHandler(logger, db)

	// 创建测试会话
	for i := 0; i < 100; i++ {
		playerID := uint32(i)
		handler.sessions[playerID] = &BridgeSession{
			PlayerID:    playerID,
			CurrentGame: "slot",
		}
	}

	bridgeData := CreateBridgeData(10, 2.0, 100000, "bonus", []uint32{1, 2, 3})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		playerID := uint32(i % 100)
		handler.TriggerGameSwitch(playerID, "slot", "animal",
			pb.ESwitchType_switch_immediate, bridgeData)
	}
}

func BenchmarkGetPlayersInGame(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	db := setupTestDB(&testing.T{})

	handler := NewBridgeHandler(logger, db)

	// 创建大量会话
	for i := 0; i < 1000; i++ {
		game := "slot"
		if i%2 == 0 {
			game = "animal"
		}
		handler.sessions[uint32(i)] = &BridgeSession{
			PlayerID:    uint32(i),
			CurrentGame: game,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.GetPlayersInGame("slot")
	}
}