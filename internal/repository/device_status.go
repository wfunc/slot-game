package repository

import (
	"context"
	"time"
	
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeviceStatusRepository 设备状态仓储接口
type DeviceStatusRepository interface {
	BaseRepository
	Create(ctx context.Context, status *models.DeviceStatus) error
	Update(ctx context.Context, status *models.DeviceStatus) error
	UpdateStatus(ctx context.Context, deviceID string, status string, extra map[string]interface{}) error
	UpdateMetrics(ctx context.Context, deviceID string, cpu, memory, disk float64) error
	UpdatePing(ctx context.Context, deviceID string) error
	FindByID(ctx context.Context, id uint) (*models.DeviceStatus, error)
	FindByDeviceID(ctx context.Context, deviceID string) (*models.DeviceStatus, error)
	FindByType(ctx context.Context, deviceType string) ([]*models.DeviceStatus, error)
	FindByStatus(ctx context.Context, status string) ([]*models.DeviceStatus, error)
	GetHealthReport(ctx context.Context) (*HealthReport, error)
	GetOfflineDevices(ctx context.Context, offlineThreshold time.Duration) ([]*models.DeviceStatus, error)
	RegisterDevice(ctx context.Context, device *models.DeviceStatus) error
}

// HealthReport 健康报告
type HealthReport struct {
	TotalDevices    int                    `json:"total_devices"`
	OnlineDevices   int                    `json:"online_devices"`
	OfflineDevices  int                    `json:"offline_devices"`
	ErrorDevices    int                    `json:"error_devices"`
	MaintenanceDevices int                 `json:"maintenance_devices"`
	AvgCPU          float64                `json:"avg_cpu"`
	AvgMemory       float64                `json:"avg_memory"`
	AvgDisk         float64                `json:"avg_disk"`
	DeviceTypes     map[string]int         `json:"device_types"`
	StatusSummary   map[string]int         `json:"status_summary"`
	AlertDevices    []*DeviceAlert         `json:"alert_devices"`
}

// DeviceAlert 设备告警
type DeviceAlert struct {
	DeviceID   string  `json:"device_id"`
	DeviceName string  `json:"device_name"`
	Type       string  `json:"type"`
	Status     string  `json:"status"`
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	Disk       float64 `json:"disk"`
	LastPing   string  `json:"last_ping"`
	AlertLevel string  `json:"alert_level"` // warning, critical
	Message    string  `json:"message"`
}

// deviceStatusRepo 设备状态仓储实现
type deviceStatusRepo struct {
	*BaseRepo
}

// NewDeviceStatusRepository 创建设备状态仓储
func NewDeviceStatusRepository(db *gorm.DB) DeviceStatusRepository {
	return &deviceStatusRepo{
		BaseRepo: NewBaseRepo(db),
	}
}

// Create 创建设备状态
func (r *deviceStatusRepo) Create(ctx context.Context, status *models.DeviceStatus) error {
	return r.db.WithContext(ctx).Create(status).Error
}

// Update 更新设备状态
func (r *deviceStatusRepo) Update(ctx context.Context, status *models.DeviceStatus) error {
	return r.db.WithContext(ctx).Save(status).Error
}

// UpdateStatus 更新设备状态
func (r *deviceStatusRepo) UpdateStatus(ctx context.Context, deviceID string, status string, extra map[string]interface{}) error {
	updates := map[string]interface{}{
		"status":       status,
		"last_ping_at": time.Now(),
	}
	
	if extra != nil {
		updates["extra"] = models.JSONMap(extra)
	}
	
	return r.db.WithContext(ctx).
		Model(&models.DeviceStatus{}).
		Where("device_id = ?", deviceID).
		Updates(updates).Error
}

// UpdateMetrics 更新设备指标
func (r *deviceStatusRepo) UpdateMetrics(ctx context.Context, deviceID string, cpu, memory, disk float64) error {
	return r.db.WithContext(ctx).
		Model(&models.DeviceStatus{}).
		Where("device_id = ?", deviceID).
		Updates(map[string]interface{}{
			"cpu":          cpu,
			"memory":       memory,
			"disk":         disk,
			"last_ping_at": time.Now(),
		}).Error
}

// UpdatePing 更新心跳时间
func (r *deviceStatusRepo) UpdatePing(ctx context.Context, deviceID string) error {
	return r.db.WithContext(ctx).
		Model(&models.DeviceStatus{}).
		Where("device_id = ?", deviceID).
		Update("last_ping_at", time.Now()).Error
}

// FindByID 根据ID查找
func (r *deviceStatusRepo) FindByID(ctx context.Context, id uint) (*models.DeviceStatus, error) {
	var status models.DeviceStatus
	err := r.db.WithContext(ctx).First(&status, id).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// FindByDeviceID 根据设备ID查找
func (r *deviceStatusRepo) FindByDeviceID(ctx context.Context, deviceID string) (*models.DeviceStatus, error) {
	var status models.DeviceStatus
	err := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		First(&status).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// FindByType 根据设备类型查找
func (r *deviceStatusRepo) FindByType(ctx context.Context, deviceType string) ([]*models.DeviceStatus, error) {
	var statuses []*models.DeviceStatus
	err := r.db.WithContext(ctx).
		Where("type = ?", deviceType).
		Order("device_name").
		Find(&statuses).Error
	return statuses, err
}

// FindByStatus 根据状态查找
func (r *deviceStatusRepo) FindByStatus(ctx context.Context, status string) ([]*models.DeviceStatus, error) {
	var statuses []*models.DeviceStatus
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("last_ping_at desc").
		Find(&statuses).Error
	return statuses, err
}

// GetHealthReport 获取健康报告
func (r *deviceStatusRepo) GetHealthReport(ctx context.Context) (*HealthReport, error) {
	report := &HealthReport{
		DeviceTypes:   make(map[string]int),
		StatusSummary: make(map[string]int),
		AlertDevices:  make([]*DeviceAlert, 0),
	}
	
	// 获取所有设备
	var devices []models.DeviceStatus
	if err := r.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return nil, err
	}
	
	report.TotalDevices = len(devices)
	
	// 统计数据
	var totalCPU, totalMemory, totalDisk float64
	activeCount := 0
	offlineThreshold := time.Now().Add(-5 * time.Minute)
	
	for _, device := range devices {
		// 状态统计
		report.StatusSummary[device.Status]++
		
		// 类型统计
		report.DeviceTypes[device.Type]++
		
		// 状态分类
		switch device.Status {
		case "online":
			// 检查心跳时间
			if device.LastPingAt.After(offlineThreshold) {
				report.OnlineDevices++
			} else {
				report.OfflineDevices++
				// 添加离线告警
				report.AlertDevices = append(report.AlertDevices, &DeviceAlert{
					DeviceID:   device.DeviceID,
					DeviceName: device.DeviceName,
					Type:       device.Type,
					Status:     "offline",
					LastPing:   device.LastPingAt.Format("2006-01-02 15:04:05"),
					AlertLevel: "warning",
					Message:    "设备离线超过5分钟",
				})
			}
		case "offline":
			report.OfflineDevices++
		case "error":
			report.ErrorDevices++
			// 添加错误告警
			report.AlertDevices = append(report.AlertDevices, &DeviceAlert{
				DeviceID:   device.DeviceID,
				DeviceName: device.DeviceName,
				Type:       device.Type,
				Status:     device.Status,
				LastPing:   device.LastPingAt.Format("2006-01-02 15:04:05"),
				AlertLevel: "critical",
				Message:    "设备状态异常",
			})
		case "maintenance":
			report.MaintenanceDevices++
		}
		
		// 计算平均指标（仅在线设备）
		if device.Status == "online" && device.LastPingAt.After(offlineThreshold) {
			totalCPU += device.CPU
			totalMemory += device.Memory
			totalDisk += device.Disk
			activeCount++
			
			// 检查资源告警
			if device.CPU > 80 {
				report.AlertDevices = append(report.AlertDevices, &DeviceAlert{
					DeviceID:   device.DeviceID,
					DeviceName: device.DeviceName,
					Type:       device.Type,
					Status:     device.Status,
					CPU:        device.CPU,
					Memory:     device.Memory,
					Disk:       device.Disk,
					AlertLevel: "warning",
					Message:    "CPU使用率过高",
				})
			}
			if device.Memory > 85 {
				report.AlertDevices = append(report.AlertDevices, &DeviceAlert{
					DeviceID:   device.DeviceID,
					DeviceName: device.DeviceName,
					Type:       device.Type,
					Status:     device.Status,
					CPU:        device.CPU,
					Memory:     device.Memory,
					Disk:       device.Disk,
					AlertLevel: "warning",
					Message:    "内存使用率过高",
				})
			}
			if device.Disk > 90 {
				report.AlertDevices = append(report.AlertDevices, &DeviceAlert{
					DeviceID:   device.DeviceID,
					DeviceName: device.DeviceName,
					Type:       device.Type,
					Status:     device.Status,
					CPU:        device.CPU,
					Memory:     device.Memory,
					Disk:       device.Disk,
					AlertLevel: "critical",
					Message:    "磁盘空间不足",
				})
			}
		}
	}
	
	// 计算平均值
	if activeCount > 0 {
		report.AvgCPU = totalCPU / float64(activeCount)
		report.AvgMemory = totalMemory / float64(activeCount)
		report.AvgDisk = totalDisk / float64(activeCount)
	}
	
	return report, nil
}

// GetOfflineDevices 获取离线设备
func (r *deviceStatusRepo) GetOfflineDevices(ctx context.Context, offlineThreshold time.Duration) ([]*models.DeviceStatus, error) {
	var devices []*models.DeviceStatus
	threshold := time.Now().Add(-offlineThreshold)
	
	err := r.db.WithContext(ctx).
		Where("status = ? OR last_ping_at < ?", "offline", threshold).
		Order("last_ping_at desc").
		Find(&devices).Error
	
	return devices, err
}

// RegisterDevice 注册设备
func (r *deviceStatusRepo) RegisterDevice(ctx context.Context, device *models.DeviceStatus) error {
	// 使用 ON CONFLICT 策略
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"device_name", "type", "status", "ip", "location", 
				"version", "last_ping_at",
			}),
		}).
		Create(device).Error
}

// WithTx 使用事务
func (r *deviceStatusRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &deviceStatusRepo{
		BaseRepo: &BaseRepo{db: tx},
	}
}