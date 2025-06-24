package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/middleware/cors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		CORS: config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         86400,
		},
	}

	// 創建測試路由
	router := gin.New()
	router.Use(cors.Middleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		CORS: config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         86400,
		},
	}

	// 創建測試路由
	router := gin.New()
	router.Use(cors.Middleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious-site.com")
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證結果
	// gin-contrib/cors 對於不允許的來源會返回 403 錯誤
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 創建測試配置（禁用 CORS）
	cfg := &config.Config{
		CORS: config.CORSConfig{
			Enabled:        false,
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         86400,
		},
	}

	// 創建測試路由
	router := gin.New()
	router.Use(cors.Middleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	// 當 CORS 被禁用時，gin-contrib/cors 仍然會設置 CORS 標頭
	// 因為它會使用空配置，但對於允許的來源仍會設置標頭
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}
