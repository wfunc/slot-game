package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/wfunc/slot-game/internal/api"
	"github.com/wfunc/slot-game/internal/config"
	"github.com/wfunc/slot-game/internal/database"
	"github.com/wfunc/slot-game/internal/errors"
	"github.com/wfunc/slot-game/internal/game"
	"github.com/wfunc/slot-game/internal/hardware"
	"github.com/wfunc/slot-game/internal/logger"
	"github.com/wfunc/slot-game/internal/service"
	"go.uber.org/zap"
)

// 版本信息
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// 服务器实例
type Server struct {
	cfg      *config.Config
	logger   *zap.Logger
	
	// 服务组件
	router           *api.Router
	httpServer       *http.Server
	recoveryManager  *game.RecoveryManager
	serialController hardware.HardwareController
	cleanupTicker    *time.Ticker
	// mqttClient    *mqtt.Client
	
	// 关闭控制
	shutdownCh chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func main() {
	// 命令行参数
	var (
		configPath  = flag.String("config", "", "配置文件路径")
		showVersion = flag.Bool("version", false, "显示版本信息")
		showHelp    = flag.Bool("help", false, "显示帮助信息")
	)
	
	flag.Parse()
	
	// 显示版本信息
	if *showVersion {
		printVersion()
		os.Exit(0)
	}
	
	// 显示帮助信息
	if *showHelp {
		printHelp()
		os.Exit(0)
	}
	
	// 加载配置
	if err := config.Init(*configPath); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}
	
	cfg := config.Get()
	
	// 初始化日志系统
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	
	// 设置系统参数
	setupSystem(&cfg.System)
	
	// 打印启动信息
	printStartInfo(cfg)
	
	// 创建服务器实例
	server := NewServer(cfg)
	
	// 启动服务器
	if err := server.Start(); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
	
	// 等待退出信号
	server.WaitForShutdown()
	
	// 优雅关闭
	if err := server.Shutdown(); err != nil {
		logger.Error("服务器关闭失败", zap.Error(err))
		os.Exit(1)
	}
	
	logger.Info("服务器已安全关闭")
}

// NewServer 创建服务器实例
func NewServer(cfg *config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Server{
		cfg:        cfg,
		logger:     logger.GetLogger(),
		shutdownCh: make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.logger.Info("正在启动推币机游戏服务器...",
		zap.String("version", Version),
		zap.String("mode", s.cfg.Server.Mode),
	)
	
	// 初始化各个组件
	if err := s.initComponents(); err != nil {
		return errors.Wrap(err, errors.ErrUnknown, "初始化组件失败")
	}
	
	// 启动各个服务
	if err := s.startServices(); err != nil {
		return errors.Wrap(err, errors.ErrUnknown, "启动服务失败")
	}
	
	// 监听配置变化
	config.Watch(func(newCfg *config.Config) {
		s.logger.Info("配置已更新，正在重新加载...")
		s.reloadConfig(newCfg)
	})
	
	s.logger.Info("服务器启动成功",
		zap.String("http", fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)),
		zap.String("websocket", fmt.Sprintf("ws://%s:%d/ws/game", s.cfg.Server.Host, s.cfg.Server.Port)),
	)
	
	return nil
}

