//go:build swagger

package api

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

// registerSwaggerRoutes 注册 Swagger 文档路由（仅在 -tags swagger 时启用）
func registerSwaggerRoutes(engine *gin.Engine) {
    // 使用 OpenAPI 路径作为文档数据源，避免强依赖本地生成包
    engine.GET("/swagger/*any", ginSwagger.WrapHandler(
        swaggerFiles.Handler,
        ginSwagger.URL("/openapi"), // 直接使用 /openapi（YAML / OpenAPI 3）
        ginSwagger.DocExpansion("none"),
    ))
}
