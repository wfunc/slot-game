package game

import (
	"time"
	
	"github.com/wfunc/slot-game/internal/game/slot"
)

// SpinResponse 转动响应
type SpinResponse struct {
	SessionID string           `json:"session_id"`
	Result    *slot.SpinResult `json:"result"`
	State     string           `json:"state"`
	TotalBet  int64            `json:"total_bet"`
	TotalWin  int64            `json:"total_win"`
	SpinCount int              `json:"spin_count"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID   string           `json:"session_id"`
	UserID      uint             `json:"user_id"`
	State       GameState        `json:"state"`
	StartTime   time.Time        `json:"start_time"`
	Duration    float64          `json:"duration"`
	SpinCount   int              `json:"spin_count"`
	TotalBet    int64            `json:"total_bet"`
	TotalWin    int64            `json:"total_win"`
	RTP         float64          `json:"rtp"`
	LastResult  *slot.SpinResult `json:"last_result,omitempty"`
	ValidEvents []string         `json:"valid_events"`
}

// UserGameStats 用户游戏统计
type UserGameStats struct {
	UserID     uint      `json:"user_id"`
	TotalGames int       `json:"total_games"`
	TotalBet   int64     `json:"total_bet"`
	TotalWin   int64     `json:"total_win"`
	BiggestWin int64     `json:"biggest_win"`
	RTP        float64   `json:"rtp"`
	LastPlayed time.Time `json:"last_played"`
}

// GameStartRequest 开始游戏请求
type GameStartRequest struct {
	UserID    uint  `json:"user_id" binding:"required"`
	BetAmount int64 `json:"bet_amount" binding:"required,min=1"`
}

// GameSpinRequest 转动请求
type GameSpinRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// GameSettleRequest 结算请求
type GameSettleRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// GameHistoryRequest 游戏历史请求
type GameHistoryRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	Limit  int  `json:"limit" binding:"min=1,max=100"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}