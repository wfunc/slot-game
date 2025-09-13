package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var (
	device   = flag.String("d", "/dev/ttyACM0", "串口设备")
	loopback = flag.Bool("loopback", false, "回环测试模式")
	monitor  = flag.Bool("monitor", false, "监控模式")
)

func main() {
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("       串口诊断工具")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 检查设备是否存在
	checkDevice(*device)

	// 2. 检查设备权限
	checkPermissions(*device)

	// 3. 检查串口是否被占用
	checkPortUsage(*device)

	// 4. 显示串口配置
	showPortConfig(*device)

	// 5. 执行测试
	if *loopback {
		fmt.Println("\n【执行回环测试】")
		fmt.Println("请确保TX和RX已短接！")
		fmt.Print("按Enter继续...")
		bufio.NewReader(os.Stdin).ReadLine()
		loopbackTest()
	} else if *monitor {
		fmt.Println("\n【监控模式】")
		monitorPort()
	} else {
		fmt.Println("\n【交互式诊断】")
		interactiveDiagnose()
	}
}

func checkDevice(device string) {
	fmt.Printf("1. 检查设备 %s ... ", device)
	if _, err := os.Stat(device); os.IsNotExist(err) {
		fmt.Println("❌ 不存在")
		fmt.Println("\n可用的串口设备:")
		listSerialPorts()
		os.Exit(1)
	}
	fmt.Println("✓ 存在")
}

func checkPermissions(device string) {
	fmt.Printf("2. 检查权限 ... ")
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("❌ 无权限\n")
		fmt.Println("   解决方法:")
		fmt.Println("   - 使用 sudo 运行")
		fmt.Printf("   - 或添加用户到dialout组: sudo usermod -a -G dialout %s\n", os.Getenv("USER"))
	} else {
		file.Close()
		fmt.Println("✓ 有读写权限")
	}
}

func checkPortUsage(device string) {
	fmt.Printf("3. 检查占用 ... ")

	// 使用 lsof 检查
	cmd := exec.Command("lsof", device)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// lsof 没有输出表示没有进程占用
		fmt.Println("✓ 未被占用")
	} else {
		fmt.Println("⚠️  被占用")
		fmt.Println("占用进程:")
		fmt.Println(string(output))
		fmt.Println("解决方法: 结束占用进程或使用 sudo kill <PID>")
	}
}

func showPortConfig(device string) {
	fmt.Printf("4. 串口配置:\n")

	cmd := exec.Command("stty", "-F", device, "-a")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("   无法读取配置: %v\n", err)
	} else {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			// 解析第一行显示主要参数
			fmt.Printf("   %s\n", lines[0])
		}
	}
}

func listSerialPorts() {
	// 列出所有可能的串口
	patterns := []string{
		"/dev/ttyS*",
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
		"/dev/tty.usb*",
	}

	for _, pattern := range patterns {
		matches, _ := exec.Command("sh", "-c", "ls "+pattern+" 2>/dev/null").Output()
		if len(matches) > 0 {
			fmt.Print(string(matches))
		}
	}
}

