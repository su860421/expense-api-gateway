package auth

import (
	"fmt"
	"net/http"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/infrastructure/jwt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JWTMiddleware JWT 認證中間件
type JWTMiddleware struct {
	config *config.Config
	logger *zap.Logger
	jwtSvc *jwt.JWTService
}

// NewJWTMiddleware 創建新的 JWT 中間件
func NewJWTMiddleware(cfg *config.Config, logger *zap.Logger) *JWTMiddleware {
	jwtSvc := jwt.NewJWTService(cfg, logger)
	return &JWTMiddleware{
		config: cfg,
		logger: logger,
		jwtSvc: jwtSvc,
	}
}

// Authenticate JWT 認證中間件 - 驗證 token 並設置轉發 headers
func (m *JWTMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從請求頭獲取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// 提取 token
		tokenString, err := m.jwtSvc.ExtractTokenFromAuthHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// 驗證 token
		authResult, err := m.jwtSvc.ValidateToken(tokenString)
		if err != nil || !authResult.Success {
			m.logger.Warn("Invalid JWT token",
				zap.String("error", err.Error()),
				zap.String("ip", c.ClientIP()),
				zap.String("user_agent", c.GetHeader("User-Agent")))

			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": authResult.Message,
			})
			c.Abort()
			return
		}

		// 設置轉發給微服務的 headers
		headers := m.jwtSvc.CreateAuthHeaders(authResult.User)
		for key, value := range headers {
			c.Request.Header.Set(key, value)
		}

		// 將基本信息存儲到上下文中（僅用於日誌和調試）
		c.Set("user_id", authResult.User.ID)
		c.Set("company_id", authResult.User.CompanyID)
		c.Set("user_role", authResult.User.Role)

		m.logger.Debug("Request authenticated",
			zap.String("user_id", authResult.User.ID),
			zap.String("company_id", authResult.User.CompanyID),
			zap.String("role", authResult.User.Role),
			zap.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// RequireRoles 角色驗證中間件
func (m *JWTMiddleware) RequireRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從 JWT claims 中獲取角色信息
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		tokenString, err := m.jwtSvc.ExtractTokenFromAuthHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		authResult, err := m.jwtSvc.ValidateToken(tokenString)
		if err != nil || !authResult.Success {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid token",
			})
			c.Abort()
			return
		}

		// 檢查用戶是否具有所需角色
		if !m.jwtSvc.ValidateRole(authResult.User, requiredRoles) {
			m.logger.Warn("Access denied - insufficient permissions",
				zap.String("user_id", authResult.User.ID),
				zap.String("user_role", authResult.User.Role),
				zap.Strings("required_roles", requiredRoles),
				zap.String("path", c.Request.URL.Path))

			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		// 設置轉發給微服務的 headers
		headers := m.jwtSvc.CreateAuthHeaders(authResult.User)
		for key, value := range headers {
			c.Request.Header.Set(key, value)
		}

		// 將基本信息存儲到上下文中
		c.Set("user_id", authResult.User.ID)
		c.Set("company_id", authResult.User.CompanyID)
		c.Set("user_role", authResult.User.Role)

		c.Next()
	}
}

// OptionalAuth 可選認證中間件（不強制要求認證）
func (m *JWTMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString, err := m.jwtSvc.ExtractTokenFromAuthHeader(authHeader)
		if err != nil {
			// 可選認證失敗時不中斷請求
			c.Next()
			return
		}

		authResult, err := m.jwtSvc.ValidateToken(tokenString)
		if err != nil || !authResult.Success {
			// 可選認證失敗時不中斷請求
			c.Next()
			return
		}

		// 設置轉發給微服務的 headers
		headers := m.jwtSvc.CreateAuthHeaders(authResult.User)
		for key, value := range headers {
			c.Request.Header.Set(key, value)
		}

		// 將基本信息存儲到上下文中
		c.Set("user_id", authResult.User.ID)
		c.Set("company_id", authResult.User.CompanyID)
		c.Set("user_role", authResult.User.Role)

		c.Next()
	}
}

// GetUserIDFromContext 從上下文中獲取用戶ID
func GetUserIDFromContext(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("user_id not found in context")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id in context")
	}

	return userIDStr, nil
}

// GetCompanyIDFromContext 從上下文中獲取公司ID
func GetCompanyIDFromContext(c *gin.Context) (string, error) {
	companyID, exists := c.Get("company_id")
	if !exists {
		return "", fmt.Errorf("company_id not found in context")
	}

	companyIDStr, ok := companyID.(string)
	if !ok {
		return "", fmt.Errorf("invalid company_id in context")
	}

	return companyIDStr, nil
}
