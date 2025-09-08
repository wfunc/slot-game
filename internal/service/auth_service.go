package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"github.com/wfunc/slot-game/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserExists         = errors.New("用户已存在")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrUserBanned         = errors.New("用户已被封禁")
	ErrSessionNotFound    = errors.New("会话不存在")
	ErrInvalidToken       = errors.New("无效的令牌")
	ErrTokenExpired       = errors.New("令牌已过期")
)

// authService 认证服务实现
type authService struct {
	db              *gorm.DB
	userRepo        repository.UserRepository
	authRepo        repository.UserAuthRepository
	sessionRepo     repository.UserSessionRepository
	walletRepo      repository.WalletRepository
	jwtManager      *utils.JWTManager
	log             *zap.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	authRepo repository.UserAuthRepository,
	sessionRepo repository.UserSessionRepository,
	walletRepo repository.WalletRepository,
	jwtManager *utils.JWTManager,
	log *zap.Logger,
) AuthService {
	return &authService{
		db:          db,
		userRepo:    userRepo,
		authRepo:    authRepo,
		sessionRepo: sessionRepo,
		walletRepo:  walletRepo,
		jwtManager:  jwtManager,
		log:         log,
	}
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// 验证输入
	if err := s.validateRegisterRequest(req); err != nil {
		return nil, err
	}
	
	// 检查用户是否已存在
	if user, _ := s.userRepo.FindByUsername(ctx, req.Username); user != nil {
		return nil, fmt.Errorf("用户名已存在")
	}
	if user, _ := s.userRepo.FindByEmail(ctx, req.Email); user != nil {
		return nil, fmt.Errorf("邮箱已被使用")
	}
	if user, _ := s.userRepo.FindByPhone(ctx, req.Phone); user != nil {
		return nil, fmt.Errorf("手机号已被使用")
	}
	
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 创建用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Phone:    req.Phone,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Status:   "active",
		Level:    1,
	}
	
	if user.Nickname == "" {
		user.Nickname = req.Username
	}
	
	if err := s.userRepo.WithTx(tx).(repository.UserRepository).Create(ctx, user); err != nil {
		tx.Rollback()
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	
	// 创建认证信息
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}
	
	auth := &models.UserAuth{
		UserID:   user.ID,
		Password: hashedPassword,
	}
	
	if err := s.authRepo.WithTx(tx).(repository.UserAuthRepository).Create(ctx, auth); err != nil {
		tx.Rollback()
		s.log.Error("Failed to create auth", zap.Error(err))
		return nil, fmt.Errorf("创建认证信息失败: %w", err)
	}
	
	// 创建钱包
	wallet := &models.Wallet{
		UserID:  user.ID,
		Balance: 0,
		Coins:   1000, // 注册赠送1000游戏币
		Points:  0,
	}
	
	if err := s.walletRepo.WithTx(tx).(repository.WalletRepository).Create(ctx, wallet); err != nil {
		tx.Rollback()
		s.log.Error("Failed to create wallet", zap.Error(err))
		return nil, fmt.Errorf("创建钱包失败: %w", err)
	}
	
	// 创建会话
	sessionID, err := utils.GenerateSessionID()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("生成会话ID失败: %w", err)
	}
	
	session := &models.UserSession{
		UserID:    user.ID,
		SessionID: sessionID,
		Token:     sessionID, // Using sessionID as token for now
		IP:        req.IP,
		UserAgent: "", // Device info not available during registration
		ExpireAt:  time.Now().Add(30 * 24 * time.Hour),
	}
	
	if err := s.sessionRepo.WithTx(tx).(repository.UserSessionRepository).Create(ctx, session); err != nil {
		tx.Rollback()
		s.log.Error("Failed to create session", zap.Error(err))
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	
	// 生成JWT令牌
	accessToken, err := s.jwtManager.GenerateAccessToken(
		user.ID, user.Username, user.Email, "user", sessionID)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}
	
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}
	
	s.log.Info("User registered successfully", zap.Uint("userID", user.ID), zap.String("username", user.Username))
	
	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetTokenExpiry("access").Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// 查找用户（支持用户名、邮箱、手机号登录）
	var user *models.User
	var err error
	
	if strings.Contains(req.Account, "@") {
		user, err = s.userRepo.FindByEmail(ctx, req.Account)
	} else if regexp.MustCompile(`^\d{11}$`).MatchString(req.Account) {
		user, err = s.userRepo.FindByPhone(ctx, req.Account)
	} else {
		user, err = s.userRepo.FindByUsername(ctx, req.Account)
	}
	
	if err != nil || user == nil {
		s.log.Warn("Login failed: user not found", zap.String("account", req.Account))
		return nil, ErrInvalidCredentials
	}
	
	// 检查用户状态
	if user.Status == "banned" {
		return nil, ErrUserBanned
	}
	
	// 获取认证信息
	auth, err := s.authRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		s.log.Error("Failed to get auth info", zap.Error(err), zap.Uint("userID", user.ID))
		return nil, ErrInvalidCredentials
	}
	
	// 验证密码
	valid, err := utils.VerifyPassword(req.Password, auth.Password)
	if err != nil || !valid {
		s.log.Warn("Login failed: invalid password", zap.Uint("userID", user.ID))
		// 记录失败次数
		_ = s.authRepo.UpdateLoginAttempts(ctx, user.ID, auth.LoginAttempts + 1)
		return nil, ErrInvalidCredentials
	}
	
	// 创建会话
	sessionID, err := utils.GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("生成会话ID失败: %w", err)
	}
	
	session := &models.UserSession{
		UserID:    user.ID,
		SessionID: sessionID,
		Token:     sessionID, // Using sessionID as token for now
		IP:        req.IP,
		UserAgent: req.Device,
		ExpireAt:  time.Now().Add(30 * 24 * time.Hour),
	}
	
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		s.log.Error("Failed to create session", zap.Error(err))
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	
	// 更新登录信息
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = req.IP
	_ = s.userRepo.Update(ctx, user)
	
	// 重置失败次数
	_ = s.authRepo.ResetLoginAttempts(ctx, user.ID)
	
	// 生成JWT令牌
	accessToken, err := s.jwtManager.GenerateAccessToken(
		user.ID, user.Username, user.Email, "user", sessionID)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}
	
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}
	
	s.log.Info("User logged in successfully", zap.Uint("userID", user.ID), zap.String("username", user.Username))
	
	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetTokenExpiry("access").Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// Logout 用户登出
