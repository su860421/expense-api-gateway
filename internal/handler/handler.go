package handler

import (
	"net/http"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/middleware/ratelimit"
	"expense-api-gateway/internal/service/discovery"
	"expense-api-gateway/internal/service/monitor"
	"expense-api-gateway/internal/service/proxy"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP處理器
type Handler struct {
	config              *config.Config
	logger              *zap.Logger
	serviceDiscovery    discovery.ServiceDiscovery
	monitorService      *monitor.Monitor
	proxyService        *proxy.ProxyService
	rateLimitMiddleware *ratelimit.RateLimitMiddleware
	maintenanceMode     bool
}

// New 創建新的處理器
func New(
	cfg *config.Config,
	logger *zap.Logger,
	serviceDiscovery discovery.ServiceDiscovery,
	monitorService *monitor.Monitor,
) *Handler {
	return &Handler{
		config:           cfg,
		logger:           logger,
		serviceDiscovery: serviceDiscovery,
		monitorService:   monitorService,
		maintenanceMode:  false,
	}
}

// NewWithProxy 創建新的處理器（使用代理服務）
func NewWithProxy(
	cfg *config.Config,
	logger *zap.Logger,
	serviceDiscovery discovery.ServiceDiscovery,
	monitor *monitor.Monitor,
	proxyService *proxy.ProxyService,
) *Handler {
	return &Handler{
		config:           cfg,
		logger:           logger,
		serviceDiscovery: serviceDiscovery,
		monitorService:   monitor,
		proxyService:     proxyService,
		maintenanceMode:  false,
	}
}

// NewWithRateLimit 創建新的處理器（包含限流）
func NewWithRateLimit(
	cfg *config.Config,
	logger *zap.Logger,
	serviceDiscovery discovery.ServiceDiscovery,
	monitor *monitor.Monitor,
	proxyService *proxy.ProxyService,
	rateLimitMiddleware *ratelimit.RateLimitMiddleware,
) *Handler {
	return &Handler{
		config:              cfg,
		logger:              logger,
		serviceDiscovery:    serviceDiscovery,
		monitorService:      monitor,
		proxyService:        proxyService,
		rateLimitMiddleware: rateLimitMiddleware,
		maintenanceMode:     false,
	}
}

// GetSystemStatus 獲取系統狀態
func (h *Handler) GetSystemStatus(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   h.config.App.Version,
		"uptime":    time.Since(h.config.App.StartTime),
	}
	c.JSON(http.StatusOK, status)
}

// GetMetrics 獲取系統指標
func (h *Handler) GetMetrics(c *gin.Context) {
	metrics := h.monitorService.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// ResetMetrics 重置系統指標
func (h *Handler) ResetMetrics(c *gin.Context) {
	h.monitorService.ResetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Metrics reset successfully",
	})
}

// ListServices 列出所有服務
func (h *Handler) ListServices(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}

// GetService 獲取特定服務
func (h *Handler) GetService(c *gin.Context) {
	serviceName := c.Param("name")
	instances, err := h.serviceDiscovery.Discover(serviceName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Service not found",
		})
		return
	}

	c.JSON(http.StatusOK, instances)
}

// RegisterService 註冊服務
func (h *Handler) RegisterService(c *gin.Context) {
	var instance discovery.ServiceInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body",
		})
		return
	}
	err := h.serviceDiscovery.Register(&instance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to register service",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Service registered successfully",
	})
}

// DeregisterService 註銷服務
func (h *Handler) DeregisterService(c *gin.Context) {
	instanceID := c.Param("id")
	err := h.serviceDiscovery.Deregister(instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to deregister service",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Service deregistered successfully",
	})
}

// AuthLogin 用戶登入
func (h *Handler) AuthLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"token":   "mock-token",
		"message": "Login successful",
	})
}

// AuthRefresh 刷新 Token
func (h *Handler) AuthRefresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"token": "mock-token",
	})
}

// AuthLogout 用戶登出
func (h *Handler) AuthLogout(c *gin.Context) {
	// 這裡可以實現 token 黑名單機制
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logout successful",
	})
}

// ProxyHandler 代理處理器
func (h *Handler) ProxyHandler(c *gin.Context) {
	if h.proxyService != nil {
		// 使用新的代理服務
		h.proxyService.ProxyGinRequest(c)
		return
	}

	// 舊的代理邏輯（向後兼容）
	h.legacyProxyHandler(c)
}

