package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIEndpoints 测试API端点的基本功能
func TestAPIEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("健康检查", func(t *testing.T) {
		router := gin.New()
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"message": "服务运行正常",
			})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "healthy", resp["status"])
	})

	t.Run("请求验证", func(t *testing.T) {
		router := gin.New()
		
		type LoginRequest struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required,min=6"`
		}
		
		router.POST("/login", func(c *gin.Context) {
			var req LoginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "登录成功",
			})
		})

		// 测试缺少必需字段
		body, _ := json.Marshal(map[string]string{
			"username": "testuser",
			// 缺少password
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 测试密码太短
		body, _ = json.Marshal(map[string]string{
			"username": "testuser",
			"password": "123", // 太短
		})
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// 测试有效请求
		body, _ = json.Marshal(map[string]string{
			"username": "testuser",
			"password": "123456",
		})
		w = httptest.NewRecorder()
		req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("分页参数", func(t *testing.T) {
		router := gin.New()
		
		router.GET("/list", func(c *gin.Context) {
			page := c.DefaultQuery("page", "1")
			pageSize := c.DefaultQuery("page_size", "10")
			
			c.JSON(http.StatusOK, gin.H{
				"page": page,
				"page_size": pageSize,
				"data": []string{"item1", "item2"},
			})
		})

		// 测试默认分页
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/list", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "1", resp["page"])
		assert.Equal(t, "10", resp["page_size"])

		// 测试自定义分页
		w = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/list?page=2&page_size=20", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "2", resp["page"])
		assert.Equal(t, "20", resp["page_size"])
	})

	t.Run("错误处理", func(t *testing.T) {
		router := gin.New()
		
		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": "INTERNAL_ERROR",
				"message": "服务器内部错误",
			})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/error", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "INTERNAL_ERROR", resp["code"])
	})
}

// TestAPIResponseFormat 测试API响应格式
func TestAPIResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 标准响应格式
	type Response struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	}
	
	router := gin.New()
	
	// 成功响应
	router.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{
			Code:    0,
			Message: "操作成功",
			Data: map[string]interface{}{
				"id":   1,
				"name": "测试数据",
			},
		})
	})
	
	// 错误响应
	router.GET("/failure", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, Response{
			Code:    40001,
			Message: "参数错误",
		})
	})
	
	t.Run("成功响应格式", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/success", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "操作成功", resp.Message)
		assert.NotNil(t, resp.Data)
	})
	
	t.Run("错误响应格式", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/failure", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		
		assert.Equal(t, 40001, resp.Code)
		assert.Equal(t, "参数错误", resp.Message)
		assert.Nil(t, resp.Data)
	})
}

// TestHTTPMethods 测试不同的HTTP方法
func TestHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 设置不同方法的路由
	router.GET("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})
	router.PUT("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT"})
	})
	router.DELETE("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "DELETE"})
	})
	
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/resource", nil)
			router.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			
			var resp map[string]string
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, method, resp["method"])
		})
	}
	
	// 测试不支持的方法
	t.Run("不支持的方法", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PATCH", "/resource", nil)
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestContentType 测试内容类型处理
func TestContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.POST("/json", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		c.JSON(http.StatusOK, data)
	})
	
	router.POST("/form", func(c *gin.Context) {
		name := c.PostForm("name")
		age := c.PostForm("age")
		c.JSON(http.StatusOK, gin.H{
			"name": name,
			"age":  age,
		})
	})
	
	t.Run("JSON内容", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "测试",
			"type": "json",
		})
		
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/json", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "测试", resp["name"])
		assert.Equal(t, "json", resp["type"])
	})
	
	t.Run("表单内容", func(t *testing.T) {
		body := bytes.NewBufferString("name=张三&age=25")
		
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/form", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "张三", resp["name"])
		assert.Equal(t, "25", resp["age"])
	})
	
	t.Run("错误的内容类型", func(t *testing.T) {
		body := bytes.NewBufferString("invalid json")
		
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/json", body)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Invalid JSON", resp["error"])
	})
}