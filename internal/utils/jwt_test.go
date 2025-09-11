package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// JWTTestSuite JWT工具测试套件
type JWTTestSuite struct {
	suite.Suite
	manager *JWTManager
}

func (suite *JWTTestSuite) SetupTest() {
	suite.manager = NewJWTManager(
		"test-secret-key",
		1*time.Hour,   // access token expiry
		7*24*time.Hour, // refresh token expiry
	)
}

// 测试创建JWT管理器
func (suite *JWTTestSuite) TestNewJWTManager() {
	manager := NewJWTManager("secret", 1*time.Hour, 24*time.Hour)
	suite.NotNil(manager)
	// 私有字段无法直接访问，通过GetTokenExpiry间接验证
	suite.Equal(1*time.Hour, manager.GetTokenExpiry("access"))
	suite.Equal(24*time.Hour, manager.GetTokenExpiry("refresh"))
}

// 测试生成访问令牌
func (suite *JWTTestSuite) TestGenerateAccessToken() {
	userID := uint(123)
	username := "testuser"
	email := "test@example.com"
	role := "user"
	sessionID := "session-123"
	
	token, err := suite.manager.GenerateAccessToken(userID, username, email, role, sessionID)
	suite.NoError(err)
	suite.NotEmpty(token)
}

// 测试生成刷新令牌
func (suite *JWTTestSuite) TestGenerateRefreshToken() {
	userID := uint(456)
	sessionID := "session-456"
	
	token, err := suite.manager.GenerateRefreshToken(userID, sessionID)
	suite.NoError(err)
	suite.NotEmpty(token)
}

// 测试验证令牌
func (suite *JWTTestSuite) TestValidateToken() {
	// 生成有效令牌
	userID := uint(789)
	username := "validuser"
	email := "valid@example.com"
	role := "admin"
	sessionID := "session-789"
	
	token, _ := suite.manager.GenerateAccessToken(userID, username, email, role, sessionID)
	
	// 验证有效令牌
	claims, err := suite.manager.ValidateToken(token)
	suite.NoError(err)
	suite.NotNil(claims)
	suite.Equal(userID, claims.UserID)
	suite.Equal(username, claims.Username)
	suite.Equal(email, claims.Email)
	suite.Equal(role, claims.Role)
	suite.Equal(sessionID, claims.SessionID)
}

// 测试验证无效令牌
func (suite *JWTTestSuite) TestValidateInvalidToken() {
	// 无效格式的令牌
	claims, err := suite.manager.ValidateToken("invalid.token.format")
	suite.Error(err)
	suite.Nil(claims)
	
	// 错误的签名
	wrongManager := NewJWTManager("wrong-secret", 1*time.Hour, 24*time.Hour)
	token, _ := wrongManager.GenerateAccessToken(1, "user", "email", "role", "session")
	claims, err = suite.manager.ValidateToken(token)
	suite.Error(err)
	suite.Nil(claims)
}

// 测试过期令牌
func (suite *JWTTestSuite) TestExpiredToken() {
	// 创建一个立即过期的管理器
	expiredManager := NewJWTManager("test-secret-key", -1*time.Hour, -1*time.Hour)
	
	token, _ := expiredManager.GenerateAccessToken(111, "expired", "expired@test.com", "user", "session")
	
	// 验证过期令牌
	claims, err := suite.manager.ValidateToken(token)
	suite.Error(err)
	suite.Nil(claims)
}

// 测试刷新访问令牌
func (suite *JWTTestSuite) TestRefreshAccessToken() {
	userID := uint(222)
	sessionID := "session-222"
	username := "refreshuser"
	email := "refresh@example.com"
	role := "user"
	
	// 生成刷新令牌
	refreshToken, _ := suite.manager.GenerateRefreshToken(userID, sessionID)
	
	// 使用刷新令牌生成新的访问令牌
	newAccessToken, err := suite.manager.RefreshAccessToken(refreshToken, username, email, role)
	suite.NoError(err)
	suite.NotEmpty(newAccessToken)
	
	// 验证新的访问令牌
	claims, err := suite.manager.ValidateToken(newAccessToken)
	suite.NoError(err)
	suite.Equal(userID, claims.UserID)
	suite.Equal(username, claims.Username)
	suite.Equal(email, claims.Email)
	suite.Equal(role, claims.Role)
}