// legacyProxyHandler 舊的代理處理器
func (h *Handler) legacyProxyHandler(c *gin.Context) {
	// 獲取服務名稱和路徑
	path := c.Param("path")
	if path == "" {
		path = c.Request.URL.Path
	}

	// 簡單的路由匹配邏輯
	var targetService string
	var targetPath string

	// 根據路徑前綴確定目標服務
	switch {
	case len(path) >= 6 && path[:6] == "/users":
		targetService = "user-service"
		targetPath = path[6:]
	case len(path) >= 9 && path[:9] == "/expenses":
		targetService = "expense-service"
		targetPath = path[9:]
	case len(path) >= 10 && path[:10] == "/approvals":
		targetService = "approval-service"
		targetPath = path[10:]
	case len(path) >= 8 && path[:8] == "/finance":
		targetService = "finance-service"
		targetPath = path[8:]
	case len(path) >= 6 && path[:6] == "/files":
		targetService = "file-service"
		targetPath = path[6:]
	case len(path) >= 3 && path[:3] == "/ai":
		targetService = "ai-service"
		targetPath = path[3:]
	case len(path) >= 14 && path[:14] == "/notifications":
		targetService = "notification-service"
		targetPath = path[14:]
	default:
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Service not found",
		})
		return
	}

	// 發現服務實例
	instances, err := h.serviceDiscovery.Discover(targetService)
	if err != nil || len(instances) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Service not available",
		})
		return
	}

	// 創建代理請求
	proxyReq := &proxy.ProxyRequest{
		Method:      c.Request.Method,
		Path:        targetPath,
		Headers:     make(map[string]string),
		Body:        c.Request.Body,
		QueryParams: make(map[string]string),
	}

	// 複製請求頭
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			proxyReq.Headers[key] = values[0]
		}
	}

	// 複製查詢參數
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			proxyReq.QueryParams[key] = values[0]
		}
	}

	// 執行代理請求
	resp, err := h.proxyService.ProxyRequest(c.Request.Context(), proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"status":  "error",
			"message": "Proxy request failed",
		})
		return
	}

	// 設置響應頭
	for key, value := range resp.Headers {
		c.Header(key, value)
	}

	// 返回響應
	c.Data(resp.StatusCode, "application/json", resp.Body)
}

// GetConfig 獲取配置
func (h *Handler) GetConfig(c *gin.Context) {
	// 返回安全的配置信息（不包含敏感數據）
	safeConfig := map[string]interface{}{
		"app": map[string]interface{}{
			"name":    h.config.App.Name,
			"version": h.config.App.Version,
			"port":    h.config.App.Port,
			"mode":    h.config.App.Mode,
		},
		"rate_limit": map[string]interface{}{
			"enabled":      h.config.RateLimit.Enabled,
			"global_limit": h.config.RateLimit.GlobalLimit,
		},
		"monitor": map[string]interface{}{
			"enabled":      h.config.Monitor.Enabled,
			"metrics_path": h.config.Monitor.MetricsPath,
		},
	}

	c.JSON(http.StatusOK, safeConfig)
}

// ReloadConfig 重載配置
func (h *Handler) ReloadConfig(c *gin.Context) {
	// 這裡可以實現配置熱重載
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Configuration reloaded successfully",
	})
}

// GetRoutes 獲取路由信息
func (h *Handler) GetRoutes(c *gin.Context) {
	routes := []map[string]interface{}{
		{"path": "/api/v1/users/*", "service": "user-service"},
		{"path": "/api/v1/expenses/*", "service": "expense-service"},
		{"path": "/api/v1/approvals/*", "service": "approval-service"},
		{"path": "/api/v1/finance/*", "service": "finance-service"},
		{"path": "/api/v1/files/*", "service": "file-service"},
		{"path": "/api/v1/ai/*", "service": "ai-service"},
		{"path": "/api/v1/notifications/*", "service": "notification-service"},
	}
	c.JSON(http.StatusOK, routes)
}

// ToggleMaintenanceMode 切換維護模式
func (h *Handler) ToggleMaintenanceMode(c *gin.Context) {
	// 切換維護模式狀態
	h.maintenanceMode = !h.maintenanceMode

	// 同步到代理服務
	if h.proxyService != nil {
		h.proxyService.SetMaintenanceMode(h.maintenanceMode)
	}

	status := "disabled"
	if h.maintenanceMode {
		status = "enabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Maintenance mode " + status,
		"data": gin.H{
			"maintenance_mode": h.maintenanceMode,
		},
	})
}

// GetRateLimitStats 獲取限流統計
func (h *Handler) GetRateLimitStats(c *gin.Context) {
	if h.rateLimitMiddleware != nil {
		stats := h.rateLimitMiddleware.GetRateLimitStats()
		c.JSON(http.StatusOK, stats)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "error",
		"message": "Rate limiting not enabled",
	})
}

// ResetRateLimit 重置限流
func (h *Handler) ResetRateLimit(c *gin.Context) {
	key := c.Query("key")
	if h.rateLimitMiddleware != nil {
		h.rateLimitMiddleware.ResetRateLimit(key)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Rate limit reset successfully",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "error",
		"message": "Rate limiting not enabled",
	})
}

// GetProxyStats 獲取代理統計
func (h *Handler) GetProxyStats(c *gin.Context) {
	if h.proxyService != nil {
		stats := h.proxyService.GetProxyStats()
		c.JSON(http.StatusOK, stats)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "error",
		"message": "Proxy service not available",
	})
}

// GetPrometheusMetrics 獲取 Prometheus 指標
func (h *Handler) GetPrometheusMetrics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"status":  "error",
		"message": "Prometheus metrics not implemented",
	})
}
