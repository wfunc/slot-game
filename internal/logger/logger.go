package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	once   sync.Once
	mu     sync.RWMutex
	
	// 模块日志器
	moduleLoggers map[string]*zap.Logger
)

// Init 初始化日志系统
func Init(cfg *config.LogConfig) error {
	var err error
	once.Do(func() {
		moduleLoggers = make(map[string]*zap.Logger)
		
		// 解析日志级别
		level := parseLevel(cfg.Level)
		
		// 创建编码器配置
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		
		// 根据格式选择编码器
		var encoder zapcore.Encoder
		if cfg.Format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}
		
		// 创建输出核心
		var cores []zapcore.Core
		
		// 控制台输出
		if cfg.Output == "stdout" || cfg.Output == "both" {
			consoleCore := zapcore.NewCore(
				encoder,
				zapcore.AddSync(os.Stdout),
				level,
			)
			cores = append(cores, consoleCore)
		}
		
		// 文件输出
		if cfg.Output == "file" || cfg.Output == "both" {
			// 确保日志目录存在
			logDir := cfg.File.Path
			if err = os.MkdirAll(logDir, 0755); err != nil {
				return
			}
			
			// 创建文件写入器（支持日志轮转）
			fileWriter := &lumberjack.Logger{
				Filename:   filepath.Join(logDir, cfg.File.Filename),
				MaxSize:    cfg.File.MaxSize,    // MB
				MaxAge:     cfg.File.MaxAge,     // days
				MaxBackups: cfg.File.MaxBackups, // 保留文件数
				Compress:   cfg.File.Compress,   // 是否压缩
			}
			
			fileCore := zapcore.NewCore(
				encoder,
				zapcore.AddSync(fileWriter),
				level,
			)
			cores = append(cores, fileCore)
			
			// 创建错误日志文件
			errorWriter := &lumberjack.Logger{
				Filename:   filepath.Join(logDir, "error.log"),
				MaxSize:    cfg.File.MaxSize,
				MaxAge:     cfg.File.MaxAge,
				MaxBackups: cfg.File.MaxBackups,
				Compress:   cfg.File.Compress,
			}
			
			errorCore := zapcore.NewCore(
				encoder,
				zapcore.AddSync(errorWriter),
				zapcore.ErrorLevel,
			)
			cores = append(cores, errorCore)
		}
		
		// 创建核心
		core := zapcore.NewTee(cores...)
		
		// 创建日志器
		logger = zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		
		sugar = logger.Sugar()
		
		// 初始化模块日志器
		if cfg.Modules != nil {
			for module, levelStr := range cfg.Modules {
				moduleLevel := parseLevel(levelStr)
				moduleCore := zapcore.NewCore(
					encoder,
					zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)),
					moduleLevel,
				)
				moduleLoggers[module] = zap.New(
					moduleCore,
					zap.AddCaller(),
					zap.AddCallerSkip(1),
				)
			}
		}
	})
	
	return err
}

// parseLevel 解析日志级别
func parseLevel(levelStr string) zapcore.Level {
	var level zapcore.Level
	switch levelStr {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}
	return level
}

// GetLogger 获取日志器
func GetLogger() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if logger == nil {
		// 如果未初始化，使用默认配置
		defaultLogger, _ := zap.NewProduction()
		return defaultLogger
	}
	return logger
}

// GetSugar 获取Sugar日志器
func GetSugar() *zap.SugaredLogger {
	mu.RLock()
	defer mu.RUnlock()
	if sugar == nil {
		return GetLogger().Sugar()
	}
	return sugar
}

// GetModuleLogger 获取模块日志器
func GetModuleLogger(module string) *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	
	if moduleLogger, ok := moduleLoggers[module]; ok {
		return moduleLogger
	}
	
	// 如果模块日志器不存在，返回默认日志器
	return GetLogger()
}

