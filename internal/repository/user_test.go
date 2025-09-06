package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// UserRepositoryTestSuite 用户仓储测试套件
type UserRepositoryTestSuite struct {
	suite.Suite
	db       *gorm.DB
	repo     UserRepository
	authRepo UserAuthRepository
	sessRepo UserSessionRepository
}

func (suite *UserRepositoryTestSuite) SetupTest() {
	suite.db = SetupTestDB()
	suite.repo = NewUserRepository(suite.db)
	suite.authRepo = NewUserAuthRepository(suite.db)
	suite.sessRepo = NewUserSessionRepository(suite.db)
}

func (suite *UserRepositoryTestSuite) TearDownTest() {
	CleanupTestDB(suite.db)
}

// TestUserRepository_Create 测试创建用户
func (suite *UserRepositoryTestSuite) TestUserRepository_Create() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Phone:    "13800138000",
		Nickname: "Test User",
		Avatar:   "avatar.jpg",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), user.ID)
	
	// 验证数据
	found, err := suite.repo.FindByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, found.Username)
	assert.Equal(suite.T(), user.Email, found.Email)
}

// TestUserRepository_FindByUsername 测试根据用户名查找
func (suite *UserRepositoryTestSuite) TestUserRepository_FindByUsername() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "findbyusername",
		Email:    "findby@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	found, err := suite.repo.FindByUsername(ctx, "findbyusername")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
	
	// 测试不存在的用户
	_, err = suite.repo.FindByUsername(ctx, "notexist")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "用户不存在")
}

// TestUserRepository_FindByEmail 测试根据邮箱查找
func (suite *UserRepositoryTestSuite) TestUserRepository_FindByEmail() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "emailuser",
		Email:    "email@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	found, err := suite.repo.FindByEmail(ctx, "email@example.com")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
}

// TestUserRepository_FindByPhone 测试根据手机号查找
func (suite *UserRepositoryTestSuite) TestUserRepository_FindByPhone() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "phoneuser",
		Email:    "phone@example.com",
		Phone:    "13900139000",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	found, err := suite.repo.FindByPhone(ctx, "13900139000")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
}

// TestUserRepository_Update 测试更新用户
func (suite *UserRepositoryTestSuite) TestUserRepository_Update() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "updateuser",
		Email:    "update@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 更新用户信息
	user.Nickname = "Updated User"
	user.Avatar = "new-avatar.jpg"
	err = suite.repo.Update(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 验证更新
	found, err := suite.repo.FindByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated User", found.Nickname)
	assert.Equal(suite.T(), "new-avatar.jpg", found.Avatar)
}

// TestUserRepository_UpdateLastLogin 测试更新最后登录时间
func (suite *UserRepositoryTestSuite) TestUserRepository_UpdateLastLogin() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "loginuser",
		Email:    "login@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	err = suite.repo.UpdateLastLogin(ctx, user.ID)
	assert.NoError(suite.T(), err)
	
	found, err := suite.repo.FindByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), found.LastLoginAt)
	assert.WithinDuration(suite.T(), time.Now(), *found.LastLoginAt, 2*time.Second)
}

// TestUserRepository_UpdateStatus 测试更新用户状态
func (suite *UserRepositoryTestSuite) TestUserRepository_UpdateStatus() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "statususer",
		Email:    "status@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	err = suite.repo.UpdateStatus(ctx, user.ID, "suspended")
	assert.NoError(suite.T(), err)
	
	found, err := suite.repo.FindByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "suspended", found.Status)
}

// TestUserRepository_GetAll 测试获取所有用户（分页）
func (suite *UserRepositoryTestSuite) TestUserRepository_GetAll() {
	ctx := context.Background()
	
	// 创建多个用户
	for i := 0; i < 5; i++ {
		user := &models.User{
			Username: fmt.Sprintf("user%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Phone:    fmt.Sprintf("1380000000%d", i),
			Status:   "active",
		}
		err := suite.repo.Create(ctx, user)
		assert.NoError(suite.T(), err)
	}
	
	// 测试分页
	pagination := &Pagination{
		Page:     1,
		PageSize: 3,
	}
	
	users, err := suite.repo.GetAll(ctx, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), users, 3)
	assert.Equal(suite.T(), int64(5), pagination.Total)
}

// TestUserRepository_Delete 测试删除用户
func (suite *UserRepositoryTestSuite) TestUserRepository_Delete() {
	ctx := context.Background()
	
	user := &models.User{
		Username: "deleteuser",
		Email:    "delete@example.com",
		Status:   "active",
	}
	
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	err = suite.repo.Delete(ctx, user.ID)
	assert.NoError(suite.T(), err)
	
	// 软删除后应该找不到
	_, err = suite.repo.FindByID(ctx, user.ID)
	assert.Error(suite.T(), err)
}

