package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
)

// STM32硬件协议测试工具
// 根据Acmlib.java实现的协议测试

var (
	acmDevice  = flag.String("acm", "/dev/ttyACM0", "ACM设备路径")
	stm32Device = flag.String("stm32", "/dev/ttyS3", "STM32设备路径")
	mode       = flag.String("mode", "bridge", "运行模式: bridge/monitor/test")
	duration   = flag.Int("duration", 60, "测试持续时间(秒)")
)

// 消息统计
type Statistics struct {
	sync.Mutex
	ACMCommands   int
	STM32Messages int
	Errors        int
	StartTime     time.Time
}

var stats = &Statistics{
	StartTime: time.Now(),
}

func main() {
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("    STM32硬件协议测试 v1.0")
	fmt.Println("========================================")
	fmt.Printf("模式: %s\n", *mode)
	fmt.Printf("ACM设备: %s\n", *acmDevice)
	fmt.Printf("STM32设备: %s\n", *stm32Device)
	fmt.Println("========================================")

	switch *mode {
	case "bridge":
		runBridgeMode()
	case "monitor":
		runMonitorMode()
	case "test":
		runTestMode()
	default:
		fmt.Println("未知模式")
	}
}

// 桥接模式：ACM命令 <-> STM32 JSON
func runBridgeMode() {
	fmt.Println("【桥接模式】")
	fmt.Println("监听ACM命令并转发到STM32...")
	
	// 打开ACM设备
	acmPort, err := openACM(*acmDevice)
	if err != nil {
		log.Fatalf("无法打开ACM设备: %v", err)
	}
	defer acmPort.Close()

	// 打开STM32设备
	stm32Port, err := openSTM32(*stm32Device)
	if err != nil {
		log.Fatalf("无法打开STM32设备: %v", err)
	}
	defer stm32Port.Close()

	// 启动双向桥接
	go bridgeACMToSTM32(acmPort, stm32Port)
	go bridgeSTM32ToACM(stm32Port, acmPort)

	// 定期显示统计
	go showStatistics()

	// 运行指定时间
	time.Sleep(time.Duration(*duration) * time.Second)
	
	fmt.Println("\n测试结束")
	printFinalStats()
}

// 监控模式：只监听不干预
func runMonitorMode() {
	fmt.Println("【监控模式】")
	fmt.Println("监听所有通信...")

	var wg sync.WaitGroup

	// 监控ACM
	if port, err := openACM(*acmDevice); err == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorACM(port)
		}()
	}

	// 监控STM32
	if port, err := openSTM32(*stm32Device); err == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			monitorSTM32(port)
		}()
	}

	// 运行指定时间
	time.Sleep(time.Duration(*duration) * time.Second)
	fmt.Println("\n监控结束")
}

// 测试模式：主动发送测试消息
func runTestMode() {
	fmt.Println("【测试模式】")
	fmt.Println("发送测试消息序列...")

	// 测试ACM命令
	if port, err := openACM(*acmDevice); err == nil {
		testACMCommands(port)
		port.Close()
	}

	time.Sleep(2 * time.Second)

	// 测试STM32 JSON
	if port, err := openSTM32(*stm32Device); err == nil {
		testSTM32Messages(port)
		port.Close()
	}

	fmt.Println("\n测试完成")
	printFinalStats()
}

func openACM(device string) (*serial.Port, error) {
	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Millisecond * 100,
	}
	return serial.OpenPort(config)
}

func openSTM32(device string) (*serial.Port, error) {
	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Millisecond * 100,
	}
	return serial.OpenPort(config)
}

// ACM -> STM32 桥接
func bridgeACMToSTM32(acm, stm32 *serial.Port) {
	buf := make([]byte, 1024)
	cmdBuf := make([]byte, 0, 1024)

	for {
		n, err := acm.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			cmdBuf = append(cmdBuf, buf[:n]...)
			
			// 查找命令结束符
			for i := 0; i < len(cmdBuf); i++ {
				if cmdBuf[i] == '\n' {
					cmd := string(cmdBuf[:i])
					cmdBuf = cmdBuf[i+1:]
					
					// 处理ACM命令
					processACMCommand(cmd, stm32, acm)
					
					stats.Lock()
					stats.ACMCommands++
					stats.Unlock()
					
					break
				}
			}
		}
	}
}

