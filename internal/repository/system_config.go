package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	
	"github.com/wfunc/slot-game/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SystemConfigRepository 系统配置仓储接口
type SystemConfigRepository interface {
	BaseRepository
	Get(ctx context.Context, key string) (*models.SystemConfig, error)
	GetString(ctx context.Context, key string, defaultValue string) string
	GetInt(ctx context.Context, key string, defaultValue int) int
	GetFloat(ctx context.Context, key string, defaultValue float64) float64
	GetBool(ctx context.Context, key string, defaultValue bool) bool
	GetJSON(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, description string) error
	SetBatch(ctx context.Context, configs map[string]interface{}) error
	Update(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	GetAll(ctx context.Context) ([]*models.SystemConfig, error)
	GetByGroup(ctx context.Context, group string) ([]*models.SystemConfig, error)
	GetPublic(ctx context.Context) ([]*models.SystemConfig, error)
	RefreshCache(ctx context.Context) error
}

// systemConfigRepo 系统配置仓储实现
type systemConfigRepo struct {
	*BaseRepo
	cache map[string]*models.SystemConfig // 内存缓存
}

// NewSystemConfigRepository 创建系统配置仓储
func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
	repo := &systemConfigRepo{
		BaseRepo: NewBaseRepo(db),
		cache:    make(map[string]*models.SystemConfig),
	}
	// 初始化缓存
	repo.RefreshCache(context.Background())
	return repo
}

// Get 获取配置
func (r *systemConfigRepo) Get(ctx context.Context, key string) (*models.SystemConfig, error) {
	// 优先从缓存读取
	if config, ok := r.cache[key]; ok {
		return config, nil
	}
	
	// 从数据库读取
	var config models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("`key` = ?", key).
		First(&config).Error
	
	if err != nil {
		return nil, err
	}
	
	// 更新缓存
	r.cache[key] = &config
	
	return &config, nil
}

// GetString 获取字符串配置
func (r *systemConfigRepo) GetString(ctx context.Context, key string, defaultValue string) string {
	config, err := r.Get(ctx, key)
	if err != nil || config == nil {
		return defaultValue
	}
	return config.Value
}

// GetInt 获取整数配置
func (r *systemConfigRepo) GetInt(ctx context.Context, key string, defaultValue int) int {
	config, err := r.Get(ctx, key)
	if err != nil || config == nil {
		return defaultValue
	}
	
	val, err := strconv.Atoi(config.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetFloat 获取浮点数配置
func (r *systemConfigRepo) GetFloat(ctx context.Context, key string, defaultValue float64) float64 {
	config, err := r.Get(ctx, key)
	if err != nil || config == nil {
		return defaultValue
	}
	
	val, err := strconv.ParseFloat(config.Value, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBool 获取布尔配置
func (r *systemConfigRepo) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	config, err := r.Get(ctx, key)
	if err != nil || config == nil {
		return defaultValue
	}
	
	val, err := strconv.ParseBool(config.Value)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetJSON 获取JSON配置
func (r *systemConfigRepo) GetJSON(ctx context.Context, key string, dest interface{}) error {
	config, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	
	if config == nil {
		return fmt.Errorf("配置项不存在: %s", key)
	}
	
	return json.Unmarshal([]byte(config.Value), dest)
}

// Set 设置配置（创建或更新）
func (r *systemConfigRepo) Set(ctx context.Context, key string, value interface{}, description string) error {
	// 转换值为字符串
	var strValue string
	var configType string
	
	switch v := value.(type) {
	case string:
		strValue = v
		configType = "string"
	case int, int32, int64:
		strValue = fmt.Sprintf("%d", v)
		configType = "int"
	case float32, float64:
		strValue = fmt.Sprintf("%f", v)
		configType = "float"
	case bool:
		strValue = strconv.FormatBool(v)
		configType = "bool"
	default:
		// 尝试JSON序列化
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		strValue = string(bytes)
		configType = "json"
	}
	
	config := &models.SystemConfig{
		Key:         key,
		Value:       strValue,
		Type:        configType,
		Description: description,
		IsPublic:    false,
	}
	
	// 使用 ON CONFLICT 策略
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "type", "description", "updated_at"}),
		}).
		Create(config).Error
	
	if err != nil {
		return err
	}
	
	// 更新缓存
	r.cache[key] = config
	
	return nil
}

