package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tarm/serial"
)

func main() {
	// 串口配置
	config := &serial.Config{
		Name:        "/dev/ttyS3",
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: time.Second * 1,
	}

	// 打开串口
	port, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("无法打开串口: %v", err)
	}
	defer port.Close()

	fmt.Println("串口已打开: /dev/ttyS3 @ 115200")
	fmt.Println("输入 'quit' 退出程序")
	fmt.Println("输入 'read' 读取数据")
	fmt.Println("其他输入将发送到串口")
	fmt.Println("----------------------------------------")

	// 启动读取协程
	go readSerial(port)

	// 读取用户输入并发送
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "quit" {
			fmt.Println("退出程序...")
			break
		}

		if input == "read" {
			// 读取模式已经在后台运行
			fmt.Println("正在持续读取串口数据...")
			continue
		}

		// 发送数据到串口
		data := []byte(input + "\r\n")
		n, err := port.Write(data)
		if err != nil {
			fmt.Printf("发送失败: %v\n", err)
		} else {
			fmt.Printf("已发送 %d 字节: %s\n", n, input)
		}
	}
}

// 持续读取串口数据
func readSerial(port *serial.Port) {
	buf := make([]byte, 256)
	for {
		n, err := port.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("读取错误: %v", err)
			}
			continue
		}
		if n > 0 {
			fmt.Printf("\n[接收] %d 字节: %s", n, string(buf[:n]))
			fmt.Print("> ") // 重新显示提示符
		}
	}
}