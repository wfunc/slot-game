package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tarm/serial"
)

// 模拟Android(Acmlib.java)端的消息处理
type AndroidMessage struct {
	MsgType string                 `json:"MsgType"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Idex    int                    `json:"idex,omitempty"`
	Code    int                    `json:"code,omitempty"`
	Action  string                 `json:"action,omitempty"`
	State   int                    `json:"state,omitempty"`
}

func main() {
	// 串口配置 - 与Acmlib.java完全一致
	config := &serial.Config{
		Name:        "/dev/ttyS3",
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2, // Java代码使用STOPB.B2
		ReadTimeout: time.Second * 1,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	fmt.Println("=== Android端模拟器 (Acmlib.java) ===")
	fmt.Println("串口: /dev/ttyS3 @ 115200 8N2")
	fmt.Println("模拟Android与STM32通信")
	fmt.Println("命令:")
	fmt.Println("  algo - 发送算法请求")
	fmt.Println("  info - 发送配置信息")
	fmt.Println("  mqtt - 请求MQTT配置")
	fmt.Println("  ver  - 查询版本")
	fmt.Println("  quit - 退出")
	fmt.Println("----------------------------------------")

	// 启动接收协程
	go receiveFromSTM32(port)

	// 读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	idex := 1000

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		cmd := scanner.Text()
		if cmd == "quit" {
			break
		}

		var msg interface{}

		switch cmd {
		case "algo":
			// 模拟发送算法请求（M2类型）
			idex++
			msg = AndroidMessage{
				MsgType: "M2",
				Code:    0,
				Idex:    idex,
				Data: map[string]interface{}{
					"function": "algo",
					"param1":   100,
					"param2":   200,
				},
			}
			fmt.Println("发送算法请求...")

		case "info":
			// 模拟发送配置信息（M1类型）
			msg = AndroidMessage{
				MsgType: "M1",
				Data: map[string]interface{}{
					"cfgData": map[string]interface{}{
						"hp30":    1,
						"timeout": 30,
						"retry":   3,
					},
					"info": map[string]interface{}{
						"version": "1.2.1",
						"device":  "Android Simulator",
					},
				},
			}
			fmt.Println("发送配置信息...")

		case "mqtt":
			// 模拟MQTT配置请求（M6类型）
			msg = map[string]interface{}{
				"MsgType": "M6",
				"toptype": 0,
				"data":    "",
			}
			fmt.Println("请求MQTT配置...")

		case "ver":
			// 模拟版本查询（M4类型）
			msg = AndroidMessage{
				MsgType: "M4",
				Action:  "wait",
			}
			fmt.Println("查询版本...")

		default:
			// 直接发送文本
			if cmd != "" {
				port.Write([]byte(cmd + "\r\n"))
				fmt.Printf("发送自定义消息: %s\n", cmd)
			}
			continue
		}

		// 发送JSON消息
		if msg != nil {
			sendMessage(port, msg)
		}
	}
}

// 发送消息到串口
func sendMessage(port *serial.Port, msg interface{}) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return
	}

	// 添加\r\n结束符
	sendData := append(jsonData, []byte("\r\n")...)
	
	n, err := port.Write(sendData)
	if err != nil {
		log.Printf("发送失败: %v", err)
	} else {
		fmt.Printf("已发送 %d 字节: %s\n", n, string(jsonData))
	}
}

// 接收STM32的消息
func receiveFromSTM32(port *serial.Port) {
	buf := make([]byte, 1024*10)
	bufLen := 0

	for {
		// 读取数据
		n, err := port.Read(buf[bufLen:])
		if err != nil {
			continue
		}
		
		if n > 0 {
			bufLen += n
			
			// 查找消息结束符\r\n
			for i := 0; i < bufLen-1; i++ {
				if buf[i] == '\r' && buf[i+1] == '\n' {
					// 提取完整消息
					msgData := buf[:i]
					
					// 解析并显示
					processSTM32Message(msgData)
					
					// 移动剩余数据
					copy(buf, buf[i+2:bufLen])
					bufLen = bufLen - i - 2
					break
				}
			}
			
			// 防止缓冲区溢出
			if bufLen >= len(buf)-100 {
				bufLen = 0
			}
		}
	}
}

// 处理STM32消息
func processSTM32Message(data []byte) {
	fmt.Printf("\n[STM32消息] %s\n", string(data))
	
	// 尝试解析JSON
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		if msgType, ok := msg["MsgType"].(string); ok {
			handleSTM32Response(msgType, msg)
		}
	}
}

// 处理STM32响应
func handleSTM32Response(msgType string, msg map[string]interface{}) {
	switch msgType {
	case "M1":
		fmt.Println("  类型: 数据消息")
		if data, ok := msg["data"].(map[string]interface{}); ok {
			if function, ok := data["function"].(string); ok {
				fmt.Printf("  功能: %s\n", function)
				if function == "algo" {
					// 算法响应
					if win, ok := data["win"].(float64); ok {
						fmt.Printf("  中奖: %.2f\n", win)
					}
					if hp30, ok := data["hp30"].(float64); ok {
						fmt.Printf("  HP30: %.0f\n", hp30)
					}
				}
			}
		}

	case "M2":
		fmt.Println("  类型: 确认消息")
		if idex, ok := msg["idex"].(float64); ok {
			fmt.Printf("  索引: %.0f\n", idex)
		}

	case "M4":
		fmt.Println("  类型: 控制消息")
		if action, ok := msg["action"].(string); ok {
			fmt.Printf("  动作: %s\n", action)
		}
		if ready, ok := msg["ready"].(string); ok {
			fmt.Printf("  就绪: %s\n", ready)
		}

	case "M6":
		fmt.Println("  类型: MQTT消息")
		if toptype, ok := msg["toptype"].(float64); ok {
			fmt.Printf("  主题类型: %.0f\n", toptype)
		}
	}
	
	fmt.Print("> ")
}