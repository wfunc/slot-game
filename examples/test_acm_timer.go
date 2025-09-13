package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wfunc/slot-game/internal/hardware"
	"github.com/wfunc/slot-game/internal/logger"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	if err := logger.Init("debug", "console", "./logs"); err != nil {
		log.Fatal("初始化日志失败:", err)
	}
	defer logger.Sync()

	log := logger.GetLogger()
	log.Info("ACM定时器测试程序启动")

	// 创建ACM配置
	config := &hardware.ACMConfig{
		Port:         "auto",           // 自动检测ACM设备
		BaudRate:     115200,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
		AutoDetect:   true,
		
		// 启用Algo定时器
		AlgoTimerEnabled:  true,
		AlgoTimerInterval: 3 * time.Second,  // 每3秒发送一次
		AlgoBet:          1,
		AlgoPrize:        100,
	}

	// 创建ACM控制器
	controller := hardware.NewACMController(config)

	// 设置消息处理器（可选）
	controller.SetMessageHandler(func(msg map[string]interface{}) {
		log.Info("收到ACM消息", zap.Any("message", msg))
	})

	// 连接ACM设备
	log.Info("正在连接ACM设备...")
	if err := controller.Connect(); err != nil {
		log.Fatal("连接ACM设备失败", zap.Error(err))
	}
	defer controller.Disconnect()

	log.Info("ACM设备连接成功，定时器已启动")
	log.Info("定时器配置", 
		zap.Duration("interval", config.AlgoTimerInterval),
		zap.Int("bet", config.AlgoBet),
		zap.Int("prize", config.AlgoPrize))

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 运行测试
	fmt.Println("\n按 Ctrl+C 退出程序")
	fmt.Println("定时器正在运行，每3秒发送一次algo命令...")
	fmt.Println("")

	// 测试运行30秒后停止定时器
	go func() {
		time.Sleep(30 * time.Second)
		log.Info("测试停止定时器")
		controller.StopAlgoTimer()
		
		time.Sleep(5 * time.Second)
		log.Info("测试重新启动定时器，使用新参数")
		controller.SetAlgoTimer(true, 2*time.Second, 2, 200)
	}()

	// 等待退出信号
	<-sigCh
	
	log.Info("收到退出信号，正在关闭...")
	controller.StopAlgoTimer()
	controller.Disconnect()
	
	log.Info("测试程序结束")
}