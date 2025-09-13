package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var (
	verbose = flag.Bool("v", false, "详细输出模式")
	test    = flag.String("test", "all", "测试类型: all/acm/stm32/loopback/hw")
)

type TestResult struct {
	Device     string
	Test       string
	Success    bool
	Message    string
	ResponseMs int64
}

func main() {
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("    硬件通信诊断工具 v2.0")
	fmt.Println("========================================")
	fmt.Println()

	results := []TestResult{}

	// 1. 系统环境检查
	fmt.Println("【1. 系统环境检查】")
	checkSystemEnvironment()

	// 2. 设备发现
	fmt.Println("\n【2. 设备发现】")
	devices := discoverDevices()

	// 3. 根据测试类型执行
	switch *test {
	case "acm":
		results = append(results, testACMDevice()...)
	case "stm32":
		results = append(results, testSTM32Device()...)
	case "loopback":
		results = append(results, testLoopback()...)
	case "hw":
		results = append(results, testHardwareProtocol()...)
	default:
		// 测试所有设备
		results = append(results, testAllDevices(devices)...)
	}

	// 4. 生成诊断报告
	generateReport(results)
}

func checkSystemEnvironment() {
	// 检查内核模块
	fmt.Print("检查串口驱动模块... ")
	cmd := exec.Command("lsmod")
	output, _ := cmd.Output()
	if strings.Contains(string(output), "ftdi_sio") || strings.Contains(string(output), "ch341") {
		fmt.Println("✓ USB串口驱动已加载")
	} else {
		fmt.Println("⚠️  未检测到常见USB串口驱动")
	}

	// 检查dmesg中的串口信息
	fmt.Print("检查系统串口日志... ")
	cmd = exec.Command("sh", "-c", "dmesg | grep -i 'tty\\|serial' | tail -5")
	output, _ = cmd.Output()
	if len(output) > 0 {
		fmt.Println("✓ 找到串口相关日志")
		if *verbose {
			fmt.Println(string(output))
		}
	}

	// 检查串口配置
	fmt.Print("检查串口默认配置... ")
	for _, dev := range []string{"/dev/ttyS3", "/dev/ttyACM0"} {
		if _, err := os.Stat(dev); err == nil {
			cmd = exec.Command("stty", "-F", dev, "-a")
			output, err := cmd.Output()
			if err == nil && *verbose {
				fmt.Printf("\n%s配置:\n%s\n", dev, output)
			}
		}
	}
	fmt.Println("✓")
}

func discoverDevices() []string {
	devices := []string{}
	
	// 检查所有可能的串口设备
	patterns := []string{
		"/dev/ttyS*",
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
	}

	for _, pattern := range patterns {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("ls %s 2>/dev/null", pattern))
		output, _ := cmd.Output()
		if len(output) > 0 {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				if line != "" {
					devices = append(devices, line)
					
					// 检查设备状态
					checkDeviceStatus(line)
				}
			}
		}
	}

	if len(devices) == 0 {
		fmt.Println("❌ 未发现任何串口设备")
	} else {
		fmt.Printf("✓ 发现 %d 个串口设备\n", len(devices))
	}

	return devices
}

func checkDeviceStatus(device string) {
	fmt.Printf("  %s: ", device)
	
	// 检查设备是否被占用
	cmd := exec.Command("lsof", device)
	output, _ := cmd.Output()
	if len(output) > 0 {
		fmt.Print("被占用 ")
	} else {
		fmt.Print("空闲 ")
	}

	// 检查权限
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		fmt.Print("无权限 ")
	} else {
		file.Close()
		fmt.Print("可访问 ")
	}

	// 特殊设备识别
	if strings.Contains(device, "ACM") {
		fmt.Print("[ACM设备] ")
	} else if strings.Contains(device, "ttyS3") {
		fmt.Print("[STM32?] ")
	}

	fmt.Println()
}

func testAllDevices(devices []string) []TestResult {
	results := []TestResult{}

	for _, device := range devices {
		fmt.Printf("\n【测试设备: %s】\n", device)
		
		// 基础连接测试
		result := testConnection(device)
		results = append(results, result)

		if result.Success {
			// 如果是ACM设备，测试命令协议
			if strings.Contains(device, "ACM") {
				results = append(results, testACMProtocol(device))
			}
			
			// 如果是ttyS3，测试JSON协议
			if strings.Contains(device, "ttyS3") {
				results = append(results, testJSONProtocol(device))
			}
		}
	}

	return results
}

