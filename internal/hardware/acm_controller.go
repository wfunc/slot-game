//go:build !noserialhw
// +build !noserialhw

package hardware

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
	"github.com/wfunc/slot-game/internal/logger"
	"github.com/wfunc/slot-game/internal/service"
	"go.uber.org/zap"
)

// DefaultACMConfig 默认ACM配置
func DefaultACMConfig() *ACMConfig {
	return &ACMConfig{
		Port:         "auto",
		BaudRate:     115200,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
		AutoDetect:   true,
		// Algo定时器默认配置
		AlgoTimerEnabled:  false,
		AlgoTimerInterval: 1 * time.Second,
		AlgoBet:           1,
		AlgoPrize:         100,
	}
}

// ACMController ACM设备控制器
type ACMController struct {
	config       *ACMConfig
	port         *serial.Port
	reconnectMgr *SerialReconnectManager // 重连管理器
	mu           sync.RWMutex
	connected    bool
	logger       *zap.Logger

	// 通道
	stopCh chan struct{}
	cmdCh  chan string
	respCh chan interface{}

	// 回调函数
	onCommand func(cmd string) (interface{}, error)
	onMessage func(msg map[string]interface{})

	// STM32控制器引用（用于桥接）
	stm32Controller *STM32Controller

	// Algo定时器
	algoTimer       *time.Ticker
	algoTimerStopCh chan struct{}

	// 串口日志服务
	serialLogService *service.SerialLogService
}