// SetBatch 批量设置配置
func (r *systemConfigRepo) SetBatch(ctx context.Context, configs map[string]interface{}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建一个使用事务的临时repo
		txRepo := &systemConfigRepo{
			BaseRepo: &BaseRepo{db: tx},
			cache:    r.cache,
		}
		for key, value := range configs {
			if err := txRepo.Set(ctx, key, value, ""); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update 更新配置值
func (r *systemConfigRepo) Update(ctx context.Context, key string, value interface{}) error {
	// 转换值为字符串
	var strValue string
	
	switch v := value.(type) {
	case string:
		strValue = v
	case int, int32, int64:
		strValue = fmt.Sprintf("%d", v)
	case float32, float64:
		strValue = fmt.Sprintf("%f", v)
	case bool:
		strValue = strconv.FormatBool(v)
	default:
		// 尝试JSON序列化
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		strValue = string(bytes)
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.SystemConfig{}).
		Where("`key` = ?", key).
		Update("value", strValue).Error
	
	if err != nil {
		return err
	}
	
	// 更新缓存
	if config, ok := r.cache[key]; ok {
		config.Value = strValue
	}
	
	return nil
}

// Delete 删除配置
func (r *systemConfigRepo) Delete(ctx context.Context, key string) error {
	err := r.db.WithContext(ctx).
		Where("`key` = ?", key).
		Delete(&models.SystemConfig{}).Error
	
	if err != nil {
		return err
	}
	
	// 删除缓存
	delete(r.cache, key)
	
	return nil
}

// GetAll 获取所有配置
func (r *systemConfigRepo) GetAll(ctx context.Context) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	err := r.db.WithContext(ctx).
		Order("`group`, `key`").
		Find(&configs).Error
	return configs, err
}

// GetByGroup 根据分组获取配置
func (r *systemConfigRepo) GetByGroup(ctx context.Context, group string) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("`group` = ?", group).
		Order("`key`").
		Find(&configs).Error
	return configs, err
}

// GetPublic 获取公开配置（可对外暴露的配置）
func (r *systemConfigRepo) GetPublic(ctx context.Context) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	err := r.db.WithContext(ctx).
		Where("is_public = ?", true).
		Order("`group`, `key`").
		Find(&configs).Error
	return configs, err
}

// RefreshCache 刷新缓存
func (r *systemConfigRepo) RefreshCache(ctx context.Context) error {
	configs, err := r.GetAll(ctx)
	if err != nil {
		return err
	}
	
	// 清空并重建缓存
	r.cache = make(map[string]*models.SystemConfig)
	for _, config := range configs {
		r.cache[config.Key] = config
	}
	
	return nil
}

// WithTx 使用事务
func (r *systemConfigRepo) WithTx(tx *gorm.DB) BaseRepository {
	return &systemConfigRepo{
		BaseRepo: &BaseRepo{db: tx},
		cache:    r.cache, // 共享缓存
	}
}

// ConfigHelper 配置辅助函数
type ConfigHelper struct {
	repo SystemConfigRepository
}

// NewConfigHelper 创建配置辅助器
func NewConfigHelper(repo SystemConfigRepository) *ConfigHelper {
	return &ConfigHelper{repo: repo}
}

// GetGameConfig 获取游戏配置
func (h *ConfigHelper) GetGameConfig(ctx context.Context) (*GameConfig, error) {
	config := &GameConfig{}
	
	// 老虎机配置
	config.Slot.MinBet = h.repo.GetInt(ctx, "game.slot.min_bet", 1)
	config.Slot.MaxBet = h.repo.GetInt(ctx, "game.slot.max_bet", 100)
	config.Slot.JackpotRate = h.repo.GetFloat(ctx, "game.slot.jackpot_rate", 0.01)
	config.Slot.RTP = h.repo.GetFloat(ctx, "game.slot.rtp", 96.5)
	
	// 推币机配置
	config.Pusher.CoinValue = h.repo.GetFloat(ctx, "game.pusher.coin_value", 0.1)
	config.Pusher.MinForce = h.repo.GetInt(ctx, "game.pusher.min_force", 5)
	config.Pusher.MaxForce = h.repo.GetInt(ctx, "game.pusher.max_force", 20)
	config.Pusher.PushInterval = h.repo.GetInt(ctx, "game.pusher.push_interval", 1000)
	
	// 钱包配置
	config.Wallet.InitialCoins = h.repo.GetInt(ctx, "wallet.initial_coins", 100)
	config.Wallet.DailyBonus = h.repo.GetInt(ctx, "wallet.daily_bonus", 50)
	config.Wallet.MaxCoins = h.repo.GetInt(ctx, "wallet.max_coins", 999999)
	
	return config, nil
}

// GameConfig 游戏配置结构
type GameConfig struct {
	Slot   SlotConfig   `json:"slot"`
	Pusher PusherConfig `json:"pusher"`
	Wallet WalletConfig `json:"wallet"`
}

// SlotConfig 老虎机配置
type SlotConfig struct {
	MinBet      int     `json:"min_bet"`
	MaxBet      int     `json:"max_bet"`
	JackpotRate float64 `json:"jackpot_rate"`
	RTP         float64 `json:"rtp"`
}

// PusherConfig 推币机配置
type PusherConfig struct {
	CoinValue    float64 `json:"coin_value"`
	MinForce     int     `json:"min_force"`
	MaxForce     int     `json:"max_force"`
	PushInterval int     `json:"push_interval"`
}

// WalletConfig 钱包配置
type WalletConfig struct {
	InitialCoins int `json:"initial_coins"`
	DailyBonus   int `json:"daily_bonus"`
	MaxCoins     int `json:"max_coins"`
}