// initComponents 初始化组件
func (s *Server) initComponents() error {
	s.logger.Info("初始化组件...")
	
	// 初始化数据库
	if err := s.initDatabase(); err != nil {
		return err
	}
	
	// 初始化HTTP路由
	if err := s.initHTTPRouter(); err != nil {
		return err
	}
	
	// 初始化游戏引擎和恢复管理器
	if err := s.initGameEngine(); err != nil {
		return err
	}
	
	// 初始化串口管理器（可选）
	if s.cfg.Serial.Enabled {
		if err := s.initSerialManager(); err != nil {
			s.logger.Warn("串口管理器初始化失败，使用模拟模式", zap.Error(err))
			// 不影响主流程，继续运行
		}
	}
	
	// TODO: 初始化MQTT客户端
	// if s.cfg.MQTT.Enabled {
	//     if err := s.initMQTTClient(); err != nil {
	//         return err
	//     }
	// }
	
	s.logger.Info("所有组件初始化完成")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase() error {
	s.logger.Info("初始化数据库...")
	
	// 初始化数据库连接
	if err := database.Init(&s.cfg.Database); err != nil {
		return errors.Wrap(err, errors.ErrDatabaseConnect, "初始化数据库连接失败")
	}
	
	// 自动迁移数据库
	if s.cfg.Database.AutoMigrate {
		s.logger.Info("执行数据库自动迁移...")
		if err := database.AutoMigrate(); err != nil {
			return errors.Wrap(err, errors.ErrDatabaseConnect, "数据库迁移失败")
		}
	}
	
	// 检查数据库连接
	if !database.IsConnected() {
		return errors.New(errors.ErrDatabaseConnect, "数据库连接检查失败")
	}
	
	s.logger.Info("数据库初始化完成")
	return nil
}

// initHTTPRouter 初始化HTTP路由
func (s *Server) initHTTPRouter() error {
	s.logger.Info("初始化HTTP路由...")
	
	// 获取数据库连接
	db := database.GetDB()
	if db == nil {
		return errors.New(errors.ErrDatabaseConnect, "数据库连接不可用")
	}
	
	// 创建服务配置
	serviceConfig := &service.Config{
		JWTSecret:          s.cfg.Security.JWT.Secret,
		AccessTokenExpiry:  time.Duration(s.cfg.Security.JWT.ExpireHours) * time.Hour,
		RefreshTokenExpiry: time.Duration(s.cfg.Security.JWT.RefreshHours) * time.Hour,
	}
	
	// 创建路由器
	s.router = api.NewRouter(db, serviceConfig, s.logger)
	
	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port),
		Handler:      s.router.GetEngine(),
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
	}
	
	s.logger.Info("HTTP路由初始化完成")
	return nil
}

// initGameEngine 初始化游戏引擎和恢复管理器
func (s *Server) initGameEngine() error {
	s.logger.Info("初始化游戏引擎和恢复管理器...")
	
	// 获取数据库连接
	db := database.GetDB()
	if db == nil {
		return errors.New(errors.ErrDatabaseConnect, "数据库连接不可用")
	}
	
	// 创建持久化器（使用数据库持久化）
	persister := game.NewDatabaseStatePersister(db)
	
	// 创建恢复管理器
	s.recoveryManager = game.NewRecoveryManager(
		s.logger,
		persister,
		db,
		30*time.Minute, // 会话超时时间
	)
	
	// 启动定时清理任务（每30分钟清理一次过期会话）
	s.cleanupTicker = time.NewTicker(30 * time.Minute)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info("启动会话清理定时任务", zap.Duration("interval", 30*time.Minute))
		
		for {
			select {
			case <-s.cleanupTicker.C:
				s.logger.Info("开始执行会话清理任务")
				if err := s.recoveryManager.CleanupExpiredSessions(s.ctx); err != nil {
					s.logger.Error("清理过期会话失败", zap.Error(err))
				}
			case <-s.ctx.Done():
				s.logger.Info("停止会话清理任务")
				return
			}
		}
	}()
	
	// 立即执行一次清理，清除启动前的过期会话
	go func() {
		time.Sleep(5 * time.Second) // 等待系统稳定
		s.logger.Info("执行启动清理任务")
		if err := s.recoveryManager.CleanupExpiredSessions(context.Background()); err != nil {
			s.logger.Error("启动清理失败", zap.Error(err))
		}
	}()
	
	s.logger.Info("游戏引擎和恢复管理器初始化完成")
	return nil
}

