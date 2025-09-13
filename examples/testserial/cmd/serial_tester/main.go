package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var (
	device   = flag.String("d", "/dev/ttyS3", "串口设备")
	baudrate = flag.Int("b", 115200, "波特率")
	stopbits = flag.Int("s", 2, "停止位(1或2)")
	mode     = flag.String("m", "both", "模式: send/recv/both/echo")
	interval = flag.Int("i", 1000, "发送间隔(毫秒)")
	message  = flag.String("msg", "TEST", "测试消息")
	verbose  = flag.Bool("v", false, "详细输出")
)

func main() {
	flag.Parse()

	// 配置串口
	var stopBit serial.StopBits
	if *stopbits == 2 {
		stopBit = serial.Stop2
	} else {
		stopBit = serial.Stop1
	}

	config := &serial.Config{
		Name:        *device,
		Baud:        *baudrate,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    stopBit,
		ReadTimeout: time.Millisecond * 100,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口 %s: %v", *device, err)
	}
	defer port.Close()

	fmt.Printf("✓ 串口已打开: %s @ %d baud, 8N%d\n", *device, *baudrate, *stopbits)
	fmt.Printf("✓ 模式: %s\n", *mode)
	fmt.Println("----------------------------------------")

	switch *mode {
	case "send":
		sendMode(port)
	case "recv":
		recvMode(port)
	case "echo":
		echoMode(port)
	case "both":
		bothMode(port)
	default:
		fmt.Println("无效模式，使用 -h 查看帮助")
	}
}

// 只发送模式
func sendMode(port *serial.Port) {
	fmt.Println("【发送模式】")
	fmt.Printf("每%dms发送: %s\n", *interval, *message)
	fmt.Println("按Ctrl+C退出")
	fmt.Println()

	count := 0
	for {
		count++
		msg := fmt.Sprintf("[%d] %s\r\n", count, *message)
		n, err := port.Write([]byte(msg))
		if err != nil {
			fmt.Printf("✗ 发送失败: %v\n", err)
		} else {
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Printf("[%s] → 发送 %d 字节: %s", timestamp, n, strings.TrimSpace(msg))
			if *verbose {
				fmt.Printf(" (HEX: %X)", []byte(msg))
			}
			fmt.Println()
		}
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
}

// 只接收模式
func recvMode(port *serial.Port) {
	fmt.Println("【接收模式】")
	fmt.Println("等待接收数据...")
	fmt.Println("按Ctrl+C退出")
	fmt.Println()

	buf := make([]byte, 1024)
	totalBytes := 0
	startTime := time.Now()

	for {
		n, err := port.Read(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "EOF") {
				fmt.Printf("✗ 读取错误: %v\n", err)
			}
			continue
		}
		if n > 0 {
			totalBytes += n
			timestamp := time.Now().Format("15:04:05.000")
			data := string(buf[:n])
			
			// 去除控制字符用于显示
			displayData := strings.ReplaceAll(data, "\r", "\\r")
			displayData = strings.ReplaceAll(displayData, "\n", "\\n")
			
			fmt.Printf("[%s] ← 接收 %d 字节: %s", timestamp, n, displayData)
			if *verbose {
				fmt.Printf("\n           HEX: %X", buf[:n])
			}
			fmt.Println()
			
			// 统计信息
			duration := time.Since(startTime).Seconds()
			if duration > 0 {
				fmt.Printf("           统计: 总计 %d 字节, %.1f 字节/秒\n", totalBytes, float64(totalBytes)/duration)
			}
		}
	}
}

