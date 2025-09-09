package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"testserial/pkg/serial"
)

// Config 配置结构
type Config struct {
	ACMPort struct {
		Device        string `json:"device"`
		DevicePattern string `json:"device_pattern"`
		BaudRate      int    `json:"baud_rate"`
		DataBits      int    `json:"data_bits"`
		StopBits      int    `json:"stop_bits"`
		Parity        string `json:"parity"`
		TimeoutMs     int    `json:"timeout_ms"`
		Description   string `json:"description"`
	} `json:"acm_port"`
	STM32Port struct {
		Device      string `json:"device"`
		BaudRate    int    `json:"baud_rate"`
		DataBits    int    `json:"data_bits"`
		StopBits    int    `json:"stop_bits"`
		Parity      string `json:"parity"`
		TimeoutMs   int    `json:"timeout_ms"`
		Description string `json:"description"`
	} `json:"stm32_port"`
	MessageTypes      map[string]string `json:"message_types"`
	TestMode          bool              `json:"test_mode"`
	LogLevel          string            `json:"log_level"`
	AutoReconnect     bool              `json:"auto_reconnect"`
	ReconnectInterval int               `json:"reconnect_interval_ms"`
}

// EnhancedSerialManager 增强型串口管理器
type EnhancedSerialManager struct {
	*serial.SerialManager
	config *Config
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	data, err := ioutil.ReadFile("../config.json")
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// NewEnhancedSerialManager 创建增强型串口管理器
func NewEnhancedSerialManager(config *Config) *EnhancedSerialManager {
	baseMgr := serial.NewSerialManager()
	return &EnhancedSerialManager{
		SerialManager: baseMgr,
		config: config,
	}
}

// ShowStatus 显示增强状态信息
func (esm *EnhancedSerialManager) ShowStatus() {
	esm.Status()
	fmt.Println("\n=== 消息类型说明 ===")
	for msgType, desc := range esm.config.MessageTypes {
		fmt.Printf("%s: %s\n", msgType, desc)
	}
	fmt.Println()
}

// 交互式命令行界面
func runInteractive(manager *EnhancedSerialManager) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\n=== 串口测试程序 (基于Java代码配置) ===")
	fmt.Println("输入 'help' 查看可用命令")
	
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		command := parts[0]
		
		switch command {
		case "help":
			printHelp()
			
		case "auto":
			// 自动连接配置中的串口
			fmt.Println("🔌 自动连接串口...")
			
			// 连接ACM设备
			if manager.config.ACMPort.Device == "auto" {
				if err := manager.InitACMPort("auto", manager.config.ACMPort.BaudRate); err != nil {
					fmt.Printf("❌ 连接ACM设备失败: %v\n", err)
				}
			} else {
				if err := manager.InitACMPort(manager.config.ACMPort.Device, manager.config.ACMPort.BaudRate); err != nil {
					fmt.Printf("❌ 连接ACM设备失败: %v\n", err)
				}
			}
			
			// 连接STM32设备
			if err := manager.InitSTM32Port(manager.config.STM32Port.Device, manager.config.STM32Port.BaudRate); err != nil {
				fmt.Printf("❌ 连接STM32设备失败: %v\n", err)
			}
			
		case "connect":
			if len(parts) < 4 {
				fmt.Println("用法: connect <acm|stm32> <设备路径> <波特率>")
				fmt.Println("示例: connect acm auto 115200")
				continue
			}
			
			portType := parts[1]
			device := parts[2]
			baud := 115200
			fmt.Sscanf(parts[3], "%d", &baud)
			
			var err error
			switch portType {
			case "acm":
				err = manager.InitACMPort(device, baud)
			case "stm32":
				err = manager.InitSTM32Port(device, baud)
			default:
				fmt.Printf("未知的端口类型: %s\n", portType)
				continue
			}
			
			if err != nil {
				fmt.Printf("连接失败: %v\n", err)
			}
			
		case "disconnect":
			if len(parts) < 2 {
				fmt.Println("用法: disconnect <acm|stm32|all>")
				continue
			}
			
			switch parts[1] {
			case "acm":
				if manager.ACMPort != nil {
					manager.ACMPort.Close()
				}
			case "stm32":
				if manager.STM32Port != nil {
					manager.STM32Port.Close()
				}
			case "all":
				manager.CloseAll()
			default:
				fmt.Printf("未知的端口类型: %s\n", parts[1])
			}
			
		case "send":
			if len(parts) < 3 {
				fmt.Println("用法: send <acm|stm32> <消息>")
				fmt.Println("示例: send acm {\"MsgType\":\"M4\",\"action\":\"wait\"}")
				continue
			}
			
			portType := parts[1]
			messageStr := strings.Join(parts[2:], " ")
			
			var msg serial.GameMessage
			if err := json.Unmarshal([]byte(messageStr), &msg); err != nil {
				fmt.Printf("JSON解析失败: %v\n", err)
				continue
			}
			
			var err error
			switch portType {
			case "acm":
				err = manager.SendGameMessage(manager.ACMPort, msg)
			case "stm32":
				err = manager.SendGameMessage(manager.STM32Port, msg)
			default:
				fmt.Printf("未知的端口类型: %s\n", portType)
				continue
			}
			
			if err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}
			
		case "cmd":
			// 发送原始命令
			if len(parts) < 3 {
				fmt.Println("用法: cmd <acm|stm32> <命令>")
				fmt.Println("示例: cmd acm ver")
				continue
			}
			
			portType := parts[1]
			command := strings.Join(parts[2:], " ")
			
			var port *serial.SerialPort
			switch portType {
			case "acm":
				port = manager.ACMPort
			case "stm32":
				port = manager.STM32Port
			default:
				fmt.Printf("未知的端口类型: %s\n", portType)
				continue
			}
			
			if err := manager.SendRawCommand(port, command); err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}
			
		case "m1", "m2", "m3", "m4", "m5", "m6":
			// 快速发送预定义消息
			runPredefinedMessage(manager, command)
			
		case "status":
			manager.ShowStatus()
			
		case "test":
			runTestSequence(manager)
			
		case "exit", "quit":
			fmt.Println("退出程序...")
			manager.CloseAll()
			return
			
		default:
			fmt.Printf("未知命令: %s\n", command)
			fmt.Println("输入 'help' 查看可用命令")
		}
	}
}

