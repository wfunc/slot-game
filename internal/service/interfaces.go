package service

import (
	"context"
	"time"

	"github.com/wfunc/slot-game/internal/models"
)

// UserService 用户服务接口
type UserService interface {
	// 用户管理
	GetUserByID(ctx context.Context, userID uint) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*models.User, error)
	UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error
	UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error
	UpdateProfile(ctx context.Context, userID uint, profile *UserProfile) error
	GetUserList(ctx context.Context, page, pageSize int) ([]*models.User, int64, error)
	SearchUsers(ctx context.Context, query string, page, pageSize int) ([]*models.User, int64, error)
	
	// 用户状态
	UpdateUserStatus(ctx context.Context, userID uint, status string) error
	BanUser(ctx context.Context, userID uint, reason string, duration time.Duration) error
	UnbanUser(ctx context.Context, userID uint) error
	
	// 用户统计
	GetUserStats(ctx context.Context, userID uint) (*UserStats, error)
	GetUserGameHistory(ctx context.Context, userID uint, page, pageSize int) ([]*models.GameResult, int64, error)
}

// AuthService 认证服务接口
type AuthService interface {
	// 注册登录
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	Logout(ctx context.Context, userID uint, token string) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	
	// 验证
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error)
	
	// 密码相关
	ResetPasswordRequest(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	VerifyEmail(ctx context.Context, token string) error
	
	// OAuth2.0
	OAuthLogin(ctx context.Context, provider string, code string) (*AuthResponse, error)
	BindOAuthAccount(ctx context.Context, userID uint, provider string, oauthID string) error
	
	// 会话管理
	GetActiveSessions(ctx context.Context, userID uint) ([]*models.UserSession, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID uint) error
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=20"`
	Email           string `json:"email" binding:"required,email"`
	Phone           string `json:"phone" binding:"required"`
	Password        string `json:"password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
	Nickname        string `json:"nickname"`
	Avatar          string `json:"avatar"`
	InviteCode      string `json:"invite_code"`
	IP              string `json:"-"` // 客户端IP，由handler设置
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account  string `json:"account" binding:"required"` // 用户名/邮箱/手机号
	Password string `json:"password" binding:"required"`
	Device   string `json:"device"`
	IP       string `json:"ip"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	TokenType    string       `json:"token_type"`
}

// TokenClaims JWT Claims
type TokenClaims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// UserProfile 用户资料
type UserProfile struct {
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Gender    string    `json:"gender"`
	Birthday  time.Time `json:"birthday"`
	City      string    `json:"city"`
	Signature string    `json:"signature"`
}

// UserStats 用户统计
type UserStats struct {
	TotalGames     int   `json:"total_games"`
	TotalWins      int   `json:"total_wins"`
	TotalBet       int64 `json:"total_bet"`
	TotalWinAmount int64 `json:"total_win_amount"`
	WinRate        float64 `json:"win_rate"`
	Level          int   `json:"level"`
	Experience     int   `json:"experience"`
	VIPLevel       int   `json:"vip_level"`
	LastLoginAt    time.Time `json:"last_login_at"`
}