// initSerialManager 初始化串口管理器
func (s *Server) initSerialManager() error {
	s.logger.Info("初始化串口管理器...")

	// 检查串口配置
	if !s.cfg.Serial.Enabled {
		s.logger.Info("串口功能未启用，跳过初始化")
		return nil
	}

	// 检查是否为调试模式（使用模拟控制器）
	if s.cfg.Serial.MockMode {
		s.logger.Info("🔧 调试模式：使用模拟控制器")
		s.serialController = hardware.NewMockController(nil)
		if err := s.serialController.Connect(); err != nil {
			return errors.Wrap(err, errors.ErrUnknown, "模拟控制器连接失败")
		}
		s.setupSerialCallbacks()
		s.logger.Info("模拟控制器已启动")
		return nil
	}

	// 检查是否有任何串口启用
	if !s.cfg.Serial.STM32.Enabled && !s.cfg.Serial.ACM.Enabled {
		s.logger.Info("没有串口被启用")
		return nil
	}

	// 为了保持与现有代码的兼容性，使用单个串口控制器
	// 如果STM32启用，使用STM32控制器
	if s.cfg.Serial.STM32.Enabled {
		s.logger.Info("STM32串口功能已启用",
			zap.String("port", s.cfg.Serial.STM32.Port),
			zap.Int("baud_rate", s.cfg.Serial.STM32.BaudRate))
		
		stm32Config := &hardware.STM32Config{
			Port:              s.cfg.Serial.STM32.Port,
			BaudRate:          s.cfg.Serial.STM32.BaudRate,
			DataBits:          s.cfg.Serial.STM32.DataBits,
			StopBits:          s.cfg.Serial.STM32.StopBits,
			ReadTimeout:       s.cfg.Serial.STM32.ReadTimeout,
			WriteTimeout:      s.cfg.Serial.STM32.WriteTimeout,
			HeartbeatInterval: s.cfg.Serial.STM32.HeartbeatInterval,
			RetryCount:        s.cfg.Serial.STM32.RetryTimes,
		}
		
		s.serialController = hardware.NewSTM32Controller(stm32Config, nil)
	} else if s.cfg.Serial.ACM.Enabled {
		// 如果只有ACM启用，使用ACM控制器
		s.logger.Info("ACM串口功能已启用",
			zap.String("port", s.cfg.Serial.ACM.Port),
			zap.Int("baud_rate", s.cfg.Serial.ACM.BaudRate),
			zap.Bool("auto_detect", s.cfg.Serial.ACM.AutoDetect))
		
		acmConfig := &hardware.ACMConfig{
			Port:         s.cfg.Serial.ACM.Port,
			BaudRate:     s.cfg.Serial.ACM.BaudRate,
			ReadTimeout:  s.cfg.Serial.ACM.ReadTimeout,
			WriteTimeout: s.cfg.Serial.ACM.WriteTimeout,
			AutoDetect:   s.cfg.Serial.ACM.AutoDetect,
		}
		
		s.serialController = hardware.NewACMController(acmConfig)
	}

	// TODO: 如果需要同时使用两个串口，后续需要实现硬件管理器的完整功能

	// 设置事件回调
	s.setupSerialCallbacks()
	
	// 启动串口连接协程（带重试）
	s.wg.Add(1)
	go s.serialConnectWithRetry()
	
	s.logger.Info("硬件管理器初始化完成，后台重试连接中...")
	
	return nil
}

// serialConnectWithRetry 串口连接重试逻辑
func (s *Server) serialConnectWithRetry() {
	defer s.wg.Done()
	
	retryInterval := 5 * time.Second // 初始重试间隔
	maxRetryInterval := 60 * time.Second // 最大重试间隔
	retryCount := 0
	
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("停止串口连接重试")
			return
		default:
			// 检查是否已连接
			if s.serialController != nil && s.serialController.IsConnected() {
				// 已连接，定期检查连接状态
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(10 * time.Second):
					// 检查连接是否断开
					if !s.serialController.IsConnected() {
						s.logger.Warn("串口连接已断开，尝试重新连接")
					}
					continue
				}
			}
			
			// 构建端口信息
			var ports []string
			if s.cfg.Serial.STM32.Enabled {
			 ports = append(ports, fmt.Sprintf("STM32:%s", s.cfg.Serial.STM32.Port))
			}
			if s.cfg.Serial.ACM.Enabled {
			 ports = append(ports, fmt.Sprintf("ACM:%s", s.cfg.Serial.ACM.Port))
			}
			
			// 尝试连接串口
			s.logger.Info("尝试连接硬件设备...", 
			zap.Strings("ports", ports),
				zap.Int("retry_count", retryCount))
			
			if err := s.serialController.Connect(); err != nil {
				retryCount++
				s.logger.Error("硬件连接失败，稍后重试",
					zap.Strings("ports", ports),
					zap.Error(err),
					zap.Int("retry_count", retryCount),
					zap.Duration("next_retry_in", retryInterval))
				
				// 等待重试
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(retryInterval):
					// 递增重试间隔（指数退避）
					if retryInterval < maxRetryInterval {
						retryInterval = time.Duration(float64(retryInterval) * 1.5)
						if retryInterval > maxRetryInterval {
							retryInterval = maxRetryInterval
						}
					}
				}
			} else {
			// 连接成功
			s.logger.Info("🎉 硬件连接成功！",
			  zap.Strings("ports", ports),
			 zap.Bool("connected", s.serialController.IsConnected()),
			 zap.Int("total_retries", retryCount))
				
				// 重置重试参数
				retryCount = 0
				retryInterval = 5 * time.Second
				
				// TODO: 后续集成游戏服务时发布连接成功事件
			}
		}
	}
}

