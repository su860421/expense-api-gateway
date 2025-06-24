package cors

import (
	"expense-api-gateway/internal/config"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Middleware CORS中間件
func Middleware(cfg *config.Config) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     cfg.CORS.AllowedMethods,
		AllowHeaders:     cfg.CORS.AllowedHeaders,
		AllowCredentials: true,
		MaxAge:           time.Duration(cfg.CORS.MaxAge) * time.Second,
	}

	// 如果配置為允許所有來源，使用AllowAllOrigins
	if len(corsConfig.AllowOrigins) == 1 && corsConfig.AllowOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
	}

	return cors.New(corsConfig)
}

// CustomCORS 自定義CORS中間件
func CustomCORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 檢查來源是否被允許
		if isAllowedOrigin(origin, cfg.CORS.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(cfg.CORS.AllowedOrigins) == 1 && cfg.CORS.AllowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isAllowedOrigin 檢查來源是否被允許
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