// runPredefinedMessage 运行预定义消息
func runPredefinedMessage(manager *EnhancedSerialManager, msgType string) {
	var msg serial.GameMessage
	
	switch msgType {
	case "m1":
		msg = serial.GameMessage{MsgType: "M1"}
		fmt.Println("发送M1基础消息...")
		
	case "m2":
		msg = serial.GameMessage{
			MsgType:  "M2",
			Code:     200,
			Function: "test",
		}
		fmt.Println("发送M2响应消息...")
		
	case "m3":
		msg = serial.GameMessage{
			MsgType: "M3",
			State:   1,
		}
		fmt.Println("发送M3状态消息...")
		
	case "m4":
		msg = serial.GameMessage{
			MsgType: "M4",
			Action:  "wait",
		}
		fmt.Println("发送M4控制消息(wait)...")
		
	case "m5":
		msg = serial.GameMessage{
			MsgType: "M5",
			UpState: 1,
		}
		fmt.Println("发送M5更新状态消息...")
		
	case "m6":
		msg = serial.GameMessage{
			MsgType: "M6",
			TopType: 0,
			Data:    json.RawMessage(`{"test":"data"}`),
		}
		fmt.Println("发送M6 MQTT消息...")
	}
	
	// 默认发送到STM32（游戏逻辑处理）
	if err := manager.SendGameMessage(manager.STM32Port, msg); err != nil {
		fmt.Printf("发送失败: %v\n", err)
	}
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println(`
可用命令:
  auto                                      - 自动连接配置文件中的串口
  
  connect <acm|stm32> <设备路径> <波特率>    - 手动连接串口
    示例: connect acm auto 115200
    示例: connect stm32 /dev/ttyS3 115200
    
  disconnect <acm|stm32|all>                - 断开串口
    
  send <acm|stm32> <JSON消息>               - 发送JSON格式消息
    示例: send stm32 {"MsgType":"M4","action":"wait"}
    
  cmd <acm|stm32> <命令>                    - 发送原始命令
    示例: cmd acm ver
    示例: cmd acm sta
    
  m1, m2, m3, m4, m5, m6                   - 发送预定义消息类型
    
  status                                    - 显示串口状态和消息类型说明
  test                                      - 运行测试序列
  help                                      - 显示此帮助信息
  exit/quit                                 - 退出程序

消息类型说明:
  M1 - 基础消息
  M2 - 响应消息 (包含code和function)
  M3 - 状态消息 (包含state)
  M4 - 控制消息 (action: wait/exit/start/fail)
  M5 - 更新状态消息 (包含upstate)
  M6 - MQTT消息 (toptype: 0/1/2)
`)
}

