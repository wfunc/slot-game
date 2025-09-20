package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// acquireMigrationLock 获取迁移锁
func acquireMigrationLock(dbPath string) (*os.File, error) {
	lockPath := dbPath + ".migration.lock"

	// 尝试创建锁文件（独占模式）
	for i := 0; i < 30; i++ {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if err == nil {
			// 成功获取锁
			logger.Debug("获取迁移锁成功", zap.String("lock", lockPath))
			return lockFile, nil
		}

		// 检查锁文件是否太旧（超过5分钟）
		if info, err := os.Stat(lockPath); err == nil {
			if time.Since(info.ModTime()) > 5*time.Minute {
				logger.Warn("迁移锁文件过期，尝试删除", zap.String("lock", lockPath))
				os.Remove(lockPath)
				continue
			}
		}

		logger.Debug("等待迁移锁...", zap.Int("attempt", i+1))
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("无法获取迁移锁，可能有其他进程正在执行迁移")
}

// releaseMigrationLock 释放迁移锁
func releaseMigrationLock(lockFile *os.File) {
	if lockFile == nil {
		return
	}

	lockPath := lockFile.Name()
	lockFile.Close()
	os.Remove(lockPath)
	logger.Debug("释放迁移锁", zap.String("lock", lockPath))
}

// getDBPath 从配置获取数据库文件路径
func getDBPath() string {
	if DB == nil {
		return "./data/slot-game.db"
	}

	// 从连接字符串获取路径
	// SQLite DSN 就是文件路径
	switch DB.Dialector.Name() {
	case "sqlite", "sqlite3":
		// 获取数据库文件路径
		if sqlDB, err := DB.DB(); err == nil {
			// 查询数据库文件路径
			row := sqlDB.QueryRow("PRAGMA database_list")
			var seq int
			var name, file string
			if err := row.Scan(&seq, &name, &file); err == nil && file != "" {
				return file
			}
		}
		return "./data/slot-game.db"
	default:
		return ""
	}
}

// CleanupStaleLocks 清理过期的锁文件
func CleanupStaleLocks() {
	patterns := []string{
		"./data/*.lock",
		"./data/slot-game.db*.lock",
		"./*.lock",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, lockFile := range matches {
			if info, err := os.Stat(lockFile); err == nil {
				if time.Since(info.ModTime()) > 10*time.Minute {
					logger.Info("清理过期锁文件", zap.String("file", lockFile))
					os.Remove(lockFile)
				}
			}
		}
	}
}