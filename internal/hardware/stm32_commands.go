package hardware

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

// ============= 硬件控制指令（0x01-0x05） =============

// DispenseCoins 上币控制
func (c *STM32Controller) DispenseCoins(count uint16, speed byte) error {
	if count == 0 || count > 9999 {
		return fmt.Errorf("invalid coin count: %d", count)
	}
	
	if speed == 0 || speed > 10 {
		speed = 5 // 默认速度
	}
	
	data := make([]byte, 3)
	binary.BigEndian.PutUint16(data[0:2], count)
	data[2] = speed
	
	err := c.sendCommand(CmdCoinDispense, data)
	if err != nil {
		c.logger.Error("Dispense coins failed",
			zap.Uint16("count", count),
			zap.Uint8("speed", speed),
			zap.Error(err))
		return err
	}
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.CoinsDispensed += count
	c.statsMu.Unlock()
	
	c.logger.Info("Coins dispensed",
		zap.Uint16("count", count),
		zap.Uint8("speed", speed))
	
	return nil
}

// RefundCoins 退币控制
func (c *STM32Controller) RefundCoins(count uint16) error {
	if count == 0 || count > 9999 {
		return fmt.Errorf("invalid refund count: %d", count)
	}
	
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, count)
	
	err := c.sendCommand(CmdCoinRefund, data)
	if err != nil {
		c.logger.Error("Refund coins failed",
			zap.Uint16("count", count),
			zap.Error(err))
		return err
	}
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.CoinsRefunded += count
	c.statsMu.Unlock()
	
	c.logger.Info("Coins refunded", zap.Uint16("count", count))
	
	return nil
}

// DispenseTickets 彩票发放
func (c *STM32Controller) DispenseTickets(count uint16) error {
	if count == 0 || count > 9999 {
		return fmt.Errorf("invalid ticket count: %d", count)
	}
	
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, count)
	
	err := c.sendCommand(CmdTicketPrint, data)
	if err != nil {
		c.logger.Error("Dispense tickets failed",
			zap.Uint16("count", count),
			zap.Error(err))
		return err
	}
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.TicketsPrinted += count
	c.statsMu.Unlock()
	
	c.logger.Info("Tickets dispensed", zap.Uint16("count", count))
	
	return nil
}

// PushControl 推币控制
func (c *STM32Controller) PushControl(action byte, param byte) error {
	data := []byte{action}
	
	// 根据动作类型添加参数
	switch action {
	case PushActionContinuous, PushActionStop:
		// 无参数
	case PushActionSingle:
		data = append(data, param) // 推币次数
	case PushActionSetSpeed:
		if param == 0 || param > 10 {
			param = 5 // 默认速度
		}
		data = append(data, param) // 速度值
	default:
		return fmt.Errorf("invalid push action: 0x%02X", action)
	}
	
	err := c.sendCommand(CmdPushControl, data)
	if err != nil {
		c.logger.Error("Push control failed",
			zap.Uint8("action", action),
			zap.Uint8("param", param),
			zap.Error(err))
		return err
	}
	
	c.logger.Info("Push control executed",
		zap.Uint8("action", action),
		zap.Uint8("param", param))
	
	return nil
}

// StartPushing 启动连续推币
func (c *STM32Controller) StartPushing() error {
	return c.PushControl(PushActionContinuous, 0)
}

// StopPushing 停止推币
func (c *STM32Controller) StopPushing() error {
	return c.PushControl(PushActionStop, 0)
}

// SinglePush 单次推币
func (c *STM32Controller) SinglePush(times byte) error {
	return c.PushControl(PushActionSingle, times)
}

// SetPushSpeed 设置推币速度
func (c *STM32Controller) SetPushSpeed(speed byte) error {
	return c.PushControl(PushActionSetSpeed, speed)
}

// SetLights 灯光控制
func (c *STM32Controller) SetLights(lightBits byte) error {
	data := []byte{lightBits}
	
	err := c.sendCommand(CmdLightControl, data)
	if err != nil {
		c.logger.Error("Set lights failed",
			zap.Uint8("bits", lightBits),
			zap.Error(err))
		return err
	}
	
	c.logger.Debug("Lights set", zap.Uint8("bits", lightBits))
	
	return nil
}

// TurnOnLight 打开指定灯
func (c *STM32Controller) TurnOnLight(light byte) error {
	return c.SetLights(light)
}