// runTestSequence 运行测试序列
func runTestSequence(manager *EnhancedSerialManager) {
	fmt.Println("\n🧪 开始测试序列（基于Java代码实际协议）...")
	
	// 测试消息列表
	testMessages := []struct {
		port string
		msg  interface{}
		desc string
		wait time.Duration
	}{
		{
			port: "acm",
			msg:  "ver",
			desc: "查询ACM版本",
			wait: 2 * time.Second,
		},
		{
			port: "acm",
			msg:  "sta",
			desc: "查询WiFi状态",
			wait: 2 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M4",
				Action:  "wait",
			},
			desc: "发送等待命令",
			wait: 1 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M1",
				Data:    json.RawMessage(`{"cmd":"get_version"}`),
			},
			desc: "获取STM32版本",
			wait: 1 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M4",
				Action:  "start",
			},
			desc: "启动游戏",
			wait: 1 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M3",
				State:   1,
			},
			desc: "设置状态",
			wait: 1 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M2",
				Code:    100,
				Function: "HP30",
			},
			desc: "HP30游戏功能",
			wait: 1 * time.Second,
		},
		{
			port: "stm32",
			msg: serial.GameMessage{
				MsgType: "M4",
				Action:  "exit",
			},
			desc: "退出游戏",
			wait: 1 * time.Second,
		},
	}
	
	for i, test := range testMessages {
		fmt.Printf("\n📝 测试 %d/%d: %s\n", i+1, len(testMessages), test.desc)
		
		var err error
		var port *serial.SerialPort
		
		// 选择端口
		switch test.port {
		case "acm":
			port = manager.ACMPort
			if port == nil || !port.IsOpen {
				fmt.Println("   ⚠️  ACM设备未连接，跳过")
				continue
			}
		case "stm32":
			port = manager.STM32Port
			if port == nil || !port.IsOpen {
				fmt.Println("   ⚠️  STM32芯片未连接，跳过")
				continue
			}
		}
		
		// 发送消息
		switch msg := test.msg.(type) {
		case string:
			err = manager.SendRawCommand(port, msg)
		case serial.GameMessage:
			err = manager.SendGameMessage(port, msg)
		}
		
		if err != nil {
			fmt.Printf("   ❌ 测试失败: %v\n", err)
		} else {
			fmt.Println("   ✅ 发送成功")
		}
		
		// 等待响应
		fmt.Printf("   ⏳ 等待响应 %v...\n", test.wait)
		time.Sleep(test.wait)
	}
	
	fmt.Println("\n✅ 测试序列完成")
}

func main() {
	// 加载配置
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	
	fmt.Println("📋 配置加载成功")
	fmt.Printf("   ACM端口: %s (自动检测: %v)\n", 
		config.ACMPort.Device, config.ACMPort.Device == "auto")
	fmt.Printf("   STM32端口: %s\n", config.STM32Port.Device)
	
	manager := NewEnhancedSerialManager(config)
	defer manager.CloseAll()
	
	// 如果配置了自动重连，启动重连协程
	if config.AutoReconnect {
		fmt.Println("🔄 自动重连已启用")
	}
	
	// 检查命令行参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "auto":
			// 自动连接模式
			fmt.Println("🔌 自动连接模式...")
			
			if config.ACMPort.Device == "auto" {
				if err := manager.InitACMPort("auto", config.ACMPort.BaudRate); err != nil {
					log.Printf("连接ACM设备失败: %v", err)
				}
			} else {
				if err := manager.InitACMPort(config.ACMPort.Device, config.ACMPort.BaudRate); err != nil {
					log.Printf("连接ACM设备失败: %v", err)
				}
			}
			
			if err := manager.InitSTM32Port(config.STM32Port.Device, config.STM32Port.BaudRate); err != nil {
				log.Printf("连接STM32设备失败: %v", err)
			}
		case "test":
			// 直接运行测试
			runTestSequence(manager)
			return
		}
	}
	
	// 运行交互式界面
	runInteractive(manager)
}