package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// LogRepositoryTestSuite 日志仓储测试套件
type LogRepositoryTestSuite struct {
	suite.Suite
	db          *gorm.DB
	sysLogRepo  SystemLogRepository
	errorRepo   ErrorLogRepository
}

// SetupSuite 设置测试套件
func (suite *LogRepositoryTestSuite) SetupSuite() {
	suite.db = SetupTestDB()
	suite.sysLogRepo = NewSystemLogRepository(suite.db)
	suite.errorRepo = NewErrorLogRepository(suite.db)
}

// TearDownSuite 清理测试套件
func (suite *LogRepositoryTestSuite) TearDownSuite() {
	CleanupTestDB(suite.db)
}

// SetupTest 每个测试前执行
func (suite *LogRepositoryTestSuite) SetupTest() {
	// 清理表数据
	suite.db.Exec("DELETE FROM system_logs")
	suite.db.Exec("DELETE FROM error_logs")
}

// TestSystemLogRepository_Create 测试创建系统日志
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_Create() {
	ctx := context.Background()
	
	log := &models.SystemLog{
		Type:      "login",
		Action:    "用户登录",
		Module:    "auth",
		UserID:    1,
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		Request:   `{"username":"testuser"}`,
		Response:  `{"success":true}`,
		Status:    "success",
		Duration:  100,
		Extra:     models.JSONMap{"username": "testuser"},
	}
	
	err := suite.sysLogRepo.Create(ctx, log)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), log.ID)
}

// TestSystemLogRepository_BatchCreate 测试批量创建系统日志
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_BatchCreate() {
	ctx := context.Background()
	
	logs := []*models.SystemLog{
		{
			Type:   "login",
			Action: "用户登录",
			Module: "auth",
			UserID: 1,
			Status: "success",
			Extra:  models.JSONMap{},
		},
		{
			Type:   "operation",
			Action: "支付操作",
			Module: "payment",
			UserID: 2,
			Status: "pending",
			Extra:  models.JSONMap{},
		},
		{
			Type:   "system",
			Action: "系统维护",
			Module: "system",
			Status: "success",
			Extra:  models.JSONMap{},
		},
	}
	
	err := suite.sysLogRepo.BatchCreate(ctx, logs)
	assert.NoError(suite.T(), err)
	
	// 验证批量创建
	var count int64
	suite.db.Model(&models.SystemLog{}).Count(&count)
	assert.Equal(suite.T(), int64(3), count)
}

// TestSystemLogRepository_FindByModule 测试按模块查询
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_FindByModule() {
	ctx := context.Background()
	
	// 创建测试数据
	logs := []*models.SystemLog{
		{Type: "login", Action: "登录", Module: "auth", Status: "success", Extra: models.JSONMap{}},
		{Type: "logout", Action: "登出", Module: "auth", Status: "success", Extra: models.JSONMap{}},
		{Type: "operation", Action: "支付", Module: "payment", Status: "success", Extra: models.JSONMap{}},
	}
	
	for _, log := range logs {
		err := suite.sysLogRepo.Create(ctx, log)
		assert.NoError(suite.T(), err)
	}
	
	// 查询auth模块的日志
	pagination := &Pagination{Page: 1, PageSize: 10}
	authLogs, err := suite.sysLogRepo.FindByModule(ctx, "auth", pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), authLogs, 2)
	assert.Equal(suite.T(), int64(2), pagination.Total)
}