// TestUserAuthRepository_Create 测试创建用户认证
func (suite *UserRepositoryTestSuite) TestUserAuthRepository_Create() {
	ctx := context.Background()
	
	// 先创建用户
	user := &models.User{
		Username: "authuser",
		Email:    "auth@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 创建认证信息
	auth := &models.UserAuth{
		UserID:       user.ID,
		Password:     "hashed_password",
		TwoFactorEnabled: false,
		LoginAttempts: 0,
	}
	
	err = suite.authRepo.Create(ctx, auth)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), auth.ID)
}

// TestUserAuthRepository_FindByUserID 测试查找用户认证信息
func (suite *UserRepositoryTestSuite) TestUserAuthRepository_FindByUserID() {
	ctx := context.Background()
	
	// 创建用户和认证信息
	user := &models.User{
		Username: "findauthuser",
		Email:    "findauth@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	auth := &models.UserAuth{
		UserID:   user.ID,
		Password: "hashed_password",
	}
	err = suite.authRepo.Create(ctx, auth)
	assert.NoError(suite.T(), err)
	
	found, err := suite.authRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), auth.ID, found.ID)
}

// TestUserAuthRepository_UpdatePassword 测试更新密码
func (suite *UserRepositoryTestSuite) TestUserAuthRepository_UpdatePassword() {
	ctx := context.Background()
	
	// 创建用户和认证信息
	user := &models.User{
		Username: "pwduser",
		Email:    "pwd@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	auth := &models.UserAuth{
		UserID:   user.ID,
		Password: "old_password",
	}
	err = suite.authRepo.Create(ctx, auth)
	assert.NoError(suite.T(), err)
	
	// 更新密码
	err = suite.authRepo.UpdatePassword(ctx, user.ID, "new_password")
	assert.NoError(suite.T(), err)
	
	found, err := suite.authRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "new_password", found.Password)
	// Note: PasswordChangedAt field doesn't exist in UserAuth model
}

// TestUserAuthRepository_UpdateLoginAttempts 测试更新登录尝试次数
func (suite *UserRepositoryTestSuite) TestUserAuthRepository_UpdateLoginAttempts() {
	ctx := context.Background()
	
	// 创建用户和认证信息
	user := &models.User{
		Username: "attemptuser",
		Email:    "attempt@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	auth := &models.UserAuth{
		UserID:        user.ID,
		Password:      "password",
		LoginAttempts: 0,
	}
	err = suite.authRepo.Create(ctx, auth)
	assert.NoError(suite.T(), err)
	
	// 更新尝试次数
	err = suite.authRepo.UpdateLoginAttempts(ctx, user.ID, 3)
	assert.NoError(suite.T(), err)
	
	found, err := suite.authRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, found.LoginAttempts)
	
	// 重置尝试次数
	err = suite.authRepo.ResetLoginAttempts(ctx, user.ID)
	assert.NoError(suite.T(), err)
	
	found, err = suite.authRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 0, found.LoginAttempts)
}

// TestUserAuthRepository_LockAccount 测试锁定账户
func (suite *UserRepositoryTestSuite) TestUserAuthRepository_LockAccount() {
	ctx := context.Background()
	
	// 创建用户和认证信息
	user := &models.User{
		Username: "lockuser",
		Email:    "lock@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	auth := &models.UserAuth{
		UserID:   user.ID,
		Password: "password",
	}
	err = suite.authRepo.Create(ctx, auth)
	assert.NoError(suite.T(), err)
	
	// 锁定账户
	lockUntil := time.Now().Add(time.Hour)
	err = suite.authRepo.LockAccount(ctx, user.ID, lockUntil)
	assert.NoError(suite.T(), err)
	
	found, err := suite.authRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), found.LockedUntil)
	assert.Equal(suite.T(), lockUntil.Unix(), found.LockedUntil.Unix())
}

// TestUserSessionRepository_Create 测试创建会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_Create() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "sessuser",
		Email:    "sess@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 创建会话
	session := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_create_%d", time.Now().UnixNano()),
		Token:     "test_token_123",
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		ExpireAt: time.Now().Add(24 * time.Hour),
	}
	
	err = suite.sessRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), session.ID)
}

// TestUserSessionRepository_FindByToken 测试根据令牌查找会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_FindByToken() {
	ctx := context.Background()
	
	// 创建用户和会话
	user := &models.User{
		Username: "tokenuser",
		Email:    "token@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	session := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_find_%d", time.Now().UnixNano()),
		Token:     "find_token_456",
		ExpireAt: time.Now().Add(24 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	
	found, err := suite.sessRepo.FindByToken(ctx, "find_token_456")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), session.ID, found.ID)
	
	// 测试过期的会话
	expiredSession := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_expired_%d", time.Now().UnixNano()),
		Token:     "expired_token",
		ExpireAt: time.Now().Add(-1 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, expiredSession)
	assert.NoError(suite.T(), err)
	
	_, err = suite.sessRepo.FindByToken(ctx, "expired_token")
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "会话不存在或已过期")
}

