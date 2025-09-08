package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"github.com/wfunc/slot-game/internal/utils"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AuthServiceTestSuite 认证服务测试套件
type AuthServiceTestSuite struct {
	suite.Suite
	db          *gorm.DB
	authService AuthService
	userService UserService
	jwtManager  *utils.JWTManager
}

// SetupSuite 设置测试套件
func (suite *AuthServiceTestSuite) SetupSuite() {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)
	
	// 自动迁移
	err = db.AutoMigrate(
		&models.User{},
		&models.UserAuth{},
		&models.UserSession{},
		&models.Wallet{},
		&models.GameResult{},
	)
	assert.NoError(suite.T(), err)
	
	suite.db = db
	
	// 创建服务
	config := DefaultConfig()
	log, _ := zap.NewDevelopment()
	
	services := NewServices(db, config, log)
	suite.authService = services.Auth
	suite.userService = services.User
	suite.jwtManager = utils.NewJWTManager(
		config.JWTSecret,
		config.AccessTokenExpiry,
		config.RefreshTokenExpiry,
	)
}

// SetupTest 每个测试前执行
func (suite *AuthServiceTestSuite) SetupTest() {
	// 清理数据
	suite.db.Exec("DELETE FROM user_sessions")
	suite.db.Exec("DELETE FROM user_auths")
	suite.db.Exec("DELETE FROM wallets")
	suite.db.Exec("DELETE FROM users")
}

// TestRegister 测试注册
func (suite *AuthServiceTestSuite) TestRegister() {
	ctx := context.Background()
	
	req := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
		Nickname:        "Test User",
	}
	
	// 注册用户
	resp, err := suite.authService.Register(ctx, req)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotEmpty(suite.T(), resp.AccessToken)
	assert.NotEmpty(suite.T(), resp.RefreshToken)
	assert.Equal(suite.T(), "testuser", resp.User.Username)
	assert.Equal(suite.T(), "test@example.com", resp.User.Email)
	
	// 验证用户已创建
	user, err := suite.userService.GetUserByUsername(ctx, "testuser")
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	
	// 验证钱包已创建
	walletRepo := repository.NewWalletRepository(suite.db)
	wallet, err := walletRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), wallet)
	assert.Equal(suite.T(), int64(1000), wallet.Coins) // 注册赠送1000游戏币
}

// TestRegisterDuplicateUsername 测试重复用户名注册
func (suite *AuthServiceTestSuite) TestRegisterDuplicateUsername() {
	ctx := context.Background()
	
	req1 := &RegisterRequest{
		Username:        "testuser",
		Email:           "test1@example.com",
		Phone:           "13800138001",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	// 第一次注册成功
	_, err := suite.authService.Register(ctx, req1)
	assert.NoError(suite.T(), err)
	
	// 第二次注册失败（用户名重复）
	req2 := &RegisterRequest{
		Username:        "testuser",
		Email:           "test2@example.com",
		Phone:           "13800138002",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	_, err = suite.authService.Register(ctx, req2)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "用户名已存在")
}

// TestLogin 测试登录
func (suite *AuthServiceTestSuite) TestLogin() {
	ctx := context.Background()
	
	// 先注册用户
	registerReq := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	_, err := suite.authService.Register(ctx, registerReq)
	assert.NoError(suite.T(), err)
	
	// 测试用户名登录
	loginReq := &LoginRequest{
		Account:  "testuser",
		Password: "password123",
		Device:   "Test Device",
		IP:       "127.0.0.1",
	}
	
	resp, err := suite.authService.Login(ctx, loginReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotEmpty(suite.T(), resp.AccessToken)
	assert.NotEmpty(suite.T(), resp.RefreshToken)
	
	// 测试邮箱登录
	loginReq.Account = "test@example.com"
	resp, err = suite.authService.Login(ctx, loginReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	
	// 测试手机号登录
	loginReq.Account = "13800138000"
	resp, err = suite.authService.Login(ctx, loginReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
}

// TestLoginInvalidPassword 测试错误密码登录
func (suite *AuthServiceTestSuite) TestLoginInvalidPassword() {
	ctx := context.Background()
	
	// 先注册用户
	registerReq := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	_, err := suite.authService.Register(ctx, registerReq)
	assert.NoError(suite.T(), err)
	
	// 使用错误密码登录
	loginReq := &LoginRequest{
		Account:  "testuser",
		Password: "wrongpassword",
		Device:   "Test Device",
		IP:       "127.0.0.1",
	}
	
	_, err = suite.authService.Login(ctx, loginReq)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrInvalidCredentials, err)
}

// TestValidateToken 测试令牌验证
func (suite *AuthServiceTestSuite) TestValidateToken() {
	ctx := context.Background()
	
	// 注册并登录
	registerReq := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	resp, err := suite.authService.Register(ctx, registerReq)
	assert.NoError(suite.T(), err)
	
	// 验证访问令牌
	claims, err := suite.authService.ValidateToken(ctx, resp.AccessToken)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), claims)
	assert.Equal(suite.T(), resp.User.ID, claims.UserID)
	assert.Equal(suite.T(), resp.User.Username, claims.Username)
	
	// 验证无效令牌
	_, err = suite.authService.ValidateToken(ctx, "invalid-token")
	assert.Error(suite.T(), err)
}

// TestRefreshToken 测试刷新令牌
func (suite *AuthServiceTestSuite) TestRefreshToken() {
	ctx := context.Background()
	
	// 注册并登录
	registerReq := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	resp, err := suite.authService.Register(ctx, registerReq)
	assert.NoError(suite.T(), err)
	
	// 等待一秒以确保新令牌不同
	time.Sleep(1 * time.Second)
	
	// 刷新令牌
	newResp, err := suite.authService.RefreshToken(ctx, resp.RefreshToken)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), newResp)
	assert.NotEmpty(suite.T(), newResp.AccessToken)
	assert.NotEqual(suite.T(), resp.AccessToken, newResp.AccessToken) // 新的访问令牌
}

// TestLogout 测试登出
func (suite *AuthServiceTestSuite) TestLogout() {
	ctx := context.Background()
	
	// 注册并登录
	registerReq := &RegisterRequest{
		Username:        "testuser",
		Email:           "test@example.com",
		Phone:           "13800138000",
		Password:        "password123",
		ConfirmPassword: "password123",
	}
	
	resp, err := suite.authService.Register(ctx, registerReq)
	assert.NoError(suite.T(), err)
	
	// 验证令牌有效
	_, err = suite.authService.ValidateToken(ctx, resp.AccessToken)
	assert.NoError(suite.T(), err)
	
	// 登出
	err = suite.authService.Logout(ctx, resp.User.ID, resp.AccessToken)
	assert.NoError(suite.T(), err)
	
	// 验证令牌无效（会话已删除）
	_, err = suite.authService.ValidateToken(ctx, resp.AccessToken)
	assert.Error(suite.T(), err)
}

// TestRunAuthServiceTestSuite 运行测试套件
func TestRunAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}