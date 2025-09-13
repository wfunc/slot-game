package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wfunc/slot-game/internal/game/slot"
)

// GameSession 游戏会话
type GameSession struct {
	SessionID    string                 `json:"session_id"`
	PlayerID     string                 `json:"player_id"`
	Balance      int64                  `json:"balance"`
	TotalBet     int64                  `json:"total_bet"`
	TotalWin     int64                  `json:"total_win"`
	GameCount    int                    `json:"game_count"`
	WinCount     int                    `json:"win_count"`
	GoldenCount  int                    `json:"golden_count"`
	WildCount    int                    `json:"wild_count"`
	CreatedAt    time.Time              `json:"created_at"`
	LastPlayedAt time.Time              `json:"last_played_at"`
	Settings     *GameSettings          `json:"settings"`
	WildState    []slot.GamePosition    `json:"wild_state"` // 当前Wild位置
}

// GameSettings 游戏设置
type GameSettings struct {
	BetAmount     int64  `json:"bet_amount"`      // 下注金额
	AutoSpin      bool   `json:"auto_spin"`       // 自动旋转
	AutoSpinCount int    `json:"auto_spin_count"` // 自动旋转次数
	SoundEnabled  bool   `json:"sound_enabled"`   // 声音开关
	Language      string `json:"language"`        // 语言设置
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	PlayerID       string        `json:"player_id"`
	InitialBalance int64         `json:"initial_balance"`
	Settings       *GameSettings `json:"settings,omitempty"`
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Session *GameSession `json:"session,omitempty"`
}

// SpinGameRequest 旋转请求
type SpinGameRequest struct {
	SessionID string `json:"session_id"`
	BetAmount int64  `json:"bet_amount"`
}

// SpinGameResponse 旋转响应
type SpinGameResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Result    *slot.GoldenWildResult `json:"result,omitempty"`
	Session   *GameSession           `json:"session,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// GetSessionResponse 获取会话响应
type GetSessionResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Session *GameSession `json:"session,omitempty"`
}

// GetStatsResponse 统计响应
type GetStatsResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	TotalBets  int64   `json:"total_bets"`
	TotalWins  int64   `json:"total_wins"`
	RTP        float64 `json:"rtp"`
	HitRate    float64 `json:"hit_rate"`
	AvgCascade float64 `json:"avg_cascade"`
}

// WebSocket消息类型
type WSMessageType string

const (
	WSMsgSpin       WSMessageType = "spin"
	WSMsgResult     WSMessageType = "result"
	WSMsgError      WSMessageType = "error"
	WSMsgHeartbeat  WSMessageType = "heartbeat"
	WSMsgSessionEnd WSMessageType = "session_end"
)

// WebSocketMessage WebSocket消息
type WebSocketMessage struct {
	Type      WSMessageType   `json:"type"`
	SessionID string          `json:"session_id"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// SimpleGameAPI 简化版游戏API服务器（不依赖gorilla/mux）
type SimpleGameAPI struct {
	engine      *slot.GoldenWildCascadeEngine
	sessions    map[string]*GameSession
	hardwareAPI *HardwareAPI // 硬件API（可选）
	mux         *http.ServeMux
}

// NewSimpleGameAPI 创建简化版游戏API
func NewSimpleGameAPI() *SimpleGameAPI {
	// 创建默认游戏配置
	cascadeConfig := slot.GetDefaultCascadeConfig()
	cascadeConfig.GridWidth = 5
	cascadeConfig.GridHeight = 4
	cascadeConfig.MinMatch = 3
	cascadeConfig.MaxCascades = 10
	cascadeConfig.CascadeMultipliers = []float64{
		1.0, 1.5, 2.0, 3.0, 5.0, 8.0, 12.0, 18.0, 25.0, 40.0,
	}

	algorithmConfig := &slot.AlgorithmConfig{
		ReelCount:   5,
		RowCount:    4,
		SymbolCount: 8,
		TargetRTP:   0.96,
		MinRTP:      0.94,
		MaxRTP:      0.98,
		SymbolWeights: [][]int{
			{18, 16, 14, 12, 12, 10, 8, 6},
			{16, 18, 14, 12, 12, 10, 8, 6},
			{14, 16, 18, 12, 12, 10, 8, 6},
			{12, 14, 16, 18, 12, 10, 8, 6},
			{12, 12, 14, 16, 18, 12, 8, 6},
		},
		PayTable: map[int][]int64{
			0: {0, 0, 20, 60, 200},  // 发财
			1: {0, 0, 25, 75, 250},  // 红中
			2: {0, 0, 30, 90, 300},  // 白板
			3: {0, 0, 15, 45, 150},  // 八万
			4: {0, 0, 12, 36, 120},  // 六筒
			5: {0, 0, 10, 30, 100},  // 六条
			6: {0, 0, 8, 24, 80},    // 三筒
			7: {0, 0, 6, 18, 60},    // 二条
		},
		Algorithm:    slot.AlgorithmTypeClassic,
		Volatility:   0.55,
		HitFrequency: 0.4,
	}

	engine := slot.NewGoldenWildCascadeEngine(algorithmConfig, cascadeConfig)

	return &SimpleGameAPI{
		engine:   engine,
		sessions: make(map[string]*GameSession),
		mux:      http.NewServeMux(),
	}
}

