package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

var (
	device = flag.String("d", "/dev/ttyS3", "串口设备")
	mode   = flag.String("m", "algo", "测试模式: algo/mqtt/version/update")
	count  = flag.Int("n", 1, "发送次数")
	delay  = flag.Int("delay", 1000, "消息间隔(毫秒)")
)

func main() {
	flag.Parse()

	// 串口配置 - 与Acmlib.java完全一致
	config := &serial.Config{
		Name:        *device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2, // 重要：2个停止位
		ReadTimeout: time.Millisecond * 100,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	fmt.Printf("✓ 串口已打开: %s @ 115200 8N2\n", *device)
	fmt.Printf("✓ 测试模式: %s\n", *mode)
	fmt.Printf("✓ 发送次数: %d\n", *count)
	fmt.Println("========================================")

	// 启动接收协程
	go receiveMessages(port)

	// 根据模式发送不同的测试消息
	switch *mode {
	case "algo":
		testAlgoProtocol(port)
	case "mqtt":
		testMqttProtocol(port)
	case "version":
		testVersionProtocol(port)
	case "update":
		testUpdateProtocol(port)
	default:
		testCustomProtocol(port)
	}

	// 继续接收
	fmt.Println("\n继续监听接收... (按Ctrl+C退出)")
	select {}
}

// 测试算法协议 (M1/M2消息)
func testAlgoProtocol(port *serial.Port) {
	fmt.Println("\n【算法协议测试】")
	idex := 1000

	for i := 0; i < *count; i++ {
		idex++
		
		// 发送M2消息（算法请求）
		msg := map[string]interface{}{
			"MsgType": "M2",
			"idex":    idex,
			"data": map[string]interface{}{
				"function": "algo",
				"param1":   100,
				"param2":   200,
			},
		}
		
		sendJSON(port, msg, fmt.Sprintf("算法请求 #%d", i+1))
		time.Sleep(time.Duration(*delay) * time.Millisecond)

		// 发送M1消息（配置）
		if i == 0 {
			cfgMsg := map[string]interface{}{
				"MsgType": "M1",
				"data": map[string]interface{}{
					"cfgData": map[string]interface{}{
						"hp30": 1,
					},
				},
			}
			sendJSON(port, cfgMsg, "配置消息")
			time.Sleep(time.Duration(*delay) * time.Millisecond)
		}
	}
}

// 测试MQTT协议 (M6消息)
func testMqttProtocol(port *serial.Port) {
	fmt.Println("\n【MQTT协议测试】")
	
	// 请求MQTT配置
	msg := map[string]interface{}{
		"MsgType": "M6",
		"toptype": 0,
		"data":    "",
	}
	
	sendJSON(port, msg, "MQTT配置请求")
	time.Sleep(2000 * time.Millisecond)

	// 发送MQTT数据
	msg2 := map[string]interface{}{
		"MsgType": "M6",
		"toptype": 1,
		"data": map[string]interface{}{
			"test": "mqtt_data",
		},
	}
	
	sendJSON(port, msg2, "MQTT数据")
}

// 测试版本协议 (M4消息)
func testVersionProtocol(port *serial.Port) {
	fmt.Println("\n【版本协议测试】")
	
	// 查询版本
	msg := map[string]interface{}{
		"MsgType": "M4",
		"cVer":    "1.0.0",
		"lVer":    "1.0.0",
		"devType": "testDevice",
		"uid":     "TEST001",
	}
	
	sendJSON(port, msg, "版本查询")
	time.Sleep(2000 * time.Millisecond)

	// 等待响应
	msg2 := map[string]interface{}{
		"MsgType": "M4",
		"action":  "wait",
	}
	
	sendJSON(port, msg2, "等待状态")
}

// 测试更新协议 (M3/M5消息)
func testUpdateProtocol(port *serial.Port) {
	fmt.Println("\n【更新协议测试】")
	
	// WiFi配置
	msg := map[string]interface{}{
		"MsgType":  "M3",
		"wifiname": "TestWiFi",
		"wifipass": "12345678",
		"path":     "/test",
	}
	
	sendJSON(port, msg, "WiFi配置")
	time.Sleep(2000 * time.Millisecond)

	// 版本更新
	msg2 := map[string]interface{}{
		"MsgType": "M5",
		"upver":   "1.2.1",
	}
	
	sendJSON(port, msg2, "版本更新")
}

// 自定义协议测试
func testCustomProtocol(port *serial.Port) {
	fmt.Println("\n【自定义协议测试】")
	
	// 构造测试消息
	messages := []map[string]interface{}{
		{
			"MsgType": "M1",
			"data": map[string]interface{}{
				"test": "hello",
			},
		},
		{
			"MsgType": "M2",
			"idex":    9999,
			"code":    0,
		},
		{
			"MsgType": "M6",
			"toptype": 0,
			"data":    "",
		},
	}

	for i, msg := range messages {
		sendJSON(port, msg, fmt.Sprintf("自定义消息 #%d", i+1))
		time.Sleep(time.Duration(*delay) * time.Millisecond)
	}
}

// 发送JSON消息
func sendJSON(port *serial.Port, msg map[string]interface{}, desc string) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
		return
	}

	// 添加\r\n结束符（重要！）
	sendData := append(jsonData, []byte("\r\n")...)
	
	n, err := port.Write(sendData)
	if err != nil {
		log.Printf("发送失败: %v", err)
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	fmt.Printf("[%s] → 发送 %s (%d字节)\n", timestamp, desc, n)
	fmt.Printf("           内容: %s\n", string(jsonData))
}

