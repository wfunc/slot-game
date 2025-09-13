package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	Serial   SerialConfig   `mapstructure:"serial"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	Game     GameConfig     `mapstructure:"game"`
	Log      LogConfig      `mapstructure:"log"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Security SecurityConfig `mapstructure:"security"`
	System   SystemConfig   `mapstructure:"system"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Mode            string        `mapstructure:"mode"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel        string        `mapstructure:"log_level"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	Host             string        `mapstructure:"host"`
	Port             int           `mapstructure:"port"`
	Path             string        `mapstructure:"path"`
	ReadBufferSize   int           `mapstructure:"read_buffer_size"`
	WriteBufferSize  int           `mapstructure:"write_buffer_size"`
	MaxMessageSize   int64         `mapstructure:"max_message_size"`
	PingInterval     time.Duration `mapstructure:"ping_interval"`
	PongTimeout      time.Duration `mapstructure:"pong_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	EnableCompression bool         `mapstructure:"enable_compression"`
}

// SerialConfig 串口配置
type SerialConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	MockMode      bool          `mapstructure:"mock_mode"`      // 调试模式（使用模拟控制器）
	Port          string        `mapstructure:"port"`
	BaudRate      int           `mapstructure:"baud_rate"`
	DataBits      int           `mapstructure:"data_bits"`
	StopBits      int           `mapstructure:"stop_bits"`
	Parity        string        `mapstructure:"parity"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	RetryTimes    int           `mapstructure:"retry_times"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	STM32         STM32Config   `mapstructure:"stm32"`
	ACM           ACMConfig     `mapstructure:"acm"`
	Bridge        BridgeConfig  `mapstructure:"bridge"`
}

// STM32Config STM32硬件串口配置
type STM32Config struct {
	Enabled           bool          `mapstructure:"enabled"`
	Port              string        `mapstructure:"port"`
	BaudRate          int           `mapstructure:"baud_rate"`
	DataBits          int           `mapstructure:"data_bits"`
	StopBits          int           `mapstructure:"stop_bits"`
	Parity            string        `mapstructure:"parity"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	RetryTimes        int           `mapstructure:"retry_times"`
	RetryInterval     time.Duration `mapstructure:"retry_interval"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// ACMConfig ACM算法模块串口配置
type ACMConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Port         string        `mapstructure:"port"`
	BaudRate     int           `mapstructure:"baud_rate"`
	DataBits     int           `mapstructure:"data_bits"`
	StopBits     int           `mapstructure:"stop_bits"`
	Parity       string        `mapstructure:"parity"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	AutoDetect   bool          `mapstructure:"auto_detect"`
}

// BridgeConfig 桥接模式配置
type BridgeConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	LogMessages bool `mapstructure:"log_messages"`
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	Broker               string        `mapstructure:"broker"`
	ClientID             string        `mapstructure:"client_id"`
	Username             string        `mapstructure:"username"`
	Password             string        `mapstructure:"password"`
	QoS                  byte          `mapstructure:"qos"`
	Retained             bool          `mapstructure:"retained"`
	CleanSession         bool          `mapstructure:"clean_session"`
	AutoReconnect        bool          `mapstructure:"auto_reconnect"`
	MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`
	KeepAlive            time.Duration `mapstructure:"keep_alive"`
	PingTimeout          time.Duration `mapstructure:"ping_timeout"`
	Topics               MQTTTopics    `mapstructure:"topics"`
}

// MQTTTopics MQTT主题配置
type MQTTTopics struct {
	Status  string `mapstructure:"status"`
	Command string `mapstructure:"command"`
	Config  string `mapstructure:"config"`
	Event   string `mapstructure:"event"`
}

// GameConfig 游戏配置
type GameConfig struct {
	Slot   SlotConfig   `mapstructure:"slot"`
	Pusher PusherConfig `mapstructure:"pusher"`
	Coin   CoinConfig   `mapstructure:"coin"`
}

// SlotConfig 老虎机配置
type SlotConfig struct {
	Reels        int                    `mapstructure:"reels"`
	Symbols      []string               `mapstructure:"symbols"`
	SpinDuration time.Duration          `mapstructure:"spin_duration"`
	WinRates     map[string]float64     `mapstructure:"win_rates"`
	Payouts      map[string]int         `mapstructure:"payouts"`
}

// PusherConfig 推币机配置
type PusherConfig struct {
	DefaultForce  int                      `mapstructure:"default_force"`
	MinForce      int                      `mapstructure:"min_force"`
	MaxForce      int                      `mapstructure:"max_force"`
	PushDuration  time.Duration            `mapstructure:"push_duration"`
	PushInterval  time.Duration            `mapstructure:"push_interval"`
	PushConfig    map[string]PushSettings  `mapstructure:"push_config"`
}

// PushSettings 推币设置
type PushSettings struct {
	Count int `mapstructure:"count"`
	Force int `mapstructure:"force"`
}