// TestSystemLogRepository_FindByType 测试按类型查询
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_FindByType() {
	ctx := context.Background()
	
	// 创建不同类型的日志
	logs := []*models.SystemLog{
		{Type: "login", Action: "登录1", Module: "auth", Status: "success", Extra: models.JSONMap{}},
		{Type: "login", Action: "登录2", Module: "auth", Status: "success", Extra: models.JSONMap{}},
		{Type: "operation", Action: "操作", Module: "game", Status: "success", Extra: models.JSONMap{}},
		{Type: "system", Action: "系统", Module: "system", Status: "success", Extra: models.JSONMap{}},
	}
	
	for _, log := range logs {
		err := suite.sysLogRepo.Create(ctx, log)
		assert.NoError(suite.T(), err)
	}
	
	// 查询login类型的日志
	pagination := &Pagination{Page: 1, PageSize: 10}
	loginLogs, err := suite.sysLogRepo.FindByType(ctx, "login", pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), loginLogs, 2)
	assert.Equal(suite.T(), int64(2), pagination.Total)
}

// TestSystemLogRepository_FindByUserID 测试按用户ID查询
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_FindByUserID() {
	ctx := context.Background()
	
	// 创建不同用户的日志
	logs := []*models.SystemLog{
		{Type: "login", Action: "用户1操作", Module: "auth", UserID: 1, Status: "success", Extra: models.JSONMap{}},
		{Type: "operation", Action: "用户1游戏", Module: "game", UserID: 1, Status: "success", Extra: models.JSONMap{}},
		{Type: "operation", Action: "用户2支付", Module: "payment", UserID: 2, Status: "success", Extra: models.JSONMap{}},
	}
	
	for _, log := range logs {
		err := suite.sysLogRepo.Create(ctx, log)
		assert.NoError(suite.T(), err)
	}
	
	// 查询用户1的日志
	pagination := &Pagination{Page: 1, PageSize: 10}
	userLogs, err := suite.sysLogRepo.FindByUserID(ctx, 1, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), userLogs, 2)
}

// TestSystemLogRepository_FindByDateRange 测试按日期范围查询
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_FindByDateRange() {
	ctx := context.Background()
	
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)
	
	// 创建不同时间的日志
	log1 := &models.SystemLog{
		Type:   "login",
		Action: "今天的日志",
		Module: "auth",
		Status: "success",
		Extra:  models.JSONMap{},
	}
	err := suite.sysLogRepo.Create(ctx, log1)
	assert.NoError(suite.T(), err)
	
	// 修改创建时间为昨天
	suite.db.Model(&models.SystemLog{}).Where("id = ?", log1.ID).
		Update("created_at", yesterday)
	
	log2 := &models.SystemLog{
		Type:   "operation",
		Action: "今天的日志2",
		Module: "game",
		Status: "success",
		Extra:  models.JSONMap{},
	}
	err = suite.sysLogRepo.Create(ctx, log2)
	assert.NoError(suite.T(), err)
	
	// 查询日期范围内的日志
	pagination := &Pagination{Page: 1, PageSize: 10}
	logs, err := suite.sysLogRepo.FindByDateRange(ctx, yesterday.Add(-1*time.Hour), tomorrow, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), logs, 2)
}

// TestSystemLogRepository_Search 测试搜索功能
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_Search() {
	ctx := context.Background()
	
	// 创建测试数据
	logs := []*models.SystemLog{
		{
			Type:   "login",
			Action: "用户登录",
			Module: "auth",
			UserID: 1,
			Status: "success",
			Extra:  models.JSONMap{},
		},
		{
			Type:   "operation",
			Action: "数据导出",
			Module: "export",
			UserID: 1,
			Status: "success",
			Extra:  models.JSONMap{},
		},
		{
			Type:   "system",
			Action: "系统备份",
			Module: "backup",
			UserID: 2,
			Status: "failed",
			Extra:  models.JSONMap{},
		},
	}
	
	for _, log := range logs {
		err := suite.sysLogRepo.Create(ctx, log)
		assert.NoError(suite.T(), err)
	}
	
	// 搜索用户1的auth模块日志
	userID := uint(1)
	query := &LogQuery{
		Module:     "auth",
		UserID:     &userID,
		Pagination: &Pagination{Page: 1, PageSize: 10},
	}
	
	results, err := suite.sysLogRepo.Search(ctx, query)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 1)
	assert.Equal(suite.T(), "用户登录", results[0].Action)
}

