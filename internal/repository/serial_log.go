package repository

import (
	"fmt"
	"time"

	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SerialLogRepository 串口日志仓库
type SerialLogRepository struct {
	db *gorm.DB
}

// NewSerialLogRepository 创建串口日志仓库
func NewSerialLogRepository(db *gorm.DB) *SerialLogRepository {
	return &SerialLogRepository{
		db: db,
	}
}

// Create 创建日志记录
func (r *SerialLogRepository) Create(log *models.SerialLog) error {
	return r.db.Create(log).Error
}

// CreateBatch 批量创建日志记录
func (r *SerialLogRepository) CreateBatch(logs []*models.SerialLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(logs, 100).Error
}

// GetByID 根据ID获取日志
func (r *SerialLogRepository) GetByID(id uint) (*models.SerialLog, error) {
	var log models.SerialLog
	err := r.db.First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetByRequestID 根据请求ID获取日志（包括请求和响应）
func (r *SerialLogRepository) GetByRequestID(requestID string) ([]*models.SerialLog, error) {
	var logs []*models.SerialLog
	err := r.db.Where("request_id = ?", requestID).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// GetBySessionID 根据会话ID获取日志
func (r *SerialLogRepository) GetBySessionID(sessionID string) ([]*models.SerialLog, error) {
	var logs []*models.SerialLog
	err := r.db.Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// Query 查询日志
func (r *SerialLogRepository) Query(query *models.SerialLogQuery) ([]*models.SerialLog, int64, error) {
	db := r.db.Model(&models.SerialLog{})

	// 构建查询条件
	if query.DeviceType != "" {
		db = db.Where("device_type = ?", query.DeviceType)
	}
	if query.Direction != "" {
		db = db.Where("direction = ?", query.Direction)
	}
	if query.Level != "" {
		db = db.Where("level = ?", query.Level)
	}
	if query.Command != "" {
		db = db.Where("command LIKE ?", "%"+query.Command+"%")
	}
	if query.Function != "" {
		db = db.Where("function = ?", query.Function)
	}
	if query.RequestID != "" {
		db = db.Where("request_id = ?", query.RequestID)
	}
	if query.SessionID != "" {
		db = db.Where("session_id = ?", query.SessionID)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", *query.EndTime)
	}
	if query.MinBet != nil {
		db = db.Where("bet >= ?", *query.MinBet)
	}
	if query.MaxBet != nil {
		db = db.Where("bet <= ?", *query.MaxBet)
	}
	if query.MinWin != nil {
		db = db.Where("win >= ?", *query.MinWin)
	}
	if query.MaxWin != nil {
		db = db.Where("win <= ?", *query.MaxWin)
	}
	if query.HasError != nil && *query.HasError {
		db = db.Where("error_msg IS NOT NULL AND error_msg != ''")
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	orderBy := query.OrderBy
	if orderBy == "" {
		orderBy = "created_at DESC"
	}
	db = db.Order(orderBy)

	// 分页
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// 查询数据
	var logs []*models.SerialLog
	if err := db.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetStats 获取统计信息
func (r *SerialLogRepository) GetStats(startTime, endTime *time.Time) (*models.SerialLogStats, error) {
	stats := &models.SerialLogStats{}
	db := r.db.Model(&models.SerialLog{})

	// 时间范围过滤
	if startTime != nil {
		db = db.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		db = db.Where("created_at <= ?", *endTime)
	}

	// 总数统计
	if err := db.Count(&stats.TotalCount).Error; err != nil {
		return nil, err
	}

	// 发送/接收统计
	if err := r.db.Model(&models.SerialLog{}).
		Where("direction = ?", "SEND").
		Count(&stats.TotalSend).Error; err != nil {
		return nil, err
	}
	stats.TotalReceive = stats.TotalCount - stats.TotalSend

	// 设备类型统计
	if err := r.db.Model(&models.SerialLog{}).
		Where("device_type LIKE ?", "ACM%").
		Count(&stats.TotalACM).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&models.SerialLog{}).
		Where("device_type LIKE ?", "STM32%").
		Count(&stats.TotalSTM32).Error; err != nil {
		return nil, err
	}

	// 错误统计
	if err := r.db.Model(&models.SerialLog{}).
		Where("error_msg IS NOT NULL AND error_msg != ''").
		Count(&stats.TotalErrors).Error; err != nil {
		return nil, err
	}

	// 游戏统计
	type GameStats struct {
		TotalBet float64
		TotalWin float64
	}
	var gameStats GameStats
	if err := r.db.Model(&models.SerialLog{}).
		Select("SUM(bet) as total_bet, SUM(win) as total_win").
		Where("bet > 0 OR win > 0").
		Scan(&gameStats).Error; err != nil {
		return nil, err
	}
	stats.TotalBet = gameStats.TotalBet
	stats.TotalWin = gameStats.TotalWin

	// 性能统计
	type DurationStats struct {
		AvgDuration float64
		MaxDuration int64
		MinDuration int64
	}
	var durationStats DurationStats
	if err := r.db.Model(&models.SerialLog{}).
		Select("AVG(duration) as avg_duration, MAX(duration) as max_duration, MIN(duration) as min_duration").
		Where("duration > 0").
		Scan(&durationStats).Error; err != nil {
		return nil, err
	}
	stats.AvgDuration = durationStats.AvgDuration
	stats.MaxDuration = durationStats.MaxDuration
	stats.MinDuration = durationStats.MinDuration

	return stats, nil
}

// GetLatest 获取最新的日志记录
func (r *SerialLogRepository) GetLatest(limit int, deviceType models.SerialLogType) ([]*models.SerialLog, error) {
	var logs []*models.SerialLog
	db := r.db.Order("created_at DESC").Limit(limit)
	if deviceType != "" {
		db = db.Where("device_type = ?", deviceType)
	}
	err := db.Find(&logs).Error
	return logs, err
}

// DeleteOldLogs 删除旧日志
func (r *SerialLogRepository) DeleteOldLogs(beforeTime time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", beforeTime).Delete(&models.SerialLog{})
	return result.RowsAffected, result.Error
}

// CleanupLogs 清理日志（保留最近N天的数据）
func (r *SerialLogRepository) CleanupLogs(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be greater than 0")
	}
	beforeTime := time.Now().AddDate(0, 0, -retentionDays)
	return r.DeleteOldLogs(beforeTime)
}

// GetAlgoCommandLogs 获取algo命令相关的日志
func (r *SerialLogRepository) GetAlgoCommandLogs(startTime, endTime *time.Time, limit int) ([]*models.SerialLog, error) {
	var logs []*models.SerialLog
	db := r.db.Where("function = ? OR command LIKE ?", "algo", "%algo%")

	if startTime != nil {
		db = db.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		db = db.Where("created_at <= ?", *endTime)
	}

	err := db.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// GetErrorLogs 获取错误日志
func (r *SerialLogRepository) GetErrorLogs(limit int) ([]*models.SerialLog, error) {
	var logs []*models.SerialLog
	err := r.db.Where("error_msg IS NOT NULL AND error_msg != ''").
		Or("level = ?", models.SerialLogLevelError).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// UpdateLogDuration 更新日志的处理时长
func (r *SerialLogRepository) UpdateLogDuration(requestID string, duration int64) error {
	return r.db.Model(&models.SerialLog{}).
		Where("request_id = ? AND direction = ?", requestID, "RECEIVE").
		Update("duration", duration).Error
}

// BulkInsertWithConflict 批量插入（忽略冲突）
func (r *SerialLogRepository) BulkInsertWithConflict(logs []*models.SerialLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(logs, 100).Error
}