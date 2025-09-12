package hardware

import (
	"encoding/binary"
	"fmt"
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
	timestamp := FormatTimestamp(time.Now())
	
	err := c.sendCommandWithTimeout(CmdHeartbeat, timestamp, 5*time.Second)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	
	c.logger.Debug("Heartbeat sent")
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
	
	c.logger.Info("Sensor event",
		zap.Uint8("type", sensorType),
		zap.Uint16("value", value))
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
	// TODO: 实现数据持久化到数据库或文件
	c.logger.Debug("Statistics saved")
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