func testConnection(device string) TestResult {
	fmt.Printf("测试基础连接... ")
	
	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return TestResult{
			Device:  device,
			Test:    "基础连接",
			Success: false,
			Message: err.Error(),
		}
	}
	defer port.Close()

	// 发送测试字节
	startTime := time.Now()
	_, err = port.Write([]byte("\r\n"))
	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
		return TestResult{
			Device:  device,
			Test:    "基础连接",
			Success: false,
			Message: fmt.Sprintf("写入失败: %v", err),
		}
	}

	// 尝试读取响应
	buf := make([]byte, 128)
	n, _ := port.Read(buf)
	responseTime := time.Since(startTime).Milliseconds()

	if n > 0 {
		fmt.Printf("✓ 有响应 (%dms)\n", responseTime)
		if *verbose {
			fmt.Printf("  响应数据: %q\n", string(buf[:n]))
		}
		return TestResult{
			Device:     device,
			Test:       "基础连接",
			Success:    true,
			Message:    fmt.Sprintf("响应: %q", string(buf[:n])),
			ResponseMs: responseTime,
		}
	} else {
		fmt.Println("⚠️  无响应")
		return TestResult{
			Device:  device,
			Test:    "基础连接",
			Success: false,
			Message: "设备无响应",
		}
	}
}

func testACMProtocol(device string) TestResult {
	fmt.Printf("测试ACM命令协议... ")
	
	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 2,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return TestResult{
			Device:  device,
			Test:    "ACM协议",
			Success: false,
			Message: err.Error(),
		}
	}
	defer port.Close()

	// 测试ver命令
	_, err = port.Write([]byte("ver\r\n"))
	if err != nil {
		return TestResult{
			Device:  device,
			Test:    "ACM协议",
			Success: false,
			Message: "发送失败",
		}
	}

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
				fmt.Println("✓ ACM协议正常")
				if *verbose {
					fmt.Printf("  版本信息: %s\n", response)
				}
				return TestResult{
					Device:  device,
					Test:    "ACM协议",
					Success: true,
					Message: "ACM命令协议正常",
				}
			}
		}
	}

	fmt.Println("❌ ACM协议异常")
	return TestResult{
		Device:  device,
		Test:    "ACM协议",
		Success: false,
		Message: "ACM协议无响应",
	}
}

func testJSONProtocol(device string) TestResult {
	fmt.Printf("测试JSON协议... ")
	
	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 2,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return TestResult{
			Device:  device,
			Test:    "JSON协议",
			Success: false,
			Message: err.Error(),
		}
	}
	defer port.Close()

	// 发送M4消息（版本查询）
	msg := map[string]interface{}{
		"MsgType": "M4",
		"action":  "wait",
	}

	jsonData, _ := json.Marshal(msg)
	sendData := append(jsonData, []byte("\r\n")...)
	
	_, err = port.Write(sendData)
	if err != nil {
		return TestResult{
			Device:  device,
			Test:    "JSON协议",
			Success: false,
			Message: "发送失败",
		}
	}

	// 读取响应
	buf := make([]byte, 1024)
	n, _ := port.Read(buf)

	if n > 0 {
		response := string(buf[:n])
		var respMsg map[string]interface{}
		if err := json.Unmarshal(buf[:n], &respMsg); err == nil {
			fmt.Println("✓ JSON协议正常")
			return TestResult{
				Device:  device,
				Test:    "JSON协议",
				Success: true,
				Message: "JSON协议正常",
			}
		} else {
			fmt.Printf("⚠️  收到非JSON响应: %q\n", response)
			return TestResult{
				Device:  device,
				Test:    "JSON协议",
				Success: false,
				Message: fmt.Sprintf("非JSON响应: %q", response),
			}
		}
	}

	fmt.Println("❌ JSON协议无响应")
	return TestResult{
		Device:  device,
		Test:    "JSON协议",
		Success: false,
		Message: "JSON协议无响应",
	}
}