// 接收消息
func receiveMessages(port *serial.Port) {
	buf := make([]byte, 1024*10)
	bufLen := 0

	for {
		n, err := port.Read(buf[bufLen:])
		if err != nil {
			continue
		}

		if n > 0 {
			bufLen += n

			// 查找消息边界 - 支持两种结束符
			// 1. JSON以\r\n结束
			// 2. ACM响应以\n>结束
			
			// 先检查JSON格式(\r\n)
			for i := 0; i < bufLen-1; i++ {
				if buf[i] == '\r' && buf[i+1] == '\n' {
					msgData := buf[:i]
					processMessage(msgData, "JSON")
					
					// 移动剩余数据
					copy(buf, buf[i+2:bufLen])
					bufLen = bufLen - i - 2
					break
				}
			}

			// 再检查ACM格式(\n>)
			for i := 0; i < bufLen-1; i++ {
				if buf[i] == '\n' && buf[i+1] == '>' {
					msgData := buf[:i]
					processMessage(msgData, "ACM")
					
					// 移动剩余数据
					copy(buf, buf[i+2:bufLen])
					bufLen = bufLen - i - 2
					break
				}
			}

			// 防止缓冲区溢出
			if bufLen >= len(buf)-100 {
				fmt.Printf("[警告] 缓冲区接近满，清空\n")
				fmt.Printf("        原始数据: %s\n", string(buf[:100]))
				bufLen = 0
			}
		}
	}
}

// 处理接收到的消息
func processMessage(data []byte, format string) {
	timestamp := time.Now().Format("15:04:05.000")
	
	if len(data) == 0 {
		return
	}

	// 清理数据
	cleaned := string(data)
	
	fmt.Printf("\n[%s] ← 接收 (%s格式, %d字节)\n", timestamp, format, len(data))
	
	// 尝试解析JSON
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err == nil {
		// 成功解析为JSON
		fmt.Printf("           类型: JSON消息\n")
		
		if msgType, ok := msg["MsgType"].(string); ok {
			fmt.Printf("           MsgType: %s\n", msgType)
			
			switch msgType {
			case "M1":
				if data, ok := msg["data"].(map[string]interface{}); ok {
					fmt.Printf("           数据: %+v\n", data)
				}
			case "M2":
				if idex, ok := msg["idex"].(float64); ok {
					fmt.Printf("           索引: %.0f\n", idex)
				}
				if code, ok := msg["code"].(float64); ok {
					fmt.Printf("           代码: %.0f\n", code)
				}
			case "M4":
				if action, ok := msg["action"].(string); ok {
					fmt.Printf("           动作: %s\n", action)
				}
				if ready, ok := msg["ready"].(string); ok {
					fmt.Printf("           就绪: %s\n", ready)
				}
			case "M6":
				if toptype, ok := msg["toptype"].(float64); ok {
					fmt.Printf("           主题类型: %.0f\n", toptype)
				}
			}
		}
	} else {
		// 非JSON格式，直接显示
		fmt.Printf("           原始: %s\n", cleaned)
	}
}