// TestSystemLogRepository_CleanupOldLogs 测试清理旧日志
func (suite *LogRepositoryTestSuite) TestSystemLogRepository_CleanupOldLogs() {
	ctx := context.Background()
	
	// 创建旧日志
	oldLog := &models.SystemLog{
		Type:   "login",
		Action: "旧日志",
		Module: "auth",
		Status: "success",
		Extra:  models.JSONMap{},
	}
	err := suite.sysLogRepo.Create(ctx, oldLog)
	assert.NoError(suite.T(), err)
	
	// 修改创建时间为31天前
	oldTime := time.Now().Add(-31 * 24 * time.Hour)
	suite.db.Model(&models.SystemLog{}).Where("id = ?", oldLog.ID).
		Update("created_at", oldTime)
	
	// 创建新日志
	newLog := &models.SystemLog{
		Type:   "operation",
		Action: "新日志",
		Module: "game",
		Status: "success",
		Extra:  models.JSONMap{},
	}
	err = suite.sysLogRepo.Create(ctx, newLog)
	assert.NoError(suite.T(), err)
	
	// 清理30天前的日志
	err = suite.sysLogRepo.CleanupOldLogs(ctx, 30)
	assert.NoError(suite.T(), err)
	
	// 验证清理结果
	var count int64
	suite.db.Model(&models.SystemLog{}).Count(&count)
	assert.Equal(suite.T(), int64(1), count)
}

// TestErrorLogRepository_Create 测试创建错误日志
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_Create() {
	ctx := context.Background()
	
	errorLog := &models.ErrorLog{
		UserID:    1,
		Level:     "error",
		Module:    "payment",
		Function:  "ProcessPayment",
		Message:   "支付处理失败",
		Stack:     "stack trace here",
		File:      "payment.go",
		Line:      123,
		Context:   models.JSONMap{"amount": 100, "currency": "CNY"},
		IP:        "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}
	
	err := suite.errorRepo.Create(ctx, errorLog)
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), errorLog.ID)
}

// TestErrorLogRepository_BatchCreate 测试批量创建错误日志
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_BatchCreate() {
	ctx := context.Background()
	
	errors := []*models.ErrorLog{
		{
			Level:    "error",
			Module:   "auth",
			Function: "Login",
			Message:  "登录失败",
			Context:  models.JSONMap{},
		},
		{
			Level:    "warning",
			Module:   "game",
			Function: "StartGame",
			Message:  "游戏启动警告",
			Context:  models.JSONMap{},
		},
		{
			Level:    "fatal",
			Module:   "system",
			Function: "Initialize",
			Message:  "系统初始化失败",
			Context:  models.JSONMap{},
		},
	}
	
	err := suite.errorRepo.BatchCreate(ctx, errors)
	assert.NoError(suite.T(), err)
	
	// 验证批量创建
	var count int64
	suite.db.Model(&models.ErrorLog{}).Count(&count)
	assert.Equal(suite.T(), int64(3), count)
}

// TestErrorLogRepository_FindByUserID 测试按用户ID查询错误日志
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_FindByUserID() {
	ctx := context.Background()
	
	// 创建不同用户的错误日志
	errors := []*models.ErrorLog{
		{UserID: 1, Level: "error", Module: "auth", Message: "用户1错误1", Context: models.JSONMap{}},
		{UserID: 1, Level: "error", Module: "game", Message: "用户1错误2", Context: models.JSONMap{}},
		{UserID: 2, Level: "error", Module: "payment", Message: "用户2错误", Context: models.JSONMap{}},
	}
	
	for _, e := range errors {
		err := suite.errorRepo.Create(ctx, e)
		assert.NoError(suite.T(), err)
	}
	
	// 查询用户1的错误
	pagination := &Pagination{Page: 1, PageSize: 10}
	userErrors, err := suite.errorRepo.FindByUserID(ctx, 1, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), userErrors, 2)
}