// STM32 -> ACM 桥接
func bridgeSTM32ToACM(stm32, acm *serial.Port) {
	buf := make([]byte, 1024)
	msgBuf := make([]byte, 0, 1024)

	for {
		n, err := stm32.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			msgBuf = append(msgBuf, buf[:n]...)
			
			// 查找JSON消息结束
			for i := 0; i < len(msgBuf)-1; i++ {
				if msgBuf[i] == '\r' && msgBuf[i+1] == '\n' {
					msg := msgBuf[:i]
					msgBuf = msgBuf[i+2:]
					
					// 处理STM32响应
					processSTM32Response(msg, acm)
					
					stats.Lock()
					stats.STM32Messages++
					stats.Unlock()
					
					break
				}
			}
		}
	}
}

// 处理ACM命令并转换为STM32 JSON
func processACMCommand(cmd string, stm32, acm *serial.Port) {
	fmt.Printf("[ACM→] %s\n", cmd)
	
	// 根据命令类型转换为JSON
	var msg map[string]interface{}
	
	if cmd == "ver" {
		msg = map[string]interface{}{
			"MsgType": "M4",
			"action":  "version",
		}
	} else if cmd == "sta" {
		msg = map[string]interface{}{
			"MsgType": "M4",
			"action":  "status",
		}
	} else if len(cmd) > 4 && cmd[:4] == "algo" {
		// 解析algo命令参数
		msg = map[string]interface{}{
			"MsgType": "M2",
			"idex":    time.Now().Unix(),
			"data": map[string]interface{}{
				"cmd": cmd,
			},
		}
	} else {
		// 未知命令，返回错误
		response := fmt.Sprintf("Command not recognised: %s\n>", cmd)
		acm.Write([]byte(response))
		return
	}

	// 发送JSON到STM32
	if msg != nil {
		jsonData, _ := json.Marshal(msg)
		sendData := append(jsonData, []byte("\r\n")...)
		stm32.Write(sendData)
		fmt.Printf("[→STM32] %s\n", string(jsonData))
	}
}

// 处理STM32响应并转换为ACM格式
func processSTM32Response(msg []byte, acm *serial.Port) {
	fmt.Printf("[STM32←] %s\n", string(msg))
	
	// 解析JSON响应
	var response map[string]interface{}
	if err := json.Unmarshal(msg, &response); err != nil {
		stats.Lock()
		stats.Errors++
		stats.Unlock()
		return
	}

	// 根据消息类型转换为ACM响应格式
	var acmResponse string
	
	if msgType, ok := response["MsgType"].(string); ok {
		switch msgType {
		case "M1":
			// 配置响应
			if data, ok := response["data"].(map[string]interface{}); ok {
				jsonData, _ := json.Marshal(data)
				acmResponse = fmt.Sprintf("%s\n>", string(jsonData))
			}
		case "M2":
			// 算法响应
			if data, ok := response["data"].(map[string]interface{}); ok {
				jsonData, _ := json.Marshal(data)
				acmResponse = fmt.Sprintf("%s\n>", string(jsonData))
			}
		case "M4":
			// 版本/状态响应
			if action, ok := response["action"].(string); ok {
				if action == "version" {
					acmResponse = fmt.Sprintf("{\"ver\":\"1.0.0\"}\n>")
				} else if action == "status" {
					acmResponse = fmt.Sprintf("{\"status\":\"ready\"}\n>")
				}
			}
		default:
			// 直接转发JSON
			acmResponse = fmt.Sprintf("%s\n>", string(msg))
		}
	}

	// 发送响应到ACM
	if acmResponse != "" {
		acm.Write([]byte(acmResponse))
		fmt.Printf("[←ACM] %s", acmResponse)
	}
}

// 监控ACM通信
func monitorACM(port *serial.Port) {
	fmt.Println("监控ACM设备...")
	buf := make([]byte, 1024)
	
	for {
		n, err := port.Read(buf)
		if err != nil {
			continue
		}
		
		if n > 0 {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s ACM] %q\n", timestamp, string(buf[:n]))
		}
	}
}