func testACMDevice() []TestResult {
	results := []TestResult{}
	device := "/dev/ttyACM0"

	fmt.Printf("\n【专项测试: ACM设备】\n")

	// 测试各种命令
	commands := []string{"ver", "sta", "algo -b 1 -p 100", "help"}

	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 2,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		results = append(results, TestResult{
			Device:  device,
			Test:    "ACM设备",
			Success: false,
			Message: err.Error(),
		})
		return results
	}
	defer port.Close()

	for _, cmd := range commands {
		fmt.Printf("测试命令 '%s'... ", cmd)
		
		_, err := port.Write([]byte(cmd + "\r\n"))
		if err != nil {
			fmt.Printf("❌ 发送失败\n")
			continue
		}

		buf := make([]byte, 4096)
		totalRead := 0
		deadline := time.Now().Add(2 * time.Second)

		for time.Now().Before(deadline) {
			n, _ := port.Read(buf[totalRead:])
			if n > 0 {
				totalRead += n
				response := string(buf[:totalRead])
				if strings.Contains(response, "\n>") {
					fmt.Printf("✓ 成功\n")
					if *verbose {
						fmt.Printf("  响应: %s\n", response)
					}
					
					results = append(results, TestResult{
						Device:  device,
						Test:    fmt.Sprintf("ACM命令-%s", cmd),
						Success: true,
						Message: "命令执行成功",
					})
					break
				}
			}
		}
	}

	return results
}

func testSTM32Device() []TestResult {
	results := []TestResult{}
	device := "/dev/ttyS3"

	fmt.Printf("\n【专项测试: STM32设备】\n")

	// 测试各种消息类型
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
				"function": "algo",
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

	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 2,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		results = append(results, TestResult{
			Device:  device,
			Test:    "STM32设备",
			Success: false,
			Message: err.Error(),
		})
		return results
	}
	defer port.Close()

	for _, msg := range messages {
		msgType := msg["MsgType"].(string)
		fmt.Printf("测试消息 %s... ", msgType)
		
		jsonData, _ := json.Marshal(msg)
		sendData := append(jsonData, []byte("\r\n")...)
		
		_, err := port.Write(sendData)
		if err != nil {
			fmt.Printf("❌ 发送失败\n")
			continue
		}

		buf := make([]byte, 1024)
		n, _ := port.Read(buf)

		if n > 0 {
			fmt.Printf("✓ 有响应\n")
			if *verbose {
				fmt.Printf("  响应: %q\n", string(buf[:n]))
			}
			
			results = append(results, TestResult{
				Device:  device,
				Test:    fmt.Sprintf("STM32-%s", msgType),
				Success: true,
				Message: "消息有响应",
			})
		} else {
			fmt.Printf("❌ 无响应\n")
			results = append(results, TestResult{
				Device:  device,
				Test:    fmt.Sprintf("STM32-%s", msgType),
				Success: false,
				Message: "消息无响应",
			})
		}
	}

	return results
}

func testLoopback() []TestResult {
	fmt.Printf("\n【回环测试】\n")
	fmt.Println("请将要测试设备的TX和RX短接")
	fmt.Print("输入设备路径 (默认/dev/ttyS3): ")
	
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	device := strings.TrimSpace(input)
	if device == "" {
		device = "/dev/ttyS3"
	}

	config := &serial.Config{
		Name:        device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return []TestResult{{
			Device:  device,
			Test:    "回环测试",
			Success: false,
			Message: err.Error(),
		}}
	}
	defer port.Close()

	testData := "LOOPBACK_TEST_12345\r\n"
	fmt.Printf("发送: %s", testData)
	
	_, err = port.Write([]byte(testData))
	if err != nil {
		return []TestResult{{
			Device:  device,
			Test:    "回环测试",
			Success: false,
			Message: "发送失败",
		}}
	}

	buf := make([]byte, 128)
	n, err := port.Read(buf)
	if err != nil || n == 0 {
		fmt.Println("❌ 回环测试失败")
		return []TestResult{{
			Device:  device,
			Test:    "回环测试",
			Success: false,
			Message: "未收到回环数据",
		}}
	}

	received := string(buf[:n])
	if received == testData {
		fmt.Printf("接收: %s", received)
		fmt.Println("✓ 回环测试成功")
		return []TestResult{{
			Device:  device,
			Test:    "回环测试",
			Success: true,
			Message: "回环测试通过",
		}}
	} else {
		fmt.Printf("接收: %s", received)
		fmt.Println("❌ 数据不匹配")
		return []TestResult{{
			Device:  device,
			Test:    "回环测试",
			Success: false,
			Message: "数据不匹配",
		}}
	}
}

