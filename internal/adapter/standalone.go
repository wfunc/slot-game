package adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

// StandaloneAdapter 单机版适配器实现（SQLite）
type StandaloneAdapter struct {
	db     *sql.DB
	config *SQLiteConfig
}

// NewStandaloneAdapter 创建单机版适配器
func NewStandaloneAdapter(config *SQLiteConfig) (*StandaloneAdapter, error) {
	if config == nil {
		config = &SQLiteConfig{
			Path: "./data/slot-game.db",
		}
	}
	
	return &StandaloneAdapter{
		config: config,
	}, nil
}

// Connect 连接数据库
func (a *StandaloneAdapter) Connect(ctx context.Context) error {
	db, err := sql.Open("sqlite3", a.config.Path)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite不支持并发写入
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	
	// 启用外键约束
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	
	// 启用WAL模式以提高性能
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("enable WAL mode: %w", err)
	}
	
	a.db = db
	
	// 初始化表结构
	if err := a.initSchema(ctx); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	
	return nil
}

// Close 关闭数据库连接
func (a *StandaloneAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Ping 测试连接
func (a *StandaloneAdapter) Ping(ctx context.Context) error {
	return a.db.PingContext(ctx)
}

// BeginTx 开始事务
func (a *StandaloneAdapter) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqliteTransaction{tx: tx}, nil
}

// CreateUser 创建用户
func (a *StandaloneAdapter) CreateUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	
	query := `
		INSERT INTO users (id, username, email, coins, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err := a.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.Coins,
		user.CreatedAt, user.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	
	return nil
}

// GetUser 获取用户
func (a *StandaloneAdapter) GetUser(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, username, email, coins, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	
	user := &User{}
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Coins,
		&user.CreatedAt, &user.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	
	return user, nil
}

// UpdateUser 更新用户
func (a *StandaloneAdapter) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()
	
	query := `
		UPDATE users
		SET username = ?, email = ?, coins = ?, updated_at = ?
		WHERE id = ?
	`
	
	result, err := a.db.ExecContext(ctx, query,
		user.Username, user.Email, user.Coins, user.UpdatedAt, user.ID,
	)
	
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return ErrNotFound
	}
	
	return nil
}

// DeleteUser 删除用户
func (a *StandaloneAdapter) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?`
	
	result, err := a.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return ErrNotFound
	}
	
	return nil
}

// ListUsers 列出用户
func (a *StandaloneAdapter) ListUsers(ctx context.Context, offset, limit int) ([]*User, error) {
	query := `
		SELECT id, username, email, coins, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := a.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	
	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Coins,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

// SaveGameRecord 保存游戏记录
func (a *StandaloneAdapter) SaveGameRecord(ctx context.Context, record *GameRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	
	if record.PlayedAt.IsZero() {
		record.PlayedAt = time.Now()
	}
	
	query := `
		INSERT INTO game_records (id, user_id, game_type, bet, win, result, played_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := a.db.ExecContext(ctx, query,
		record.ID, record.UserID, record.GameType,
		record.Bet, record.Win, record.Result, record.PlayedAt,
	)
	
	if err != nil {
		return fmt.Errorf("save game record: %w", err)
	}
	
	return nil
}

// GetGameRecord 获取游戏记录
func (a *StandaloneAdapter) GetGameRecord(ctx context.Context, id string) (*GameRecord, error) {
	query := `
		SELECT id, user_id, game_type, bet, win, result, played_at
		FROM game_records
		WHERE id = ?
	`
	
	record := &GameRecord{}
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.UserID, &record.GameType,
		&record.Bet, &record.Win, &record.Result, &record.PlayedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get game record: %w", err)
	}
	
	return record, nil
}

// ListGameRecords 列出游戏记录
func (a *StandaloneAdapter) ListGameRecords(ctx context.Context, userID string, offset, limit int) ([]*GameRecord, error) {
	query := `
		SELECT id, user_id, game_type, bet, win, result, played_at
		FROM game_records
		WHERE user_id = ?
		ORDER BY played_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := a.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list game records: %w", err)
	}
	defer rows.Close()
	
	var records []*GameRecord
	for rows.Next() {
		record := &GameRecord{}
		err := rows.Scan(
			&record.ID, &record.UserID, &record.GameType,
			&record.Bet, &record.Win, &record.Result, &record.PlayedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	
	return records, nil
}

// GetUserStats 获取用户统计
func (a *StandaloneAdapter) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_games,
			COALESCE(SUM(bet), 0) as total_bet,
			COALESCE(SUM(win), 0) as total_win,
			MAX(played_at) as last_played_at
		FROM game_records
		WHERE user_id = ?
	`
	
	stats := &UserStats{UserID: userID}
	var lastPlayed sql.NullTime
	
	err := a.db.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalGames, &stats.TotalBet, &stats.TotalWin, &lastPlayed,
	)
	
	if err != nil {
		return nil, fmt.Errorf("get user stats: %w", err)
	}
	
	if lastPlayed.Valid {
		stats.LastPlayedAt = &lastPlayed.Time
	}
	
	// 计算胜率
	if stats.TotalGames > 0 {
		// 统计获胜场次
		winQuery := `SELECT COUNT(*) FROM game_records WHERE user_id = ? AND win > bet`
		var winCount int64
		if err := a.db.QueryRowContext(ctx, winQuery, userID).Scan(&winCount); err == nil {
			stats.WinRate = float64(winCount) / float64(stats.TotalGames)
		}
	}
	
	return stats, nil
}

