package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

// 模拟STM32发送的JSON消息格式
type STM32Message struct {
	MsgType string      `json:"MsgType"`
	Data    interface{} `json:"data,omitempty"`
	Idex    int         `json:"idex,omitempty"`
	Action  string      `json:"action,omitempty"`
}

func main() {
	// 串口配置 - 与Java代码一致
	config := &serial.Config{
		Name:        "/dev/ttyS3",
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2, // 注意：Java代码使用2个停止位
		ReadTimeout: time.Second * 1,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	fmt.Println("=== STM32模拟器启动 ===")
	fmt.Println("串口: /dev/ttyS3 @ 115200 8N2")
	fmt.Println("模拟STM32与Android通信协议")
	fmt.Println("----------------------------------------")

	// 启动接收协程
	go receiveMessages(port)

	// 测试消息序列
	testMessages := []STM32Message{
		// M1类型消息 - 数据传输
		{
			MsgType: "M1",
			Data: map[string]interface{}{
				"function": "algo",
				"win":      100.5,
				"hp30":     1,
				"one":      1,
				"free":     0,
			},
		},
		// M2类型消息 - 带索引的数据
		{
			MsgType: "M2",
			Idex:    1001,
			Data: map[string]interface{}{
				"code":     0,
				"function": "test",
			},
		},
		// M4类型消息 - 版本检查
		{
			MsgType: "M4",
			Action:  "wait",
		},
		// M6类型消息 - MQTT相关
		{
			MsgType: "M6",
			Data: map[string]interface{}{
				"toptype": 0,
				"devid":   "TEST-DEVICE-001",
			},
		},
	}

	// 发送测试消息
	for i, msg := range testMessages {
		time.Sleep(2 * time.Second)
		
		// 序列化为JSON
		jsonData, err := json.Marshal(msg)
		if err != nil {
			log.Printf("JSON序列化失败: %v", err)
			continue
		}

		// 添加\r\n结束符（与Java代码一致）
		sendData := append(jsonData, []byte("\r\n")...)
		
		fmt.Printf("\n[发送 %d] %s", i+1, string(jsonData))
		
		n, err := port.Write(sendData)
		if err != nil {
			log.Printf("发送失败: %v", err)
		} else {
			fmt.Printf(" (已发送 %d 字节)", n)
		}
	}

	// 保持运行，继续接收消息
	fmt.Println("\n\n按 Ctrl+C 退出...")
	select {}
}

// 接收并解析消息
func receiveMessages(port *serial.Port) {
	scanner := bufio.NewScanner(port)
	fmt.Println("\n[接收线程启动]")

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fmt.Printf("\n[接收] %s", line)

		// 尝试解析JSON
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			// 根据消息类型处理
			if msgType, ok := msg["MsgType"].(string); ok {
				handleMessage(msgType, msg, port)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("读取错误: %v", err)
	}
}

// 处理接收到的消息
func handleMessage(msgType string, msg map[string]interface{}, port *serial.Port) {
	fmt.Printf("\n[处理] 消息类型: %s", msgType)

	switch msgType {
	case "M1":
		// 收到M1，回复M2确认
		response := STM32Message{
			MsgType: "M2",
			Idex:    int(time.Now().Unix() % 10000),
			Data: map[string]interface{}{
				"code": 0,
				"msg":  "received",
			},
		}
		sendResponse(port, response)

	case "M2":
		// 算法相关响应
		if idex, ok := msg["idex"].(float64); ok {
			fmt.Printf(" - 索引: %.0f", idex)
		}

	case "M4":
		// 版本相关
		if action, ok := msg["action"].(string); ok {
			fmt.Printf(" - 动作: %s", action)
			if action == "wait" {
				// 回复ready
				response := map[string]interface{}{
					"MsgType": "M4",
					"ready":   "ok",
				}
				sendResponse(port, response)
			}
		}

	case "M6":
		// MQTT相关
		fmt.Printf(" - MQTT消息处理")
	}
}

// 发送响应
func sendResponse(port *serial.Port, response interface{}) {
	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("响应序列化失败: %v", err)
		return
	}

	sendData := append(jsonData, []byte("\r\n")...)
	fmt.Printf("\n[自动回复] %s", string(jsonData))
	
	if _, err := port.Write(sendData); err != nil {
		log.Printf("回复发送失败: %v", err)
	}
}