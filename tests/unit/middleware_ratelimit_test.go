package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/middleware/ratelimit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRateLimitMiddleware_GlobalLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			Enabled:     true,
			GlobalLimit: 2, // 每分鐘最多 2 個請求
			IPLimit: config.RateLimitRule{
				Requests: 10,
				Window:   time.Minute,
			},
			UserLimit: config.RateLimitRule{
				Requests: 20,
				Window:   time.Minute,
			},
		},
	}

	// 創建限流中間件
	logger := zap.NewNop()
	middleware := ratelimit.NewRateLimitMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(middleware.GlobalRateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 測試前兩個請求應該成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "前兩個請求應該成功")
	}

	// 第三個請求應該被限流
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第三個請求應該被限流")
}

func TestRateLimitMiddleware_IPLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			Enabled:     true,
			GlobalLimit: 1000,
			IPLimit: config.RateLimitRule{
				Requests: 3, // 每個 IP 每分鐘最多 3 個請求
				Window:   time.Minute,
			},
		},
	}

	// 創建限流中間件
	logger := zap.NewNop()
	middleware := ratelimit.NewRateLimitMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(middleware.IPRateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 測試前三個請求應該成功
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345" // 設置固定 IP
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "前三個請求應該成功")
	}

	// 第四個請求應該被限流
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第四個請求應該被限流")
}

func TestRateLimitMiddleware_UserLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			Enabled:     true,
			GlobalLimit: 1000,
			UserLimit: config.RateLimitRule{
				Requests: 2, // 每個用戶每分鐘最多 2 個請求
				Window:   time.Minute,
			},
		},
	}

	// 創建限流中間件
	logger := zap.NewNop()
	middleware := ratelimit.NewRateLimitMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(middleware.UserRateLimit())
	router.GET("/test", func(c *gin.Context) {
		// 模擬用戶認證，設置用戶 ID
		c.Set("user_id", "user123")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 測試前兩個請求應該成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "前兩個請求應該成功")
	}

	// 第三個請求應該被限流
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第三個請求應該被限流")
}

func TestRateLimitMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置（禁用限流）
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			Enabled: false,
		},
	}

	// 創建限流中間件
	logger := zap.NewNop()
	middleware := ratelimit.NewRateLimitMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(middleware.GlobalRateLimit())
	router.Use(middleware.IPRateLimit())
	router.Use(middleware.UserRateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 即使發送多個請求，也應該都成功（因為限流被禁用）
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "當限流被禁用時，所有請求都應該成功")
	}
}

func TestRateLimitMiddleware_Reset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			Enabled:     true,
			GlobalLimit: 1, // 每分鐘最多 1 個請求
		},
	}

	// 創建限流中間件
	logger := zap.NewNop()
	middleware := ratelimit.NewRateLimitMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(middleware.GlobalRateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 第一個請求應該成功
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "第一個請求應該成功")

	// 第二個請求應該被限流
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "第二個請求應該被限流")

	// 重置限流器
	middleware.ResetRateLimit("global")

	// 重置後的第一個請求應該成功
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "重置後的第一個請求應該成功")
}