// setupSerialCallbacks 设置串口事件回调
func (s *Server) setupSerialCallbacks() {
	if s.serialController == nil {
		return
	}
	
	// 投币检测回调
	s.serialController.SetCoinInsertedCallback(func(count byte) {
		s.logger.Info("💰 投币检测", zap.Uint8("count", count))
		// TODO: 后续集成游戏服务时处理投币事件
	})
	
	// 回币检测回调
	s.serialController.SetCoinReturnedCallback(func(data *hardware.CoinReturnData) {
		s.logger.Info("🪙 回币检测", 
			zap.Uint8("front", data.FrontCount),
			zap.Uint8("left", data.LeftCount),
			zap.Uint8("right", data.RightCount))
		// TODO: 后续集成游戏服务时处理回币事件
	})
	
	// 按键事件回调
	s.serialController.SetButtonPressedCallback(func(event *hardware.ButtonEvent) {
		s.logger.Info("🔘 按键事件",
			zap.Uint8("type", event.KeyType),
			zap.Uint8("code", event.KeyCode),
			zap.Uint8("action", event.Action))
		// TODO: 后续集成游戏服务时处理按键事件
	})
	
	// 故障报告回调
	s.serialController.SetFaultReportCallback(func(event *hardware.FaultEvent) {
		s.logger.Error("⚠️ 硬件故障",
			zap.Uint8("code", event.FaultCode),
			zap.Uint8("level", event.Level))
		// TODO: 后续集成游戏服务时处理故障事件
	})
}

// startServices 启动服务
func (s *Server) startServices() error {
	s.logger.Info("启动服务...")
	
	// 启动HTTP服务器
	s.wg.Add(1)
	go s.startHTTPServer()
	
	// TODO: 启动WebSocket服务器
	// s.wg.Add(1)
	// go s.startWebSocketServer()
	
	// TODO: 启动串口通信
	// s.wg.Add(1)
	// go s.startSerialCommunication()
	
	// TODO: 启动MQTT客户端
	// if s.cfg.MQTT.Enabled {
	//     s.wg.Add(1)
	//     go s.startMQTTClient()
	// }
	
	// TODO: 启动监控服务
	// if s.cfg.Monitor.Enabled {
	//     s.wg.Add(1)
	//     go s.startMonitor()
	// }
	
	s.logger.Info("所有服务启动完成")
	return nil
}

