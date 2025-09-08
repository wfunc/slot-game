package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wfunc/slot-game/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
	userService service.UserService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService service.AuthService, userService service.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "注册信息"
// @Success 200 {object} service.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "请求参数错误",
			Details: err.Error(),
		})
		return
	}
	
	// 获取客户端IP
	req.IP = c.ClientIP()
	
	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "REGISTER_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名/邮箱/手机号登录
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "登录信息"
// @Success 200 {object} service.AuthResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "请求参数错误",
			Details: err.Error(),
		})
		return
	}
	
	// 获取客户端信息
	req.IP = c.ClientIP()
	req.Device = c.GetHeader("User-Agent")
	
	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusUnauthorized
		if err == service.ErrUserBanned {
			status = http.StatusForbidden
		}
		
		c.JSON(status, ErrorResponse{
			Code:    "LOGIN_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 退出登录并清除会话
// @Tags Auth
// @Security Bearer
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "未登录",
		})
		return
	}
	
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "NO_TOKEN",
			Message: "缺少令牌",
		})
		return
	}
	
	err := h.authService.Logout(c.Request.Context(), userID.(uint), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "LOGOUT_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "登出成功",
	})
}

// RefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "刷新令牌"
// @Success 200 {object} service.AuthResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "请求参数错误",
			Details: err.Error(),
		})
		return
	}
	
	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "REFRESH_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

// GetProfile 获取用户资料
// @Summary 获取当前用户资料
// @Description 获取登录用户的详细信息
// @Tags Auth
// @Security Bearer
// @Success 200 {object} UserProfileResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "未登录",
		})
		return
	}
	
	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "用户不存在",
		})
		return
	}
	
	// 获取用户统计
	stats, _ := h.userService.GetUserStats(c.Request.Context(), userID.(uint))
	
	c.JSON(http.StatusOK, UserProfileResponse{
		User:  user,
		Stats: stats,
	})
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前用户的个人信息
// @Tags Auth
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body service.UserProfile true "用户资料"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "未登录",
		})
		return
	}
	
	var profile service.UserProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "请求参数错误",
			Details: err.Error(),
		})
		return
	}
	
	err := h.userService.UpdateProfile(c.Request.Context(), userID.(uint), &profile)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "更新成功",
	})
}

// UpdatePassword 更新密码
// @Summary 修改密码
// @Description 修改当前用户的登录密码
// @Tags Auth
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body UpdatePasswordRequest true "密码信息"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/auth/password [put]
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "未登录",
		})
		return
	}
	
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "请求参数错误",
			Details: err.Error(),
		})
		return
	}
	
	err := h.userService.UpdatePassword(c.Request.Context(), userID.(uint), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "UPDATE_FAILED",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "密码修改成功",
	})
}

// extractToken 从请求中提取令牌
func extractToken(c *gin.Context) string {
	// 从Header中获取
	bearerToken := c.GetHeader("Authorization")
	if bearerToken != "" {
		parts := strings.Split(bearerToken, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	
	// 从Query参数中获取
	return c.Query("token")
}

// 请求和响应结构体

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdatePasswordRequest 更新密码请求
type UpdatePasswordRequest struct {
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// UserProfileResponse 用户资料响应
type UserProfileResponse struct {
	User  interface{} `json:"user"`
	Stats interface{} `json:"stats,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}