func (s *authService) Logout(ctx context.Context, userID uint, token string) error {
	// 验证令牌
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return ErrInvalidToken
	}
	
	// 删除会话
	if err := s.sessionRepo.Delete(ctx, claims.SessionID); err != nil {
		s.log.Error("Failed to delete session", zap.Error(err), zap.String("sessionID", claims.SessionID))
		return fmt.Errorf("删除会话失败: %w", err)
	}
	
	s.log.Info("User logged out successfully", zap.Uint("userID", userID))
	return nil
}

// RefreshToken 刷新令牌
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// 验证刷新令牌
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}
	
	if claims.TokenType != "refresh" {
		return nil, errors.New("不是刷新令牌")
	}
	
	// 检查会话是否有效
	session, err := s.sessionRepo.FindByToken(ctx, claims.SessionID)
	if err != nil || session == nil {
		return nil, ErrSessionNotFound
	}
	
	if session.ExpireAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}
	
	// 获取用户信息
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	
	// 生成新的访问令牌
	accessToken, err := s.jwtManager.GenerateAccessToken(
		user.ID, user.Username, user.Email, "user", claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}
	
	s.log.Info("Token refreshed successfully", zap.Uint("userID", user.ID))
	
	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetTokenExpiry("access").Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken 验证令牌