// TestErrorLogRepository_FindUnresolved 测试查询未解决的错误
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_FindUnresolved() {
	ctx := context.Background()
	
	// 创建已解决和未解决的错误
	resolvedError := &models.ErrorLog{
		Level:      "error",
		Module:     "auth",
		Message:    "已解决的错误",
		IsResolved: true,
		Context:    models.JSONMap{},
	}
	err := suite.errorRepo.Create(ctx, resolvedError)
	assert.NoError(suite.T(), err)
	
	unresolvedError1 := &models.ErrorLog{
		Level:      "error",
		Module:     "payment",
		Message:    "未解决的错误1",
		IsResolved: false,
		Context:    models.JSONMap{},
	}
	err = suite.errorRepo.Create(ctx, unresolvedError1)
	assert.NoError(suite.T(), err)
	
	unresolvedError2 := &models.ErrorLog{
		Level:      "error",
		Module:     "game",
		Message:    "未解决的错误2",
		IsResolved: false,
		Context:    models.JSONMap{},
	}
	err = suite.errorRepo.Create(ctx, unresolvedError2)
	assert.NoError(suite.T(), err)
	
	// 查询未解决的错误
	pagination := &Pagination{Page: 1, PageSize: 10}
	unresolved, err := suite.errorRepo.FindUnresolved(ctx, pagination)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), unresolved, 2)
}

// TestErrorLogRepository_MarkAsResolved 测试标记错误为已解决
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_MarkAsResolved() {
	ctx := context.Background()
	
	// 创建未解决的错误
	errorLog := &models.ErrorLog{
		Level:      "error",
		Module:     "payment",
		Message:    "支付错误",
		IsResolved: false,
		Context:    models.JSONMap{},
	}
	err := suite.errorRepo.Create(ctx, errorLog)
	assert.NoError(suite.T(), err)
	
	// 标记为已解决
	err = suite.errorRepo.MarkAsResolved(ctx, errorLog.ID, 1, "问题已修复")
	assert.NoError(suite.T(), err)
	
	// 验证已解决
	found, err := suite.errorRepo.FindByID(ctx, errorLog.ID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), found.IsResolved)
	assert.Equal(suite.T(), uint(1), found.ResolvedBy)
	assert.NotNil(suite.T(), found.ResolvedAt)
}

// TestErrorLogRepository_GetStatistics 测试获取错误统计
func (suite *LogRepositoryTestSuite) TestErrorLogRepository_GetStatistics() {
	ctx := context.Background()
	
	// 创建各种错误日志
	errors := []*models.ErrorLog{
		{Level: "error", Module: "auth", Message: "认证错误", Context: models.JSONMap{}},
		{Level: "error", Module: "auth", Message: "认证错误2", Context: models.JSONMap{}},
		{Level: "warning", Module: "payment", Message: "支付警告", Context: models.JSONMap{}},
		{Level: "fatal", Module: "system", Message: "系统崩溃", Context: models.JSONMap{}},
	}
	
	for _, e := range errors {
		err := suite.errorRepo.Create(ctx, e)
		assert.NoError(suite.T(), err)
	}
	
	// 获取统计信息
	stats, err := suite.errorRepo.GetStatistics(ctx, time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	assert.Equal(suite.T(), 4, stats.TotalErrors)
	assert.Equal(suite.T(), 3, len(stats.ErrorsByModule))
	assert.Equal(suite.T(), 2, stats.ErrorsByModule["auth"])
	assert.Equal(suite.T(), 1, stats.ErrorsByModule["payment"])
	assert.Equal(suite.T(), 1, stats.ErrorsByModule["system"])
	// ErrorsByLevel 不存在，只有 ErrorsByModule 和 ErrorsByCode
}

// TestLogRepositorySuite 运行日志仓储测试套件
func TestLogRepositorySuite(t *testing.T) {
	suite.Run(t, new(LogRepositoryTestSuite))
}