// 回显模式（接收后立即发回）
func echoMode(port *serial.Port) {
	fmt.Println("【回显模式】")
	fmt.Println("接收数据后立即发回")
	fmt.Println("按Ctrl+C退出")
	fmt.Println()

	buf := make([]byte, 1024)
	echoCount := 0

	for {
		n, err := port.Read(buf)
		if err != nil {
			continue
		}
		if n > 0 {
			echoCount++
			timestamp := time.Now().Format("15:04:05.000")
			data := buf[:n]
			
			fmt.Printf("[%s] ← 接收 %d 字节\n", timestamp, n)
			
			// 立即回显
			wn, werr := port.Write(data)
			if werr != nil {
				fmt.Printf("           ✗ 回显失败: %v\n", werr)
			} else {
				fmt.Printf("           → 回显 %d 字节 (第%d次)\n", wn, echoCount)
			}
		}
	}
}

// 交互模式（同时收发）
func bothMode(port *serial.Port) {
	fmt.Println("【交互模式】")
	fmt.Println("输入文本发送，同时显示接收的数据")
	fmt.Println("命令:")
	fmt.Println("  :test N  - 发送N条测试消息")
	fmt.Println("  :hex     - 切换HEX显示")
	fmt.Println("  :stats   - 显示统计信息")
	fmt.Println("  :clear   - 清屏")
	fmt.Println("  :quit    - 退出")
	fmt.Println("----------------------------------------")

	// 接收统计
	stats := struct {
		sent     int
		received int
		errors   int
	}{}

	showHex := false

	// 启动接收协程
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := port.Read(buf)
			if err != nil {
				if !strings.Contains(err.Error(), "EOF") {
					stats.errors++
				}
				continue
			}
			if n > 0 {
				stats.received += n
				timestamp := time.Now().Format("15:04:05.000")
				data := string(buf[:n])
				
				// 格式化显示
				lines := strings.Split(data, "\n")
				for _, line := range lines {
					if line = strings.TrimSpace(line); line != "" {
						fmt.Printf("[%s] ← %s", timestamp, line)
						if showHex {
							fmt.Printf(" (HEX: %X)", []byte(line))
						}
						fmt.Println()
					}
				}
			}
		}
	}()

	// 读取用户输入
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		
		// 处理命令
		if strings.HasPrefix(input, ":") {
			handleCommand(input, port, &stats, &showHex)
			continue
		}

		// 发送数据
		if input != "" {
			data := []byte(input + "\r\n")
			n, err := port.Write(data)
			if err != nil {
				fmt.Printf("✗ 发送失败: %v\n", err)
				stats.errors++
			} else {
				stats.sent += n
				timestamp := time.Now().Format("15:04:05.000")
				fmt.Printf("[%s] → 发送 %d 字节\n", timestamp, n)
			}
		}
	}
}

func handleCommand(cmd string, port *serial.Port, stats *struct{ sent, received, errors int }, showHex *bool) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case ":test":
		count := 10
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &count)
		}
		fmt.Printf("发送 %d 条测试消息...\n", count)
		for i := 1; i <= count; i++ {
			msg := fmt.Sprintf("TEST_%d_%d\r\n", time.Now().Unix(), i)
			n, err := port.Write([]byte(msg))
			if err != nil {
				fmt.Printf("✗ 发送失败: %v\n", err)
				stats.errors++
			} else {
				stats.sent += n
				fmt.Printf("→ 测试消息 %d/%d 已发送 (%d字节)\n", i, count, n)
			}
			time.Sleep(100 * time.Millisecond)
		}

	case ":hex":
		*showHex = !*showHex
		status := "关闭"
		if *showHex {
			status = "开启"
		}
		fmt.Printf("HEX显示: %s\n", status)

	case ":stats":
		fmt.Println("=== 统计信息 ===")
		fmt.Printf("发送: %d 字节\n", stats.sent)
		fmt.Printf("接收: %d 字节\n", stats.received)
		fmt.Printf("错误: %d 次\n", stats.errors)

	case ":clear":
		fmt.Print("\033[2J\033[H")
		fmt.Println("屏幕已清空")

	case ":quit":
		fmt.Println("退出程序...")
		os.Exit(0)

	default:
		fmt.Println("未知命令:", cmd)
	}
}