// ACMMessage ACM消息格式
type ACMMessage struct {
	MsgType  string                 `json:"MsgType"`
	Function string                 `json:"function,omitempty"`
	Command  string                 `json:"command,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// NewACMController 创建ACM控制器
func NewACMController(config *ACMConfig) *ACMController {
	if config == nil {
		config = DefaultACMConfig()
	}

	return &ACMController{
		config: config,
		logger: logger.GetLogger(),
		stopCh: make(chan struct{}),
		cmdCh:  make(chan string, 100),
		respCh: make(chan interface{}, 100),
	}
}

// SetSTM32Controller 设置STM32控制器引用（用于桥接）
func (c *ACMController) SetSTM32Controller(stm32 *STM32Controller) {
	c.stm32Controller = stm32
}

// SetSerialLogService 设置串口日志服务
func (c *ACMController) SetSerialLogService(service *service.SerialLogService) {
	c.serialLogService = service
}

// Connect 连接ACM设备
func (c *ACMController) Connect() error {
	c.mu.Lock()

	if c.connected {
		c.mu.Unlock()
		return nil
	}

	c.logger.Info("ACM Connect() 函数开始执行",
		zap.Bool("algo_enabled", c.config.AlgoTimerEnabled),
		zap.Duration("algo_interval", c.config.AlgoTimerInterval))

	// 初始化实际的连接配置
	serialCfg := &serial.Config{
		Baud:        c.config.BaudRate,
		Size:        8,
		StopBits:    serial.Stop2,
		Parity:      serial.ParityNone,
		ReadTimeout: 100 * time.Millisecond,
	}

	// 创建重连管理器
	c.reconnectMgr = NewSerialReconnectManager("ACM", "ttyACM", serialCfg)
	c.logger.Info("重连管理器已创建")

	// 设置回调
	c.reconnectMgr.SetCallbacks(
		// 连接成功回调
		func(port *serial.Port) error {
			c.logger.Info("ACM连接回调开始执行")

			// 缓存配置（避免后续访问时的潜在锁问题）
			algoEnabled := c.config.AlgoTimerEnabled
			algoInterval := c.config.AlgoTimerInterval

			// 更新状态
			c.port = port
			c.connected = true

			// 创建通道
			c.stopCh = make(chan struct{})
			c.cmdCh = make(chan string, 10)
			c.respCh = make(chan interface{}, 10)

			c.logger.Info("ACM通道已创建，准备启动后台任务")

			// 启动后台任务
			go c.readLoop()
			go c.processLoop()

			c.logger.Info("ACM后台任务已启动")

			c.logger.Info("步骤1: 准备获取设备路径")
			// 获取设备路径（简化处理，避免潜在的锁问题）
			devicePath := "/dev/ttyACM"
			c.logger.Info("步骤2: 设备路径获取完成", zap.String("path", devicePath))

			c.logger.Info("ACM连接成功",
				zap.String("device", devicePath))

			c.logger.Info("步骤3: 准备等待设备就绪")
			// 等待设备就绪
			time.Sleep(500 * time.Millisecond)
			c.logger.Info("步骤4: 设备就绪等待完成")

			c.logger.Info("设备就绪，检查Algo定时器配置",
				zap.Bool("enabled", algoEnabled))

			c.logger.Info("步骤5: 检查是否启动Algo定时器",
				zap.Bool("enabled", algoEnabled))

			// 启动Algo定时器
			if algoEnabled {
				c.logger.Info("步骤6: Algo定时器已启用，准备启动...",
					zap.Bool("enabled", algoEnabled),
					zap.Duration("interval", algoInterval))
				// 异步启动定时器，避免阻塞
				go func() {
					c.logger.Info("步骤7: Algo定时器goroutine已启动，等待500ms")
					time.Sleep(500 * time.Millisecond)
					c.logger.Info("步骤8: 准备调用startAlgoTimer()")
					c.startAlgoTimer()
					c.logger.Info("步骤9: startAlgoTimer()调用完成")
				}()
				c.logger.Info("步骤10: Algo定时器goroutine已创建")
			} else {
				c.logger.Info("Algo定时器未启用",
					zap.Bool("enabled", algoEnabled))
			}

			c.logger.Info("步骤11: 准备返回nil")
			c.logger.Info("ACM连接回调执行完成")
			return nil
		},
		// 断开连接回调
		func() {
			c.connected = false
			c.stopAlgoTimer()
			if c.stopCh != nil {
				close(c.stopCh)
				c.stopCh = nil
			}
			c.logger.Info("ACM连接已断开")
		},
		// 重连成功回调
		func(port *serial.Port) error {
			c.logger.Info("ACM重连回调开始执行")

			// 缓存配置（避免后续访问时的潜在锁问题）
			algoEnabled := c.config.AlgoTimerEnabled
			algoInterval := c.config.AlgoTimerInterval

			// 更新状态
			c.port = port
			c.connected = true

			// 创建通道
			c.stopCh = make(chan struct{})
			c.cmdCh = make(chan string, 10)
			c.respCh = make(chan interface{}, 10)

			// 重启后台任务
			go c.readLoop()
			go c.processLoop()

			// 获取设备路径（简化处理）
			devicePath := "/dev/ttyACM"

			c.logger.Info("ACM重连成功",
				zap.String("new_device", devicePath))

			// 重启Algo定时器
			if algoEnabled {
				c.logger.Info("重连后重启Algo定时器...",
					zap.Bool("enabled", algoEnabled),
					zap.Duration("interval", algoInterval))
				// 异步启动定时器，避免阻塞
				go func() {
					time.Sleep(500 * time.Millisecond)
					c.startAlgoTimer()
				}()
			} else {
				c.logger.Info("重连后Algo定时器未启用",
					zap.Bool("enabled", algoEnabled))
			}

			c.logger.Info("ACM重连回调执行完成")
			return nil
		},
	)

	// 在调用Start()之前释放锁，避免死锁
	c.mu.Unlock()

	c.logger.Info("准备启动重连管理器...")
	// 启动重连管理器（不持有锁）
	err := c.reconnectMgr.Start()
	if err != nil {
		c.logger.Error("重连管理器启动失败", zap.Error(err))
		return err
	}
	c.logger.Info("重连管理器启动成功")
	return nil
}

// Disconnect 断开连接
func (c *ACMController) Disconnect() error {
	c.mu.Lock()

	if !c.connected {
		c.mu.Unlock()
		return nil
	}

	c.logger.Info("正在断开ACM连接...")

	// 标记为未连接
	c.connected = false

	// 首先关闭停止通道，通知所有goroutine退出（包括Algo定时器）
	if c.stopCh != nil {
		c.logger.Info("关闭停止通道，通知所有goroutine退出...")
		close(c.stopCh)
		c.stopCh = nil
	}

	// 释放锁以避免stopAlgoTimer中的死锁
	c.mu.Unlock()

	// 停止Algo定时器（这会关闭algoTimerStopCh）
	c.logger.Info("停止Algo定时器...")
	c.stopAlgoTimer()

	// 停止重连管理器（这可能会阻塞，所以不持有锁）
	c.mu.Lock()
	mgr := c.reconnectMgr
	c.reconnectMgr = nil
	c.mu.Unlock()

	if mgr != nil {
		c.logger.Info("停止重连管理器...")
		mgr.Stop()
	}

	// 等待goroutine退出
	c.logger.Info("等待goroutine退出...")
	time.Sleep(200 * time.Millisecond)

	// 清理资源
	c.mu.Lock()

	// 关闭命令通道
	if c.cmdCh != nil {
		c.logger.Info("关闭命令通道...")
		close(c.cmdCh)
		c.cmdCh = nil
	}

	// 关闭串口
	if c.port != nil {
		c.logger.Info("关闭串口...")
		err := c.port.Close()
		if err != nil {
			c.logger.Warn("关闭ACM串口时出错", zap.Error(err))
		}
		c.port = nil
	}

	c.logger.Info("ACM控制器已断开")
	c.mu.Unlock()

	return nil
}

// IsConnected 检查连接状态
func (c *ACMController) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// findACMDevice 自动查找ACM设备
func (c *ACMController) findACMDevice() string {
	c.logger.Info("开始自动检测ACM设备...")

	// Linux设备路径
	patterns := []string{
		"/dev/ttyACM*",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			c.logger.Debug("扫描设备失败", zap.String("pattern", pattern), zap.Error(err))
			continue
		}

		c.logger.Debug("扫描到设备", zap.String("pattern", pattern), zap.Strings("devices", matches))

		// 优先选择ACM设备
		for _, device := range matches {
			if strings.Contains(device, "ACM") {
				c.logger.Info("找到ACM设备", zap.String("device", device))
				return device
			}
		}

		// 如果没有ACM，返回第一个USB设备
		if len(matches) > 0 {
			c.logger.Info("未找到ACM设备，使用USB设备", zap.String("device", matches[0]))
			return matches[0]
		}
	}

	// macOS设备路径
	macPatterns := []string{
		"/dev/cu.usbmodem*",
		"/dev/tty.usbmodem*",
	}

	for _, pattern := range macPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

// readLoop 读取循环
func (c *ACMController) readLoop() {
	defer func() {
		c.logger.Info("ACM读取循环已退出")
	}()

	buffer := make([]byte, 1024)
	var msgBuffer string

	for {
		select {
		case <-c.stopCh:
			c.logger.Info("收到停止信号，退出读取循环")
			return
		default:
			// 检查串口是否有效
			if c.port == nil {
				c.logger.Warn("串口已关闭，退出读取循环")
				return
			}

			// 读取数据
			n, err := c.port.Read(buffer)
			if err != nil {
				// 重要：EOF不是致命错误，某些USB-CDC设备会定期发送EOF
				// 参考测试工具的实现，EOF应该被忽略
				if strings.Contains(err.Error(), "EOF") {
					// EOF是正常的，继续读取
					continue
				}
				// 忽略超时错误
				if !strings.Contains(err.Error(), "timeout") {
					c.logger.Debug("读取ACM数据错误", zap.Error(err))
				}
				// 只有在真正的错误时才考虑断开
				// 例如：device not configured, broken pipe等
				if strings.Contains(err.Error(), "device not configured") ||
					strings.Contains(err.Error(), "broken pipe") ||
					strings.Contains(err.Error(), "input/output error") {
					c.logger.Error("ACM设备断开连接", zap.Error(err))
					c.mu.Lock()
					c.connected = false
					c.mu.Unlock()
					return
				}
				// 短暂休眠避免CPU占用过高
				time.Sleep(10 * time.Millisecond)
				continue
			}

			if n > 0 {
				receivedData := string(buffer[:n])
				msgBuffer += receivedData

				// 打印原始接收数据（十六进制和ASCII）
				c.logger.Debug("ACM接收原始数据",
					zap.String("ascii", receivedData),
					zap.String("hex", fmt.Sprintf("% X", buffer[:n])),
					zap.Int("bytes", n))

				// 处理完整的消息（以\n或\r\n结尾）
				for {
					idx := strings.Index(msgBuffer, "\n")
					if idx == -1 {
						break
					}

					msg := strings.TrimSpace(msgBuffer[:idx])
					msgBuffer = msgBuffer[idx+1:]

					if msg != "" {
						// 跳过 "end" 和 ">" 标记
						if msg == "end" || msg == ">" {
							c.logger.Debug("收到结束标记", zap.String("marker", msg))
							continue
						}

						// 记录所有消息，帮助了解ACM设备的响应格式
						if strings.Contains(msg, "help") || strings.Contains(msg, "Help") ||
							strings.Contains(msg, "command") || strings.Contains(msg, "Command") {
							c.logger.Info("ACM帮助信息", zap.String("message", msg))
						} else if strings.HasPrefix(msg, "{") && strings.HasSuffix(msg, "}") {
							c.logger.Info("ACM JSON响应", zap.String("message", msg))
						} else {
							c.logger.Debug("ACM接收消息", zap.String("message", msg))
						}
						c.processMessage(msg)
					}
				}
			}
		}
	}
}

// processMessage 处理接收到的消息
func (c *ACMController) processMessage(msg string) {
	c.logger.Debug("收到ACM消息", zap.String("message", msg))

	// 尝试解析为JSON
	if strings.HasPrefix(msg, "{") {
		var jsonMsg map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &jsonMsg); err == nil {
			// 成功解析为JSON，作为ACM消息处理
			c.handleJSONMessage(jsonMsg)
			return
		} else {
			// JSON解析失败，可能是格式错误或不完整的JSON
			c.logger.Debug("JSON解析失败，忽略", zap.String("message", msg), zap.Error(err))
			return
		}
	}

	// 只有非JSON格式的消息才作为命令处理
	c.handleCommand(msg)
}

// handleJSONMessage 处理JSON消息
func (c *ACMController) handleJSONMessage(msg map[string]interface{}) {
	msgType, _ := msg["MsgType"].(string)

	switch msgType {
	case "M1": // APP控制消息
		c.handleAppControl(msg)
	case "M2": // ACM响应消息
		c.handleACMResponse(msg)
	case "M4": // 游戏控制消息
		c.handleGameControl(msg)
	default:
		if c.onMessage != nil {
			c.onMessage(msg)
		}
	}
}

// handleCommand 处理简单命令
func (c *ACMController) handleCommand(cmd string) {
	// 移除可能的结束符
	cmd = strings.TrimSuffix(cmd, ">")
	cmd = strings.TrimSpace(cmd)

	// 忽略空命令和提示符
	if cmd == "" || cmd == ">" {
		return
	}

	// 忽略ACM设备的错误响应，避免循环
	if strings.Contains(cmd, "Command not recognised") ||
		strings.Contains(cmd, "Enter 'help'") {
		c.logger.Debug("忽略ACM错误响应", zap.String("message", cmd))
		return
	}

	// 忽略看起来像JSON但解析失败的数据（避免将ACM的JSON响应当作命令）
	if strings.HasPrefix(cmd, "{") || strings.Contains(cmd, "\"code\"") ||
		strings.Contains(cmd, "\"msg\"") || strings.Contains(cmd, "\"ident\"") {
		c.logger.Debug("忽略非命令格式的数据", zap.String("message", cmd))
		return
	}

	var response interface{}
	var err error

	// 如果有命令处理器，使用它
	if c.onCommand != nil {
		response, err = c.onCommand(cmd)
	} else {
		// 默认命令处理
		response, err = c.defaultCommandHandler(cmd)
	}

	// 发送响应
	if err != nil {
		c.sendResponse(map[string]interface{}{
			"error": err.Error(),
		})
	} else if response != nil {
		c.sendResponse(response)
	}
}

// defaultCommandHandler 默认命令处理器
func (c *ACMController) defaultCommandHandler(cmd string) (interface{}, error) {
	// 处理algo命令（格式：algo -b 1 -p 100）
	if strings.HasPrefix(cmd, "algo") {
		return c.handleAlgoCommand(cmd)
	}

	switch cmd {
	case "ver":
		return map[string]interface{}{
			"version": "1.0.0",
			"device":  "ACM Controller",
		}, nil

	case "sta":
		return map[string]interface{}{
			"status":    "ready",
			"connected": c.IsConnected(),
		}, nil

	case "test":
		return map[string]interface{}{
			"result":    "ok",
			"timestamp": time.Now().Unix(),
		}, nil

	default:
		// 如果有STM32控制器，尝试转发命令
		if c.stm32Controller != nil {
			return c.forwardToSTM32(cmd)
		}
		return nil, fmt.Errorf("未知命令: %s", cmd)
	}
}

// forwardToSTM32 转发命令到STM32
func (c *ACMController) forwardToSTM32(cmd string) (interface{}, error) {
	// 这里实现ACM命令到STM32协议的转换
	// 根据命令类型构建相应的STM32帧

	switch cmd {
	case "coin_status":
		// 查询币数状态
		err := c.stm32Controller.QueryStatus(0x01)
		if err != nil {
			return nil, err
		}
		// 等待响应...
		return map[string]interface{}{
			"coins": c.stm32Controller.GetStatistics().CoinsInserted,
		}, nil

	case "start_game":
		// 开始游戏
		if c.stm32Controller.gameLogic != nil {
			c.stm32Controller.gameLogic.StartGame(1)
		}
		return map[string]interface{}{
			"result": "game_started",
		}, nil

	default:
		return nil, fmt.Errorf("无法转发命令: %s", cmd)
	}
}

// sendResponse 发送响应
func (c *ACMController) sendResponse(data interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接")
	}

	var response []byte

	switch v := data.(type) {
	case string:
		response = []byte(v + "\n>")
	case []byte:
		response = append(v, []byte("\n>")...)
	default:
		// 转换为JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		response = append(jsonData, []byte("\n>")...)
	}

	// 打印发送数据（ASCII和十六进制）
	c.logger.Info("ACM发送数据",
		zap.String("ascii", string(response)),
		zap.String("hex", fmt.Sprintf("% X", response)),
		zap.Int("bytes", len(response)))

	n, err := c.port.Write(response)
	if err != nil {
		c.logger.Error("发送ACM响应失败", zap.Error(err))
		// 触发重连
		if c.reconnectMgr != nil {
			c.reconnectMgr.HandleError(err)
		}
		return err
	}

	c.logger.Debug("ACM发送成功", zap.Int("bytes_written", n))
	return nil
}

// processLoop 处理循环
func (c *ACMController) processLoop() {
	defer func() {
		c.logger.Info("ACM处理循环已退出")
	}()

	for {
		select {
		case <-c.stopCh:
			c.logger.Info("收到停止信号，退出处理循环")
			return
		case cmd, ok := <-c.cmdCh:
			if !ok {
				c.logger.Info("命令通道已关闭，退出处理循环")
				return
			}
			// 重要修复：发送命令到串口，而不是处理它
			// 使用\r\n作为行尾符，与测试工具一致
			cmdBytes := []byte(cmd + "\r\n")
			c.logger.Debug("发送命令到ACM设备",
				zap.String("command", cmd),
				zap.String("hex", fmt.Sprintf("% X", cmdBytes)),
				zap.Int("bytes", len(cmdBytes)))

			if c.port != nil {
				n, err := c.port.Write(cmdBytes)
				if err != nil {
					c.logger.Error("发送命令失败", zap.Error(err))
					// 触发重连
					if c.reconnectMgr != nil {
						c.reconnectMgr.HandleError(err)
					}
				} else {
					c.logger.Debug("命令发送成功", zap.Int("bytes_written", n))
				}
			}
		case resp, ok := <-c.respCh:
			if !ok {
				c.logger.Info("响应通道已关闭，退出处理循环")
				return
			}
			c.sendResponse(resp)
		}
	}
}

// handleAppControl 处理APP控制消息
func (c *ACMController) handleAppControl(msg map[string]interface{}) {
	action, _ := msg["action"].(string)

	switch action {
	case "coin_insert":
		// 模拟投币
		if c.stm32Controller != nil && c.stm32Controller.gameLogic != nil {
			count, _ := msg["count"].(float64)
			c.stm32Controller.gameLogic.AddCredits(byte(count))
		}

	case "game_start":
		// 开始游戏
		if c.stm32Controller != nil && c.stm32Controller.gameLogic != nil {
			c.stm32Controller.gameLogic.StartGame(1)
		}

	case "refund":
		// 退币
		if c.stm32Controller != nil {
			c.stm32Controller.RefundCoins(1)
		}
	}
}

// handleACMResponse 处理ACM响应消息
func (c *ACMController) handleACMResponse(msg map[string]interface{}) {
	function, _ := msg["function"].(string)

	switch function {
	case "algo":
		// 算法响应
		c.logger.Info("收到算法响应",
			zap.Any("win", msg["win"]),
			zap.Any("hp30", msg["hp30"]))
	}
}

// handleGameControl 处理游戏控制消息
func (c *ACMController) handleGameControl(msg map[string]interface{}) {
	action, _ := msg["action"].(string)

	switch action {
	case "wait":
		c.logger.Info("游戏等待中")
	case "start":
		c.logger.Info("游戏开始")
	case "end":
		c.logger.Info("游戏结束")
	}
}

// SendCommand 发送命令
func (c *ACMController) SendCommand(cmd string) error {
	if !c.IsConnected() {
		return fmt.Errorf("未连接")
	}

	select {
	case c.cmdCh <- cmd:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("命令队列已满")
	}
}

// SendMessage 发送JSON消息
func (c *ACMController) SendMessage(msg interface{}) error {
	return c.sendResponse(msg)
}

// SetCommandHandler 设置命令处理器
func (c *ACMController) SetCommandHandler(handler func(cmd string) (interface{}, error)) {
	c.onCommand = handler
}

// SetMessageHandler 设置消息处理器
func (c *ACMController) SetMessageHandler(handler func(msg map[string]interface{})) {
	c.onMessage = handler
}

// GetStatistics 获取统计信息
func (c *ACMController) GetStatistics() *CoinStatistics {
	// ACM设备不追踪币的统计信息，返回空统计
	return &CoinStatistics{
		CoinsInserted:      0,
		CoinsDispensed:     0,
		CoinsReturnedFront: 0,
		CoinsReturnedLeft:  0,
		CoinsReturnedRight: 0,
		CoinsRefunded:      0,
		TicketsPrinted:     0,
		FaultCount:         0,
		RecoveryCount:      0,
		GameDuration:       0,
		Timestamp:          time.Now(),
		ReturnRate:         0,
	}
}

// handleAlgoCommand 处理算法命令
// 格式: algo -b 1 -p 100
// 返回JSON格式的算法结果
func (c *ACMController) handleAlgoCommand(cmd string) (map[string]interface{}, error) {
	// 解析参数
	parts := strings.Fields(cmd)
	bet := 1
	prize := 100

	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "-b":
			if i+1 < len(parts) {
				fmt.Sscanf(parts[i+1], "%d", &bet)
			}
		case "-p":
			if i+1 < len(parts) {
				fmt.Sscanf(parts[i+1], "%d", &prize)
			}
		}
	}

	// 生成算法结果（这里是模拟，实际应该调用算法模块）
	win := c.calculateWin(bet, prize)
	hp30 := c.calculateHP30()

	// 构建返回的JSON响应
	response := map[string]interface{}{
		"code":     0,
		"msg":      "success",
		"ident":    c.generateIdent(),
		"function": "algo",
		"prize":    prize,
		"bet":      bet,
		"algo": map[string]interface{}{
			"part": []interface{}{
				[]interface{}{},
				map[string]interface{}{
					"l1": c.generateLine(),
					"l2": c.generateLine(),
					"l3": c.generateLine(),
					"l4": c.generateLine(),
					"l5": c.generateLine(),
				},
			},
		},
		"hp30": hp30,
		"win":  win,
		"chk":  c.generateChecksum(bet, prize, win),
	}

	c.logger.Info("处理algo命令",
		zap.Int("bet", bet),
		zap.Int("prize", prize),
		zap.Float64("win", win))

	return response, nil
}

// calculateWin 计算中奖金额
func (c *ACMController) calculateWin(bet int, prize int) float64 {
	// 简单的中奖概率计算
	// 实际应该根据游戏逻辑和概率配置计算
	randVal := time.Now().UnixNano() % 100
	if randVal < 10 { // 10%概率中奖
		return float64(bet) * 2.5
	}
	return 0.0
}

// calculateHP30 计算HP30值
func (c *ACMController) calculateHP30() int {
	// 模拟HP30计算
	return 0
}

// generateIdent 生成标识符
func (c *ACMController) generateIdent() int {
	return int(time.Now().Unix() % 10000)
}

// generateLine 生成一条线的数据
func (c *ACMController) generateLine() []int {
	line := make([]int, 5)
	for i := 0; i < 5; i++ {
		line[i] = int(time.Now().UnixNano() % 8) // 0-7的随机数
	}
	return line
}

// generateChecksum 生成校验和
func (c *ACMController) generateChecksum(bet int, prize int, win float64) string {
	// 简单的校验和生成（实际应该使用更安全的算法）
	data := fmt.Sprintf("%d:%d:%.4f:%d", bet, prize, win, time.Now().Unix())
	hash := fmt.Sprintf("%x", data)
	if len(hash) > 40 {
		hash = hash[:40]
	}
	return hash
}

// SendAlgoCommand 发送algo命令并等待响应
func (c *ACMController) SendAlgoCommand(bet int, prize int) (map[string]interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("ACM设备未连接")
	}

	// 构建命令
	cmd := fmt.Sprintf("algo -b %d -p %d", bet, prize)
	cmdBytes := []byte(cmd + "\r\n") // 修复：使用\r\n与测试工具一致

	// 打印发送的命令（ASCII和十六进制）
	c.logger.Info("ACM发送algo命令",
		zap.String("command", cmd),
		zap.String("hex", fmt.Sprintf("% X", cmdBytes)),
		zap.Int("bytes", len(cmdBytes)))

	// 发送命令
	n, err := c.port.Write(cmdBytes)
	if err != nil {
		return nil, fmt.Errorf("发送algo命令失败: %w", err)
	}

	c.logger.Debug("algo命令发送成功", zap.Int("bytes_written", n))

	// 这里应该等待并解析响应
	// 实际实现中需要添加超时和响应解析逻辑

	// 模拟返回响应
	return c.handleAlgoCommand(cmd)
}

// 以下是为了实现HardwareController接口的必需方法
// ACM设备不支持这些操作，返回错误或空操作

// DispenseCoins 出币控制 - ACM不支持
func (c *ACMController) DispenseCoins(count uint16, speed byte) error {
	return fmt.Errorf("ACM controller does not support coin dispensing")
}

// RefundCoins 退币控制 - ACM不支持
func (c *ACMController) RefundCoins(count uint16) error {
	return fmt.Errorf("ACM controller does not support coin refund")
}

// PrintTickets 打印票据 - ACM不支持
func (c *ACMController) PrintTickets(count uint16) error {
	return fmt.Errorf("ACM controller does not support ticket printing")
}

// PushControl 推币控制 - ACM不支持
func (c *ACMController) PushControl(action byte, param byte) error {
	return fmt.Errorf("ACM controller does not support push control")
}

// StartPushing 开始推币 - ACM不支持
func (c *ACMController) StartPushing() error {
	return fmt.Errorf("ACM controller does not support pushing")
}

// StopPushing 停止推币 - ACM不支持
func (c *ACMController) StopPushing() error {
	return fmt.Errorf("ACM controller does not support pushing")
}

// SetPushSpeed 设置推币速度 - ACM不支持
func (c *ACMController) SetPushSpeed(speed byte) error {
	return fmt.Errorf("ACM controller does not support push speed control")
}

// PushCoin 推币 - ACM不支持
func (c *ACMController) PushCoin(force int, duration time.Duration) error {
	return fmt.Errorf("ACM controller does not support coin pushing")
}

// LightControl 灯光控制 - ACM不支持
func (c *ACMController) LightControl(pattern byte, brightness byte, duration byte) error {
	return fmt.Errorf("ACM controller does not support light control")
}

// QueryStatus 查询状态 - ACM不支持
func (c *ACMController) QueryStatus(statusType byte) error {
	return fmt.Errorf("ACM controller does not support status query")
}

// SendHeartbeat 发送心跳 - ACM不支持
func (c *ACMController) SendHeartbeat() error {
	// ACM设备不需要心跳
	return nil
}

// FaultRecovery 故障恢复 - ACM不支持
func (c *ACMController) FaultRecovery(faultCode byte, action byte, retryCount byte) error {
	return fmt.Errorf("ACM controller does not support fault recovery")
}

// SetCoinInsertedCallback 设置投币回调 - ACM不支持
func (c *ACMController) SetCoinInsertedCallback(callback func(count byte)) {
	// ACM设备不提供投币事件
}

// SetCoinReturnedCallback 设置退币回调 - ACM不支持
func (c *ACMController) SetCoinReturnedCallback(callback func(data *CoinReturnData)) {
	// ACM设备不提供退币事件
}

// SetButtonPressedCallback 设置按键回调 - ACM不支持
func (c *ACMController) SetButtonPressedCallback(callback func(event *ButtonEvent)) {
	// ACM设备不提供按键事件
}

// SetFaultReportCallback 设置故障回调 - ACM不支持
func (c *ACMController) SetFaultReportCallback(callback func(event *FaultEvent)) {
	// ACM设备不提供故障事件
}

// startAlgoTimer 启动Algo定时器
func (c *ACMController) startAlgoTimer() {
	// 使用互斥锁保护定时器状态
	c.mu.Lock()

	if c.algoTimer != nil {
		c.logger.Info("Algo定时器已在运行，跳过启动")
		c.mu.Unlock()
		return // 定时器已经在运行
	}

	c.logger.Info("正在启动Algo定时器...",
		zap.Duration("interval", c.config.AlgoTimerInterval),
		zap.Int("bet", c.config.AlgoBet),
		zap.Int("prize", c.config.AlgoPrize))

	c.algoTimerStopCh = make(chan struct{})
	c.algoTimer = time.NewTicker(c.config.AlgoTimerInterval)

	// 释放锁，让goroutine可以独立运行
	c.mu.Unlock()

	go func() {
		c.logger.Info("Algo定时器线程已启动",
			zap.Duration("interval", c.config.AlgoTimerInterval))

		// 立即发送第一个命令
		c.logger.Info("立即发送第一个algo命令...")
		c.sendAlgoCommandAsync()

		for {
			select {
			case <-c.algoTimer.C:
				c.logger.Info("定时器触发，发送algo命令")
				c.sendAlgoCommandAsync()
			case <-c.algoTimerStopCh:
				c.logger.Info("Algo定时器线程已停止")
				return
			case <-c.stopCh:
				c.logger.Info("Algo定时器因控制器停止而退出")
				return
			}
		}
	}()
}

// stopAlgoTimer 停止Algo定时器
func (c *ACMController) stopAlgoTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("stopAlgoTimer开始执行")

	if c.algoTimer != nil {
		c.logger.Info("停止定时器ticker...")
		c.algoTimer.Stop()
		c.algoTimer = nil
	}

	if c.algoTimerStopCh != nil {
		c.logger.Info("关闭algoTimerStopCh通道...")
		close(c.algoTimerStopCh)
		c.algoTimerStopCh = nil
	}

	c.logger.Info("stopAlgoTimer执行完成")
}

// sendAlgoCommandAsync 异步发送algo命令
func (c *ACMController) sendAlgoCommandAsync() {
	go func() {
		cmd := fmt.Sprintf("algo -b %d -p %d", c.config.AlgoBet, c.config.AlgoPrize)

		c.logger.Info("【定时algo】发送命令",
			zap.String("command", cmd),
			zap.Time("timestamp", time.Now()))

		// 发送命令并获取响应
		response, err := c.SendAlgoCommand(c.config.AlgoBet, c.config.AlgoPrize)
		if err != nil {
			c.logger.Error("【定时algo】命令执行失败",
				zap.Error(err),
				zap.String("command", cmd))

			// 检查是否是串口I/O错误，触发重连
			if c.reconnectMgr != nil {
				c.reconnectMgr.HandleError(err)
			}
			return
		}

		// 记录响应（检查字段是否存在）
		if response != nil {
			c.logger.Info("【定时algo】命令响应成功",
				zap.Any("response", response))

			// 安全地获取字段值
			if win, ok := response["win"].(float64); ok {
				c.logger.Info("【定时algo】结果", zap.Float64("win", win))
			}
		}
	}()
}

// SetAlgoTimer 动态设置Algo定时器
func (c *ACMController) SetAlgoTimer(enabled bool, interval time.Duration, bet int, prize int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 停止现有定时器
	c.stopAlgoTimer()

	// 更新配置
	c.config.AlgoTimerEnabled = enabled
	c.config.AlgoTimerInterval = interval
	c.config.AlgoBet = bet
	c.config.AlgoPrize = prize

	// 如果启用且已连接，启动新定时器
	if enabled && c.connected {
		c.startAlgoTimer()
		c.logger.Info("Algo定时器已更新",
			zap.Bool("enabled", enabled),
			zap.Duration("interval", interval),
			zap.Int("bet", bet),
			zap.Int("prize", prize))
	}
}

// StartAlgoTimer 启动Algo定时器（外部调用）
func (c *ACMController) StartAlgoTimer() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("ACM设备未连接")
	}

	if c.algoTimer != nil {
		return fmt.Errorf("Algo定时器已在运行")
	}

	c.config.AlgoTimerEnabled = true
	c.startAlgoTimer()

	return nil
}

// StopAlgoTimer 停止Algo定时器（外部调用）
func (c *ACMController) StopAlgoTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config.AlgoTimerEnabled = false
	c.stopAlgoTimer()

	c.logger.Info("Algo定时器已停止")
}
