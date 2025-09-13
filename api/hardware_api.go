package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	
	"github.com/wfunc/slot-game/internal/hardware"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

var (
	ErrInvalidCommand = errors.New("invalid command")
)

// HardwareAPI 硬件API服务
type HardwareAPI struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	manager         *hardware.HardwareManager
	server          *http.Server
}

// HardwareStatusResponse 硬件状态响应
type HardwareStatusResponse struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	Connected  bool                   `json:"connected"`
	Running    bool                   `json:"running"`
	Statistics map[string]interface{} `json:"statistics,omitempty"`
}

// HardwareCommandRequest 硬件命令请求
type HardwareCommandRequest struct {
	Command    string                 `json:"command"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// HardwareCommandResponse 硬件命令响应
type HardwareCommandResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHardwareAPI 创建硬件API
func NewHardwareAPI(manager *hardware.HardwareManager) *HardwareAPI {
	return &HardwareAPI{
		logger:  logger.GetLogger(),
		manager: manager,
	}
}

// Initialize 初始化硬件API
func (api *HardwareAPI) Initialize(config *hardware.HardwareConfig) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	
	// 创建硬件管理器
	api.manager = hardware.NewHardwareManager(config)
	
	// 初始化硬件
	if err := api.manager.Initialize(); err != nil {
		api.logger.Error("硬件初始化失败", zap.Error(err))
		return err
	}
	
	// 注册事件处理器
	api.registerEventHandlers()
	
	// 如果配置了自动启动
	if config.AutoStart {
		if err := api.manager.Start(); err != nil {
			api.logger.Error("硬件启动失败", zap.Error(err))
			return err
		}
	}
	
	api.logger.Info("硬件API初始化完成")
	return nil
}

// registerEventHandlers 注册事件处理器
func (api *HardwareAPI) registerEventHandlers() {
	// 投币事件
	api.manager.RegisterEventHandler("coin_inserted", func(event interface{}) {
		count := event.(byte)
		api.logger.Info("API: 投币事件", zap.Uint8("count", count))
		// 这里可以广播WebSocket消息给前端
	})
	
	// 回币事件
	api.manager.RegisterEventHandler("coin_returned", func(event interface{}) {
		data := event.(*hardware.CoinReturnData)
		api.logger.Info("API: 回币事件", 
			zap.Uint8("front", data.FrontCount),
			zap.Uint8("left", data.LeftCount),
			zap.Uint8("right", data.RightCount))
		// 这里可以广播WebSocket消息给前端
	})
	
	// 按键事件
	api.manager.RegisterEventHandler("button_pressed", func(event interface{}) {
		btnEvent := event.(*hardware.ButtonEvent)
		api.logger.Info("API: 按键事件", 
			zap.Uint8("keyCode", btnEvent.KeyCode))
		// 这里可以广播WebSocket消息给前端
	})
	
	// 故障事件
	api.manager.RegisterEventHandler("fault", func(event interface{}) {
		faultEvent := event.(*hardware.FaultEvent)
		api.logger.Error("API: 故障事件", 
			zap.Uint8("code", faultEvent.FaultCode),
			zap.Uint8("level", faultEvent.Level))
		// 这里可以发送告警通知
	})
}

// SetupRoutes 设置HTTP路由
func (api *HardwareAPI) SetupRoutes(mux *http.ServeMux) {
	// 硬件状态
	mux.HandleFunc("/api/hardware/status", api.handleStatus)
	
	// 硬件控制
	mux.HandleFunc("/api/hardware/start", api.handleStart)
	mux.HandleFunc("/api/hardware/stop", api.handleStop)
	mux.HandleFunc("/api/hardware/command", api.handleCommand)
	
	// 游戏控制
	mux.HandleFunc("/api/hardware/dispense", api.handleDispenseCoins)
	mux.HandleFunc("/api/hardware/refund", api.handleRefundCoins)
	mux.HandleFunc("/api/hardware/ticket", api.handlePrintTickets)
	mux.HandleFunc("/api/hardware/light", api.handleLightControl)
	mux.HandleFunc("/api/hardware/push", api.handlePushControl)
	
	// 统计数据
	mux.HandleFunc("/api/hardware/statistics", api.handleStatistics)
}

// handleStatus 处理状态请求
func (api *HardwareAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	api.mu.RLock()
	defer api.mu.RUnlock()
	
	response := HardwareStatusResponse{
		Success:   true,
		Connected: api.manager != nil,
		Running:   api.manager != nil && api.manager.IsRunning(),
	}
	
	if api.manager != nil {
		response.Statistics = api.manager.GetStatistics()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStart 处理启动请求
func (api *HardwareAPI) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	api.mu.Lock()
	defer api.mu.Unlock()
	
	if api.manager == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未初始化",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	if err := api.manager.Start(); err != nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	response := HardwareCommandResponse{
		Success: true,
		Message: "硬件启动成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStop 处理停止请求
func (api *HardwareAPI) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	api.mu.Lock()
	defer api.mu.Unlock()
	
	if api.manager == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未初始化",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	if err := api.manager.Stop(); err != nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	response := HardwareCommandResponse{
		Success: true,
		Message: "硬件停止成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCommand 处理通用命令
func (api *HardwareAPI) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req HardwareCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	gameLogic := api.manager.GetGameLogic()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	var err error
	var data interface{}
	
	// 处理不同的命令
	switch req.Command {
	case "query_status":
		queryType := byte(0x01)
		if v, ok := req.Parameters["type"].(float64); ok {
			queryType = byte(v)
		}
		err = controller.QueryStatus(queryType)
		
	case "set_mode":
		if v, ok := req.Parameters["mode"].(float64); ok {
			gameLogic.SetCurrentMode(hardware.GameMode(v))
			data = map[string]interface{}{"mode": v}
		}
		
	case "reset":
		gameLogic.Reset()
		data = "游戏状态已重置"
		
	default:
		err = ErrInvalidCommand
	}
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data:    data,
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDispenseCoins 处理上币请求
func (api *HardwareAPI) handleDispenseCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Count uint16 `json:"count"`
		Speed byte   `json:"speed"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	if req.Speed == 0 {
		req.Speed = 5 // 默认速度
	}
	
	err := controller.DispenseCoins(req.Count, req.Speed)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"count": req.Count,
			"speed": req.Speed,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRefundCoins 处理退币请求
func (api *HardwareAPI) handleRefundCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Count uint16 `json:"count"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	err := controller.RefundCoins(req.Count)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"count": req.Count,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePrintTickets 处理打印彩票请求
func (api *HardwareAPI) handlePrintTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Count uint16 `json:"count"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	err := controller.PrintTickets(req.Count)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"count": req.Count,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLightControl 处理灯光控制请求
func (api *HardwareAPI) handleLightControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Pattern    byte `json:"pattern"`
		Brightness byte `json:"brightness"`
		Duration   byte `json:"duration"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// 设置默认值
	if req.Brightness == 0 {
		req.Brightness = 100
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	err := controller.LightControl(req.Pattern, req.Brightness, req.Duration)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"pattern":    req.Pattern,
			"brightness": req.Brightness,
			"duration":   req.Duration,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePushControl 处理推币控制请求
func (api *HardwareAPI) handlePushControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Action    byte `json:"action"`
		Parameter byte `json:"parameter"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	err := controller.PushControl(req.Action, req.Parameter)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"action":    req.Action,
			"parameter": req.Parameter,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStatistics 处理统计请求
func (api *HardwareAPI) handleStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	api.mu.RLock()
	defer api.mu.RUnlock()
	
	if api.manager == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未初始化",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	stats := api.manager.GetStatistics()
	
	response := HardwareCommandResponse{
		Success: true,
		Data:    stats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
// GetStatus 获取状态（HTTP处理器）
func (api *HardwareAPI) GetStatus(w http.ResponseWriter, r *http.Request) {
	api.handleStatus(w, r)
}

// DispenseCoins 上币（HTTP处理器）
func (api *HardwareAPI) DispenseCoins(w http.ResponseWriter, r *http.Request) {
	api.handleDispenseCoins(w, r)
}

// RefundCoins 退币（HTTP处理器）
func (api *HardwareAPI) RefundCoins(w http.ResponseWriter, r *http.Request) {
	api.handleRefundCoins(w, r)
}

// PrintTickets 打印彩票（HTTP处理器）
func (api *HardwareAPI) PrintTickets(w http.ResponseWriter, r *http.Request) {
	api.handlePrintTickets(w, r)
}

// PushControl 推币控制（HTTP处理器）
func (api *HardwareAPI) PushControl(w http.ResponseWriter, r *http.Request) {
	api.handlePushControl(w, r)
}

// LightControl 灯光控制（HTTP处理器）
func (api *HardwareAPI) LightControl(w http.ResponseWriter, r *http.Request) {
	api.handleLightControl(w, r)
}

// SetMode 设置模式（HTTP处理器）
func (api *HardwareAPI) SetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Mode byte `json:"mode"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	gameLogic := api.manager.GetGameLogic()
	api.mu.RUnlock()
	
	if gameLogic != nil {
		gameLogic.SetCurrentMode(hardware.GameMode(req.Mode))
	}
	
	response := HardwareCommandResponse{
		Success: true,
		Data: map[string]interface{}{
			"mode": req.Mode,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SetDifficulty 设置难度（HTTP处理器）
func (api *HardwareAPI) SetDifficulty(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Level byte `json:"level"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	gameLogic := api.manager.GetGameLogic()
	api.mu.RUnlock()
	
	if gameLogic != nil {
		gameLogic.SetDifficulty(req.Level)
	}
	
	response := HardwareCommandResponse{
		Success: true,
		Data: map[string]interface{}{
			"level": req.Level,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FaultRecovery 故障恢复（HTTP处理器）
func (api *HardwareAPI) FaultRecovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		FaultCode  byte `json:"fault_code"`
		Action     byte `json:"action"`
		RetryCount byte `json:"retry_count"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	api.mu.RLock()
	controller := api.manager.GetController()
	api.mu.RUnlock()
	
	if controller == nil {
		response := HardwareCommandResponse{
			Success: false,
			Message: "硬件未连接",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	
	err := controller.FaultRecovery(req.FaultCode, req.Action, req.RetryCount)
	
	response := HardwareCommandResponse{
		Success: err == nil,
		Data: map[string]interface{}{
			"fault_code":  req.FaultCode,
			"action":      req.Action,
			"retry_count": req.RetryCount,
		},
	}
	
	if err != nil {
		response.Message = err.Error()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStatistics 获取统计信息（HTTP处理器）
func (api *HardwareAPI) GetStatistics(w http.ResponseWriter, r *http.Request) {
	api.handleStatistics(w, r)
}
