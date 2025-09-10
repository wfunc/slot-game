//go:build !swagger

package api

import "github.com/gin-gonic/gin"

// registerSwaggerRoutes 是空实现，用于非 swagger 构建
func registerSwaggerRoutes(engine *gin.Engine) {}