// TurnOffAllLights 关闭所有灯
func (c *STM32Controller) TurnOffAllLights() error {
	return c.SetLights(LightNone)
}

// TurnOnAllLights 打开所有灯
func (c *STM32Controller) TurnOnAllLights() error {
	return c.SetLights(LightAll)
}

// ============= 状态管理（0x21-0x25） =============

// QueryStatus 查询设备状态
func (c *STM32Controller) QueryStatus(queryType byte) error {
	data := []byte{queryType}
	
	err := c.sendCommand(CmdStatusQuery, data)
	if err != nil {
		c.logger.Error("Query status failed",
			zap.Uint8("type", queryType),
			zap.Error(err))
		return err
	}
	
	return nil
}

// RecoverFault 故障恢复
func (c *STM32Controller) RecoverFault(faultCode byte, action byte, param byte) error {
	data := []byte{faultCode, action, param}
	
	err := c.sendCommand(CmdFaultRecovery, data)
	if err != nil {
		c.logger.Error("Fault recovery failed",
			zap.Uint8("fault", faultCode),
			zap.Uint8("action", action),
			zap.Error(err))
		return err
	}
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.RecoveryCount++
	c.statsMu.Unlock()
	
	c.logger.Info("Fault recovery executed",
		zap.Uint8("fault", faultCode),
		zap.Uint8("action", action))
	
	return nil
}

// ============= 系统指令（0x31） =============

// SendHeartbeat 发送心跳
func (c *STM32Controller) SendHeartbeat() error {
	// 构建心跳数据（包含版本号）
	data := make([]byte, 5)
	copy(data[0:4], FormatTimestamp(time.Now()))
	data[4] = 0x01 // 协议版本号 v1.1
	
	err := c.sendCommandWithTimeout(CmdHeartbeat, data, 5*time.Second)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	
	c.logger.Debug("Heartbeat sent", zap.Uint8("version", data[4]))
	return nil
}

// ============= 事件处理函数 =============

// handleCoinInserted 处理投币事件
func (c *STM32Controller) handleCoinInserted(frame *Frame) {
	if len(frame.Data) < 1 {
		c.logger.Error("Invalid coin inserted data")
		return
	}
	
	count := frame.Data[0]
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.CoinsInserted += uint16(count)
	c.statsMu.Unlock()
	
	// 更新游戏逻辑
	if c.gameLogic != nil {
		c.gameLogic.AddCredits(count)
	}
	
	// 调用回调
	if c.onCoinInserted != nil {
		c.onCoinInserted(count)
	}
	
	c.logger.Info("Coin inserted", zap.Uint8("count", count))
}

// handleCoinReturned 处理回币事件（优化后的格式）
func (c *STM32Controller) handleCoinReturned(frame *Frame) {
	if len(frame.Data) < 3 {
		c.logger.Error("Invalid coin return data")
		return
	}
	
	// 新数据格式：[前方数量][左侧数量][右侧数量]
	data := &CoinReturnData{
		FrontCount: frame.Data[0],
		LeftCount:  frame.Data[1],
		RightCount: frame.Data[2],
	}
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.CoinsReturnedFront += uint16(data.FrontCount)
	c.stats.CoinsReturnedLeft += uint16(data.LeftCount)
	c.stats.CoinsReturnedRight += uint16(data.RightCount)
	
	// 计算回币率
	totalReturned := c.stats.CoinsReturnedFront + 
					c.stats.CoinsReturnedLeft + 
					c.stats.CoinsReturnedRight
	if totalReturned > 0 {
		c.stats.ReturnRate = float64(c.stats.CoinsReturnedFront) / float64(totalReturned) * 100
	}
	c.statsMu.Unlock()
	
	// 更新游戏逻辑
	if c.gameLogic != nil {
		if data.FrontCount > 0 {
			c.gameLogic.AddPlayerCoins(data.FrontCount)
		}
		c.gameLogic.UpdateReturnRate(c.stats.ReturnRate)
	}
	
	// 调用回调
	if c.onCoinReturned != nil {
		c.onCoinReturned(data)
	}
	
	// 立即持久化数据（防断电丢失）
	c.saveStatistics()
	
	c.logger.Info("Coins returned",
		zap.Uint8("front", data.FrontCount),
		zap.Uint8("left", data.LeftCount),
		zap.Uint8("right", data.RightCount),
		zap.Float64("returnRate", c.stats.ReturnRate))
}

