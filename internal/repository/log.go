package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
)

// SystemLogRepository 系统日志仓储接口
type SystemLogRepository interface {
	BaseRepository
	Create(ctx context.Context, log *models.SystemLog) error
	BatchCreate(ctx context.Context, logs []*models.SystemLog) error
	FindByID(ctx context.Context, id uint) (*models.SystemLog, error)
	FindByType(ctx context.Context, logType string, pagination *Pagination) ([]*models.SystemLog, error)
	FindByModule(ctx context.Context, module string, pagination *Pagination) ([]*models.SystemLog, error)
	FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.SystemLog, error)
	FindByDateRange(ctx context.Context, start, end time.Time, pagination *Pagination) ([]*models.SystemLog, error)
	Search(ctx context.Context, query *LogQuery) ([]*models.SystemLog, error)
	CleanupOldLogs(ctx context.Context, days int) error
}

// LogQuery 日志查询条件
type LogQuery struct {
	Level      string    `json:"level"`
	Module     string    `json:"module"`
	UserID     *uint     `json:"user_id"`
	Action     string    `json:"action"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Pagination *Pagination
}

// systemLogRepo 系统日志仓储实现
type systemLogRepo struct {
	*BaseRepo
}

// NewSystemLogRepository 创建系统日志仓储
func NewSystemLogRepository(db *gorm.DB) SystemLogRepository {
	return &systemLogRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建日志
func (r *systemLogRepo) Create(ctx context.Context, log *models.SystemLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate 批量创建日志
func (r *systemLogRepo) BatchCreate(ctx context.Context, logs []*models.SystemLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

// FindByID 根据ID查找
func (r *systemLogRepo) FindByID(ctx context.Context, id uint) (*models.SystemLog, error) {
	var log models.SystemLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("日志不存在")
		}
		return nil, err
	}
	return &log, nil
}

// FindByType 根据类型查找
func (r *systemLogRepo) FindByType(ctx context.Context, logType string, pagination *Pagination) ([]*models.SystemLog, error) {
	var logs []*models.SystemLog
	query := r.db.WithContext(ctx).Model(&models.SystemLog{}).Where("type = ?", logType)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindByModule 根据模块查找
func (r *systemLogRepo) FindByModule(ctx context.Context, module string, pagination *Pagination) ([]*models.SystemLog, error) {
	var logs []*models.SystemLog
	query := r.db.WithContext(ctx).Model(&models.SystemLog{}).Where("module = ?", module)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindByUserID 根据用户ID查找
func (r *systemLogRepo) FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.SystemLog, error) {
	var logs []*models.SystemLog
	query := r.db.WithContext(ctx).Model(&models.SystemLog{}).Where("user_id = ?", userID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindByDateRange 根据日期范围查找
func (r *systemLogRepo) FindByDateRange(ctx context.Context, start, end time.Time, pagination *Pagination) ([]*models.SystemLog, error) {
	var logs []*models.SystemLog
	query := r.db.WithContext(ctx).Model(&models.SystemLog{}).Where("created_at BETWEEN ? AND ?", start, end)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// Search 搜索日志
func (r *systemLogRepo) Search(ctx context.Context, q *LogQuery) ([]*models.SystemLog, error) {
	var logs []*models.SystemLog
	query := r.db.WithContext(ctx).Model(&models.SystemLog{})
	
	// 构建查询条件
	if q.Level != "" {
		// For backward compatibility, map Level to Type field
		query = query.Where("type = ?", q.Level)
	}
	if q.Module != "" {
		query = query.Where("module = ?", q.Module)
	}
	if q.UserID != nil {
		query = query.Where("user_id = ?", *q.UserID)
	}
	if q.Action != "" {
		query = query.Where("action LIKE ?", "%"+q.Action+"%")
	}
	if !q.StartTime.IsZero() && !q.EndTime.IsZero() {
		query = query.Where("created_at BETWEEN ? AND ?", q.StartTime, q.EndTime)
	}
	
	// 获取总数
	if q.Pagination != nil {
		var total int64
		query.Count(&total)
		q.Pagination.Total = total
		
		// 分页查询
		query = query.
			Limit(q.Pagination.PageSize).
			Offset((q.Pagination.Page - 1) * q.Pagination.PageSize)
	}
	
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// CleanupOldLogs 清理旧日志
func (r *systemLogRepo) CleanupOldLogs(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	return r.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&models.SystemLog{}).Error
}

// WithTx 使用事务
func (r *systemLogRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &systemLogRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}

// ErrorLogRepository 错误日志仓储接口
type ErrorLogRepository interface {
	BaseRepository
	Create(ctx context.Context, log *models.ErrorLog) error
	BatchCreate(ctx context.Context, logs []*models.ErrorLog) error
	FindByID(ctx context.Context, id uint) (*models.ErrorLog, error)
	FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.ErrorLog, error)
	FindByErrorCode(ctx context.Context, code string, pagination *Pagination) ([]*models.ErrorLog, error)
	FindByStatus(ctx context.Context, status string, pagination *Pagination) ([]*models.ErrorLog, error)
	FindUnresolved(ctx context.Context, pagination *Pagination) ([]*models.ErrorLog, error)
	MarkAsResolved(ctx context.Context, id uint, resolvedBy uint, resolution string) error
	GetStatistics(ctx context.Context, start, end time.Time) (*ErrorStatistics, error)
}

// ErrorStatistics 错误统计
type ErrorStatistics struct {
	TotalErrors      int            `json:"total_errors"`
	ResolvedErrors   int            `json:"resolved_errors"`
	UnresolvedErrors int            `json:"unresolved_errors"`
	ErrorsByCode     map[string]int `json:"errors_by_code"`
	ErrorsByModule   map[string]int `json:"errors_by_module"`
	TopUsers         []UserErrorStat `json:"top_users"`
}

// UserErrorStat 用户错误统计
type UserErrorStat struct {
	UserID     uint `json:"user_id"`
	ErrorCount int  `json:"error_count"`
}

// errorLogRepo 错误日志仓储实现
type errorLogRepo struct {
	*BaseRepo
}

// NewErrorLogRepository 创建错误日志仓储
func NewErrorLogRepository(db *gorm.DB) ErrorLogRepository {
	return &errorLogRepo{
		BaseRepo: &BaseRepo{db: db},
	}
}

// Create 创建错误日志
func (r *errorLogRepo) Create(ctx context.Context, log *models.ErrorLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate 批量创建错误日志
func (r *errorLogRepo) BatchCreate(ctx context.Context, logs []*models.ErrorLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

// FindByID 根据ID查找
func (r *errorLogRepo) FindByID(ctx context.Context, id uint) (*models.ErrorLog, error) {
	var log models.ErrorLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("错误日志不存在")
		}
		return nil, err
	}
	return &log, nil
}

// FindByUserID 根据用户ID查找
func (r *errorLogRepo) FindByUserID(ctx context.Context, userID uint, pagination *Pagination) ([]*models.ErrorLog, error) {
	var logs []*models.ErrorLog
	query := r.db.WithContext(ctx).Model(&models.ErrorLog{}).Where("user_id = ?", userID)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindByErrorCode 根据错误代码查找
func (r *errorLogRepo) FindByErrorCode(ctx context.Context, code string, pagination *Pagination) ([]*models.ErrorLog, error) {
	var logs []*models.ErrorLog
	query := r.db.WithContext(ctx).Model(&models.ErrorLog{}).Where("error_code = ?", code)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindByStatus 根据状态查找
func (r *errorLogRepo) FindByStatus(ctx context.Context, status string, pagination *Pagination) ([]*models.ErrorLog, error) {
	var logs []*models.ErrorLog
	query := r.db.WithContext(ctx).Model(&models.ErrorLog{}).Where("status = ?", status)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// FindUnresolved 查找未解决的错误
func (r *errorLogRepo) FindUnresolved(ctx context.Context, pagination *Pagination) ([]*models.ErrorLog, error) {
	var logs []*models.ErrorLog
	query := r.db.WithContext(ctx).Model(&models.ErrorLog{}).Where("is_resolved = ?", false)
	
	// 获取总数
	var total int64
	query.Count(&total)
	pagination.Total = total
	
	// 分页查询
	err := query.
		Limit(pagination.PageSize).
		Offset((pagination.Page - 1) * pagination.PageSize).
		Order("created_at DESC").
		Find(&logs).Error
	
	return logs, err
}

// MarkAsResolved 标记为已解决
func (r *errorLogRepo) MarkAsResolved(ctx context.Context, id uint, resolvedBy uint, resolution string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_resolved": true,
			"resolved_by": resolvedBy,
			"resolved_at": &now,
		}).Error
}

// GetStatistics 获取错误统计
func (r *errorLogRepo) GetStatistics(ctx context.Context, start, end time.Time) (*ErrorStatistics, error) {
	stats := &ErrorStatistics{
		ErrorsByCode:   make(map[string]int),
		ErrorsByModule: make(map[string]int),
	}
	
	// 获取总错误数
	var totalErrors int64
	r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Count(&totalErrors)
	stats.TotalErrors = int(totalErrors)
	
	// 获取已解决错误数
	var resolvedErrors int64
	r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("created_at BETWEEN ? AND ? AND is_resolved = ?", start, end, true).
		Count(&resolvedErrors)
	stats.ResolvedErrors = int(resolvedErrors)
	
	stats.UnresolvedErrors = stats.TotalErrors - stats.ResolvedErrors
	
	// 按错误级别统计（用ErrorsByCode字段存储）
	var levelStats []struct {
		Level string
		Count int
	}
	r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select("level, COUNT(*) as count").
		Group("level").
		Scan(&levelStats)
	
	for _, ls := range levelStats {
		stats.ErrorsByCode[ls.Level] = ls.Count
	}
	
	// 按模块统计
	var moduleStats []struct {
		Module string
		Count  int
	}
	r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select("module, COUNT(*) as count").
		Group("module").
		Scan(&moduleStats)
	
	for _, ms := range moduleStats {
		stats.ErrorsByModule[ms.Module] = ms.Count
	}
	
	// 获取错误最多的用户
	var userStats []UserErrorStat
	r.db.WithContext(ctx).
		Model(&models.ErrorLog{}).
		Where("created_at BETWEEN ? AND ? AND user_id IS NOT NULL", start, end).
		Select("user_id, COUNT(*) as error_count").
		Group("user_id").
		Order("error_count DESC").
		Limit(10).
		Scan(&userStats)
	
	stats.TopUsers = userStats
	
	return stats, nil
}

// WithTx 使用事务
func (r *errorLogRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &errorLogRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}