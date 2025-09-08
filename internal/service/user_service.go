package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"github.com/wfunc/slot-game/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// userService 用户服务实现
type userService struct {
	db          *gorm.DB
	userRepo    repository.UserRepository
	authRepo    repository.UserAuthRepository
	walletRepo  repository.WalletRepository
	gameRepo    repository.GameResultRepository
	log         *zap.Logger
}

// NewUserService 创建用户服务
func NewUserService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	authRepo repository.UserAuthRepository,
	walletRepo repository.WalletRepository,
	gameRepo repository.GameResultRepository,
	log *zap.Logger,
) UserService {
	return &userService{
		db:         db,
		userRepo:   userRepo,
		authRepo:   authRepo,
		walletRepo: walletRepo,
		gameRepo:   gameRepo,
		log:        log,
	}
}

// GetUserByID 根据ID获取用户
func (s *userService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user by ID", zap.Error(err), zap.Uint("userID", userID))
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *userService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		s.log.Error("Failed to get user by username", zap.Error(err), zap.String("username", username))
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.log.Error("Failed to get user by email", zap.Error(err), zap.String("email", email))
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

// GetUserByPhone 根据手机号获取用户
func (s *userService) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	user, err := s.userRepo.FindByPhone(ctx, phone)
	if err != nil {
		s.log.Error("Failed to get user by phone", zap.Error(err), zap.String("phone", phone))
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

// UpdateUser 更新用户信息
func (s *userService) UpdateUser(ctx context.Context, userID uint, updates map[string]interface{}) error {
	// 获取用户
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}
	
	// 应用更新（只更新允许的字段）
	if nickname, ok := updates["nickname"].(string); ok {
		user.Nickname = nickname
	}
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	if phone, ok := updates["phone"].(string); ok {
		user.Phone = phone
	}
	if avatar, ok := updates["avatar"].(string); ok {
		user.Avatar = avatar
	}
	if status, ok := updates["status"].(string); ok {
		user.Status = status
	}
	
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update user", zap.Error(err), zap.Uint("userID", userID))
		return fmt.Errorf("更新用户失败: %w", err)
	}
	
	s.log.Info("User updated successfully", zap.Uint("userID", userID), zap.Any("updates", updates))
	return nil
}

// UpdatePassword 更新密码
func (s *userService) UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// 获取认证信息
	auth, err := s.authRepo.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取认证信息失败: %w", err)
	}
	
	// 验证旧密码
	valid, err := utils.VerifyPassword(oldPassword, auth.Password)
	if err != nil || !valid {
		return errors.New("旧密码不正确")
	}
	
	// 验证新密码
	if len(newPassword) < 6 {
		return errors.New("新密码长度至少6个字符")
	}
	
	// 加密新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	
	// 更新密码
	if err := s.authRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		s.log.Error("Failed to update password", zap.Error(err), zap.Uint("userID", userID))
		return fmt.Errorf("更新密码失败: %w", err)
	}
	
	s.log.Info("Password updated successfully", zap.Uint("userID", userID))
	return nil
}

// UpdateProfile 更新用户资料
func (s *userService) UpdateProfile(ctx context.Context, userID uint, profile *UserProfile) error {
	// 获取用户
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("获取用户失败: %w", err)
	}
	
	// 应用更新
	updated := false
	if profile.Nickname != "" {
		user.Nickname = profile.Nickname
		updated = true
	}
	if profile.Avatar != "" {
		user.Avatar = profile.Avatar
		updated = true
	}
	// Note: Gender, Birthday, City, Signature are in UserProfile table, not User table
	// This would need a separate UserProfile repository to update properly
	// For now, just update what's in the User model
	
	if !updated {
		return errors.New("没有需要更新的内容")
	}
	
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("Failed to update profile", zap.Error(err), zap.Uint("userID", userID))
		return fmt.Errorf("更新资料失败: %w", err)
	}
	
	s.log.Info("Profile updated successfully", zap.Uint("userID", userID))
	return nil
}

// GetUserList 获取用户列表
func (s *userService) GetUserList(ctx context.Context, page, pageSize int) ([]*models.User, int64, error) {
	pagination := repository.NewPagination(page, pageSize)
	users, err := s.userRepo.GetAll(ctx, pagination)
	if err != nil {
		s.log.Error("Failed to get user list", zap.Error(err))
		return nil, 0, fmt.Errorf("获取用户列表失败: %w", err)
	}
	return users, pagination.Total, nil
}