// handleButtonPressed 处理按键事件
func (c *STM32Controller) handleButtonPressed(frame *Frame) {
	if len(frame.Data) < 3 {
		c.logger.Error("Invalid button event data")
		return
	}
	
	event := &ButtonEvent{
		KeyType: frame.Data[0],
		KeyCode: frame.Data[1],
		Action:  frame.Data[2],
	}
	
	// 如果有附加数据
	if len(frame.Data) > 3 {
		event.ExtraData = frame.Data[3:]
	}
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	// 处理游戏按键
	if event.KeyType == KeyTypeGame && event.Action == KeyActionDown {
		c.handleGameButton(event.KeyCode)
	}
	
	// 调用回调
	if c.onButtonPressed != nil {
		c.onButtonPressed(event)
	}
	
	c.logger.Info("Button pressed",
		zap.Uint8("type", event.KeyType),
		zap.Uint8("code", event.KeyCode),
		zap.Uint8("action", event.Action))
}

// handleGameButton 处理游戏按键
func (c *STM32Controller) handleGameButton(keyCode byte) {
	if c.gameLogic == nil {
		return
	}
	
	switch keyCode {
	case KeyStart:
		// 开始按键
		if c.gameLogic.HasCredits() {
			coinCount := c.gameLogic.GetPendingCoins()
			
			// 通过上币接口落币，开始游戏
			c.DispenseCoins(coinCount, 5)
			
			// 开始游戏逻辑
			c.gameLogic.StartGame(coinCount)
			
			// 启动推币
			c.StartPushing()
		}
		
	case KeyRefundTicket:
		// 退币/彩票按键（根据内部模式）
		mode := c.gameLogic.GetCurrentMode()
		if mode == 0x01 { // 退币模式
			if coins := c.gameLogic.GetRefundableCoins(); coins > 0 {
				c.RefundCoins(coins)
				c.gameLogic.DeductCoins(coins)
			}
		} else if mode == 0x02 { // 彩票模式
			if tickets := c.gameLogic.GetAvailableTickets(); tickets > 0 {
				c.DispenseTickets(tickets)
				c.gameLogic.RedeemTickets(tickets)
			}
		}
	}
}

// handleSensorEvent 处理传感器事件
func (c *STM32Controller) handleSensorEvent(frame *Frame) {
	if len(frame.Data) < 3 {
		c.logger.Error("Invalid sensor event data")
		return
	}
	
	sensorType := frame.Data[0]
	value := binary.BigEndian.Uint16(frame.Data[1:3])
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	// 根据传感器类型处理不同事件
	switch sensorType {
	case SensorCoinLevel:
		c.handleCoinLevelSensor(value)
	case SensorTicketLevel:
		c.handleTicketLevelSensor(value)
	case SensorTemperature:
		c.handleTemperatureSensor(value)
	case SensorVibration:
		c.handleVibrationSensor(value)
	case SensorDoor:
		c.handleDoorSensor(value)
	default:
		c.logger.Warn("Unknown sensor type",
			zap.Uint8("type", sensorType),
			zap.Uint16("value", value))
	}
}

