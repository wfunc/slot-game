package database

import (
	"context"
	"fmt"
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