// 监控STM32通信
func monitorSTM32(port *serial.Port) {
	fmt.Println("监控STM32设备...")
	buf := make([]byte, 1024)
	
	for {
		n, err := port.Read(buf)
		if err != nil {
			continue
		}
		
		if n > 0 {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s STM32] %q\n", timestamp, string(buf[:n]))
		}
	}
}

// 测试ACM命令
func testACMCommands(port *serial.Port) {
	fmt.Println("\n测试ACM命令:")
	
	commands := []string{
		"ver",
		"sta",
		"algo -b 1 -p 100",
		"help",
	}

	for _, cmd := range commands {
		fmt.Printf("发送: %s\n", cmd)
		port.Write([]byte(cmd + "\r\n"))
		
		// 读取响应
		buf := make([]byte, 1024)
		totalRead := 0
		deadline := time.Now().Add(2 * time.Second)
		
		for time.Now().Before(deadline) {
			n, _ := port.Read(buf[totalRead:])
			if n > 0 {
				totalRead += n
				response := string(buf[:totalRead])
				if strings.Contains(response, "\n>") {
					fmt.Printf("响应: %s\n", response)
					break
				}
			}
		}
		
		time.Sleep(500 * time.Millisecond)
	}
}

// 测试STM32消息
func testSTM32Messages(port *serial.Port) {
	fmt.Println("\n测试STM32消息:")
	
	messages := []map[string]interface{}{
		{
			"MsgType": "M1",
			"data": map[string]interface{}{
				"cfgData": map[string]interface{}{
					"hp30": 1,
				},
			},
		},
		{
			"MsgType": "M2",
			"idex":    1000,
			"data": map[string]interface{}{
				"test": "hello",
			},
		},
		{
			"MsgType": "M4",
			"action":  "wait",
		},
		{
			"MsgType": "M6",
			"toptype": 0,
			"data":    "",
		},
	}

	for _, msg := range messages {
		jsonData, _ := json.Marshal(msg)
		fmt.Printf("发送: %s\n", string(jsonData))
		
		sendData := append(jsonData, []byte("\r\n")...)
		port.Write(sendData)
		
		// 读取响应
		buf := make([]byte, 1024)
		n, _ := port.Read(buf)
		
		if n > 0 {
			fmt.Printf("响应: %q\n", string(buf[:n]))
		} else {
			fmt.Println("无响应")
		}
		
		time.Sleep(500 * time.Millisecond)
	}
}

// 显示统计信息
func showStatistics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats.Lock()
			elapsed := time.Since(stats.StartTime).Seconds()
			fmt.Printf("\n[统计] 运行时间: %.0fs | ACM命令: %d | STM32消息: %d | 错误: %d\n",
				elapsed, stats.ACMCommands, stats.STM32Messages, stats.Errors)
			stats.Unlock()
		}
	}
}

// 打印最终统计
func printFinalStats() {
	stats.Lock()
	defer stats.Unlock()
	
	elapsed := time.Since(stats.StartTime).Seconds()
	fmt.Println("\n========================================")
	fmt.Println("           最终统计")
	fmt.Println("========================================")
	fmt.Printf("运行时间: %.1f 秒\n", elapsed)
	fmt.Printf("ACM命令数: %d\n", stats.ACMCommands)
	fmt.Printf("STM32消息数: %d\n", stats.STM32Messages)
	fmt.Printf("错误数: %d\n", stats.Errors)
	
	if stats.ACMCommands > 0 {
		fmt.Printf("ACM命令速率: %.2f/秒\n", float64(stats.ACMCommands)/elapsed)
	}
	if stats.STM32Messages > 0 {
		fmt.Printf("STM32消息速率: %.2f/秒\n", float64(stats.STM32Messages)/elapsed)
	}
	if stats.Errors > 0 {
		fmt.Printf("错误率: %.2f%%\n", float64(stats.Errors)/float64(stats.ACMCommands+stats.STM32Messages)*100)
	}
}