package main

import (
	"flag"
	"fmt"
	"os"
	
	"github.com/wfunc/slot-game/api"
	"github.com/wfunc/slot-game/internal/hardware"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

func main() {
	// 命令行参数
	var (
		port         = flag.String("port", "8080", "API服务端口")
		enableHW     = flag.Bool("hw", false, "启用硬件集成")
		mockHW       = flag.Bool("mock", false, "使用模拟硬件")
		serialPort   = flag.String("serial", "/dev/ttyS3", "串口设备")
		logLevel     = flag.String("log", "info", "日志级别(debug/info/error)")
	)
	flag.Parse()
	
	// 初始化日志
	logger.InitLogger(*logLevel, "simple-api.log")
	log := logger.GetLogger()
	
	fmt.Println("=== 推币机游戏API服务 ===")
	fmt.Printf("端口: %s\n", *port)
	
	// 创建简化版游戏API
	gameAPI := api.NewSimpleGameAPI()
	
	// 如果启用硬件集成
	if *enableHW {
		fmt.Println("正在初始化硬件集成...")
		
		// 创建硬件配置
		hwConfig := &hardware.HardwareConfig{
			SerialPort:     *serialPort,
			BaudRate:       115200,
			InitialCredits: 0,
			GameMode:       hardware.ModeCoinRefund,
			AutoStart:      true,
			DebugMode:      *logLevel == "debug",
			MockMode:       *mockHW,
		}
		
		// 创建硬件管理器
		hwManager := hardware.NewHardwareManager(hwConfig)
		
		// 初始化硬件
		if err := hwManager.Initialize(); err != nil {
			log.Error("硬件初始化失败", zap.Error(err))
			fmt.Printf("⚠️ 硬件初始化失败: %v\n", err)
			fmt.Println("将在无硬件模式下运行")
		} else {
			// 启动硬件管理器
			if err := hwManager.Start(); err != nil {
				log.Error("硬件启动失败", zap.Error(err))
				fmt.Printf("⚠️ 硬件启动失败: %v\n", err)
				fmt.Println("将在无硬件模式下运行")
			} else {
				fmt.Printf("✅ 硬件集成已启用 (串口: %s)\n", *serialPort)
				
				// 创建硬件API并注册路由
				hwAPI := api.NewHardwareAPI(hwManager)
				gameAPI.RegisterHardwareAPI(hwAPI)
				
				// 确保程序退出时停止硬件
				defer func() {
					fmt.Println("正在关闭硬件连接...")
					if err := hwManager.Stop(); err != nil {
						log.Error("硬件停止失败", zap.Error(err))
					}
				}()
			}
		}
	} else {
		fmt.Println("ℹ️ 硬件集成未启用 (使用 -hw 参数启用)")
	}
	
	fmt.Println("")
	fmt.Println("API端点:")
	fmt.Println("  游戏: http://localhost:" + *port + "/api/game/")
	fmt.Println("  旋转: http://localhost:" + *port + "/api/spin")
	fmt.Println("  统计: http://localhost:" + *port + "/api/stats")
	if *enableHW {
		fmt.Println("  硬件: http://localhost:" + *port + "/api/hardware/")
	}
	fmt.Println("")
	fmt.Println("按 Ctrl+C 停止服务...")
	
	// 启动服务器
	if err := gameAPI.Start(*port); err != nil {
		log.Fatal("服务器启动失败", zap.Error(err))
		os.Exit(1)
	}
}