// SearchUsers 搜索用户
func (s *userService) SearchUsers(ctx context.Context, query string, page, pageSize int) ([]*models.User, int64, error) {
	// TODO: Implement search functionality
	// For now, just return all users (search needs to be implemented in repository)
	pagination := repository.NewPagination(page, pageSize)
	users, err := s.userRepo.GetAll(ctx, pagination)
	if err != nil {
		s.log.Error("Failed to search users", zap.Error(err), zap.String("query", query))
		return nil, 0, fmt.Errorf("搜索用户失败: %w", err)
	}
	return users, pagination.Total, nil
}

// UpdateUserStatus 更新用户状态
func (s *userService) UpdateUserStatus(ctx context.Context, userID uint, status string) error {
	validStatuses := map[string]bool{
		"active":   true,
		"inactive": true,
		"banned":   true,
		"frozen":   true,
	}
	
	if !validStatuses[status] {
		return errors.New("无效的状态")
	}
	
	// 使用仓储层的 UpdateStatus 方法
	if err := s.userRepo.UpdateStatus(ctx, userID, status); err != nil {
		s.log.Error("Failed to update user status", zap.Error(err), zap.Uint("userID", userID), zap.String("status", status))
		return fmt.Errorf("更新状态失败: %w", err)
	}
	
	s.log.Info("User status updated", zap.Uint("userID", userID), zap.String("status", status))
	return nil
}

// BanUser 封禁用户
func (s *userService) BanUser(ctx context.Context, userID uint, reason string, duration time.Duration) error {
	// 更新用户状态为 banned
	if err := s.userRepo.UpdateStatus(ctx, userID, "banned"); err != nil {
		s.log.Error("Failed to ban user", zap.Error(err), zap.Uint("userID", userID))
		return fmt.Errorf("封禁用户失败: %w", err)
	}
	
	// TODO: Store ban reason and duration in a separate table if needed
	
	s.log.Info("User banned", zap.Uint("userID", userID), zap.String("reason", reason), zap.Duration("duration", duration))
	return nil
}

// UnbanUser 解封用户
func (s *userService) UnbanUser(ctx context.Context, userID uint) error {
	// 更新用户状态为 active
	if err := s.userRepo.UpdateStatus(ctx, userID, "active"); err != nil {
		s.log.Error("Failed to unban user", zap.Error(err), zap.Uint("userID", userID))
		return fmt.Errorf("解封用户失败: %w", err)
	}
	
	s.log.Info("User unbanned", zap.Uint("userID", userID))
	return nil
}

// GetUserStats 获取用户统计
func (s *userService) GetUserStats(ctx context.Context, userID uint) (*UserStats, error) {
	// 获取用户信息
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	
	// 获取钱包信息
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		s.log.Warn("Failed to get wallet", zap.Error(err), zap.Uint("userID", userID))
	}
	
	// 获取游戏统计
	pagination := &repository.Pagination{
		Page:     1,
		PageSize: 1000, // 获取最近1000条记录统计
	}
	results, err := s.gameRepo.FindWinsByUserID(ctx, userID, pagination)
	if err != nil {
		s.log.Warn("Failed to get game results", zap.Error(err), zap.Uint("userID", userID))
	}
	
	// 计算统计数据
	stats := &UserStats{
		Level:       user.Level,
		Experience:  user.Experience,
		VIPLevel:    user.VipLevel,
		LastLoginAt: func() time.Time {
			if user.LastLoginAt != nil {
				return *user.LastLoginAt
			}
			return time.Time{}
		}(),
	}
	
	if wallet != nil {
		stats.TotalBet = wallet.TotalBet
		stats.TotalWinAmount = wallet.TotalWin
	}
	
	if len(results) > 0 {
		stats.TotalGames = len(results)
		for _, result := range results {
			if result.WinAmount > 0 {
				stats.TotalWins++
			}
		}
		if stats.TotalGames > 0 {
			stats.WinRate = float64(stats.TotalWins) / float64(stats.TotalGames) * 100
		}
	}
	
	return stats, nil
}

// GetUserGameHistory 获取用户游戏历史
func (s *userService) GetUserGameHistory(ctx context.Context, userID uint, page, pageSize int) ([]*models.GameResult, int64, error) {
	pagination := &repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}
	
	results, err := s.gameRepo.FindWinsByUserID(ctx, userID, pagination)
	if err != nil {
		s.log.Error("Failed to get game history", zap.Error(err), zap.Uint("userID", userID))
		return nil, 0, fmt.Errorf("获取游戏历史失败: %w", err)
	}
	
	return results, pagination.Total, nil
}