// TestUserSessionRepository_FindByUserID 测试查找用户的所有会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_FindByUserID() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "multisessuser",
		Email:    "multisess@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 创建多个会话
	for i := 0; i < 3; i++ {
		session := &models.UserSession{
			UserID:    user.ID,
			SessionID: fmt.Sprintf("session_user_%d_%d", i, time.Now().UnixNano()),
			Token:     fmt.Sprintf("token_%d", i),
			ExpireAt: time.Now().Add(24 * time.Hour),
		}
		err = suite.sessRepo.Create(ctx, session)
		assert.NoError(suite.T(), err)
	}
	
	sessions, err := suite.sessRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), sessions, 3)
}

// TestUserSessionRepository_UpdateLastActive 测试更新最后活动时间
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_UpdateLastActive() {
	ctx := context.Background()
	
	// 创建用户和会话
	user := &models.User{
		Username: "activeuser",
		Email:    "active@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	session := &models.UserSession{
		UserID:       user.ID,
		SessionID:    fmt.Sprintf("session_active_%d", time.Now().UnixNano()),
		Token:        "active_token",
		ExpireAt:    time.Now().Add(24 * time.Hour),
		LastActiveAt: time.Now().Add(-1 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	
	err = suite.sessRepo.UpdateLastActive(ctx, "active_token")
	assert.NoError(suite.T(), err)
	
	found, err := suite.sessRepo.FindByToken(ctx, "active_token")
	assert.NoError(suite.T(), err)
	assert.WithinDuration(suite.T(), time.Now(), found.LastActiveAt, 2*time.Second)
}

// TestUserSessionRepository_Delete 测试删除会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_Delete() {
	ctx := context.Background()
	
	// 创建用户和会话
	user := &models.User{
		Username: "delsessuser",
		Email:    "delsess@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	session := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_delete_%d", time.Now().UnixNano()),
		Token:     "delete_token",
		ExpireAt: time.Now().Add(24 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, session)
	assert.NoError(suite.T(), err)
	
	err = suite.sessRepo.Delete(ctx, "delete_token")
	assert.NoError(suite.T(), err)
	
	_, err = suite.sessRepo.FindByToken(ctx, "delete_token")
	assert.Error(suite.T(), err)
}

// TestUserSessionRepository_DeleteByUserID 测试删除用户的所有会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_DeleteByUserID() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "delallsessuser",
		Email:    "delallsess@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 创建多个会话
	for i := 0; i < 3; i++ {
		session := &models.UserSession{
			UserID:    user.ID,
			SessionID: fmt.Sprintf("session_del_%d_%d", i, time.Now().UnixNano()),
			Token:     fmt.Sprintf("del_token_%d", i),
			ExpireAt: time.Now().Add(24 * time.Hour),
		}
		err = suite.sessRepo.Create(ctx, session)
		assert.NoError(suite.T(), err)
	}
	
	err = suite.sessRepo.DeleteByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	
	sessions, err := suite.sessRepo.FindByUserID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), sessions, 0)
}

// TestUserSessionRepository_CleanupExpired 测试清理过期会话
func (suite *UserRepositoryTestSuite) TestUserSessionRepository_CleanupExpired() {
	ctx := context.Background()
	
	// 创建用户
	user := &models.User{
		Username: "cleanupuser",
		Email:    "cleanup@example.com",
		Status:   "active",
	}
	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 创建过期和未过期的会话
	expiredSession := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_exp_cleanup_%d", time.Now().UnixNano()),
		Token:     "expired_cleanup",
		ExpireAt: time.Now().Add(-1 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, expiredSession)
	assert.NoError(suite.T(), err)
	
	activeSession := &models.UserSession{
		UserID:    user.ID,
		SessionID: fmt.Sprintf("session_act_cleanup_%d", time.Now().UnixNano()),
		Token:     "active_cleanup",
		ExpireAt: time.Now().Add(24 * time.Hour),
	}
	err = suite.sessRepo.Create(ctx, activeSession)
	assert.NoError(suite.T(), err)
	
	err = suite.sessRepo.CleanupExpired(ctx)
	assert.NoError(suite.T(), err)
	
	// 验证过期的被删除，未过期的保留
	_, err = suite.sessRepo.FindByToken(ctx, "active_cleanup")
	assert.NoError(suite.T(), err)
	
	// 直接查询过期会话（绕过FindByToken的时间检查）
	var count int64
	suite.db.Model(&models.UserSession{}).Where("token = ?", "expired_cleanup").Count(&count)
	assert.Equal(suite.T(), int64(0), count)
}

// TestUserRepository_WithTx 测试事务支持
func (suite *UserRepositoryTestSuite) TestUserRepository_WithTx() {
	ctx := context.Background()
	
	// 开始事务
	tx := suite.db.Begin()
	defer tx.Rollback()
	
	txRepo := suite.repo.WithTx(tx)
	
	user := &models.User{
		Username: "txuser",
		Email:    "tx@example.com",
		Status:   "active",
	}
	
	err := txRepo.(UserRepository).Create(ctx, user)
	assert.NoError(suite.T(), err)
	
	// 事务内可以查到
	found, err := txRepo.(UserRepository).FindByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, found.Username)
	
	// 回滚后查不到
	tx.Rollback()
	
	_, err = suite.repo.FindByID(ctx, user.ID)
	assert.Error(suite.T(), err)
}

func TestUserRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}