func loopbackTest() {
	config := &serial.Config{
		Name:        *device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 1,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	// 发送测试数据
	testData := "LOOPBACK_TEST_123\r\n"
	fmt.Printf("发送: %s", testData)

	n, err := port.Write([]byte(testData))
	if err != nil {
		fmt.Printf("❌ 发送失败: %v\n", err)
		return
	}
	fmt.Printf("已发送 %d 字节\n", n)

	// 读取响应
	buf := make([]byte, 128)
	fmt.Print("等待接收...")

	n, err = port.Read(buf)
	if err != nil {
		fmt.Printf("❌ 接收失败: %v\n", err)
		fmt.Println("\n可能原因:")
		fmt.Println("- TX和RX未短接")
		fmt.Println("- 串口硬件故障")
		fmt.Println("- 驱动问题")
		return
	}

	received := string(buf[:n])
	fmt.Printf("\n接收: %s", received)

	if received == testData {
		fmt.Println("✓ 回环测试成功！")
	} else {
		fmt.Printf("⚠️  数据不匹配\n")
		fmt.Printf("期望: %q\n", testData)
		fmt.Printf("实际: %q\n", received)
	}
}

func monitorPort() {
	config := &serial.Config{
		Name:        *device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Millisecond * 100,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	fmt.Println("监控中... (显示所有接收到的数据)")
	fmt.Println("按Ctrl+C退出")
	fmt.Println("----------------------------------------")

	buf := make([]byte, 1)
	count := 0
	lastTime := time.Now()

	for {
		n, err := port.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			count++
			now := time.Now()

			// 如果超过1秒没有数据，打印时间戳
			if now.Sub(lastTime) > time.Second {
				fmt.Printf("\n[%s] ", now.Format("15:04:05"))
			}

			// 打印字符
			if buf[0] >= 32 && buf[0] < 127 {
				fmt.Printf("%c", buf[0])
			} else {
				fmt.Printf("[%02X]", buf[0])
			}

			lastTime = now

			// 如果是换行符，打印统计
			if buf[0] == '\n' {
				fmt.Printf(" (总计: %d 字节)", count)
			}
		}
	}
}

func interactiveDiagnose() {
	fmt.Println("\n诊断步骤:")
	fmt.Println("1. 首先测试发送功能")
	fmt.Println("2. 然后测试接收功能")
	fmt.Println("3. 最后测试双向通信")
	fmt.Println()

	config := &serial.Config{
		Name:        *device,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop2,
		ReadTimeout: time.Second * 2,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	// 步骤1: 测试发送
	fmt.Println("【步骤1: 发送测试】")
	testMsg := `{"MsgType":"M2","idex":9999,"data":"DIAGNOSE"}` + "\r\n"
	fmt.Printf("发送JSON消息: %s", testMsg)

	n, err := port.Write([]byte(testMsg))
	if err != nil {
		fmt.Printf("❌ 发送失败: %v\n", err)
	} else {
		fmt.Printf("✓ 已发送 %d 字节\n", n)
	}

	// 步骤2: 测试接收
	fmt.Println("\n【步骤2: 接收测试】")
	fmt.Println("等待2秒接收数据...")

	buf := make([]byte, 1024)
	n, err = port.Read(buf)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			fmt.Println("⚠️  接收超时 - 没有收到任何数据")
			fmt.Println("\n可能的原因:")
			fmt.Println("1. 对端设备(STM32/ACM)未运行")
			fmt.Println("2. 对端设备未正确连接到串口")
			fmt.Println("3. TX/RX接线错误（可能需要交叉）")
			fmt.Println("4. 波特率或参数不匹配")
			fmt.Println("5. 对端程序没有响应该消息类型")
		} else {
			fmt.Printf("❌ 接收错误: %v\n", err)
		}
	} else if n > 0 {
		fmt.Printf("✓ 接收到 %d 字节\n", n)
		fmt.Printf("数据: %s\n", string(buf[:n]))
		fmt.Printf("HEX: % X\n", buf[:n])
	}

	// 步骤3: 连续测试
	fmt.Println("\n【步骤3: 连续监听】")
	fmt.Println("持续监听5秒...")

	endTime := time.Now().Add(5 * time.Second)
	totalReceived := 0

	for time.Now().Before(endTime) {
		n, err := port.Read(buf)
		if err == nil && n > 0 {
			totalReceived += n
			fmt.Printf("接收: %s", string(buf[:n]))
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n总计接收: %d 字节\n", totalReceived)

	// 诊断建议
	fmt.Println("\n========================================")
	fmt.Println("诊断建议:")

	if totalReceived == 0 {
		fmt.Println("❌ 没有接收到任何数据")
		fmt.Println()
		fmt.Println("建议检查:")
		fmt.Println("1. 确认对端设备是否在运行")
		fmt.Println("   - 检查STM32是否已启动")
		fmt.Println("   - 检查ACM设备是否已连接")
		fmt.Println()
		fmt.Println("2. 验证物理连接:")
		fmt.Println("   - TX → RX")
		fmt.Println("   - RX → TX")
		fmt.Println("   - GND → GND")
		fmt.Println()
		fmt.Println("3. 尝试回环测试:")
		fmt.Printf("   ./serial_diagnose -loopback\n")
		fmt.Println()
		fmt.Println("4. 使用其他工具验证:")
		fmt.Println("   minicom -D /dev/ttyS3 -b 115200")
		fmt.Println("   screen /dev/ttyS3 115200")
	} else {
		fmt.Println("✓ 串口通信正常")
		fmt.Println("可以使用 protocol_tester 进行协议测试")
	}
}
