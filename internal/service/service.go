package service

import (
	"time"

	"github.com/wfunc/slot-game/internal/repository"
	"github.com/wfunc/slot-game/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config 服务配置
type Config struct {
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		JWTSecret:          "your-secret-key-change-in-production",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

// Services 服务集合
type Services struct {
	Auth AuthService
	User UserService
}

// NewServices 创建服务集合
func NewServices(db *gorm.DB, config *Config, log *zap.Logger) *Services {
	// 初始化仓储
	userRepo := repository.NewUserRepository(db)
	authRepo := repository.NewUserAuthRepository(db)
	sessionRepo := repository.NewUserSessionRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	gameResultRepo := repository.NewGameResultRepository(db)
	
	// 初始化JWT管理器
	jwtManager := utils.NewJWTManager(
		config.JWTSecret,
		config.AccessTokenExpiry,
		config.RefreshTokenExpiry,
	)
	
	// 初始化服务
	authService := NewAuthService(
		db,
		userRepo,
		authRepo,
		sessionRepo,
		walletRepo,
		jwtManager,
		log,
	)
	
	userService := NewUserService(
		db,
		userRepo,
		authRepo,
		walletRepo,
		gameResultRepo,
		log,
	)
	
	return &Services{
		Auth: authService,
		User: userService,
	}
}