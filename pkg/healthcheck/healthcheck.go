package healthcheck

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Status 健康檢查狀態
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckFunc 健康檢查函數類型
type CheckFunc func(ctx context.Context) error

// Check 健康檢查項目
type Check struct {
	Name     string
	CheckFn  CheckFunc
	Timeout  time.Duration
	Critical bool
}

// HealthChecker 健康檢查器
type HealthChecker struct {
	checks []Check
	mutex  sync.RWMutex
}

// Result 健康檢查結果
type Result struct {
	Status  Status                 `json:"status"`
	Checks  map[string]CheckResult `json:"checks"`
	Version string                 `json:"version,omitempty"`
	Uptime  time.Duration          `json:"uptime"`
}

// CheckResult 單個檢查結果
type CheckResult struct {
	Status   Status        `json:"status"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration"`
}

// New 創建新的健康檢查器
func New() *HealthChecker {
	return &HealthChecker{
		checks: make([]Check, 0),
	}
}

// AddCheck 添加健康檢查項目
func (h *HealthChecker) AddCheck(check Check) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if check.Timeout == 0 {
		check.Timeout = 5 * time.Second
	}

	h.checks = append(h.checks, check)
}

// Check 執行所有健康檢查
func (h *HealthChecker) Check(ctx context.Context) Result {
	h.mutex.RLock()
	checks := make([]Check, len(h.checks))
	copy(checks, h.checks)
	h.mutex.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy
	hasCriticalFailure := false

	for _, check := range checks {
		result := h.runCheck(ctx, check)
		results[check.Name] = result

		if result.Status == StatusUnhealthy {
			if check.Critical {
				hasCriticalFailure = true
			} else if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	if hasCriticalFailure {
		overallStatus = StatusUnhealthy
	}

	return Result{
		Status: overallStatus,
		Checks: results,
		Uptime: time.Since(startTime),
	}
}

// runCheck 執行單個檢查
func (h *HealthChecker) runCheck(ctx context.Context, check Check) CheckResult {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	err := check.CheckFn(checkCtx)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:   StatusUnhealthy,
			Message:  err.Error(),
			Duration: duration,
		}
	}

	return CheckResult{
		Status:   StatusHealthy,
		Duration: duration,
	}
}

// Handler 返回 Gin 路由處理器
func (h *HealthChecker) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		result := h.Check(c.Request.Context())

		statusCode := http.StatusOK
		if result.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if result.Status == StatusDegraded {
			statusCode = http.StatusOK // 降級狀態仍返回200
		}

		c.JSON(statusCode, result)
	}
}

// 應用程式啟動時間
var startTime = time.Now()

// DatabaseCheck 數據庫健康檢查
func DatabaseCheck(db interface{}) CheckFunc {
	return func(ctx context.Context) error {
		// 這裡可以根據實際數據庫類型實現檢查邏輯
		// 例如：執行簡單的查詢
		return nil
	}
}

// RedisCheck Redis健康檢查
func RedisCheck(client interface{}) CheckFunc {
	return func(ctx context.Context) error {
		// 這裡可以根據實際Redis客戶端實現檢查邏輯
		// 例如：執行PING命令
		return nil
	}
}

// HTTPCheck HTTP服務健康檢查
func HTTPCheck(url string) CheckFunc {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return http.ErrNotSupported
		}

		return nil
	}
}
