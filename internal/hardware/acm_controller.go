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
	"go.uber.org/zap"
)

// ACMConfig ACM设备配置
type ACMConfig struct {
	Port         string        // 串口端口（"auto"表示自动检测）
	BaudRate     int           // 波特率
	ReadTimeout  time.Duration // 读超时
	WriteTimeout time.Duration // 写超时
	AutoDetect   bool          // 是否自动检测设备

	// Algo命令定时器配置
	AlgoTimerEnabled  bool          // 是否启用algo定时器
	AlgoTimerInterval time.Duration // algo命令发送间隔
	AlgoBet           int           // algo命令的bet参数
	AlgoPrize         int           // algo命令的prize参数
}

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
		AlgoTimerInterval: 5 * time.Second,
		AlgoBet:           1,
		AlgoPrize:         100,
	}
}

// ACMController ACM设备控制器
type ACMController struct {
	config    *ACMConfig
	port      *serial.Port
	mu        sync.RWMutex
	connected bool
	logger    *zap.Logger

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

// Connect 连接ACM设备
func (c *ACMController) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// 自动检测设备
	if c.config.AutoDetect || c.config.Port == "auto" {
		device := c.findACMDevice()
		if device == "" {
			return fmt.Errorf("未找到ACM设备")
		}
		c.config.Port = device
		c.logger.Info("自动检测到ACM设备", zap.String("device", device))
	}

	// 配置串口
	cfg := &serial.Config{
		Name:        c.config.Port,
		Baud:        c.config.BaudRate,
		Size:        8,
		StopBits:    serial.Stop2,
		Parity:      serial.ParityNone,
		ReadTimeout: c.config.ReadTimeout,
	}

	// 打开串口
	port, err := serial.OpenPort(cfg)
	if err != nil {
		c.logger.Error("打开ACM串口失败",
			zap.String("port", c.config.Port),
			zap.Error(err))
		return fmt.Errorf("打开ACM串口失败: %w", err)
	}

	c.port = port
	c.connected = true
	c.stopCh = make(chan struct{})
	c.cmdCh = make(chan string, 10)
	c.respCh = make(chan interface{}, 10)

	// 启动后台任务
	go c.readLoop()
	go c.processLoop()

	c.logger.Info("ACM控制器已连接",
		zap.String("port", c.config.Port),
		zap.Int("baudrate", c.config.BaudRate))

	// 等待设备就绪并发送help命令
	time.Sleep(200 * time.Millisecond)
	c.logger.Info("发送help命令查询支持的命令列表")
	if err := c.SendCommand("help"); err != nil {
		c.logger.Warn("发送help命令失败", zap.Error(err))
	}

	// 启动Algo定时器（如果启用）
	if c.config.AlgoTimerEnabled {
		// 延迟启动，等help命令响应完成
		time.Sleep(500 * time.Millisecond)
		c.startAlgoTimer()
		c.logger.Info("Algo定时器已启动",
			zap.Duration("interval", c.config.AlgoTimerInterval),
			zap.Int("bet", c.config.AlgoBet),
			zap.Int("prize", c.config.AlgoPrize))
	}

	return nil
}

// Disconnect 断开连接
func (c *ACMController) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.logger.Info("正在断开ACM连接...")

	// 停止Algo定时器
	c.stopAlgoTimer()

	// 标记为未连接
	c.connected = false

	// 关闭停止通道，通知所有goroutine退出
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}

	// 关闭命令通道
	if c.cmdCh != nil {
		close(c.cmdCh)
		c.cmdCh = nil
	}

	// 短暂等待goroutine退出
	time.Sleep(100 * time.Millisecond)

	// 关闭串口
	if c.port != nil {
		err := c.port.Close()
		if err != nil {
			c.logger.Warn("关闭ACM串口时出错", zap.Error(err))
		}
		c.port = nil
	}

	c.logger.Info("ACM控制器已断开")

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

			// 设置短超时避免阻塞
			c.port.Flush()
			n, err := c.port.Read(buffer)
			if err != nil {
				// EOF表示连接断开
				if err.Error() == "EOF" || strings.Contains(err.Error(), "EOF") {
					c.logger.Error("ACM连接断开(EOF)，退出读取循环")
					c.mu.Lock()
					c.connected = false
					c.mu.Unlock()
					return
				}
				// 忽略超时错误
				if !strings.Contains(err.Error(), "timeout") {
					c.logger.Debug("读取ACM数据错误", zap.Error(err))
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
						// 记录所有消息，帮助了解ACM设备的响应格式
						if strings.Contains(msg, "help") || strings.Contains(msg, "Help") ||
						   strings.Contains(msg, "command") || strings.Contains(msg, "Command") {
							c.logger.Info("ACM帮助信息", zap.String("message", msg))
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
			c.handleJSONMessage(jsonMsg)
			return
		}
	}

	// 处理简单命令
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
			c.handleCommand(cmd)
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
	cmdBytes := []byte(cmd + "\n")

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
	if c.algoTimer != nil {
		return // 定时器已经在运行
	}

	c.algoTimerStopCh = make(chan struct{})
	c.algoTimer = time.NewTicker(c.config.AlgoTimerInterval)

	go func() {
		c.logger.Info("Algo定时器线程已启动")

		// 立即发送第一个命令
		c.sendAlgoCommandAsync()

		for {
			select {
			case <-c.algoTimer.C:
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
	if c.algoTimer != nil {
		c.algoTimer.Stop()
		c.algoTimer = nil
	}

	if c.algoTimerStopCh != nil {
		close(c.algoTimerStopCh)
		c.algoTimerStopCh = nil
	}
}

// sendAlgoCommandAsync 异步发送algo命令
func (c *ACMController) sendAlgoCommandAsync() {
	go func() {
		cmd := fmt.Sprintf("algo -b %d -p %d", c.config.AlgoBet, c.config.AlgoPrize)

		c.logger.Debug("定时发送algo命令",
			zap.String("command", cmd),
			zap.Time("timestamp", time.Now()))

		// 发送命令并获取响应
		response, err := c.SendAlgoCommand(c.config.AlgoBet, c.config.AlgoPrize)
		if err != nil {
			c.logger.Error("定时algo命令执行失败",
				zap.Error(err),
				zap.String("command", cmd))
			return
		}

		// 记录响应
		c.logger.Info("定时algo命令响应",
			zap.Any("response", response),
			zap.Float64("win", response["win"].(float64)),
			zap.Int("hp30", response["hp30"].(int)))
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