func testHardwareProtocol() []TestResult {
	fmt.Printf("\n【硬件协议测试】\n")
	fmt.Println("模拟完整的通信流程:")
	fmt.Println("1. ACM接收命令")
	fmt.Println("2. 转换为JSON发送到STM32")
	fmt.Println("3. 接收STM32响应")
	fmt.Println("4. 返回结果给ACM")
	
	results := []TestResult{}
	
	// 这里可以实现更复杂的协议测试
	fmt.Println("\n⚠️  此测试需要两个设备都连接正常")
	
	return results
}

func generateReport(results []TestResult) {
	fmt.Println("\n========================================")
	fmt.Println("           诊断报告")
	fmt.Println("========================================")

	// 统计
	total := len(results)
	success := 0
	for _, r := range results {
		if r.Success {
			success++
		}
	}

	fmt.Printf("\n测试总数: %d\n", total)
	fmt.Printf("成功: %d\n", success)
	fmt.Printf("失败: %d\n", total-success)
	fmt.Printf("成功率: %.1f%%\n", float64(success)/float64(total)*100)

	// 详细结果
	fmt.Println("\n详细结果:")
	fmt.Println("设备\t\t测试\t\t结果\t响应时间")
	fmt.Println("------------------------------------------------------------")
	for _, r := range results {
		status := "❌"
		if r.Success {
			status = "✓"
		}
		fmt.Printf("%-15s\t%-15s\t%s\t", r.Device, r.Test, status)
		if r.ResponseMs > 0 {
			fmt.Printf("%dms", r.ResponseMs)
		}
		fmt.Println()
		if *verbose && r.Message != "" {
			fmt.Printf("  → %s\n", r.Message)
		}
	}

	// 建议
	fmt.Println("\n诊断建议:")
	
	hasACM := false
	hasSTM32 := false
	acmWorks := false
	stm32Works := false

	for _, r := range results {
		if strings.Contains(r.Device, "ACM") {
			hasACM = true
			if r.Success {
				acmWorks = true
			}
		}
		if strings.Contains(r.Device, "ttyS3") {
			hasSTM32 = true
			if r.Success {
				stm32Works = true
			}
		}
	}

	if hasACM && acmWorks {
		fmt.Println("✓ ACM设备工作正常，可以接收命令")
	} else if hasACM && !acmWorks {
		fmt.Println("⚠️  ACM设备存在但无法正常工作")
		fmt.Println("  - 检查设备权限")
		fmt.Println("  - 确认波特率设置")
	} else {
		fmt.Println("❌ 未发现ACM设备")
		fmt.Println("  - 检查USB连接")
		fmt.Println("  - 检查驱动安装")
	}

	if hasSTM32 && stm32Works {
		fmt.Println("✓ STM32通信正常")
	} else if hasSTM32 && !stm32Works {
		fmt.Println("⚠️  /dev/ttyS3存在但无响应")
		fmt.Println("  - STM32可能未连接或未上电")
		fmt.Println("  - 检查TX/RX接线是否正确")
		fmt.Println("  - 验证STM32固件是否正在运行")
		fmt.Println("  - 尝试回环测试验证硬件")
	} else {
		fmt.Println("❌ 未发现/dev/ttyS3")
		fmt.Println("  - 检查硬件连接")
		fmt.Println("  - 验证串口配置")
	}

	// 生成测试脚本建议
	fmt.Println("\n推荐的下一步测试:")
	if !stm32Works {
		fmt.Println("1. 执行回环测试: ./hardware_diagnose -test loopback")
		fmt.Println("2. 检查硬件连接: TX→RX, RX→TX, GND→GND")
		fmt.Println("3. 使用示波器验证信号")
	}
	if acmWorks && !stm32Works {
		fmt.Println("4. 验证ACM到STM32的桥接逻辑")
	}
}