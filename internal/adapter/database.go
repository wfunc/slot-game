package adapter

import (
	"context"
	"errors"
	"time"
)

// DatabaseAdapter 数据库适配器接口
// 支持双模式：线上版(PostgreSQL+Redis) 和 单机版(SQLite)
type DatabaseAdapter interface {
	// 基础操作
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
	
	// 事务操作
	BeginTx(ctx context.Context) (Transaction, error)
	
	// 用户操作
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*User, error)
	
	// 游戏记录操作
	SaveGameRecord(ctx context.Context, record *GameRecord) error
	GetGameRecord(ctx context.Context, id string) (*GameRecord, error)
	ListGameRecords(ctx context.Context, userID string, offset, limit int) ([]*GameRecord, error)
	
	// 统计操作
	GetUserStats(ctx context.Context, userID string) (*UserStats, error)
	GetDailyStats(ctx context.Context, date time.Time) (*DailyStats, error)
	
	// 缓存操作（仅线上版支持）
	SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetCache(ctx context.Context, key string) (interface{}, error)
	DeleteCache(ctx context.Context, key string) error
}

// Transaction 事务接口
type Transaction interface {
	Commit() error
	Rollback() error
	Exec(ctx context.Context, query string, args ...interface{}) error
}

// User 用户模型
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Coins     int64     `json:"coins"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GameRecord 游戏记录
type GameRecord struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	GameType   string    `json:"game_type"`
	Bet        int64     `json:"bet"`
	Win        int64     `json:"win"`
	Result     string    `json:"result"`
	PlayedAt   time.Time `json:"played_at"`
}

// UserStats 用户统计
type UserStats struct {
	UserID      string  `json:"user_id"`
	TotalGames  int64   `json:"total_games"`
	TotalBet    int64   `json:"total_bet"`
	TotalWin    int64   `json:"total_win"`
	WinRate     float64 `json:"win_rate"`
	LastPlayedAt *time.Time `json:"last_played_at"`
}

// DailyStats 每日统计
type DailyStats struct {
	Date       time.Time `json:"date"`
	ActiveUsers int64    `json:"active_users"`
	TotalGames int64     `json:"total_games"`
	TotalBet   int64     `json:"total_bet"`
	TotalWin   int64     `json:"total_win"`
	Revenue    int64     `json:"revenue"`
}

// AdapterType 适配器类型
type AdapterType string

const (
	AdapterTypeOnline     AdapterType = "online"     // 线上版：PostgreSQL + Redis
	AdapterTypeStandalone AdapterType = "standalone" // 单机版：SQLite
)

// Config 适配器配置
type Config struct {
	Type AdapterType `yaml:"type"`
	
	// 线上版配置
	PostgreSQL *PostgreSQLConfig `yaml:"postgresql,omitempty"`
	Redis      *RedisConfig      `yaml:"redis,omitempty"`
	
	// 单机版配置
	SQLite *SQLiteConfig `yaml:"sqlite,omitempty"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
	MaxConns int    `yaml:"max_conns"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// SQLiteConfig SQLite配置
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// Errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicateKey  = errors.New("duplicate key")
	ErrInvalidData   = errors.New("invalid data")
	ErrNotSupported  = errors.New("operation not supported")
	ErrConnection    = errors.New("connection error")
)

// Factory 创建适配器的工厂函数
func NewAdapter(config *Config) (DatabaseAdapter, error) {
	switch config.Type {
	case AdapterTypeOnline:
		return NewOnlineAdapter(config.PostgreSQL, config.Redis)
	case AdapterTypeStandalone:
		return NewStandaloneAdapter(config.SQLite)
	default:
		return nil, errors.New("unknown adapter type")
	}
}