// handleCoinLevelSensor 处理币仓余量传感器
func (c *STM32Controller) handleCoinLevelSensor(value uint16) {
	c.logger.Info("Coin level sensor triggered",
		zap.Uint16("level", value))
	
	// 币仓余量报警阈值
	const lowLevelThreshold = 10
	const criticalLevelThreshold = 5
	
	if value <= criticalLevelThreshold {
		c.logger.Error("CRITICAL: Coin hopper almost empty",
			zap.Uint16("remaining", value))
		// 触发紧急补币流程
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultNoResource,
				Level:     FaultLevelCritical,
				ExtraInfo: []byte{SensorCoinLevel, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	} else if value <= lowLevelThreshold {
		c.logger.Warn("Low coin level detected",
			zap.Uint16("remaining", value))
		// 触发预警
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultNoResource,
				Level:     FaultLevelWarning,
				ExtraInfo: []byte{SensorCoinLevel, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	}
}

// handleTicketLevelSensor 处理彩票余量传感器
func (c *STM32Controller) handleTicketLevelSensor(value uint16) {
	c.logger.Info("Ticket level sensor triggered",
		zap.Uint16("level", value))
	
	// 彩票余量报警阈值
	const lowLevelThreshold = 50
	const criticalLevelThreshold = 10
	
	if value <= criticalLevelThreshold {
		c.logger.Error("CRITICAL: Ticket paper almost empty",
			zap.Uint16("remaining", value))
		// 触发紧急换纸流程
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultTicketNoPaper,
				Level:     FaultLevelCritical,
				ExtraInfo: []byte{SensorTicketLevel, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	} else if value <= lowLevelThreshold {
		c.logger.Warn("Low ticket paper level detected",
			zap.Uint16("remaining", value))
		// 触发预警
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultTicketNoPaper,
				Level:     FaultLevelWarning,
				ExtraInfo: []byte{SensorTicketLevel, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	}
}

// handleTemperatureSensor 处理温度传感器
func (c *STM32Controller) handleTemperatureSensor(value uint16) {
	// 温度值为实际温度*10（如250表示25.0°C）
	temperature := float64(value) / 10.0
	c.logger.Info("Temperature sensor reading",
		zap.Float64("temperature", temperature))
	
	const warningTemp = 45.0  // 45°C警告
	const criticalTemp = 55.0 // 55°C严重
	
	if temperature >= criticalTemp {
		c.logger.Error("CRITICAL: Over temperature detected",
			zap.Float64("temperature", temperature))
		// 触发过热保护
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultOverTemperature,
				Level:     FaultLevelCritical,
				ExtraInfo: []byte{SensorTemperature, byte(value >> 8), byte(value & 0xFF)},
			})
		}
		// 自动停止推币电机以降温
		c.StopPushing()
	} else if temperature >= warningTemp {
		c.logger.Warn("High temperature warning",
			zap.Float64("temperature", temperature))
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultOverTemperature,
				Level:     FaultLevelWarning,
				ExtraInfo: []byte{SensorTemperature, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	}
}

// handleVibrationSensor 处理震动传感器
func (c *STM32Controller) handleVibrationSensor(value uint16) {
	c.logger.Info("Vibration sensor triggered",
		zap.Uint16("intensity", value))
	
	const abnormalVibrationThreshold = 500 // 异常震动阈值
	
	if value >= abnormalVibrationThreshold {
		c.logger.Warn("Abnormal vibration detected",
			zap.Uint16("intensity", value))
		// 可能有人在撞击机器或机器故障
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultSensorAbnormal,
				Level:     FaultLevelWarning,
				ExtraInfo: []byte{SensorVibration, byte(value >> 8), byte(value & 0xFF)},
			})
		}
	}
}

// handleDoorSensor 处理门开关传感器
func (c *STM32Controller) handleDoorSensor(value uint16) {
	isOpen := value > 0
	c.logger.Info("Door sensor status",
		zap.Bool("is_open", isOpen))
	
	if isOpen {
		c.logger.Warn("Machine door opened")
		// 门被打开，可能是维护或非法访问
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultSensorAbnormal,
				Level:     FaultLevelWarning,
				ExtraInfo: []byte{SensorDoor, 0x01, 0x00}, // 0x01表示门开
			})
		}
		// 暂停游戏
		if c.gameLogic != nil && c.gameLogic.HasCredits() {
			c.logger.Info("Pausing game due to door open")
		}
	} else {
		c.logger.Info("Machine door closed")
		// 门已关闭，恢复正常
		if c.onFaultReport != nil {
			c.onFaultReport(&FaultEvent{
				FaultCode: FaultSensorAbnormal,
				Level:     FaultLevelInfo,
				ExtraInfo: []byte{SensorDoor, 0x00, 0x00}, // 0x00表示门关
			})
		}
	}
}

// handleStatusReport 处理状态上报
func (c *STM32Controller) handleStatusReport(frame *Frame) {
	if len(frame.Data) < 10 {
		c.logger.Error("Invalid status report data")
		return
	}
	
	status := &DeviceStatus{
		CoinMotor:     frame.Data[0],
		ReturnMotor:   frame.Data[1],
		PushMotor:     frame.Data[2],
		TicketPrinter: frame.Data[3],
		CoinCount:     binary.BigEndian.Uint16(frame.Data[4:6]),
		TicketCount:   binary.BigEndian.Uint16(frame.Data[6:8]),
		Temperature:   frame.Data[8],
		ErrorFlags:    frame.Data[9],
	}
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	c.logger.Info("Status report received",
		zap.Any("status", status))
}

