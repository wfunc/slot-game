package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wfunc/slot-game/internal/logger"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SerialLogService 串口日志服务
type SerialLogService struct {
	repo      *repository.SerialLogRepository
	logger    *zap.Logger
	mu        sync.Mutex
	buffer    []*models.SerialLog
	bufferCh  chan *models.SerialLog
	stopCh    chan struct{}
	sessionID string
}

// NewSerialLogService 创建串口日志服务
func NewSerialLogService(db *gorm.DB) *SerialLogService {
	service := &SerialLogService{
		repo:      repository.NewSerialLogRepository(db),
		logger:    logger.GetLogger(),
		buffer:    make([]*models.SerialLog, 0, 100),
		bufferCh:  make(chan *models.SerialLog, 1000),
		stopCh:    make(chan struct{}),
		sessionID: uuid.New().String(),
	}

	// 启动后台写入协程
	go service.backgroundWriter()

	return service
}

// backgroundWriter 后台写入协程
func (s *SerialLogService) backgroundWriter() {
	ticker := time.NewTicker(5 * time.Second) // 每5秒批量写入一次
	defer ticker.Stop()

	for {
		select {
		case log := <-s.bufferCh:
			s.mu.Lock()
			s.buffer = append(s.buffer, log)
			// 如果缓冲区满了，立即写入
			if len(s.buffer) >= 100 {
				s.flushBuffer()
			}
			s.mu.Unlock()

		case <-ticker.C:
			s.mu.Lock()
			s.flushBuffer()
			s.mu.Unlock()

		case <-s.stopCh:
			// 退出前写入剩余的日志
			s.mu.Lock()
			s.flushBuffer()
			s.mu.Unlock()
			return
		}
	}
}

// flushBuffer 写入缓冲区的日志到数据库
func (s *SerialLogService) flushBuffer() {
	if len(s.buffer) == 0 {
		return
	}

	if err := s.repo.CreateBatch(s.buffer); err != nil {
		s.logger.Error("批量写入串口日志失败", zap.Error(err))
	} else {
		s.logger.Debug("批量写入串口日志成功", zap.Int("count", len(s.buffer)))
	}

	// 清空缓冲区
	s.buffer = s.buffer[:0]
}

