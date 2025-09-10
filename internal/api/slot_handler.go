package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wfunc/slot-game/internal/game"
	"github.com/wfunc/slot-game/internal/middleware"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
)

// SlotHandler 老虎机游戏处理器
type SlotHandler struct {
	gameService *game.GameService
	walletRepo  repository.WalletRepository
	wsHandler   *WebSocketHandler
	logger      *zap.Logger
}

// NewSlotHandler 创建老虎机处理器
func NewSlotHandler(gameService *game.GameService, walletRepo repository.WalletRepository, wsHandler *WebSocketHandler, logger *zap.Logger) *SlotHandler {
	return &SlotHandler{
		gameService: gameService,
		walletRepo:  walletRepo,
		wsHandler:   wsHandler,
		logger:      logger,
	}
}

// StartRequest 开始游戏请求
type StartRequest struct {
	BetAmount int64 `json:"bet_amount" binding:"required,min=100"`
}

// StartResponse 开始游戏响应
type StartResponse struct {
	SessionID string `json:"session_id"`
	Balance   int64  `json:"balance"`
	Message   string `json:"message"`
}

// SpinRequest 转动请求
type SpinRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// SpinResponse 转动响应
type SpinResponse struct {
	Result   interface{} `json:"result"`
	Balance  int64       `json:"balance"`
	State    string      `json:"state"`
	TotalBet int64       `json:"total_bet"`
	TotalWin int64       `json:"total_win"`
}

// SettleRequest 结算请求
type SettleRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// SettleResponse 结算响应
type SettleResponse struct {
	TotalBet  int64  `json:"total_bet"`
	TotalWin  int64  `json:"total_win"`
	Balance   int64  `json:"balance"`
	RTP       float64 `json:"rtp"`
	Message   string `json:"message"`
}

