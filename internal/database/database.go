package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wfunc/slot-game/internal/config"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	// DB 全局数据库实例
	DB *gorm.DB
)

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) error {
	var (
		dialector gorm.Dialector
		err       error
	)

	// 根据配置选择数据库驱动
	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DSN)
	case "sqlite", "sqlite3":
		// SQLite模式下，自动创建数据目录
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return fmt.Errorf("创建SQLite数据目录失败: %w", err)
		}
		dialector = sqlite.Open(cfg.DSN)
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}

	// 配置GORM日志
	logLevel := gormlogger.Info
	switch cfg.LogLevel {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	}

	// 创建自定义日志适配器
	gormLogger := NewGormLogger(logger.GetLogger(), logLevel)

	// 连接数据库
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger:                                   gormLogger,
		SkipDefaultTransaction:                   true,  // 跳过默认事务
		PrepareStmt:                              true,  // 预编译语句
		DisableForeignKeyConstraintWhenMigrating: false, // 迁移时启用外键约束
	})

	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// SQLite特殊优化：立即启用WAL模式和性能优化
	if cfg.Driver == "sqlite" || cfg.Driver == "sqlite3" {
		if err := optimizeSQLiteImmediately(); err != nil {
			logger.Warn("SQLite优化失败", zap.Error(err))
		}
	}

	// 获取底层SQL数据库实例
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logger.Info("数据库连接成功",
		zap.String("driver", cfg.Driver),
		zap.Int("max_idle", cfg.MaxIdleConns),
		zap.Int("max_open", cfg.MaxOpenConns),
	)

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// IsConnected 检查数据库是否连接
func IsConnected() bool {
	if DB == nil {
		return false
	}
	
	sqlDB, err := DB.DB()
	if err != nil {
		return false
	}
	
	if err := sqlDB.Ping(); err != nil {
		return false
	}
	
	return true
}

// Transaction 执行事务
func Transaction(fn func(*gorm.DB) error) error {
	return DB.Transaction(fn)
}

// GormLogger GORM日志适配器
type GormLogger struct {
	logger   *zap.Logger
	logLevel gormlogger.LogLevel
}

// NewGormLogger 创建GORM日志适配器
func NewGormLogger(logger *zap.Logger, level gormlogger.LogLevel) *GormLogger {
	return &GormLogger{
		logger:   logger,
		logLevel: level,
	}
}

// LogMode 设置日志级别
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	l.logLevel = level
	return l
}

// Info 输出信息日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		l.logger.Sugar().Infof(msg, data...)
	}
}

// Warn 输出警告日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.Sugar().Warnf(msg, data...)
	}
}

// Error 输出错误日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		l.logger.Sugar().Errorf(msg, data...)
	}
}

// Trace 输出SQL追踪日志
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.logLevel >= gormlogger.Error:
		l.logger.Error("SQL执行错误",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
		)
	case elapsed > time.Second && l.logLevel >= gormlogger.Warn:
		l.logger.Warn("SQL执行缓慢",
			zap.String("sql", sql),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
		)
	case l.logLevel >= gormlogger.Info:
		l.logger.Debug("SQL执行",
			zap.String("sql", sql),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
		)
	}
}

// optimizeSQLiteImmediately 立即优化SQLite性能，在连接后立即执行
func optimizeSQLiteImmediately() error {
	// 最关键的优化：立即启用WAL模式以避免锁问题
	pragmas := []string{
		"PRAGMA journal_mode = WAL",           // 必须首先执行，避免锁问题
		"PRAGMA busy_timeout = 30000",         // 设置忙等待超时为30秒
		"PRAGMA synchronous = NORMAL",         // 降低同步级别，提高写入性能
		"PRAGMA cache_size = -2000000",        // 设置缓存大小为2GB
		"PRAGMA temp_store = MEMORY",          // 使用内存存储临时表
		"PRAGMA mmap_size = 30000000000",      // 使用内存映射提高读取性能
		"PRAGMA foreign_keys = ON",            // 启用外键约束
	}

	for _, pragma := range pragmas {
		if err := DB.Exec(pragma).Error; err != nil {
			// WAL模式失败是严重问题，其他可以忽略
			if strings.Contains(pragma, "journal_mode") {
				return fmt.Errorf("启用WAL模式失败: %w", err)
			}
			logger.Warn("设置SQLite参数失败", zap.String("pragma", pragma), zap.Error(err))
		}
	}

	logger.Info("SQLite WAL模式和性能优化已启用")
	return nil
}

// ensureSQLiteDir 确保SQLite数据文件目录存在
func ensureSQLiteDir(dsn string) error {
	// 获取数据文件的目录路径
	dir := filepath.Dir(dsn)

	// 如果是内存数据库，直接返回
	if dsn == ":memory:" || dir == "." || dir == "" {
		return nil
	}

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 创建目录（包括父目录）
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		logger.Info("创建SQLite数据目录", zap.String("path", dir))
	}

	return nil
}