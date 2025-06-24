package logging

import (
	"time"

	"expense-api-gateway/internal/service/monitor"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware 日誌中間件
func Middleware(logger *zap.Logger, monitor *monitor.Monitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 處理請求
		c.Next()

		// 計算處理時間
		latency := time.Since(start)

		// 獲取請求信息
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()
		userAgent := c.Request.UserAgent()

		// 構建完整路徑
		if raw != "" {
			path = path + "?" + raw
		}

		// 記錄日誌
		logger.Info("HTTP Request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.Int("body_size", bodySize),
			zap.String("user_agent", userAgent),
		)

		// 記錄監控指標
		if monitor != nil {
			monitor.RecordRequest(c.Request.URL.Path, statusCode, latency)
		}
	}
}

// ErrorLogger 錯誤日誌中間件
func ErrorLogger(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			logger.Error("Panic recovered",
				zap.String("error", err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
			)
		}
		c.AbortWithStatus(500)
	})
}
