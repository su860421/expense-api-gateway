package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/middleware/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

// MemoryRateLimiter 內存限流器實現
type MemoryRateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

// NewMemoryRateLimiter 創建新的內存限流器
func NewMemoryRateLimiter(limit int, window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow 檢查是否允許請求
func (r *MemoryRateLimiter) Allow(key string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	// 清理過期的請求記錄
	if times, exists := r.requests[key]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(windowStart) {
				validTimes = append(validTimes, t)
			}
		}
		r.requests[key] = validTimes
	} else {
		r.requests[key] = make([]time.Time, 0)
	}

	// 檢查是否超過限制
	if len(r.requests[key]) >= r.limit {
		return false
	}

	// 記錄當前請求
	r.requests[key] = append(r.requests[key], now)
	return true
}

// Reset 重置限流器
func (r *MemoryRateLimiter) Reset(key string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.requests, key)
}

// RateLimitMiddleware 限流中間件
type RateLimitMiddleware struct {
	config        *config.Config
	logger        *zap.Logger
	limiters      map[string]RateLimiter
	globalLimiter RateLimiter
	mutex         sync.RWMutex
}

// NewRateLimitMiddleware 創建新的限流中間件
func NewRateLimitMiddleware(cfg *config.Config, logger *zap.Logger) *RateLimitMiddleware {
	middleware := &RateLimitMiddleware{
		config:   cfg,
		logger:   logger,
		limiters: make(map[string]RateLimiter),
	}

	// 創建全局限流器
	if cfg.RateLimit.Enabled {
		middleware.globalLimiter = NewMemoryRateLimiter(
			cfg.RateLimit.GlobalLimit,
			time.Minute, // 1分鐘窗口
		)

		// 創建 IP 限流器
		if cfg.RateLimit.IPLimit.Requests > 0 {
			middleware.limiters["ip"] = NewMemoryRateLimiter(
				cfg.RateLimit.IPLimit.Requests,
				cfg.RateLimit.IPLimit.Window,
			)
		}

		// 創建用戶限流器
		if cfg.RateLimit.UserLimit.Requests > 0 {
			middleware.limiters["user"] = NewMemoryRateLimiter(
				cfg.RateLimit.UserLimit.Requests,
				cfg.RateLimit.UserLimit.Window,
			)
		}
	}

	return middleware
}

