package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wfunc/slot-game/internal/models"
	"github.com/wfunc/slot-game/internal/service"
)

// SerialLogAPI 串口日志API
type SerialLogAPI struct {
	service *service.SerialLogService
}

// NewSerialLogAPI 创建串口日志API
func NewSerialLogAPI(service *service.SerialLogService) *SerialLogAPI {
	return &SerialLogAPI{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (api *SerialLogAPI) RegisterRoutes(router *gin.RouterGroup) {
	logs := router.Group("/serial-logs")
	{
		logs.GET("", api.QueryLogs)           // 查询日志列表
		logs.GET("/latest", api.GetLatestLogs) // 获取最新日志
		logs.GET("/stats", api.GetStats)       // 获取统计信息
		logs.GET("/algo", api.GetAlgoLogs)     // 获取algo命令日志
		logs.GET("/errors", api.GetErrorLogs)  // 获取错误日志
		logs.POST("/cleanup", api.CleanupLogs) // 清理旧日志
		logs.GET("/export", api.ExportLogs)    // 导出日志
	}
}

// QueryLogs 查询日志列表
func (api *SerialLogAPI) QueryLogs(c *gin.Context) {
	query := &models.SerialLogQuery{}

	// 解析查询参数
	if deviceType := c.Query("device_type"); deviceType != "" {
		query.DeviceType = models.SerialLogType(deviceType)
	}
	query.Direction = c.Query("direction")
	if level := c.Query("level"); level != "" {
		query.Level = models.SerialLogLevel(level)
	}
	query.Command = c.Query("command")
	query.Function = c.Query("function")
	query.RequestID = c.Query("request_id")
	query.SessionID = c.Query("session_id")

	// 时间范围
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query.EndTime = &t
		}
	}

	// 金额范围
	if minBet := c.Query("min_bet"); minBet != "" {
		if v, err := strconv.ParseFloat(minBet, 64); err == nil {
			query.MinBet = &v
		}
	}
	if maxBet := c.Query("max_bet"); maxBet != "" {
		if v, err := strconv.ParseFloat(maxBet, 64); err == nil {
			query.MaxBet = &v
		}
	}
	if minWin := c.Query("min_win"); minWin != "" {
		if v, err := strconv.ParseFloat(minWin, 64); err == nil {
			query.MinWin = &v
		}
	}
	if maxWin := c.Query("max_win"); maxWin != "" {
		if v, err := strconv.ParseFloat(maxWin, 64); err == nil {
			query.MaxWin = &v
		}
	}

	// 是否有错误
	if hasError := c.Query("has_error"); hasError == "true" {
		b := true
		query.HasError = &b
	}

	// 分页参数
	query.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	query.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	query.OrderBy = c.DefaultQuery("order_by", "created_at DESC")

	// 查询日志
	logs, total, err := api.service.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "查询失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"total": total,
		"limit": query.Limit,
		"offset": query.Offset,
	})
}

// GetLatestLogs 获取最新日志
func (api *SerialLogAPI) GetLatestLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	deviceType := models.SerialLogType(c.Query("device_type"))

	logs, err := api.service.GetLatestLogs(limit, deviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"count": len(logs),
	})
}

// GetStats 获取统计信息
func (api *SerialLogAPI) GetStats(c *gin.Context) {
	var startTime, endTime *time.Time

	// 解析时间范围
	if start := c.Query("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = &t
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = &t
		}
	}

	stats, err := api.service.GetStats(startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取统计失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAlgoLogs 获取algo命令日志
func (api *SerialLogAPI) GetAlgoLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var startTime, endTime *time.Time

	// 解析时间范围
	if start := c.Query("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = &t
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = &t
		}
	}

	logs, err := api.service.GetAlgoCommandLogs(startTime, endTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取algo日志失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"count": len(logs),
	})
}

// GetErrorLogs 获取错误日志
func (api *SerialLogAPI) GetErrorLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	logs, err := api.service.GetErrorLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取错误日志失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"count": len(logs),
	})
}

// CleanupLogs 清理旧日志
func (api *SerialLogAPI) CleanupLogs(c *gin.Context) {
	// 需要管理员权限
	// TODO: 添加权限检查

	// 获取保留天数
	retentionDays, _ := strconv.Atoi(c.DefaultPostForm("retention_days", "30"))
	if retentionDays < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "保留天数必须大于0",
		})
		return
	}

	count, err := api.service.CleanupOldLogs(retentionDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "清理失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "清理成功",
		"deleted": count,
		"retention_days": retentionDays,
	})
}

// ExportLogs 导出日志
func (api *SerialLogAPI) ExportLogs(c *gin.Context) {
	query := &models.SerialLogQuery{}

	// 解析查询参数（与QueryLogs相同）
	if deviceType := c.Query("device_type"); deviceType != "" {
		query.DeviceType = models.SerialLogType(deviceType)
	}
	query.Direction = c.Query("direction")
	query.Function = c.Query("function")
	query.RequestID = c.Query("request_id")
	query.SessionID = c.Query("session_id")

	// 时间范围
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query.EndTime = &t
		}
	}

	// 导出限制
	query.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "1000"))

	// 导出日志
	data, err := api.service.ExportLogs(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "导出失败",
			"message": err.Error(),
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=serial_logs_export.json")
	c.Data(http.StatusOK, "application/json", data)
}