// Sync 同步日志缓冲区
func Sync() error {
	mu.RLock()
	defer mu.RUnlock()
	
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// 便捷方法

// Debug 输出调试日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 输出信息日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 输出警告日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 输出错误日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Debugf 格式化输出调试日志
func Debugf(template string, args ...interface{}) {
	GetSugar().Debugf(template, args...)
}

// Infof 格式化输出信息日志
func Infof(template string, args ...interface{}) {
	GetSugar().Infof(template, args...)
}

// Warnf 格式化输出警告日志
func Warnf(template string, args ...interface{}) {
	GetSugar().Warnf(template, args...)
}

// Errorf 格式化输出错误日志
func Errorf(template string, args ...interface{}) {
	GetSugar().Errorf(template, args...)
}

// Fatalf 格式化输出致命错误日志并退出程序
func Fatalf(template string, args ...interface{}) {
	GetSugar().Fatalf(template, args...)
}

// With 创建带有字段的日志器
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// WithModule 创建带有模块名的日志器
func WithModule(module string) *zap.Logger {
	return GetModuleLogger(module)
}

// LogRequest 记录请求日志
func LogRequest(method, path string, statusCode int, latency time.Duration, clientIP string) {
	GetLogger().Info("request",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", statusCode),
		zap.Duration("latency", latency),
		zap.String("client_ip", clientIP),
	)
}

// LogError 记录错误日志（带堆栈）
func LogError(err error, msg string, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	GetLogger().Error(msg, fields...)
}

// LogPanic 记录panic日志
func LogPanic(recovered interface{}, stack []byte) {
	GetLogger().Error("panic recovered",
		zap.Any("panic", recovered),
		zap.ByteString("stack", stack),
	)
}

// LogGameEvent 记录游戏事件
func LogGameEvent(event string, sessionID string, data map[string]interface{}) {
	GetModuleLogger("game").Info("game_event",
		zap.String("event", event),
		zap.String("session_id", sessionID),
		zap.Any("data", data),
	)
}

// LogSerialCommand 记录串口命令
func LogSerialCommand(cmd string, response string, success bool) {
	logger := GetModuleLogger("serial")
	if success {
		logger.Info("serial_command",
			zap.String("command", cmd),
			zap.String("response", response),
		)
	} else {
		logger.Error("serial_command_failed",
			zap.String("command", cmd),
			zap.String("response", response),
		)
	}
}

// LogWebSocketMessage 记录WebSocket消息
func LogWebSocketMessage(direction string, messageType string, payload interface{}) {
	GetModuleLogger("websocket").Debug("ws_message",
		zap.String("direction", direction), // "send" or "receive"
		zap.String("type", messageType),
		zap.Any("payload", payload),
	)
}

// LogMQTTMessage 记录MQTT消息
func LogMQTTMessage(topic string, action string, payload interface{}) {
	GetModuleLogger("mqtt").Info("mqtt_message",
		zap.String("topic", topic),
		zap.String("action", action), // "publish" or "receive"
		zap.Any("payload", payload),
	)
}

// LogDatabaseOperation 记录数据库操作
func LogDatabaseOperation(operation string, table string, duration time.Duration, err error) {
	logger := GetModuleLogger("database")
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("table", table),
		zap.Duration("duration", duration),
	}
	
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("database_operation_failed", fields...)
	} else {
		logger.Debug("database_operation", fields...)
	}
}

// RotateLogs 手动触发日志轮转（如果需要）
func RotateLogs() error {
	// lumberjack会自动处理日志轮转
	// 这里可以添加额外的轮转逻辑
	return nil
}

// SetLevel 动态设置日志级别
func SetLevel(levelStr string) {
	level := parseLevel(levelStr)
	
	mu.Lock()
	defer mu.Unlock()
	
	// 重新创建日志器以应用新的级别
	cfg := config.Get()
	if cfg != nil {
		cfg.Log.Level = levelStr
		Init(&cfg.Log)
	}
}

// Cleanup 清理日志资源
func Cleanup() {
	if err := Sync(); err != nil {
		fmt.Printf("Failed to sync logger: %v\n", err)
	}
}