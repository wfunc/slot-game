package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// JWTClaims 自定义JWT Claims
type JWTClaims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	TokenType string `json:"token_type"` // access or refresh
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secretKey          string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:          secretKey,
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

// GenerateAccessToken 生成访问令牌
func (j *JWTManager) GenerateAccessToken(userID uint, username, email, role, sessionID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(j.accessTokenExpiry)
	
	claims := &JWTClaims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "slot-game",
			Subject:   username,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// GenerateRefreshToken 生成刷新令牌
func (j *JWTManager) GenerateRefreshToken(userID uint, sessionID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(j.refreshTokenExpiry)
	
	claims := &JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "slot-game",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken 验证令牌
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	
	// 检查是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrExpiredToken
	}
	
	return claims, nil
}

// RefreshAccessToken 使用刷新令牌生成新的访问令牌
func (j *JWTManager) RefreshAccessToken(refreshToken string, username, email, role string) (string, error) {
	claims, err := j.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}
	
	// 确保是刷新令牌
	if claims.TokenType != "refresh" {
		return "", errors.New("not a refresh token")
	}
	
	// 生成新的访问令牌
	return j.GenerateAccessToken(claims.UserID, username, email, role, claims.SessionID)
}

// GetTokenExpiry 获取令牌过期时间
func (j *JWTManager) GetTokenExpiry(tokenType string) time.Duration {
	if tokenType == "refresh" {
		return j.refreshTokenExpiry
	}
	return j.accessTokenExpiry
}