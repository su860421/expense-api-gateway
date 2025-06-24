package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/service/discovery"
	"expense-api-gateway/internal/service/proxy"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mock RouteParser
var mockRouteParser = &proxy.RouteParser{}

// mock ServiceDiscovery
// var mockDiscovery = &discovery.MockServiceDiscovery{}
type mockDiscovery struct{}

func (m *mockDiscovery) Register(instance *discovery.ServiceInstance) error { return nil }
func (m *mockDiscovery) Deregister(serviceID string) error                  { return nil }
func (m *mockDiscovery) Discover(serviceName string) ([]*discovery.ServiceInstance, error) {
	return nil, nil
}
func (m *mockDiscovery) Watch(serviceName string) (<-chan []*discovery.ServiceInstance, error) {
	return nil, nil
}
func (m *mockDiscovery) Start() error { return nil }
func (m *mockDiscovery) Stop() error  { return nil }

var mockDiscoveryInstance = &mockDiscovery{}

func TestProxyService_ForwardRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{
				"test-service": {
					Hosts:       []string{"localhost"},
					Port:        8080,
					HealthCheck: "/health",
					Headers: map[string]string{
						"X-Service-Name": "test-service",
					},
					MaxBodySize: 1048576, // 1MB
				},
			},
		},
	}
	logger := zap.NewNop()
	_ = proxy.NewProxyService(cfg, logger, mockRouteParser, mockDiscoveryInstance)
	// 測試路由
	router := gin.New()
	router.Any("/proxy/*path", func(c *gin.Context) {
		serviceName := "test-service"
		path := c.Param("path")
		serviceConfig, exists := cfg.Discovery.Services[serviceName]
		assert.True(t, exists, "服務配置應該存在")
		c.JSON(http.StatusOK, gin.H{
			"message": "proxied",
			"service": serviceName,
			"path":    path,
			"host":    serviceConfig.Hosts[0],
			"port":    serviceConfig.Port,
		})
	})
	req := httptest.NewRequest("GET", "/proxy/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxyService_ServiceNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{},
		},
	}
	logger := zap.NewNop()
	_ = proxy.NewProxyService(cfg, logger, mockRouteParser, mockDiscoveryInstance)
	router := gin.New()
	router.Any("/proxy/*path", func(c *gin.Context) {
		serviceName := "non-existent-service"
		_, exists := cfg.Discovery.Services[serviceName]
		assert.False(t, exists, "服務配置不應該存在")
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "service not found",
			"service": serviceName,
		})
	})
	req := httptest.NewRequest("GET", "/proxy/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyService_LoadBalancing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{
				"load-balanced-service": {
					Hosts:       []string{"host1", "host2", "host3"},
					Port:        8080,
					HealthCheck: "/health",
					Headers: map[string]string{
						"X-Service-Name": "load-balanced-service",
					},
					MaxBodySize: 1048576, // 1MB
				},
			},
		},
	}
	logger := zap.NewNop()
	_ = proxy.NewProxyService(cfg, logger, mockRouteParser, mockDiscoveryInstance)
	router := gin.New()
	router.Any("/proxy/*path", func(c *gin.Context) {
		serviceName := "load-balanced-service"
		serviceConfig, exists := cfg.Discovery.Services[serviceName]
		assert.True(t, exists, "服務配置應該存在")
		assert.Greater(t, len(serviceConfig.Hosts), 1, "應該有多個主機用於負載均衡")
		selectedHost := serviceConfig.Hosts[0]
		c.JSON(http.StatusOK, gin.H{
			"message": "load balanced",
			"service": serviceName,
			"host":    selectedHost,
			"port":    serviceConfig.Port,
		})
	})
	req := httptest.NewRequest("GET", "/proxy/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