// RegisterHardwareAPI 注册硬件API
func (api *SimpleGameAPI) RegisterHardwareAPI(hwAPI *HardwareAPI) {
	api.hardwareAPI = hwAPI
	
	// 注册硬件API路由
	api.mux.HandleFunc("/api/hardware/status", api.corsWrapper(hwAPI.GetStatus))
	api.mux.HandleFunc("/api/hardware/dispense", api.corsWrapper(hwAPI.DispenseCoins))
	api.mux.HandleFunc("/api/hardware/refund", api.corsWrapper(hwAPI.RefundCoins))
	api.mux.HandleFunc("/api/hardware/ticket", api.corsWrapper(hwAPI.PrintTickets))
	api.mux.HandleFunc("/api/hardware/push", api.corsWrapper(hwAPI.PushControl))
	api.mux.HandleFunc("/api/hardware/light", api.corsWrapper(hwAPI.LightControl))
	api.mux.HandleFunc("/api/hardware/mode", api.corsWrapper(hwAPI.SetMode))
	api.mux.HandleFunc("/api/hardware/difficulty", api.corsWrapper(hwAPI.SetDifficulty))
	api.mux.HandleFunc("/api/hardware/fault", api.corsWrapper(hwAPI.FaultRecovery))
	api.mux.HandleFunc("/api/hardware/stats", api.corsWrapper(hwAPI.GetStatistics))
}

// Start 启动HTTP服务器
func (api *SimpleGameAPI) Start(port string) error {
	// 设置游戏API路由
	api.mux.HandleFunc("/health", api.corsWrapper(api.HealthCheck))
	api.mux.HandleFunc("/api/game/session", api.corsWrapper(api.SessionHandler))
	api.mux.HandleFunc("/api/game/session/", api.corsWrapper(api.SessionHandler))
	api.mux.HandleFunc("/api/spin", api.corsWrapper(api.SpinHandler))
	api.mux.HandleFunc("/api/stats", api.corsWrapper(api.StatsHandler))
	
	// 兼容旧版API路径
	api.mux.HandleFunc("/api/v1/session", api.corsWrapper(api.SessionHandler))
	api.mux.HandleFunc("/api/v1/session/", api.corsWrapper(api.SessionHandler))
	api.mux.HandleFunc("/api/v1/spin", api.corsWrapper(api.SpinHandler))
	api.mux.HandleFunc("/api/v1/stats/", api.corsWrapper(api.StatsHandler))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      api.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 游戏API服务器启动在端口 :%s", port)
	
	return server.ListenAndServe()
}

// corsWrapper CORS包装器
func (api *SimpleGameAPI) corsWrapper(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头部
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Content-Type", "application/json")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 日志记录
		start := time.Now()
		handler.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.RequestURI, time.Since(start))
	}
}

// HealthCheck 健康检查
func (api *SimpleGameAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"sessions":  len(api.sessions),
	}
	api.sendJSON(w, response, http.StatusOK)
}

// SessionHandler 会话处理器
func (api *SimpleGameAPI) SessionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		api.CreateSession(w, r)
	case "GET":
		// 从URL中提取session ID（简化版解析）
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/session/")
		if path != "" {
			api.GetSession(w, r, path)
		} else {
			api.sendError(w, "Session ID required", http.StatusBadRequest)
		}
	case "DELETE":
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/session/")
		if path != "" {
			api.DeleteSession(w, r, path)
		} else {
			api.sendError(w, "Session ID required", http.StatusBadRequest)
		}
	default:
		api.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateSession 创建游戏会话
func (api *SimpleGameAPI) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证请求参数
	if req.PlayerID == "" {
		api.sendError(w, "Player ID is required", http.StatusBadRequest)
		return
	}
	if req.InitialBalance <= 0 {
		api.sendError(w, "Initial balance must be positive", http.StatusBadRequest)
		return
	}

	// 创建会话
	sessionID := fmt.Sprintf("session_%s_%d", req.PlayerID, time.Now().Unix())
	
	settings := req.Settings
	if settings == nil {
		settings = &GameSettings{
			BetAmount:     100,
			AutoSpin:      false,
			AutoSpinCount: 0,
			SoundEnabled:  true,
			Language:      "zh-CN",
		}
	}

	session := &GameSession{
		SessionID:    sessionID,
		PlayerID:     req.PlayerID,
		Balance:      req.InitialBalance,
		TotalBet:     0,
		TotalWin:     0,
		GameCount:    0,
		WinCount:     0,
		GoldenCount:  0,
		WildCount:    0,
		CreatedAt:    time.Now(),
		LastPlayedAt: time.Now(),
		Settings:     settings,
		WildState:    []slot.GamePosition{},
	}

	api.sessions[sessionID] = session

	response := CreateSessionResponse{
		Success: true,
		Message: "Session created successfully",
		Session: session,
	}

	api.sendJSON(w, response, http.StatusCreated)
}