// GetDailyStats 获取每日统计
func (a *StandaloneAdapter) GetDailyStats(ctx context.Context, date time.Time) (*DailyStats, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	query := `
		SELECT 
			COUNT(DISTINCT user_id) as active_users,
			COUNT(*) as total_games,
			COALESCE(SUM(bet), 0) as total_bet,
			COALESCE(SUM(win), 0) as total_win
		FROM game_records
		WHERE played_at >= ? AND played_at < ?
	`
	
	stats := &DailyStats{Date: startOfDay}
	
	err := a.db.QueryRowContext(ctx, query, startOfDay, endOfDay).Scan(
		&stats.ActiveUsers, &stats.TotalGames, &stats.TotalBet, &stats.TotalWin,
	)
	
	if err != nil {
		return nil, fmt.Errorf("get daily stats: %w", err)
	}
	
	// 计算收益（投注总额 - 获胜总额）
	stats.Revenue = stats.TotalBet - stats.TotalWin
	
	return stats, nil
}

// SetCache 设置缓存（单机版使用内存表模拟）
func (a *StandaloneAdapter) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 将值序列化为JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}
	
	expiry := time.Now().Add(ttl)
	
	query := `
		INSERT OR REPLACE INTO cache (key, value, expiry)
		VALUES (?, ?, ?)
	`
	
	_, err = a.db.ExecContext(ctx, query, key, string(data), expiry)
	if err != nil {
		return fmt.Errorf("set cache: %w", err)
	}
	
	return nil
}

// GetCache 获取缓存
func (a *StandaloneAdapter) GetCache(ctx context.Context, key string) (interface{}, error) {
	query := `
		SELECT value, expiry FROM cache WHERE key = ?
	`
	
	var data string
	var expiry time.Time
	
	err := a.db.QueryRowContext(ctx, query, key).Scan(&data, &expiry)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cache: %w", err)
	}
	
	// 检查是否过期
	if time.Now().After(expiry) {
		// 删除过期缓存
		a.DeleteCache(ctx, key)
		return nil, ErrNotFound
	}
	
	// 反序列化值
	var value interface{}
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return nil, fmt.Errorf("unmarshal cache value: %w", err)
	}
	
	return value, nil
}

// DeleteCache 删除缓存
func (a *StandaloneAdapter) DeleteCache(ctx context.Context, key string) error {
	query := `DELETE FROM cache WHERE key = ?`
	
	_, err := a.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}
	
	return nil
}

// initSchema 初始化数据库表结构
func (a *StandaloneAdapter) initSchema(ctx context.Context) error {
	schemas := []string{
		// 用户表
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			coins INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		
		// 游戏记录表
		`CREATE TABLE IF NOT EXISTS game_records (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			game_type TEXT NOT NULL,
			bet INTEGER NOT NULL,
			win INTEGER NOT NULL,
			result TEXT NOT NULL,
			played_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		
		// 缓存表（模拟Redis）
		`CREATE TABLE IF NOT EXISTS cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			expiry DATETIME NOT NULL
		)`,
		
		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_game_records_user_id ON game_records(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_game_records_played_at ON game_records(played_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_expiry ON cache(expiry)`,
	}
	
	for _, schema := range schemas {
		if _, err := a.db.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("execute schema: %w", err)
		}
	}
	
	// 定期清理过期缓存
	go a.cleanExpiredCache(ctx)
	
	return nil
}

// cleanExpiredCache 定期清理过期缓存
func (a *StandaloneAdapter) cleanExpiredCache(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			query := `DELETE FROM cache WHERE expiry < ?`
			a.db.ExecContext(ctx, query, time.Now())
		}
	}
}

// sqliteTransaction SQLite事务实现
type sqliteTransaction struct {
	tx *sql.Tx
}

func (t *sqliteTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *sqliteTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *sqliteTransaction) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}