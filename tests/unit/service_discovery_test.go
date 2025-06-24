package unit

import (
	"testing"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/service/discovery"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestServiceDiscovery_RegisterAndDiscover(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{
				"test-service": {
					Hosts: []string{"localhost"},
					Port:  8080,
				},
			},
		},
	}

	logger, _ := zap.NewDevelopment()
	discoveryService := discovery.New(cfg, logger)

	// 啟動服務發現
	err := discoveryService.Start()
	assert.NoError(t, err)
	defer discoveryService.Stop()

	// 註冊服務實例
	instance := &discovery.ServiceInstance{
		ID:      "test-instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Health:  discovery.HealthStatusHealthy,
	}

	err = discoveryService.Register(instance)
	assert.NoError(t, err)

	// 發現服務實例
	instances, err := discoveryService.Discover("test-service")
	assert.NoError(t, err)
	assert.Len(t, instances, 2) // 1個預註冊 + 1個手動註冊

	// 驗證實例信息
	found := false
	for _, inst := range instances {
		if inst.ID == "test-instance-1" {
			assert.Equal(t, "test-service", inst.Name)
			assert.Equal(t, "localhost", inst.Address)
			assert.Equal(t, 8080, inst.Port)
			assert.Equal(t, discovery.HealthStatusHealthy, inst.Health)
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestServiceDiscovery_Deregister(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{},
		},
	}

	logger, _ := zap.NewDevelopment()
	discoveryService := discovery.New(cfg, logger)

	// 啟動服務發現
	err := discoveryService.Start()
	assert.NoError(t, err)
	defer discoveryService.Stop()

	// 註冊服務實例
	instance := &discovery.ServiceInstance{
		ID:      "test-instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Health:  discovery.HealthStatusHealthy,
	}

	err = discoveryService.Register(instance)
	assert.NoError(t, err)

	// 驗證註冊成功
	instances, err := discoveryService.Discover("test-service")
	assert.NoError(t, err)
	assert.Len(t, instances, 1)

	// 註銷服務實例
	err = discoveryService.Deregister("test-instance-1")
	assert.NoError(t, err)

	// 驗證註銷成功 - 服務可能仍然存在但實例被移除
	instances, err = discoveryService.Discover("test-service")
	// 註銷後可能返回空列表或服務不存在
	if err == nil {
		assert.Len(t, instances, 0)
	} else {
		// 如果服務不存在，這是正常的
		assert.Contains(t, err.Error(), "not found")
	}
}

func TestServiceDiscovery_Watch(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{},
		},
	}

	logger, _ := zap.NewDevelopment()
	discoveryService := discovery.New(cfg, logger)

	// 啟動服務發現
	err := discoveryService.Start()
	assert.NoError(t, err)
	defer discoveryService.Stop()

	// 監聽服務變化
	watchCh, err := discoveryService.Watch("test-service")
	assert.NoError(t, err)

	// 註冊服務實例
	instance := &discovery.ServiceInstance{
		ID:      "test-instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Health:  discovery.HealthStatusHealthy,
	}

	err = discoveryService.Register(instance)
	assert.NoError(t, err)

	// 等待通知
	select {
	case instances := <-watchCh:
		assert.Len(t, instances, 1)
		assert.Equal(t, "test-instance-1", instances[0].ID)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for watch notification")
	}
}

func TestServiceDiscovery_HealthStatus(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Services: map[string]config.ServiceConfig{},
		},
	}

	logger, _ := zap.NewDevelopment()
	discoveryService := discovery.New(cfg, logger)

	// 啟動服務發現
	err := discoveryService.Start()
	assert.NoError(t, err)
	defer discoveryService.Stop()

	// 註冊健康和不健康的實例
	healthyInstance := &discovery.ServiceInstance{
		ID:      "healthy-instance",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Health:  discovery.HealthStatusHealthy,
	}

	unhealthyInstance := &discovery.ServiceInstance{
		ID:      "unhealthy-instance",
		Name:    "test-service",
		Address: "localhost",
		Port:    8081,
		Health:  discovery.HealthStatusUnhealthy,
	}

	err = discoveryService.Register(healthyInstance)
	assert.NoError(t, err)

	err = discoveryService.Register(unhealthyInstance)
	assert.NoError(t, err)

	// 發現服務實例 - 只應該返回健康的實例
	instances, err := discoveryService.Discover("test-service")
	assert.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Equal(t, "healthy-instance", instances[0].ID)
	assert.Equal(t, discovery.HealthStatusHealthy, instances[0].Health)
}
