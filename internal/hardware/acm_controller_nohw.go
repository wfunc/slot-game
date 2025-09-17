//go:build noserialhw
// +build noserialhw

package hardware

import (
	"fmt"
	"sync"
	"time"

	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

// ACMController ACM控制器（无硬件版本）
type ACMController struct {
	config      ACMConfig
	logger      *zap.Logger
	isConnected bool
	mu          sync.RWMutex
	stats       CoinStatistics
	statsMu     sync.RWMutex
}

// NewACMController 创建ACM控制器（无硬件版本）
func NewACMController(config ACMConfig) (*ACMController, error) {
	logger := logger.NewLogger("acm_nohw", "info", "")

	c := &ACMController{
		config: config,
		logger: logger,
	}

	c.logger.Info("ACM Controller initialized (no hardware version)",
		zap.String("mode", "simulation"))

	return c, nil
}

// Connect 模拟连接
func (c *ACMController) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Simulating ACM connection")
	c.isConnected = true

	return nil
}

// Disconnect 模拟断开连接
func (c *ACMController) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Disconnecting simulated ACM")
	c.isConnected = false

	return nil
}

// IsConnected 检查连接状态
func (c *ACMController) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// SendCommand 发送命令（模拟）
func (c *ACMController) SendCommand(cmd string) error {
	if !c.IsConnected() {
		return fmt.Errorf("ACM not connected (simulation)")
	}

	c.logger.Debug("Simulated command sent", zap.String("command", cmd))
	return nil
}

// GetStatistics 获取统计信息
func (c *ACMController) GetStatistics() CoinStatistics {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

// UpdateStatistics 更新统计信息
func (c *ACMController) UpdateStatistics(stats CoinStatistics) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats = stats
}

// Start 启动控制器
func (c *ACMController) Start() error {
	return c.Connect()
}

// Stop 停止控制器
func (c *ACMController) Stop() error {
	return c.Disconnect()
}

// Reset 重置控制器
func (c *ACMController) Reset() error {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	c.stats = CoinStatistics{
		LastInsertTime: time.Now(),
		LastReturnTime: time.Now(),
	}

	c.logger.Info("ACM controller reset (simulation)")
	return nil
}