// handleFaultReport 处理故障上报
func (c *STM32Controller) handleFaultReport(frame *Frame) {
	if len(frame.Data) < 2 {
		c.logger.Error("Invalid fault report data")
		return
	}
	
	event := &FaultEvent{
		FaultCode: frame.Data[0],
		Level:     frame.Data[1],
	}
	
	if len(frame.Data) > 2 {
		event.ExtraInfo = frame.Data[2:]
	}
	
	// 发送ACK确认
	c.sendACKResponse(frame.Sequence, frame.Command, StatusSuccess)
	
	// 更新统计
	c.statsMu.Lock()
	c.stats.FaultCount++
	c.statsMu.Unlock()
	
	// 调用回调
	if c.onFaultReport != nil {
		c.onFaultReport(event)
	}
	
	// 根据严重级别处理
	switch event.Level {
	case FaultLevelCritical:
		c.logger.Error("Critical fault detected",
			zap.Uint8("code", event.FaultCode))
		// 可能需要停止游戏
	case FaultLevelError:
		c.logger.Error("Fault detected",
			zap.Uint8("code", event.FaultCode))
	case FaultLevelWarning:
		c.logger.Warn("Warning detected",
			zap.Uint8("code", event.FaultCode))
	default:
		c.logger.Info("Info fault",
			zap.Uint8("code", event.FaultCode))
	}
}

// handleProgress 处理执行进度
func (c *STM32Controller) handleProgress(frame *Frame) {
	if len(frame.Data) < 6 {
		c.logger.Error("Invalid progress data")
		return
	}
	
	report := &ProgressReport{
		OriginalCmd: frame.Data[0],
		Completed:   binary.BigEndian.Uint16(frame.Data[1:3]),
		Total:       binary.BigEndian.Uint16(frame.Data[3:5]),
		Status:      frame.Data[5],
	}
	
	c.logger.Info("Progress update",
		zap.Uint8("cmd", report.OriginalCmd),
		zap.Uint16("completed", report.Completed),
		zap.Uint16("total", report.Total),
		zap.Uint8("status", report.Status))
}

// handleHeartbeat 处理心跳响应
func (c *STM32Controller) handleHeartbeat(frame *Frame) {
	if len(frame.Data) < 8 {
		c.logger.Error("Invalid heartbeat response")
		return
	}
	
	// timestamp := ParseTimestamp(frame.Data[0:4])
	// uptime := binary.BigEndian.Uint32(frame.Data[4:8])
	
	// 检查是否有版本信息（协议v1.1）
	if len(frame.Data) >= 9 {
		stm32Version := frame.Data[8]
		c.logger.Debug("Heartbeat received", 
			zap.Uint8("stm32_version", stm32Version),
			zap.Uint8("frame_version", frame.Version))
		
		// 版本协商：如果版本不匹配，可以记录或处理
		if frame.Version != 0x01 && stm32Version != 0x01 {
			c.logger.Warn("Protocol version mismatch",
				zap.Uint8("expected", 0x01),
				zap.Uint8("received", stm32Version))
		}
	}
	
	c.logger.Debug("Heartbeat response received")
}

// sendACKResponse 发送ACK响应
func (c *STM32Controller) sendACKResponse(origSeq uint16, origCmd byte, status byte) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], origSeq)
	data[2] = origCmd
	data[3] = status
	
	// 使用偶数序列号（STM32上报）
	seq := origSeq + 1
	if seq%2 == 1 {
		seq++
	}
	
	frame := NewFrame(CmdACK, seq, data)
	if err := c.writeFrame(frame); err != nil {
		c.logger.Error("Send ACK failed", zap.Error(err))
	}
}

