package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/domain"
	"expense-api-gateway/internal/infrastructure/jwt"
	"expense-api-gateway/internal/middleware/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestJWTMiddleware_ValidToken(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-very-long-for-testing",
			Expiration:        1 * time.Hour,
			RefreshExpiration: 24 * time.Hour,
		},
	}

	// 創建 JWT 服務
	jwtService := jwt.NewJWTService(cfg, zap.NewNop())
	logger, _ := zap.NewDevelopment()

	// 創建測試用戶
	testUser := &domain.AuthUser{
		ID:        "1",
		Email:     "test@example.com",
		Role:      "user",
		CompanyID: "1",
		Roles:     []string{"user"},
	}

	// 生成測試 Token
	tokenPair, err := jwtService.GenerateToken(testUser)
	assert.NoError(t, err)

	// 設置路由
	r := gin.New()
	jwtMiddleware := auth.NewJWTMiddleware(cfg, logger)
	r.Use(jwtMiddleware.Authenticate())
	r.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		assert.True(t, exists)
		assert.Equal(t, testUser.ID, userID)

		companyID, exists := c.Get("company_id")
		assert.True(t, exists)
		assert.Equal(t, testUser.CompanyID, companyID)

		userRole, exists := c.Get("user_role")
		assert.True(t, exists)
		assert.Equal(t, testUser.Role, userRole)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-very-long-for-testing",
			Expiration:        1 * time.Hour,
			RefreshExpiration: 24 * time.Hour,
		},
	}

	logger, _ := zap.NewDevelopment()

	// 設置路由
	r := gin.New()
	jwtMiddleware := auth.NewJWTMiddleware(cfg, logger)
	r.Use(jwtMiddleware.Authenticate())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求 - 無 Token
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret-key-very-long-for-testing",
			Expiration:        1 * time.Hour,
			RefreshExpiration: 24 * time.Hour,
		},
	}

	logger, _ := zap.NewDevelopment()

	// 設置路由
	r := gin.New()
	jwtMiddleware := auth.NewJWTMiddleware(cfg, logger)
	r.Use(jwtMiddleware.Authenticate())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建測試請求 - 過期 Token
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
