package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"testserial/pkg/serial"
)

// 交互式命令行界面
func runInteractive(manager *serial.SerialManager) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\n=== 串口测试程序（基础版）===")
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
			
		case "connect":
			if len(parts) < 4 {
				fmt.Println("用法: connect <acm|stm32> <设备路径> <波特率>")
				fmt.Println("示例: connect acm /dev/ttyUSB0 115200")
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
				fmt.Println("示例: send acm {\"type\":\"command\",\"command\":\"status\"}")
				continue
			}
			
			portType := parts[1]
			messageStr := strings.Join(parts[2:], " ")
			
			var msg serial.Message
			if err := json.Unmarshal([]byte(messageStr), &msg); err != nil {
				// 如果不是JSON，创建简单消息
				msg = serial.Message{
					Type:    "command",
					Command: messageStr,
				}
			}
			
			var err error
			switch portType {
			case "acm":
				err = manager.SendToACM(msg)
			case "stm32":
				err = manager.SendToSTM32(msg)
			default:
				fmt.Printf("未知的端口类型: %s\n", portType)
				continue
			}
			
			if err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}
			
		case "status":
			manager.Status()
			
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

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println(`
可用命令:
  connect <acm|stm32> <设备路径> <波特率>  - 连接串口
    示例: connect acm /dev/ttyUSB0 115200
    
  disconnect <acm|stm32|all>               - 断开串口
    示例: disconnect all
    
  send <acm|stm32> <消息>                  - 发送消息
    示例: send acm {"type":"command","command":"status"}
    
  status                                    - 显示串口状态
  test                                      - 运行测试序列
  help                                      - 显示此帮助信息
  exit/quit                                 - 退出程序
`)
}

// runTestSequence 运行测试序列
func runTestSequence(manager *serial.SerialManager) {
	fmt.Println("\n开始测试序列...")
	
	// 测试消息列表
	testMessages := []struct {
		port string
		msg  serial.Message
		desc string
	}{
		{
			port: "acm",
			msg: serial.Message{
				Type:    "command",
				Command: "status",
			},
			desc: "查询ACM设备状态",
		},
		{
			port: "stm32",
			msg: serial.Message{
				Type:    "command",
				Command: "get_chip_id",
			},
			desc: "获取STM32芯片ID",
		},
		{
			port: "acm",
			msg: serial.Message{
				Type:    "command",
				Command: "led_control",
				Data:    json.RawMessage(`{"led":1,"state":"on"}`),
			},
			desc: "控制LED灯",
		},
	}
	
	for _, test := range testMessages {
		fmt.Printf("\n📝 测试: %s\n", test.desc)
		
		var err error
		switch test.port {
		case "acm":
			if manager.ACMPort == nil || !manager.ACMPort.IsOpen {
				fmt.Println("   ⚠️  ACM设备未连接，跳过")
				continue
			}
			err = manager.SendToACM(test.msg)
		case "stm32":
			if manager.STM32Port == nil || !manager.STM32Port.IsOpen {
				fmt.Println("   ⚠️  STM32芯片未连接，跳过")
				continue
			}
			err = manager.SendToSTM32(test.msg)
		}
		
		if err != nil {
			fmt.Printf("   ❌ 测试失败: %v\n", err)
		} else {
			fmt.Println("   ✅ 发送成功")
		}
		
		// 等待响应
		time.Sleep(time.Second)
	}
	
	fmt.Println("\n测试序列完成")
}

func main() {
	manager := serial.NewSerialManager()
	defer manager.CloseAll()
	
	// 检查命令行参数
	if len(os.Args) > 1 {
		// 如果有参数，尝试自动连接
		if len(os.Args) >= 3 {
			acmDevice := os.Args[1]
			stm32Device := os.Args[2]
			
			fmt.Printf("尝试连接:\n")
			fmt.Printf("  ACM设备: %s\n", acmDevice)
			fmt.Printf("  STM32设备: %s\n", stm32Device)
			
			if err := manager.InitACMPort(acmDevice, 115200); err != nil {
				log.Printf("连接ACM设备失败: %v", err)
			}
			
			if err := manager.InitSTM32Port(stm32Device, 115200); err != nil {
				log.Printf("连接STM32设备失败: %v", err)
			}
		}
	}
	
	// 运行交互式界面
	runInteractive(manager)
}