// GetSession 获取游戏会话
func (api *SimpleGameAPI) GetSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	session, exists := api.sessions[sessionID]
	if !exists {
		api.sendError(w, "Session not found", http.StatusNotFound)
		return
	}

	response := GetSessionResponse{
		Success: true,
		Message: "Session retrieved successfully",
		Session: session,
	}

	api.sendJSON(w, response, http.StatusOK)
}

// DeleteSession 删除游戏会话
func (api *SimpleGameAPI) DeleteSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if _, exists := api.sessions[sessionID]; !exists {
		api.sendError(w, "Session not found", http.StatusNotFound)
		return
	}

	delete(api.sessions, sessionID)

	response := map[string]interface{}{
		"success": true,
		"message": "Session deleted successfully",
	}

	api.sendJSON(w, response, http.StatusOK)
}

// SpinHandler 旋转处理器
func (api *SimpleGameAPI) SpinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SpinGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取会话
	session, exists := api.sessions[req.SessionID]
	if !exists {
		api.sendError(w, "Session not found", http.StatusNotFound)
		return
	}

	// 验证下注金额
	if req.BetAmount <= 0 {
		api.sendError(w, "Bet amount must be positive", http.StatusBadRequest)
		return
	}
	if req.BetAmount > session.Balance {
		api.sendError(w, "Insufficient balance", http.StatusBadRequest)
		return
	}

	// 执行游戏旋转
	spinRequest := &slot.SpinRequest{
		GameRequest: &slot.GameRequest{
			SessionID: req.SessionID,
			BetAmount: req.BetAmount,
			Metadata: map[string]interface{}{
				"game_type":  "golden_wild_mahjong",
				"player_id":  session.PlayerID,
				"wild_state": session.WildState,
			},
		},
		ThemeID:     "mahjong",
		EnableTheme: false,
	}

	ctx := context.Background()
	result, err := api.engine.SpinWithGoldenWild(ctx, spinRequest)
	if err != nil {
		log.Printf("Game spin failed: %v", err)
		api.sendError(w, "Game execution failed", http.StatusInternalServerError)
		return
	}

	// 更新会话状态
	session.Balance -= req.BetAmount
	session.TotalBet += req.BetAmount
	session.GameCount++
	session.LastPlayedAt = time.Now()

	if result.IsWin {
		session.Balance += result.TotalWin
		session.TotalWin += result.TotalWin
		session.WinCount++
	}

	// 统计金色符号和Wild
	goldenInThisRound := len(result.GoldenSymbols)
	session.GoldenCount += goldenInThisRound

	wildInThisRound := len(result.WildPositions)
	session.WildCount += wildInThisRound

	// 更新Wild状态
	session.WildState = result.WildPositions

	response := SpinGameResponse{
		Success:   true,
		Message:   "Spin completed successfully",
		Result:    result,
		Session:   session,
		Timestamp: time.Now(),
	}

	api.sendJSON(w, response, http.StatusOK)
}

// StatsHandler 统计处理器
func (api *SimpleGameAPI) StatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL中提取session ID
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/stats/")
	if sessionID == "" {
		api.sendError(w, "Session ID required", http.StatusBadRequest)
		return
	}

	session, exists := api.sessions[sessionID]
	if !exists {
		api.sendError(w, "Session not found", http.StatusNotFound)
		return
	}

	rtp := float64(0)
	if session.TotalBet > 0 {
		rtp = float64(session.TotalWin) / float64(session.TotalBet)
	}

	hitRate := float64(0)
	if session.GameCount > 0 {
		hitRate = float64(session.WinCount) / float64(session.GameCount)
	}

	response := GetStatsResponse{
		Success:    true,
		Message:    "Stats retrieved successfully",
		TotalBets:  session.TotalBet,
		TotalWins:  session.TotalWin,
		RTP:        rtp,
		HitRate:    hitRate,
		AvgCascade: 0, // 需要从引擎获取
	}

	api.sendJSON(w, response, http.StatusOK)
}

// 辅助方法

// sendJSON 发送JSON响应
func (api *SimpleGameAPI) sendJSON(w http.ResponseWriter, data interface{}, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError 发送错误响应
func (api *SimpleGameAPI) sendError(w http.ResponseWriter, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"message":   message,
		"timestamp": time.Now(),
	}
	api.sendJSON(w, response, status)
}

// 从查询参数获取整数
func getIntParam(r *http.Request, key string, defaultValue int) int {
	if value := r.URL.Query().Get(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}