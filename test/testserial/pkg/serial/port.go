package serial

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tarm/serial"
)

// SerialPort 串口管理结构
type SerialPort struct {
	Name       string
	Port       *serial.Port
	Config     *serial.Config
	IsOpen     bool
	mu         sync.RWMutex
	readBuffer chan []byte
	stopChan   chan bool
}

// Message JSON消息格式
type Message struct {
	Type      string          `json:"type"`
	Command   string          `json:"command,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// GameMessage 游戏消息格式（基于Java代码）
type GameMessage struct {
	MsgType   string          `json:"MsgType"`           // M1-M6
	Code      int             `json:"code,omitempty"`    // M2响应码
	Function  string          `json:"function,omitempty"` // M2功能
	State     int             `json:"state,omitempty"`   // M3状态
	Action    string          `json:"action,omitempty"`  // M4动作
	UpState   int             `json:"upstate,omitempty"` // M5更新状态
	TopType   int             `json:"toptype,omitempty"` // M6 MQTT类型
	Data      json.RawMessage `json:"data,omitempty"`    // 附加数据
	Timestamp int64           `json:"timestamp,omitempty"`
}

// NewSerialPort 创建新的串口实例
func NewSerialPort(name string, config *serial.Config) *SerialPort {
	return &SerialPort{
		Name:       name,
		Config:     config,
		IsOpen:     false,
		readBuffer: make(chan []byte, 100),
		stopChan:   make(chan bool),
	}
}

// Open 打开串口
func (sp *SerialPort) Open() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.IsOpen {
		return fmt.Errorf("串口 %s 已经打开", sp.Name)
	}

	port, err := serial.OpenPort(sp.Config)
	if err != nil {
		return fmt.Errorf("打开串口 %s 失败: %v", sp.Name, err)
	}

	sp.Port = port
	sp.IsOpen = true

	// 启动读取协程
	go sp.readLoop()

	fmt.Printf("✅ 串口 %s 已打开 (设备: %s, 波特率: %d)\n", 
		sp.Name, sp.Config.Name, sp.Config.Baud)
	return nil
}

// Close 关闭串口
func (sp *SerialPort) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.IsOpen {
		return fmt.Errorf("串口 %s 未打开", sp.Name)
	}

	sp.stopChan <- true
	err := sp.Port.Close()
	if err != nil {
		return fmt.Errorf("关闭串口 %s 失败: %v", sp.Name, err)
	}

	sp.IsOpen = false
	fmt.Printf("✅ 串口 %s 已关闭\n", sp.Name)
	return nil
}

// Write 写入数据到串口
func (sp *SerialPort) Write(data []byte) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if !sp.IsOpen {
		return fmt.Errorf("串口 %s 未打开", sp.Name)
	}

	n, err := sp.Port.Write(data)
	if err != nil {
		return fmt.Errorf("写入串口 %s 失败: %v", sp.Name, err)
	}

	fmt.Printf("📤 [%s] 发送 %d 字节: %s\n", sp.Name, n, string(data))
	return nil
}

// readLoop 读取循环
func (sp *SerialPort) readLoop() {
	buffer := make([]byte, 1024)
	for {
		select {
		case <-sp.stopChan:
			return
		default:
			if sp.Port != nil && sp.IsOpen {
				n, err := sp.Port.Read(buffer)
				if err != nil {
					if sp.IsOpen {
						fmt.Printf("❌ [%s] 读取错误: %v\n", sp.Name, err)
					}
					continue
				}
				if n > 0 {
					data := make([]byte, n)
					copy(data, buffer[:n])
					sp.readBuffer <- data
					sp.handleReceivedData(data)
				}
			}
		}
	}
}

// handleReceivedData 处理接收到的数据
func (sp *SerialPort) handleReceivedData(data []byte) {
	fmt.Printf("📥 [%s] 收到 %d 字节: %s\n", sp.Name, len(data), string(data))
	
	// 尝试解析为JSON
	var msg Message
	if err := json.Unmarshal(data, &msg); err == nil {
		fmt.Printf("   📋 消息类型: %s, 命令: %s\n", msg.Type, msg.Command)
		
		// 根据消息类型处理
		switch msg.Type {
		case "response":
			fmt.Printf("   ✅ 收到响应\n")
		case "event":
			fmt.Printf("   🔔 收到事件通知\n")
		case "error":
			fmt.Printf("   ❌ 收到错误消息\n")
		default:
			fmt.Printf("   ❓ 未知消息类型\n")
		}
	}
}

// GetReadBuffer 获取读取缓冲区通道
func (sp *SerialPort) GetReadBuffer() <-chan []byte {
	return sp.readBuffer
}