package serial

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// SerialManager 串口管理器
type SerialManager struct {
	ACMPort   *SerialPort
	STM32Port *SerialPort
}

// NewSerialManager 创建串口管理器
func NewSerialManager() *SerialManager {
	return &SerialManager{}
}

// FindACMDevice 自动查找ACM设备
func FindACMDevice() string {
	// Linux设备路径
	linuxPaths := []string{"/dev/ttyACM*", "/dev/ttyUSB*"}
	
	for _, pattern := range linuxPaths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		
		for _, device := range matches {
			if strings.Contains(device, "ACM") {
				fmt.Printf("🔍 找到ACM设备: %s\n", device)
				return device
			}
		}
	}
	
	// macOS设备路径
	macPaths := []string{"/dev/cu.usbmodem*", "/dev/tty.usbmodem*"}
	for _, pattern := range macPaths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			fmt.Printf("🔍 找到设备: %s\n", matches[0])
			return matches[0]
		}
	}
	
	return ""
}

// InitACMPort 初始化ACM设备串口
func (sm *SerialManager) InitACMPort(device string, baud int) error {
	// 如果配置为auto，自动查找ACM设备
	if device == "" || device == "auto" {
		device = FindACMDevice()
		if device == "" {
			return fmt.Errorf("未找到ACM设备")
		}
	}
	
	config := &serial.Config{
		Name:        device,
		Baud:        baud,
		ReadTimeout: time.Second * 1,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
	}

	sm.ACMPort = NewSerialPort("ACM设备", config)
	return sm.ACMPort.Open()
}

// InitSTM32Port 初始化STM32芯片串口
func (sm *SerialManager) InitSTM32Port(device string, baud int) error {
	config := &serial.Config{
		Name:        device,
		Baud:        baud,
		ReadTimeout: time.Second * 1,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
	}

	sm.STM32Port = NewSerialPort("STM32芯片", config)
	return sm.STM32Port.Open()
}

// SendToACM 发送消息到ACM设备
func (sm *SerialManager) SendToACM(msg Message) error {
	if sm.ACMPort == nil {
		return fmt.Errorf("ACM设备未初始化")
	}

	msg.Timestamp = time.Now().Unix()
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	return sm.ACMPort.Write(data)
}

// SendToSTM32 发送消息到STM32芯片
func (sm *SerialManager) SendToSTM32(msg Message) error {
	if sm.STM32Port == nil {
		return fmt.Errorf("STM32芯片未初始化")
	}

	msg.Timestamp = time.Now().Unix()
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	return sm.STM32Port.Write(data)
}

// SendGameMessage 发送游戏消息
func (sm *SerialManager) SendGameMessage(port *SerialPort, msg GameMessage) error {
	if port == nil || !port.IsOpen {
		return fmt.Errorf("串口未初始化或未打开")
	}

	msg.Timestamp = time.Now().Unix()
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// Java代码中会在消息后添加\r\n
	data = append(data, []byte("\r\n")...)
	return port.Write(data)
}

// SendRawCommand 发送原始命令（如ver, sta等）
func (sm *SerialManager) SendRawCommand(port *SerialPort, command string) error {
	if port == nil || !port.IsOpen {
		return fmt.Errorf("串口未初始化或未打开")
	}
	
	// Java代码中的命令格式
	data := []byte(command)
	if !strings.HasSuffix(command, "\r\n") {
		data = append(data, []byte("\r\n")...)
	}
	
	return port.Write(data)
}

// CloseAll 关闭所有串口
func (sm *SerialManager) CloseAll() {
	if sm.ACMPort != nil && sm.ACMPort.IsOpen {
		sm.ACMPort.Close()
	}
	if sm.STM32Port != nil && sm.STM32Port.IsOpen {
		sm.STM32Port.Close()
	}
}

// Status 显示状态
func (sm *SerialManager) Status() {
	fmt.Println("\n=== 串口状态 ===")
	
	if sm.ACMPort != nil {
		status := "❌ 关闭"
		if sm.ACMPort.IsOpen {
			status = "✅ 打开"
		}
		fmt.Printf("ACM设备: %s (设备: %s)\n", status, sm.ACMPort.Config.Name)
	} else {
		fmt.Println("ACM设备: ❌ 未初始化")
	}
	
	if sm.STM32Port != nil {
		status := "❌ 关闭"
		if sm.STM32Port.IsOpen {
			status = "✅ 打开"
		}
		fmt.Printf("STM32芯片: %s (设备: %s)\n", status, sm.STM32Port.Config.Name)
	} else {
		fmt.Println("STM32芯片: ❌ 未初始化")
	}
	fmt.Println()
}