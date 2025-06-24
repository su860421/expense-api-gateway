package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"expense-api-gateway/internal/config"

	"go.uber.org/zap"
)

// ServiceInstance 服務實例
type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags"`
	Meta     map[string]string `json:"meta"`
	Health   HealthStatus      `json:"health"`
	LastSeen time.Time         `json:"last_seen"`
}

// HealthStatus 健康狀態
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusCritical  HealthStatus = "critical"
)

// ServiceDiscovery 服務發現接口
type ServiceDiscovery interface {
	Register(instance *ServiceInstance) error
	Deregister(serviceID string) error
	Discover(serviceName string) ([]*ServiceInstance, error)
	Watch(serviceName string) (<-chan []*ServiceInstance, error)
	Start() error
	Stop() error
}

// InMemoryDiscovery 內存服務發現實現
type InMemoryDiscovery struct {
	services map[string]map[string]*ServiceInstance
	watchers map[string][]chan []*ServiceInstance
	mutex    sync.RWMutex
	config   *config.Config
	logger   *zap.Logger
	stopCh   chan struct{}
}

// New 創建新的服務發現實例
func New(cfg *config.Config, logger *zap.Logger) ServiceDiscovery {
	return &InMemoryDiscovery{
		services: make(map[string]map[string]*ServiceInstance),
		watchers: make(map[string][]chan []*ServiceInstance),
		config:   cfg,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// Start 啟動服務發現
func (d *InMemoryDiscovery) Start() error {
	d.logger.Info("Starting service discovery")

	// 預註冊配置中的服務
	for serviceName, serviceConfig := range d.config.Discovery.Services {
		for _, host := range serviceConfig.Hosts {
			serviceInstance := &ServiceInstance{
				ID:       fmt.Sprintf("%s-%s", serviceName, host),
				Name:     serviceName,
				Address:  host,
				Port:     serviceConfig.Port,
				Tags:     []string{},
				Meta:     make(map[string]string),
				Health:   HealthStatusHealthy,
				LastSeen: time.Now(),
			}

			if err := d.Register(serviceInstance); err != nil {
				d.logger.Error("Failed to register service instance",
					zap.String("service", serviceName),
					zap.String("instance", host),
					zap.Error(err))
			}
		}
	}

	// 啟動健康檢查協程
	go d.healthCheckLoop()

	return nil
}

// Stop 停止服務發現
func (d *InMemoryDiscovery) Stop() error {
	d.logger.Info("Stopping service discovery")
	close(d.stopCh)
	return nil
}

// Register 註冊服務實例
func (d *InMemoryDiscovery) Register(instance *ServiceInstance) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.services[instance.Name] == nil {
		d.services[instance.Name] = make(map[string]*ServiceInstance)
	}

	instance.LastSeen = time.Now()
	d.services[instance.Name][instance.ID] = instance

	d.logger.Info("Service instance registered",
		zap.String("service", instance.Name),
		zap.String("id", instance.ID),
		zap.String("address", instance.Address))

	// 通知監聽者
	d.notifyWatchers(instance.Name)

	return nil
}

// Deregister 註銷服務實例
func (d *InMemoryDiscovery) Deregister(serviceID string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for serviceName, instances := range d.services {
		if _, exists := instances[serviceID]; exists {
			delete(instances, serviceID)
			d.logger.Info("Service instance deregistered",
				zap.String("service", serviceName),
				zap.String("id", serviceID))

			// 通知監聽者
			d.notifyWatchers(serviceName)
			return nil
		}
	}

	return fmt.Errorf("service instance not found: %s", serviceID)
}

// Discover 發現服務實例
func (d *InMemoryDiscovery) Discover(serviceName string) ([]*ServiceInstance, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	instances, exists := d.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	var result []*ServiceInstance
	for _, instance := range instances {
		if instance.Health == HealthStatusHealthy {
			result = append(result, instance)
		}
	}

	return result, nil
}

// Watch 監聽服務變化
func (d *InMemoryDiscovery) Watch(serviceName string) (<-chan []*ServiceInstance, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	ch := make(chan []*ServiceInstance, 1)
	d.watchers[serviceName] = append(d.watchers[serviceName], ch)

	// 立即發送當前狀態
	go func() {
		if instances, err := d.Discover(serviceName); err == nil {
			select {
			case ch <- instances:
			case <-d.stopCh:
			}
		}
	}()

	return ch, nil
}

// notifyWatchers 通知監聽者
func (d *InMemoryDiscovery) notifyWatchers(serviceName string) {
	if watchers, exists := d.watchers[serviceName]; exists {
		if instances, err := d.Discover(serviceName); err == nil {
			for _, watcher := range watchers {
				select {
				case watcher <- instances:
				default:
					// 非阻塞發送
				}
			}
		}
	}
}

// healthCheckLoop 健康檢查循環
func (d *InMemoryDiscovery) healthCheckLoop() {
	// 簡化的健康檢查，每30秒檢查一次
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.performHealthChecks()
		case <-d.stopCh:
			return
		}
	}
}

// performHealthChecks 執行健康檢查
func (d *InMemoryDiscovery) performHealthChecks() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for serviceName, instances := range d.services {
		for _, instance := range instances {
			go d.checkInstanceHealth(serviceName, instance)
		}
	}
}

// checkInstanceHealth 檢查實例健康狀態
func (d *InMemoryDiscovery) checkInstanceHealth(serviceName string, instance *ServiceInstance) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用配置中的健康檢查路徑
	healthPath := "/health"
	if serviceConfig, exists := d.config.GetServiceConfig(serviceName); exists && serviceConfig.HealthCheck != "" {
		healthPath = serviceConfig.HealthCheck
	}

	healthURL := fmt.Sprintf("http://%s:%d%s", instance.Address, instance.Port, healthPath)

	// 這裡可以實現實際的HTTP健康檢查
	// 目前簡化為模擬檢查
	healthy := d.simulateHealthCheck(ctx, healthURL)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	oldHealth := instance.Health
	if healthy {
		instance.Health = HealthStatusHealthy
		instance.LastSeen = time.Now()
	} else {
		instance.Health = HealthStatusUnhealthy
	}

	// 如果健康狀態改變，通知監聽者
	if oldHealth != instance.Health {
		d.logger.Info("Service instance health changed",
			zap.String("service", serviceName),
			zap.String("id", instance.ID),
			zap.String("old_health", string(oldHealth)),
			zap.String("new_health", string(instance.Health)))

		d.notifyWatchers(serviceName)
	}
}

// simulateHealthCheck 模擬健康檢查
func (d *InMemoryDiscovery) simulateHealthCheck(ctx context.Context, url string) bool {
	// 這裡可以實現實際的HTTP請求
	// 目前返回true表示健康
	return true
}