// 测试获取令牌过期时间
func (suite *JWTTestSuite) TestGetTokenExpiry() {
	// 访问令牌过期时间
	accessExpiry := suite.manager.GetTokenExpiry("access")
	suite.Equal(1*time.Hour, accessExpiry)
	
	// 刷新令牌过期时间
	refreshExpiry := suite.manager.GetTokenExpiry("refresh")
	suite.Equal(7*24*time.Hour, refreshExpiry)
	
	// 未知类型默认返回访问令牌过期时间
	unknownExpiry := suite.manager.GetTokenExpiry("unknown")
	suite.Equal(1*time.Hour, unknownExpiry)
}

// 测试令牌类型
func (suite *JWTTestSuite) TestTokenTypes() {
	userID := uint(333)
	sessionID := "session-333"
	
	// 访问令牌
	accessToken, _ := suite.manager.GenerateAccessToken(userID, "user", "email", "role", sessionID)
	accessClaims, _ := suite.manager.ValidateToken(accessToken)
	suite.Equal("access", accessClaims.TokenType)
	
	// 刷新令牌
	refreshToken, _ := suite.manager.GenerateRefreshToken(userID, sessionID)
	refreshClaims, _ := suite.manager.ValidateToken(refreshToken)
	suite.Equal("refresh", refreshClaims.TokenType)
}

// 测试空参数
func (suite *JWTTestSuite) TestEmptyParameters() {
	// 空用户名
	token, err := suite.manager.GenerateAccessToken(1, "", "email", "role", "session")
	suite.NoError(err)
	suite.NotEmpty(token)
	
	// 空邮箱
	token, err = suite.manager.GenerateAccessToken(1, "user", "", "role", "session")
	suite.NoError(err)
	suite.NotEmpty(token)
	
	// 空角色
	token, err = suite.manager.GenerateAccessToken(1, "user", "email", "", "session")
	suite.NoError(err)
	suite.NotEmpty(token)
	
	// 空会话ID
	token, err = suite.manager.GenerateAccessToken(1, "user", "email", "role", "")
	suite.NoError(err)
	suite.NotEmpty(token)
}

// 测试并发生成令牌
func (suite *JWTTestSuite) TestConcurrentTokenGeneration() {
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			userID := uint(id)
			username := fmt.Sprintf("user%d", id)
			email := fmt.Sprintf("user%d@test.com", id)
			sessionID := fmt.Sprintf("session-%d", id)
			
			token, err := suite.manager.GenerateAccessToken(userID, username, email, "user", sessionID)
			suite.NoError(err)
			suite.NotEmpty(token)
			done <- true
		}(i)
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// 测试无效的刷新令牌
func (suite *JWTTestSuite) TestRefreshWithInvalidToken() {
	// 使用访问令牌尝试刷新
	accessToken, _ := suite.manager.GenerateAccessToken(1, "user", "email", "role", "session")
	newToken, err := suite.manager.RefreshAccessToken(accessToken, "user", "email", "role")
	suite.Error(err) // 应该失败，因为不是刷新令牌
	suite.Empty(newToken)
	
	// 使用无效令牌
	newToken, err = suite.manager.RefreshAccessToken("invalid.token", "user", "email", "role")
	suite.Error(err)
	suite.Empty(newToken)
}

// 测试令牌的标准声明
func (suite *JWTTestSuite) TestStandardClaims() {
	token, _ := suite.manager.GenerateAccessToken(1, "user", "email", "role", "session")
	claims, _ := suite.manager.ValidateToken(token)
	
	// 验证标准声明 - JWT使用Unix时间戳
	suite.NotNil(claims.IssuedAt)
	suite.NotNil(claims.ExpiresAt)
	
	// 比较Unix时间戳
	issuedTime := claims.IssuedAt.Unix()
	expiresTime := claims.ExpiresAt.Unix()
	suite.Greater(expiresTime, issuedTime)
}

func TestJWTSuite(t *testing.T) {
	suite.Run(t, new(JWTTestSuite))
}