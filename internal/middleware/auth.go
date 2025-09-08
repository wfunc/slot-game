package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wfunc/slot-game/internal/service"
)

// AuthMiddleware JWT认证中间件
type AuthMiddleware struct {
	authService service.AuthService
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(authService service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth 需要认证的中间件
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "NO_TOKEN",
				"message": "缺少认证令牌",
			})
			c.Abort()
			return
		}
		
		// 验证令牌
		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_TOKEN",
				"message": "无效的令牌",
				"details": err.Error(),
			})
			c.Abort()
			return
		}
		
		// 将用户信息存入上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("sessionID", claims.SessionID)
		c.Set("token", token)
		
		c.Next()
	}
}

// OptionalAuth 可选认证的中间件（不强制要求登录）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token != "" {
			// 尝试验证令牌
			claims, err := m.authService.ValidateToken(c.Request.Context(), token)
			if err == nil {
				// 令牌有效，设置用户信息
				c.Set("userID", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("email", claims.Email)
				c.Set("role", claims.Role)
				c.Set("sessionID", claims.SessionID)
				c.Set("token", token)
			}
		}
		
		c.Next()
	}
}

// RequireRole 需要特定角色的中间件
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先确保已认证
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "NO_TOKEN",
				"message": "缺少认证令牌",
			})
			c.Abort()
			return
		}
		
		// 验证令牌
		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    "INVALID_TOKEN",
				"message": "无效的令牌",
				"details": err.Error(),
			})
			c.Abort()
			return
		}
		
		// 检查角色
		hasRole := false
		for _, role := range roles {
			if claims.Role == role {
				hasRole = true
				break
			}
		}
		
		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    "INSUFFICIENT_PERMISSION",
				"message": "权限不足",
			})
			c.Abort()
			return
		}
		
		// 设置用户信息
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("sessionID", claims.SessionID)
		c.Set("token", token)
		
		c.Next()
	}
}

// extractToken 从请求中提取令牌
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	// 1. 从Authorization Header获取 (Bearer Token)
	bearerToken := c.GetHeader("Authorization")
	if bearerToken != "" {
		parts := strings.Split(bearerToken, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}
	
	// 2. 从X-Access-Token Header获取
	if token := c.GetHeader("X-Access-Token"); token != "" {
		return token
	}
	
	// 3. 从Cookie获取
	if token, err := c.Cookie("access_token"); err == nil && token != "" {
		return token
	}
	
	// 4. 从Query参数获取（不推荐用于生产环境）
	if token := c.Query("token"); token != "" {
		return token
	}
	
	return ""
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(uint); ok {
			return id, true
		}
	}
	return 0, false
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, bool) {
	if username, exists := c.Get("username"); exists {
		if name, ok := username.(string); ok {
			return name, true
		}
	}
	return "", false
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(c *gin.Context) (string, bool) {
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(string); ok {
			return r, true
		}
	}
	return "", false
}

// GetSessionID 从上下文获取会话ID
func GetSessionID(c *gin.Context) (string, bool) {
	if sessionID, exists := c.Get("sessionID"); exists {
		if id, ok := sessionID.(string); ok {
			return id, true
		}
	}
	return "", false
}

// IsAuthenticated 检查是否已认证
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("userID")
	return exists
}

// HasRole 检查是否有特定角色
func HasRole(c *gin.Context, role string) bool {
	if userRole, exists := GetUserRole(c); exists {
		return userRole == role
	}
	return false
}

// HasAnyRole 检查是否有任一角色
func HasAnyRole(c *gin.Context, roles ...string) bool {
	if userRole, exists := GetUserRole(c); exists {
		for _, role := range roles {
			if userRole == role {
				return true
			}
		}
	}
	return false
}