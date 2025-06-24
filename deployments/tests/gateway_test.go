package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/router"
	"expense-api-gateway/internal/service/discovery"
	"expense-api-gateway/internal/service/monitor"
	"expense-api-gateway/pkg/healthcheck"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// createTestConfig 創建測試配置
func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "test",
		},
		Monitor: config.MonitorConfig{
			Enabled: true,
		},
		RateLimit: config.RateLimitConfig{
			Enabled: false,
		},
		Security: config.SecurityConfig{
			CORS: config.CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"*"},
				AllowCredentials: true,
			},
		},
	}
}

func TestHealthEndpoint(t *testing.T) {
	// 創建測試配置
	cfg := createTestConfig()

	// 創建依賴
	logger, _ := zap.NewDevelopment()
	serviceDiscovery := discovery.New(cfg, logger)
	monitorService := monitor.New(cfg, logger)
	healthChecker := healthcheck.New()

	// 設置路由
	r := router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestSystemStatusEndpoint(t *testing.T) {
	// 創建測試配置
	cfg := createTestConfig()

	// 創建依賴
	logger, _ := zap.NewDevelopment()
	serviceDiscovery := discovery.New(cfg, logger)
	monitorService := monitor.New(cfg, logger)
	healthChecker := healthcheck.New()

	// 設置路由
	r := router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/api/v1/system/status", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
	assert.Contains(t, w.Body.String(), "ok")
}

func TestGetMetricsEndpoint(t *testing.T) {
	// 創建測試配置
	cfg := createTestConfig()

	// 創建依賴
	logger, _ := zap.NewDevelopment()
	serviceDiscovery := discovery.New(cfg, logger)
	monitorService := monitor.New(cfg, logger)
	healthChecker := healthcheck.New()

	// 設置路由
	r := router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/api/v1/system/metrics", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "request_count")
	assert.Contains(t, w.Body.String(), "error_count")
}

func TestGetRoutesEndpoint(t *testing.T) {
	// 創建測試配置
	cfg := createTestConfig()

	// 創建依賴
	logger, _ := zap.NewDevelopment()
	serviceDiscovery := discovery.New(cfg, logger)
	monitorService := monitor.New(cfg, logger)
	healthChecker := healthcheck.New()

	// 設置路由
	r := router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/admin/routes", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "endpoints")
	assert.Contains(t, w.Body.String(), "/health")
}

func TestMaintenanceModeToggle(t *testing.T) {
	// 創建測試配置
	cfg := createTestConfig()

	// 創建依賴
	logger, _ := zap.NewDevelopment()
	serviceDiscovery := discovery.New(cfg, logger)
	monitorService := monitor.New(cfg, logger)
	healthChecker := healthcheck.New()

	// 設置路由
	r := router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)

	// 創建測試請求
	req, _ := http.NewRequest("POST", "/admin/maintenance", nil)
	w := httptest.NewRecorder()

	// 執行請求
	r.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "maintenance_mode")
}