// saveStatistics 保存统计数据（用于断电保护）
func (c *STM32Controller) saveStatistics() {
	c.statsMu.RLock()
	stats := c.stats
	stats.Timestamp = time.Now()
	c.statsMu.RUnlock()
	
	// 计算回币率
	if stats.CoinsInserted > 0 {
		totalReturned := stats.CoinsReturnedFront + stats.CoinsReturnedLeft + stats.CoinsReturnedRight
		stats.ReturnRate = float64(totalReturned) / float64(stats.CoinsInserted)
	}
	
	// 使用事务支持的批量保存
	if err := c.saveStatisticsWithTransaction(&stats); err != nil {
		c.logger.Error("Failed to save statistics with transaction", zap.Error(err))
		// 降级到普通保存
		filename := fmt.Sprintf("data/statistics_%s.json", time.Now().Format("20060102"))
		if err := c.saveStatsToFile(filename, &stats); err != nil {
			c.logger.Error("Failed to save statistics to file", zap.Error(err))
		}
	}
	
	c.logger.Info("Statistics saved",
		zap.Uint16("coins_inserted", stats.CoinsInserted),
		zap.Uint16("coins_dispensed", stats.CoinsDispensed),
		zap.Float64("return_rate", stats.ReturnRate))
}

// saveStatisticsWithTransaction 使用事务保存统计数据
func (c *STM32Controller) saveStatisticsWithTransaction(stats *CoinStatistics) error {
	// 确保目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("create data directory failed: %w", err)
	}
	
	// 生成文件名
	filename := fmt.Sprintf("data/statistics_%s.json", time.Now().Format("20060102"))
	tempFile := filename + ".tmp"
	backupFile := filename + ".bak"
	
	// 1. 写入临时文件
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal statistics failed: %w", err)
	}
	
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("write temp file failed: %w", err)
	}
	
	// 2. 如果原文件存在，创建备份
	if _, err := os.Stat(filename); err == nil {
		if err := os.Rename(filename, backupFile); err != nil {
			os.Remove(tempFile) // 清理临时文件
			return fmt.Errorf("backup original file failed: %w", err)
		}
	}
	
	// 3. 将临时文件重命名为目标文件
	if err := os.Rename(tempFile, filename); err != nil {
		// 尝试恢复备份
		if _, berr := os.Stat(backupFile); berr == nil {
			os.Rename(backupFile, filename)
		}
		return fmt.Errorf("rename temp file failed: %w", err)
	}
	
	// 4. 删除备份文件（可选）
	os.Remove(backupFile)
	
	return nil
}

// saveStatsToFile 保存统计数据到文件
func (c *STM32Controller) saveStatsToFile(filename string, stats *CoinStatistics) error {
	// 确保目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("create data directory failed: %w", err)
	}
	
	// 将统计数据转换为JSON
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal statistics failed: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write statistics file failed: %w", err)
	}
	
	return nil
}

// loadStatistics 加载统计数据
func (c *STM32Controller) loadStatistics() error {
	// 查找今天的统计文件
	filename := fmt.Sprintf("data/statistics_%s.json", time.Now().Format("20060102"))
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			c.logger.Info("No existing statistics file, starting fresh")
			return nil
		}
		return fmt.Errorf("read statistics file failed: %w", err)
	}
	
	var stats CoinStatistics
	if err := json.Unmarshal(data, &stats); err != nil {
		return fmt.Errorf("unmarshal statistics failed: %w", err)
	}
	
	c.statsMu.Lock()
	c.stats = stats
	c.statsMu.Unlock()
	
	c.logger.Info("Statistics loaded",
		zap.String("file", filename),
		zap.Uint16("coins_inserted", stats.CoinsInserted),
		zap.Uint16("coins_dispensed", stats.CoinsDispensed))
	
	return nil
}

// GetStatistics 获取统计数据
func (c *STM32Controller) GetStatistics() *CoinStatistics {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	
	stats := c.stats
	return &stats
}

// ResetStatistics 重置统计数据
func (c *STM32Controller) ResetStatistics() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	
	c.stats = CoinStatistics{
		Timestamp: time.Now(),
	}
	
	c.logger.Info("Statistics reset")
}

// SetCoinInsertedCallback 设置投币回调
func (c *STM32Controller) SetCoinInsertedCallback(callback func(count byte)) {
	c.onCoinInserted = callback
}

// SetCoinReturnedCallback 设置回币回调
func (c *STM32Controller) SetCoinReturnedCallback(callback func(data *CoinReturnData)) {
	c.onCoinReturned = callback
}

// SetButtonPressedCallback 设置按键回调
func (c *STM32Controller) SetButtonPressedCallback(callback func(event *ButtonEvent)) {
	c.onButtonPressed = callback
}

// SetFaultReportCallback 设置故障回调
func (c *STM32Controller) SetFaultReportCallback(callback func(event *FaultEvent)) {
	c.onFaultReport = callback
}