// LogACMSend 记录ACM发送日志
func (s *SerialLogService) LogACMSend(command string, rawData []byte, hexData string, requestID string) {
	log := &models.SerialLog{
		DeviceType:  models.SerialLogTypeACMSend,
		Direction:   "SEND",
		Level:       models.SerialLogLevelInfo,
		Command:     command,
		RawData:     string(rawData),
		HexData:     hexData,
		BytesCount:  len(rawData),
		RequestID:   requestID,
		SessionID:   s.sessionID,
		CreatedAt:   time.Now(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 解析命令类型和参数
	s.parseACMCommand(log, command)

	// 异步写入
	select {
	case s.bufferCh <- log:
	default:
		s.logger.Warn("串口日志缓冲区满，丢弃日志")
	}
}

// LogACMReceive 记录ACM接收日志
func (s *SerialLogService) LogACMReceive(rawData string, hexData string, jsonMsg map[string]interface{}, requestID string) {
	log := &models.SerialLog{
		DeviceType:  models.SerialLogTypeACMReceive,
		Direction:   "RECEIVE",
		Level:       models.SerialLogLevelInfo,
		RawData:     rawData,
		HexData:     hexData,
		BytesCount:  len(rawData),
		RequestID:   requestID,
		SessionID:   s.sessionID,
		CreatedAt:   time.Now(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 如果有JSON数据
	if jsonMsg != nil {
		log.JSONData = models.JSONData(jsonMsg)

		// 提取关键信息
		if function, ok := jsonMsg["function"].(string); ok {
			log.Function = function
		}
		if code, ok := jsonMsg["code"].(float64); ok {
			log.ResponseCode = int(code)
		}
		if msg, ok := jsonMsg["msg"].(string); ok {
			log.ResponseMsg = msg
		}
		if ident, ok := jsonMsg["ident"].(float64); ok {
			log.Ident = int(ident)
		}

		// 提取游戏数据
		if bet, ok := jsonMsg["bet"].(float64); ok {
			log.Bet = bet
		}
		if prize, ok := jsonMsg["prize"].(float64); ok {
			log.Prize = prize
		}
		if win, ok := jsonMsg["win"].(float64); ok {
			log.Win = win
		}

		// 如果有错误
		if log.ResponseCode != 0 {
			log.Level = models.SerialLogLevelError
			if errMsg, ok := jsonMsg["error"].(string); ok {
				log.ErrorMsg = errMsg
			}
		}
	}

	// 异步写入
	select {
	case s.bufferCh <- log:
	default:
		s.logger.Warn("串口日志缓冲区满，丢弃日志")
	}
}

// LogSTM32Send 记录STM32发送日志
func (s *SerialLogService) LogSTM32Send(frameData []byte, hexData string, command uint8, requestID string) {
	log := &models.SerialLog{
		DeviceType:  models.SerialLogTypeSTM32Send,
		Direction:   "SEND",
		Level:       models.SerialLogLevelInfo,
		RawData:     fmt.Sprintf("CMD:0x%02X", command),
		HexData:     hexData,
		BytesCount:  len(frameData),
		Command:     fmt.Sprintf("0x%02X", command),
		RequestID:   requestID,
		SessionID:   s.sessionID,
		CreatedAt:   time.Now(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 异步写入
	select {
	case s.bufferCh <- log:
	default:
		s.logger.Warn("串口日志缓冲区满，丢弃日志")
	}
}

// LogSTM32Receive 记录STM32接收日志
func (s *SerialLogService) LogSTM32Receive(frameData []byte, hexData string, command uint8, requestID string) {
	log := &models.SerialLog{
		DeviceType:  models.SerialLogTypeSTM32Receive,
		Direction:   "RECEIVE",
		Level:       models.SerialLogLevelInfo,
		RawData:     fmt.Sprintf("CMD:0x%02X", command),
		HexData:     hexData,
		BytesCount:  len(frameData),
		Command:     fmt.Sprintf("0x%02X", command),
		RequestID:   requestID,
		SessionID:   s.sessionID,
		CreatedAt:   time.Now(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 异步写入
	select {
	case s.bufferCh <- log:
	default:
		s.logger.Warn("串口日志缓冲区满，丢弃日志")
	}
}

// LogError 记录错误日志
func (s *SerialLogService) LogError(deviceType models.SerialLogType, direction string, errorMsg string, rawData string) {
	log := &models.SerialLog{
		DeviceType:  deviceType,
		Direction:   direction,
		Level:       models.SerialLogLevelError,
		ErrorMsg:    errorMsg,
		RawData:     rawData,
		SessionID:   s.sessionID,
		CreatedAt:   time.Now(),
		Timestamp:   time.Now().UnixMilli(),
	}

	// 异步写入
	select {
	case s.bufferCh <- log:
	default:
		s.logger.Warn("串口日志缓冲区满，丢弃日志")
	}
}

// parseACMCommand 解析ACM命令
func (s *SerialLogService) parseACMCommand(log *models.SerialLog, command string) {
	// 解析命令类型
	parts := strings.Fields(command)
	if len(parts) > 0 {
		log.Function = parts[0]

		// 特殊处理 algo 命令
		if parts[0] == "algo" {
			for i := 1; i < len(parts); i += 2 {
				if i+1 >= len(parts) {
					break
				}
				switch parts[i] {
				case "-b":
					if bet, err := parseFloat(parts[i+1]); err == nil {
						log.Bet = bet
					}
				case "-p":
					if prize, err := parseFloat(parts[i+1]); err == nil {
						log.Prize = prize
					}
				}
			}
		}
	}
}

// parseFloat 解析浮点数
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// Query 查询日志
func (s *SerialLogService) Query(query *models.SerialLogQuery) ([]*models.SerialLog, int64, error) {
	return s.repo.Query(query)
}

// GetStats 获取统计信息
func (s *SerialLogService) GetStats(startTime, endTime *time.Time) (*models.SerialLogStats, error) {
	return s.repo.GetStats(startTime, endTime)
}

// GetLatestLogs 获取最新的日志
func (s *SerialLogService) GetLatestLogs(limit int, deviceType models.SerialLogType) ([]*models.SerialLog, error) {
	return s.repo.GetLatest(limit, deviceType)
}

// GetAlgoCommandLogs 获取algo命令日志
func (s *SerialLogService) GetAlgoCommandLogs(startTime, endTime *time.Time, limit int) ([]*models.SerialLog, error) {
	return s.repo.GetAlgoCommandLogs(startTime, endTime, limit)
}

// GetErrorLogs 获取错误日志
func (s *SerialLogService) GetErrorLogs(limit int) ([]*models.SerialLog, error) {
	return s.repo.GetErrorLogs(limit)
}

// CleanupOldLogs 清理旧日志
func (s *SerialLogService) CleanupOldLogs(retentionDays int) (int64, error) {
	return s.repo.CleanupLogs(retentionDays)
}

// ExportLogs 导出日志为JSON格式
func (s *SerialLogService) ExportLogs(query *models.SerialLogQuery) ([]byte, error) {
	logs, _, err := s.Query(query)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(logs, "", "  ")
}

// GenerateRequestID 生成请求ID
func (s *SerialLogService) GenerateRequestID() string {
	return uuid.New().String()
}

// Close 关闭服务
func (s *SerialLogService) Close() {
	close(s.stopCh)
	// 等待一段时间确保数据写入完成
	time.Sleep(1 * time.Second)
}