package api

import (
	"github.com/gin-gonic/gin"
	"github.com/wfunc/slot-game/internal/middleware"
	"github.com/wfunc/slot-game/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Router API路由器
type Router struct {
	engine      *gin.Engine
	db          *gorm.DB
	services    *service.Services
	authHandler *AuthHandler
	authMiddleware *middleware.AuthMiddleware
	log         *zap.Logger
}

// NewRouter 创建路由器
func NewRouter(db *gorm.DB, config *service.Config, log *zap.Logger) *Router {
	// 创建Gin引擎
	engine := gin.New()
	
	// 全局中间件
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	
	// 创建服务
	services := service.NewServices(db, config, log)
	
	// 创建处理器
	authHandler := NewAuthHandler(services.Auth, services.User)
	
	// 创建中间件
	authMiddleware := middleware.NewAuthMiddleware(services.Auth)
	
	router := &Router{
		engine:         engine,
		db:             db,
		services:       services,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
		log:            log,
	}
	
	// 设置路由
	router.setupRoutes()
	
	return router
}

// setupRoutes 设置路由
func (r *Router) setupRoutes() {
	// 健康检查
	r.engine.GET("/health", r.healthCheck)
	
	// API v1路由组
	v1 := r.engine.Group("/api/v1")
	{
		// 认证相关路由（不需要认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.RefreshToken)
			
			// 需要认证的路由
			authRequired := auth.Group("")
			authRequired.Use(r.authMiddleware.RequireAuth())
			{
				authRequired.POST("/logout", r.authHandler.Logout)
				authRequired.GET("/profile", r.authHandler.GetProfile)
				authRequired.PUT("/profile", r.authHandler.UpdateProfile)
				authRequired.PUT("/password", r.authHandler.UpdatePassword)
			}
		}
		
		// 用户相关路由（需要认证）
		users := v1.Group("/users")
		users.Use(r.authMiddleware.RequireAuth())
		{
			// TODO: 实现用户管理API
			// users.GET("", r.userHandler.GetUsers)
			// users.GET("/:id", r.userHandler.GetUser)
			// users.PUT("/:id", r.userHandler.UpdateUser)
			// users.DELETE("/:id", r.userHandler.DeleteUser)
		}
		
		// 游戏相关路由（需要认证）
		games := v1.Group("/games")
		games.Use(r.authMiddleware.RequireAuth())
		{
			// TODO: 实现游戏API
			// games.GET("", r.gameHandler.GetGames)
			// games.GET("/:id", r.gameHandler.GetGame)
			// games.POST("/:id/play", r.gameHandler.PlayGame)
		}
		
		// 老虎机游戏路由
		slot := v1.Group("/slot")
		slot.Use(r.authMiddleware.RequireAuth())
		{
			// TODO: 实现老虎机API
			// slot.GET("/machines", r.slotHandler.GetMachines)
			// slot.POST("/spin", r.slotHandler.Spin)
			// slot.GET("/history", r.slotHandler.GetHistory)
		}
		
		// 推币机游戏路由
		pusher := v1.Group("/pusher")
		pusher.Use(r.authMiddleware.RequireAuth())
		{
			// TODO: 实现推币机API
			// pusher.GET("/machines", r.pusherHandler.GetMachines)
			// pusher.POST("/push", r.pusherHandler.Push)
			// pusher.GET("/drops", r.pusherHandler.GetDrops)
		}
		
		// 钱包相关路由（需要认证）
		wallet := v1.Group("/wallet")
		wallet.Use(r.authMiddleware.RequireAuth())
		{
			// TODO: 实现钱包API
			// wallet.GET("/balance", r.walletHandler.GetBalance)
			// wallet.POST("/deposit", r.walletHandler.Deposit)
			// wallet.POST("/withdraw", r.walletHandler.Withdraw)
			// wallet.GET("/transactions", r.walletHandler.GetTransactions)
		}
		
		// 管理员路由（需要管理员权限）
		admin := v1.Group("/admin")
		admin.Use(r.authMiddleware.RequireRole("admin"))
		{
			// TODO: 实现管理员API
			// admin.GET("/users", r.adminHandler.GetUsers)
			// admin.PUT("/users/:id/status", r.adminHandler.UpdateUserStatus)
			// admin.GET("/stats", r.adminHandler.GetStatistics)
			// admin.GET("/logs", r.adminHandler.GetLogs)
		}
	}
	
	// WebSocket路由
	ws := r.engine.Group("/ws")
	ws.Use(r.authMiddleware.RequireAuth())
	{
		// TODO: 实现WebSocket
		// ws.GET("/game", r.wsHandler.GameWebSocket)
		// ws.GET("/chat", r.wsHandler.ChatWebSocket)
	}
	
	// 静态文件服务
	r.engine.Static("/static", "./static")
	
	// 404处理
	r.engine.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"code":    "NOT_FOUND",
			"message": "接口不存在",
		})
	})
}

// healthCheck 健康检查
func (r *Router) healthCheck(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := r.db.DB()
	if err != nil {
		c.JSON(500, gin.H{
			"status":  "unhealthy",
			"message": "数据库连接失败",
		})
		return
	}
	
	if err := sqlDB.Ping(); err != nil {
		c.JSON(500, gin.H{
			"status":  "unhealthy",
			"message": "数据库ping失败",
		})
		return
	}
	
	c.JSON(200, gin.H{
		"status":  "healthy",
		"message": "服务运行正常",
	})
}

// Run 运行服务器
func (r *Router) Run(addr string) error {
	r.log.Info("Starting API server", zap.String("address", addr))
	return r.engine.Run(addr)
}

// GetEngine 获取Gin引擎（用于测试）
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}