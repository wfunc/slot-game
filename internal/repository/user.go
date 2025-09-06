package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	BaseRepository
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	GetAll(ctx context.Context, pagination *Pagination) ([]*models.User, error)
	UpdateLastLogin(ctx context.Context, userID uint) error
	UpdateStatus(ctx context.Context, userID uint, status string) error
}

// userRepo 用户仓储实现
type userRepo struct {
	*BaseRepo
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建用户
func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户（软删除）
func (r *userRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// FindByID 根据ID查找用户
func (r *userRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername 根据用户名查找
func (r *userRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查找
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// FindByPhone 根据手机号查找
func (r *userRepo) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// GetAll 获取所有用户（分页）
func (r *userRepo) GetAll(ctx context.Context, pagination *Pagination) ([]*models.User, error) {
	var users []*models.User
	query := r.db.WithContext(ctx).Model(&models.User{})
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&users).Error
	
	return users, err
}

// UpdateLastLogin 更新最后登录时间
func (r *userRepo) UpdateLastLogin(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", now).Error
}

// UpdateStatus 更新用户状态
func (r *userRepo) UpdateStatus(ctx context.Context, userID uint, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status).Error
}

// WithTx 使用事务
func (r *userRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &userRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// UserAuthRepository 用户认证仓储接口
type UserAuthRepository interface {
	BaseRepository
	Create(ctx context.Context, auth *models.UserAuth) error
	Update(ctx context.Context, auth *models.UserAuth) error
	FindByUserID(ctx context.Context, userID uint) (*models.UserAuth, error)
	UpdatePassword(ctx context.Context, userID uint, hashedPassword string) error
	UpdateLoginAttempts(ctx context.Context, userID uint, attempts int) error
	ResetLoginAttempts(ctx context.Context, userID uint) error
	LockAccount(ctx context.Context, userID uint, until time.Time) error
}

// userAuthRepo 用户认证仓储实现
type userAuthRepo struct {
	*BaseRepo
}

// NewUserAuthRepository 创建用户认证仓储
func NewUserAuthRepository(db *gorm.DB) UserAuthRepository {
	return &userAuthRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建用户认证信息
func (r *userAuthRepo) Create(ctx context.Context, auth *models.UserAuth) error {
	return r.db.WithContext(ctx).Create(auth).Error
}

// Update 更新用户认证信息
func (r *userAuthRepo) Update(ctx context.Context, auth *models.UserAuth) error {
	return r.db.WithContext(ctx).Save(auth).Error
}

// FindByUserID 根据用户ID查找认证信息
func (r *userAuthRepo) FindByUserID(ctx context.Context, userID uint) (*models.UserAuth, error) {
	var auth models.UserAuth
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("认证信息不存在")
		}
		return nil, err
	}
	return &auth, nil
}

// UpdatePassword 更新密码
func (r *userAuthRepo) UpdatePassword(ctx context.Context, userID uint, hashedPassword string) error {
	return r.db.WithContext(ctx).
		Model(&models.UserAuth{}).
		Where("user_id = ?", userID).
		Update("password", hashedPassword).Error
}

// UpdateLoginAttempts 更新登录尝试次数
func (r *userAuthRepo) UpdateLoginAttempts(ctx context.Context, userID uint, attempts int) error {
	return r.db.WithContext(ctx).
		Model(&models.UserAuth{}).
		Where("user_id = ?", userID).
		Update("login_attempts", attempts).Error
}

// ResetLoginAttempts 重置登录尝试次数
func (r *userAuthRepo) ResetLoginAttempts(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.UserAuth{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"login_attempts": 0,
			"locked_until": nil,
		}).Error
}

// LockAccount 锁定账户
func (r *userAuthRepo) LockAccount(ctx context.Context, userID uint, until time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.UserAuth{}).
		Where("user_id = ?", userID).
		Update("locked_until", until).Error
}

// WithTx 使用事务
func (r *userAuthRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &userAuthRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// UserSessionRepository 用户会话仓储接口
type UserSessionRepository interface {
	BaseRepository
	Create(ctx context.Context, session *models.UserSession) error
	FindByToken(ctx context.Context, token string) (*models.UserSession, error)
	FindByUserID(ctx context.Context, userID uint) ([]*models.UserSession, error)
	UpdateLastActive(ctx context.Context, token string) error
	Delete(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID uint) error
	CleanupExpired(ctx context.Context) error
}

// userSessionRepo 用户会话仓储实现
type userSessionRepo struct {
	*BaseRepo
}

// NewUserSessionRepository 创建用户会话仓储
func NewUserSessionRepository(db *gorm.DB) UserSessionRepository {
	return &userSessionRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建会话
func (r *userSessionRepo) Create(ctx context.Context, session *models.UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// FindByToken 根据令牌查找会话
func (r *userSessionRepo) FindByToken(ctx context.Context, token string) (*models.UserSession, error) {
	var session models.UserSession
	err := r.db.WithContext(ctx).
		Where("token = ? AND expire_at > ?", token, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在或已过期")
		}
		return nil, err
	}
	return &session, nil
}

// FindByUserID 查找用户的所有会话
func (r *userSessionRepo) FindByUserID(ctx context.Context, userID uint) ([]*models.UserSession, error) {
	var sessions []*models.UserSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND expire_at > ?", userID, time.Now()).
		Find(&sessions).Error
	return sessions, err
}

// UpdateLastActive 更新最后活动时间
func (r *userSessionRepo) UpdateLastActive(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Model(&models.UserSession{}).
		Where("token = ?", token).
		Update("last_active_at", time.Now()).Error
}

// Delete 删除会话
func (r *userSessionRepo) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&models.UserSession{}).Error
}

// DeleteByUserID 删除用户的所有会话
func (r *userSessionRepo) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.UserSession{}).Error
}

// CleanupExpired 清理过期会话
func (r *userSessionRepo) CleanupExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expire_at < ?", time.Now()).
		Delete(&models.UserSession{}).Error
}

// WithTx 使用事务
func (r *userSessionRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &userSessionRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}