// WaitForShutdown 等待关闭信号
func (s *Server) WaitForShutdown() {
	// 创建信号通道
	sigCh := make(chan os.Signal, 1)
	
	// 监听系统信号
	signal.Notify(sigCh,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // kill命令
		syscall.SIGQUIT, // Ctrl+\
	)
	
	// 等待信号
	sig := <-sigCh
	s.logger.Info("收到退出信号", zap.String("signal", sig.String()))
	
	// 发送关闭信号
	close(s.shutdownCh)
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown() error {
	s.logger.Info("正在优雅关闭服务器...")
	
	// 创建超时上下文
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer cancel()
	
	// 停止接收新请求
	s.logger.Info("停止接收新请求...")
	
	// 取消主上下文，触发所有goroutine退出
	s.cancel()
	
	// 等待所有服务关闭
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	
	// 等待关闭完成或超时
	select {
	case <-done:
		s.logger.Info("所有服务已正常关闭")
	case <-shutdownCtx.Done():
		s.logger.Warn("关闭超时，强制退出")
		return errors.New(errors.ErrTimeout, "关闭超时")
	}
	
	// 关闭各个组件
	if err := s.closeComponents(); err != nil {
		s.logger.Error("关闭组件失败", zap.Error(err))
		return err
	}
	
	// 同步日志
	if err := logger.Sync(); err != nil {
		fmt.Printf("同步日志失败: %v\n", err)
	}
	
	return nil
}

// closeComponents 关闭组件
func (s *Server) closeComponents() error {
	s.logger.Info("关闭组件...")
	
	// 停止定时清理任务
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
		s.logger.Info("定时清理任务已停止")
	}
	
	// 关闭串口连接
	if s.serialController != nil {
		if err := s.serialController.Disconnect(); err != nil {
			s.logger.Error("关闭串口失败", zap.Error(err))
		} else {
			s.logger.Info("串口连接已关闭")
		}
	}
	
	// 关闭数据库连接
	if err := database.Close(); err != nil {
		s.logger.Error("关闭数据库失败", zap.Error(err))
	}
	
	// TODO: 关闭MQTT连接
	// if s.mqttClient != nil {
	//     s.mqttClient.Disconnect(1000)
	// }
	
	// TODO: 关闭WebSocket服务器
	// if s.wsServer != nil {
	//     s.wsServer.Close()
	// }
	
	s.logger.Info("所有组件已关闭")
	return nil
}

// startHTTPServer 启动HTTP服务器
func (s *Server) startHTTPServer() {
	defer s.wg.Done()
	
	s.logger.Info("启动HTTP服务器",
		zap.String("address", s.httpServer.Addr),
	)
	
	// 启动服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()
	
	// 等待关闭信号
	<-s.ctx.Done()
	
	// 优雅关闭HTTP服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("HTTP服务器关闭失败", zap.Error(err))
	}
	
	s.logger.Info("HTTP服务器已关闭")
}

// reloadConfig 重新加载配置
func (s *Server) reloadConfig(newCfg *config.Config) {
	s.cfg = newCfg
	
	// TODO: 应用新配置到各个组件
	// 例如：更新日志级别、调整游戏参数等
	
	s.logger.Info("配置重新加载完成")
}

// setupSystem 设置系统参数
func setupSystem(cfg *config.SystemConfig) {
	// 设置时区
	if cfg.Timezone != "" {
		if loc, err := time.LoadLocation(cfg.Timezone); err == nil {
			time.Local = loc
		}
	}
	
	// 设置最大处理器数
	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}
	
	// 设置文件描述符限制（Unix系统）
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err == nil {
		rLimit.Cur = rLimit.Max
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	}
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("推币机游戏服务器\n")
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("构建时间: %s\n", BuildTime)
	fmt.Printf("Git提交: %s\n", GitCommit)
	fmt.Printf("Go版本: %s\n", runtime.Version())
	fmt.Printf("操作系统: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("推币机游戏服务器")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  slot-game-server [选项]")
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("环境变量:")
	fmt.Println("  SLOT_GAME_ENV          运行环境 (development/production/test)")
	fmt.Println("  SLOT_GAME_CONFIG       配置文件路径")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  slot-game-server -config=/path/to/config.yaml")
	fmt.Println("  slot-game-server -version")
}

// printStartInfo 打印启动信息
func printStartInfo(cfg *config.Config) {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║     _____ _       _      _____                               ║
║    /  ___| |     | |    |  __ \                              ║
║    \ ` + "`" + `--.| | ___ | |_   | |  \/ __ _ _ __ ___   ___          ║
║     ` + "`" + `--. \ |/ _ \| __|  | | __ / _` + "`" + ` | '_ ` + "`" + ` _ \ / _ \         ║
║    /\__/ / | (_) | |_   | |_\ \ (_| | | | | | |  __/         ║
║    \____/|_|\___/ \__|   \____/\__,_|_| |_| |_|\___|         ║
║                                                               ║
║                   推币机游戏后端服务器                        ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	fmt.Printf("版本: %s | 模式: %s | PID: %d\n", Version, cfg.Server.Mode, os.Getpid())
	fmt.Printf("配置文件: %s\n", config.GetString("config_file"))
	fmt.Println("═══════════════════════════════════════════════════════════════")
}