// HistoryResponse 历史记录响应
type HistoryResponse struct {
	Records interface{} `json:"records"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PageSize int        `json:"page_size"`
}

// SessionInfoResponse 会话信息响应
type SessionInfoResponse struct {
	SessionID   string      `json:"session_id"`
	State       string      `json:"state"`
	SpinCount   int         `json:"spin_count"`
	TotalBet    int64       `json:"total_bet"`
	TotalWin    int64       `json:"total_win"`
	RTP         float64     `json:"rtp"`
	Duration    float64     `json:"duration"`
	LastResult  interface{} `json:"last_result,omitempty"`
}

// Start 开始游戏
// @Summary 开始游戏
// @Description 扣除投注并创建会话，返回 session_id 与余额
// @Tags Slot
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body StartRequest true "开始游戏请求"
// @Success 200 {object} StartResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/start [post]
func (h *SlotHandler) Start(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 验证投注金额
	if req.BetAmount < 100 || req.BetAmount > 10000 {
		c.JSON(400, gin.H{"error": "投注金额必须在100-10000之间"})
		return
	}

	// 生成会话ID
	sessionID := generateSessionID()

	// 调用游戏服务开始游戏
	err := h.gameService.StartGame(c.Request.Context(), userID, sessionID, req.BetAmount)
	if err != nil {
		h.logger.Error("开始游戏失败",
			zap.Uint("user_id", userID),
			zap.String("session_id", sessionID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 获取当前余额
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取钱包失败", zap.Error(err))
		c.JSON(500, gin.H{"error": "获取余额失败"})
		return
	}

	h.logger.Info("游戏开始",
		zap.Uint("user_id", userID),
		zap.String("session_id", sessionID),
		zap.Int64("bet_amount", req.BetAmount),
		zap.Int64("balance", wallet.Balance))

	response := StartResponse{
		SessionID: sessionID,
		Balance:   wallet.Balance,
		Message:   "游戏已开始，请执行转动",
	}

	// 发送WebSocket消息
	if h.wsHandler != nil {
		h.wsHandler.SendGameStart(userID, sessionID, response)
		h.wsHandler.SendBalanceUpdate(userID, wallet.Balance)
	}

	c.JSON(200, response)
}

// Spin 执行转动
// @Summary 单次转动
// @Description 在已有会话内执行一次转动
// @Tags Slot
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body SpinRequest true "转动请求"
// @Success 200 {object} SpinResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/spin [post]
func (h *SlotHandler) Spin(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req SpinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 执行转动
	result, err := h.gameService.Spin(c.Request.Context(), req.SessionID)
	if err != nil {
		h.logger.Error("转动失败",
			zap.String("session_id", req.SessionID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 获取当前余额
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取钱包失败", zap.Error(err))
		wallet = &models.Wallet{Balance: 0} // 使用默认值避免nil
	}

	h.logger.Info("转动完成",
		zap.String("session_id", req.SessionID),
		zap.Any("result", result))

	response := SpinResponse{
		Result:   result.Result,
		Balance:  wallet.Balance,
		State:    result.State,
		TotalBet: result.TotalBet,
		TotalWin: result.TotalWin,
	}

	// 发送WebSocket消息
	if h.wsHandler != nil {
		h.wsHandler.SendGameResult(userID, req.SessionID, response)
		h.wsHandler.SendBalanceUpdate(userID, wallet.Balance)
		h.wsHandler.SendGameState(req.SessionID, map[string]interface{}{
			"state":     result.State,
			"total_bet": result.TotalBet,
			"total_win": result.TotalWin,
		})
	}

	c.JSON(200, response)
}

// Settle 结算游戏
// @Summary 结算会话
// @Description 对当前会话进行结算，返回统计数据
// @Tags Slot
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body SettleRequest true "结算请求"
// @Success 200 {object} SettleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/settle [post]
func (h *SlotHandler) Settle(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req SettleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 获取会话信息
	sessionInfo, err := h.gameService.GetSessionInfo(c.Request.Context(), req.SessionID)
	if err != nil {
		h.logger.Error("获取会话信息失败",
			zap.String("session_id", req.SessionID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": "会话不存在"})
		return
	}

	// 执行结算
	err = h.gameService.Settle(c.Request.Context(), req.SessionID)
	if err != nil {
		h.logger.Error("结算失败",
			zap.String("session_id", req.SessionID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 获取当前余额
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取钱包失败", zap.Error(err))
		wallet = &models.Wallet{Balance: 0}
	}

	h.logger.Info("游戏结算",
		zap.String("session_id", req.SessionID),
		zap.Int64("total_bet", sessionInfo.TotalBet),
		zap.Int64("total_win", sessionInfo.TotalWin))

	c.JSON(200, SettleResponse{
		TotalBet:  sessionInfo.TotalBet,
		TotalWin:  sessionInfo.TotalWin,
		Balance:   wallet.Balance,
		RTP:       sessionInfo.RTP,
		Message:   "游戏已结算",
	})
}

// GetHistory 获取游戏历史
// @Summary 用户历史记录
// @Description 获取当前用户的最近游戏历史
// @Tags Slot
// @Security Bearer
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量（<=100）"
// @Success 200 {object} HistoryResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/history [get]
func (h *SlotHandler) GetHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	// 获取分页参数
	page := 1
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	// 限制页面大小
	if pageSize > 100 {
		pageSize = 100
	}

	// 获取游戏历史
	records, err := h.gameService.GetUserGameHistory(c.Request.Context(), userID, pageSize)
	if err != nil {
		h.logger.Error("获取游戏历史失败",
			zap.Uint("user_id", userID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": "获取历史记录失败"})
		return
	}

	c.JSON(200, HistoryResponse{
		Records:  records,
		Total:    len(records),
		Page:     page,
		PageSize: pageSize,
	})
}

// GetSessionInfo 获取会话信息
// @Summary 会话信息
// @Description 获取指定会话的当前状态与统计
// @Tags Slot
// @Security Bearer
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} SessionInfoResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/slot/session/{id} [get]
func (h *SlotHandler) GetSessionInfo(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(400, gin.H{"error": "会话ID不能为空"})
		return
	}

	// 获取会话信息
	info, err := h.gameService.GetSessionInfo(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("获取会话信息失败",
			zap.String("session_id", sessionID),
			zap.Error(err))
		c.JSON(404, gin.H{"error": "会话不存在"})
		return
	}

	// 验证会话属于当前用户
	if info.UserID != userID {
		c.JSON(403, gin.H{"error": "无权访问此会话"})
		return
	}

	c.JSON(200, SessionInfoResponse{
		SessionID:   info.SessionID,
		State:       string(info.State),
		SpinCount:   info.SpinCount,
		TotalBet:    info.TotalBet,
		TotalWin:    info.TotalWin,
		RTP:         info.RTP,
		Duration:    info.Duration,
		LastResult:  info.LastResult,
	})
}

// BatchSpin 批量转动
// @Summary 批量转动
// @Description 连续多次转动，支持中奖/大奖阈值提前停止
// @Tags Slot
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body game.BatchSpinRequest true "批量转动请求"
// @Success 200 {object} game.BatchSpinResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/batch-spin [post]
func (h *SlotHandler) BatchSpin(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	var req game.BatchSpinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 验证转动次数
	if req.SpinCount < 1 || req.SpinCount > 100 {
		c.JSON(400, gin.H{"error": "转动次数必须在1-100之间"})
		return
	}

	// 执行批量转动
	result, err := h.gameService.BatchSpin(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("批量转动失败",
			zap.String("session_id", req.SessionID),
			zap.Int("spin_count", req.SpinCount),
			zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 获取当前余额
	wallet, err := h.walletRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取钱包失败", zap.Error(err))
	}

	h.logger.Info("批量转动完成",
		zap.String("session_id", req.SessionID),
		zap.Int("total_spins", result.TotalSpins),
		zap.Int64("total_win", result.TotalWin))

	// 发送WebSocket批量结果通知
	if h.wsHandler != nil {
		// 发送批量转动结果
		h.wsHandler.SendGameResult(userID, req.SessionID, result)
		// 更新余额
		if wallet != nil {
			h.wsHandler.SendBalanceUpdate(userID, wallet.Balance)
		}
		// 更新游戏状态
		h.wsHandler.SendGameState(req.SessionID, map[string]interface{}{
			"state":         result.State,
			"total_bet":     result.TotalBet,
			"total_win":     result.TotalWin,
			"total_spins":   result.TotalSpins,
			"stopped_early": result.StoppedEarly,
			"stop_reason":   result.StopReason,
		})
	}

	c.JSON(200, result)
}

// GetUserStats 获取用户统计
// @Summary 用户统计
// @Description 获取当前用户的游戏统计信息（RTP、最大中奖等）
// @Tags Slot
// @Security Bearer
// @Produce json
// @Success 200 {object} game.UserGameStats
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/slot/stats [get]
func (h *SlotHandler) GetUserStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists || userID == 0 {
		c.JSON(401, gin.H{"error": "未登录"})
		return
	}

	stats, err := h.gameService.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("获取用户统计失败",
			zap.Uint("user_id", userID),
			zap.Error(err))
		c.JSON(500, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(200, stats)
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	return fmt.Sprintf("slot_%s_%d", uuid.New().String()[:8], time.Now().Unix())
}