// CoinConfig 币数管理配置
type CoinConfig struct {
	PricePerCoin  float64 `mapstructure:"price_per_coin"`
	MinBet        int     `mapstructure:"min_bet"`
	MaxBet        int     `mapstructure:"max_bet"`
	InitialCoins  int     `mapstructure:"initial_coins"`
	MaxCoins      int     `mapstructure:"max_coins"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level   string                 `mapstructure:"level"`
	Format  string                 `mapstructure:"format"`
	Output  string                 `mapstructure:"output"`
	File    LogFileConfig          `mapstructure:"file"`
	Modules map[string]string      `mapstructure:"modules"`
}

// LogFileConfig 日志文件配置
type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled             bool           `mapstructure:"enabled"`
	MetricsInterval     time.Duration  `mapstructure:"metrics_interval"`
	HealthCheckInterval time.Duration  `mapstructure:"health_check_interval"`
	Alerts              AlertsConfig   `mapstructure:"alerts"`
}

// AlertsConfig 告警配置
type AlertsConfig struct {
	CPUThreshold     float64 `mapstructure:"cpu_threshold"`
	MemoryThreshold  float64 `mapstructure:"memory_threshold"`
	ErrorRate        float64 `mapstructure:"error_rate"`
	ResponseTime     int     `mapstructure:"response_time"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	RequestsPerMinute  int  `mapstructure:"requests_per_minute"`
	Burst              int  `mapstructure:"burst"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret       string `mapstructure:"secret"`
	ExpireHours  int    `mapstructure:"expire_hours"`
	RefreshHours int    `mapstructure:"refresh_hours"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Algorithm string `mapstructure:"algorithm"`
	Key       string `mapstructure:"key"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Timezone string       `mapstructure:"timezone"`
	MaxProcs int          `mapstructure:"max_procs"`
	Cache    CacheConfig  `mapstructure:"cache"`
	Backup   BackupConfig `mapstructure:"backup"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Size    int           `mapstructure:"size"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// BackupConfig 备份配置
type BackupConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Interval time.Duration `mapstructure:"interval"`
	Path     string        `mapstructure:"path"`
	KeepDays int           `mapstructure:"keep_days"`
}

var (
	cfg  *Config
	once sync.Once
	mu   sync.RWMutex
	v    *viper.Viper
)

// Init 初始化配置
func Init(configPath string) error {
	var err error
	once.Do(func() {
		v = viper.New()
		
		// 设置配置文件路径
		if configPath != "" {
			v.SetConfigFile(configPath)
		} else {
			v.SetConfigName("config")
			v.SetConfigType("yaml")
			v.AddConfigPath("./config")
			v.AddConfigPath(".")
		}
		
		// 设置环境变量前缀
		v.SetEnvPrefix("SLOT_GAME")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()
		
		// 设置默认值
		setDefaults(v)
		
		// 读取配置文件
		if err = v.ReadInConfig(); err != nil {
			// 如果配置文件不存在，使用默认配置
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return
			}
		}
		
		// 解析配置到结构体
		cfg = &Config{}
		if err = v.Unmarshal(cfg); err != nil {
			return
		}
		
		// 替换MQTT主题中的变量
		replaceMQTTTopics()
	})
	
	return err
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "development")
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")
	
	// 数据库默认配置
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "./data/slot-game.db")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.log_level", "info")
	v.SetDefault("database.auto_migrate", true)
	
	// WebSocket默认配置
	v.SetDefault("websocket.host", "0.0.0.0")
	v.SetDefault("websocket.port", 8081)
	v.SetDefault("websocket.path", "/ws")
	v.SetDefault("websocket.read_buffer_size", 1024)
	v.SetDefault("websocket.write_buffer_size", 1024)
	v.SetDefault("websocket.max_message_size", 8192)
	v.SetDefault("websocket.ping_interval", "30s")
	v.SetDefault("websocket.pong_timeout", "60s")
	v.SetDefault("websocket.write_timeout", "10s")
	v.SetDefault("websocket.enable_compression", true)
	
	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "both")
	v.SetDefault("log.file.path", "./logs")
	v.SetDefault("log.file.filename", "slot-game.log")
	v.SetDefault("log.file.max_size", 100)
	v.SetDefault("log.file.max_age", 30)
	v.SetDefault("log.file.max_backups", 7)
	v.SetDefault("log.file.compress", true)
}

// replaceMQTTTopics 替换MQTT主题中的变量
func replaceMQTTTopics() {
	if cfg == nil || !cfg.MQTT.Enabled {
		return
	}
	
	clientID := cfg.MQTT.ClientID
	cfg.MQTT.Topics.Status = strings.ReplaceAll(cfg.MQTT.Topics.Status, "{client_id}", clientID)
	cfg.MQTT.Topics.Command = strings.ReplaceAll(cfg.MQTT.Topics.Command, "{client_id}", clientID)
	cfg.MQTT.Topics.Config = strings.ReplaceAll(cfg.MQTT.Topics.Config, "{client_id}", clientID)
	cfg.MQTT.Topics.Event = strings.ReplaceAll(cfg.MQTT.Topics.Event, "{client_id}", clientID)
}

// Get 获取配置实例
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// Watch 监听配置文件变化
func Watch(callback func(*Config)) {
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		mu.Lock()
		defer mu.Unlock()
		
		newCfg := &Config{}
		if err := v.Unmarshal(newCfg); err != nil {
			fmt.Printf("配置重载失败: %v\n", err)
			return
		}
		
		cfg = newCfg
		replaceMQTTTopics()
		
		if callback != nil {
			callback(cfg)
		}
		
		fmt.Println("配置已重新加载")
	})
}

// GetString 获取字符串配置
func GetString(key string) string {
	return v.GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return v.GetInt(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return v.GetBool(key)
}

// GetFloat64 获取浮点数配置
func GetFloat64(key string) float64 {
	return v.GetFloat64(key)
}

// GetDuration 获取时间间隔配置
func GetDuration(key string) time.Duration {
	return v.GetDuration(key)
}

// IsSet 检查配置项是否存在
func IsSet(key string) bool {
	return v.IsSet(key)
}

// Set 动态设置配置值
func Set(key string, value interface{}) {
	v.Set(key, value)
}