func (s *authService) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	
	// 检查会话是否有效
	session, err := s.sessionRepo.FindByToken(ctx, claims.SessionID)
	if err != nil || session == nil {
		return nil, ErrSessionNotFound
	}
	
	if session.ExpireAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}
	
	return &TokenClaims{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID,
		IssuedAt:  claims.IssuedAt.Unix(),
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// ValidateSession 验证会话
func (s *authService) ValidateSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	session, err := s.sessionRepo.FindByToken(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	if session.ExpireAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}
	
	return session, nil
}

// GetActiveSessions 获取活跃会话
func (s *authService) GetActiveSessions(ctx context.Context, userID uint) ([]*models.UserSession, error) {
	return s.sessionRepo.FindByUserID(ctx, userID)
}

// RevokeSession 撤销会话
func (s *authService) RevokeSession(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// RevokeAllSessions 撤销所有会话
func (s *authService) RevokeAllSessions(ctx context.Context, userID uint) error {
	sessions, err := s.sessionRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	
	for _, session := range sessions {
		if err := s.sessionRepo.Delete(ctx, session.SessionID); err != nil {
			s.log.Error("Failed to revoke session", zap.Error(err), zap.String("sessionID", session.SessionID))
		}
	}
	
	return nil
}

// ResetPasswordRequest 请求重置密码
func (s *authService) ResetPasswordRequest(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return ErrUserNotFound
	}
	
	// 生成重置令牌
	resetToken, err := utils.GenerateRandomString(32)
	if err != nil {
		return fmt.Errorf("生成重置令牌失败: %w", err)
	}
	
	// 保存令牌（这里应该保存到缓存或数据库）
	// TODO: 实现令牌存储逻辑
	_ = resetToken // 暂时未使用，待实现令牌存储
	
	// 发送邮件（这里应该调用邮件服务）
	// TODO: 实现邮件发送逻辑
	
	s.log.Info("Password reset requested", zap.Uint("userID", user.ID), zap.String("email", email))
	
	return nil
}

// ResetPassword 重置密码
func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// 验证令牌（这里应该从缓存或数据库获取）
	// TODO: 实现令牌验证逻辑
	
	// 获取用户ID（从令牌中）
	var userID uint = 0 // TODO: 从令牌获取
	
	// 更新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	
	if err := s.authRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	
	// 撤销所有会话
	_ = s.RevokeAllSessions(ctx, userID)
	
	s.log.Info("Password reset successfully", zap.Uint("userID", userID))
	
	return nil
}

// VerifyEmail 验证邮箱
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	// TODO: 实现邮箱验证逻辑
	return nil
}

// OAuthLogin OAuth登录
func (s *authService) OAuthLogin(ctx context.Context, provider string, code string) (*AuthResponse, error) {
	// TODO: 实现OAuth登录逻辑
	return nil, errors.New("OAuth登录暂未实现")
}

// BindOAuthAccount 绑定OAuth账号
func (s *authService) BindOAuthAccount(ctx context.Context, userID uint, provider string, oauthID string) error {
	// TODO: 实现OAuth账号绑定逻辑
	return errors.New("OAuth账号绑定暂未实现")
}

// validateRegisterRequest 验证注册请求
func (s *authService) validateRegisterRequest(req *RegisterRequest) error {
	// 验证用户名
	if len(req.Username) < 3 || len(req.Username) > 20 {
		return errors.New("用户名长度必须在3-20个字符之间")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(req.Username) {
		return errors.New("用户名只能包含字母、数字和下划线")
	}
	
	// 验证邮箱
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(req.Email) {
		return errors.New("邮箱格式不正确")
	}
	
	// 验证手机号
	if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
		return errors.New("手机号格式不正确")
	}
	
	// 验证密码
	if len(req.Password) < 6 {
		return errors.New("密码长度至少6个字符")
	}
	
	if req.Password != req.ConfirmPassword {
		return errors.New("两次输入的密码不一致")
	}
	
	return nil
}