// GlobalRateLimit 全局限流中間件
func (m *RateLimitMiddleware) GlobalRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.RateLimit.Enabled || m.globalLimiter == nil {
			c.Next()
			return
		}

		if !m.globalLimiter.Allow("global") {
			m.logger.Warn("Global rate limit exceeded",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Too many requests",
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit IP 限流中間件
func (m *RateLimitMiddleware) IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.RateLimit.Enabled {
			c.Next()
			return
		}

		limiter, exists := m.limiters["ip"]
		if !exists {
			c.Next()
			return
		}

		clientIP := m.getClientIP(c)
		key := fmt.Sprintf("ip:%s", clientIP)

		if !limiter.Allow(key) {
			m.logger.Warn("IP rate limit exceeded",
				zap.String("ip", clientIP),
				zap.String("path", c.Request.URL.Path))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "IP rate limit exceeded",
				"code":    "IP_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimit 用戶限流中間件
func (m *RateLimitMiddleware) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.RateLimit.Enabled {
			c.Next()
			return
		}

		limiter, exists := m.limiters["user"]
		if !exists {
			c.Next()
			return
		}

		// 嘗試從上下文中獲取用戶ID
		userID, err := auth.GetUserIDFromContext(c)
		if err != nil {
			// 如果沒有用戶信息，使用 IP 作為備用
			clientIP := m.getClientIP(c)
			key := fmt.Sprintf("user:anonymous:%s", clientIP)

			if !limiter.Allow(key) {
				m.logger.Warn("Anonymous user rate limit exceeded",
					zap.String("ip", clientIP),
					zap.String("path", c.Request.URL.Path))

				c.JSON(http.StatusTooManyRequests, gin.H{
					"status":  "error",
					"message": "User rate limit exceeded",
					"code":    "USER_RATE_LIMIT_EXCEEDED",
				})
				c.Abort()
				return
			}
		} else {
			key := fmt.Sprintf("user:%s", userID)

			if !limiter.Allow(key) {
				m.logger.Warn("User rate limit exceeded",
					zap.String("user_id", userID),
					zap.String("path", c.Request.URL.Path))

				c.JSON(http.StatusTooManyRequests, gin.H{
					"status":  "error",
					"message": "User rate limit exceeded",
					"code":    "USER_RATE_LIMIT_EXCEEDED",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// APIRateLimit API 端點限流中間件
func (m *RateLimitMiddleware) APIRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.RateLimit.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		// 檢查是否有針對此 API 的限流配置
		apiLimit, exists := m.config.RateLimit.APILimit[path]
		if !exists {
			c.Next()
			return
		}

		// 創建或獲取 API 限流器
		limiterKey := fmt.Sprintf("api:%s:%s", method, path)
		limiter := m.getOrCreateAPILimiter(limiterKey, apiLimit.Requests, apiLimit.Window)

		// 生成限流鍵
		var rateLimitKey string
		userID, err := auth.GetUserIDFromContext(c)
		if err != nil {
			// 使用 IP 作為備用
			clientIP := m.getClientIP(c)
			rateLimitKey = fmt.Sprintf("%s:ip:%s", limiterKey, clientIP)
		} else {
			rateLimitKey = fmt.Sprintf("%s:user:%s", limiterKey, userID)
		}

		if !limiter.Allow(rateLimitKey) {
			m.logger.Warn("API rate limit exceeded",
				zap.String("path", path),
				zap.String("method", method),
				zap.String("user_id", userID),
				zap.String("ip", m.getClientIP(c)))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "API rate limit exceeded",
				"code":    "API_RATE_LIMIT_EXCEEDED",
				"path":    path,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getOrCreateAPILimiter 獲取或創建 API 限流器
func (m *RateLimitMiddleware) getOrCreateAPILimiter(key string, limit int, window time.Duration) RateLimiter {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if limiter, exists := m.limiters[key]; exists {
		return limiter
	}

	limiter := NewMemoryRateLimiter(limit, window)
	m.limiters[key] = limiter
	return limiter
}

// getClientIP 獲取客戶端 IP
func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	// 檢查 X-Forwarded-For 頭
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 檢查 X-Real-IP 頭
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// 使用客戶端 IP
	return c.ClientIP()
}

// GetRateLimitHeaders 獲取限流相關的響應頭
func (m *RateLimitMiddleware) GetRateLimitHeaders(c *gin.Context) {
	if !m.config.RateLimit.Enabled {
		return
	}

	// 設置限流相關的響應頭
	c.Header("X-RateLimit-Limit", strconv.Itoa(m.config.RateLimit.GlobalLimit))
	c.Header("X-RateLimit-Remaining", "0") // 這裡可以實現更精確的計算
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
}

// ResetRateLimit 重置限流器（用於管理端點）
func (m *RateLimitMiddleware) ResetRateLimit(key string) {
	if limiter, exists := m.limiters[key]; exists {
		limiter.Reset(key)
	}
	if m.globalLimiter != nil {
		m.globalLimiter.Reset("global")
	}
}

// GetRateLimitStats 獲取限流統計信息
func (m *RateLimitMiddleware) GetRateLimitStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["enabled"] = m.config.RateLimit.Enabled
	stats["global_limit"] = m.config.RateLimit.GlobalLimit
	stats["ip_limit"] = m.config.RateLimit.IPLimit
	stats["user_limit"] = m.config.RateLimit.UserLimit
	stats["api_limits"] = len(m.config.RateLimit.APILimit)

	return stats
}
