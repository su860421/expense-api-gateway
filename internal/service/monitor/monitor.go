package monitor

import (
	"sync"
	"time"

	"expense-api-gateway/internal/config"

	"go.uber.org/zap"
)

// Metrics 監控指標
type Metrics struct {
	RequestCount    int64                       `json:"request_count"`
	ErrorCount      int64                       `json:"error_count"`
	ResponseTime    time.Duration               `json:"avg_response_time"`
	StatusCodes     map[int]int64               `json:"status_codes"`
	EndpointMetrics map[string]*EndpointMetrics `json:"endpoint_metrics"`
	LastUpdated     time.Time                   `json:"last_updated"`
}

// EndpointMetrics 端點指標
type EndpointMetrics struct {
	RequestCount int64         `json:"request_count"`
	ErrorCount   int64         `json:"error_count"`
	ResponseTime time.Duration `json:"avg_response_time"`
	StatusCodes  map[int]int64 `json:"status_codes"`
}

// Monitor 監控服務
type Monitor struct {
	config  *config.Config
	logger  *zap.Logger
	metrics *Metrics
	mutex   sync.RWMutex
	stopCh  chan struct{}
}

// New 創建新的監控服務
func New(cfg *config.Config, logger *zap.Logger) *Monitor {
	return &Monitor{
		config: cfg,
		logger: logger,
		metrics: &Metrics{
			StatusCodes:     make(map[int]int64),
			EndpointMetrics: make(map[string]*EndpointMetrics),
			LastUpdated:     time.Now(),
		},
		stopCh: make(chan struct{}),
	}
}

// Start 啟動監控服務
func (m *Monitor) Start() {
	if !m.config.Monitor.Enabled {
		m.logger.Info("Monitor service is disabled")
		return
	}

	m.logger.Info("Starting monitor service")
	go m.metricsCollectionLoop()
}

// Stop 停止監控服務
func (m *Monitor) Stop() {
	m.logger.Info("Stopping monitor service")
	close(m.stopCh)
}

// RecordRequest 記錄請求
func (m *Monitor) RecordRequest(endpoint string, statusCode int, responseTime time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 更新全局指標
	m.metrics.RequestCount++
	m.metrics.StatusCodes[statusCode]++

	if statusCode >= 400 {
		m.metrics.ErrorCount++
	}

	// 計算平均響應時間
	if m.metrics.RequestCount == 1 {
		m.metrics.ResponseTime = responseTime
	} else {
		m.metrics.ResponseTime = (m.metrics.ResponseTime + responseTime) / 2
	}

	// 更新端點指標
	if m.metrics.EndpointMetrics[endpoint] == nil {
		m.metrics.EndpointMetrics[endpoint] = &EndpointMetrics{
			StatusCodes: make(map[int]int64),
		}
	}

	endpointMetric := m.metrics.EndpointMetrics[endpoint]
	endpointMetric.RequestCount++
	endpointMetric.StatusCodes[statusCode]++

	if statusCode >= 400 {
		endpointMetric.ErrorCount++
	}

	// 計算端點平均響應時間
	if endpointMetric.RequestCount == 1 {
		endpointMetric.ResponseTime = responseTime
	} else {
		endpointMetric.ResponseTime = (endpointMetric.ResponseTime + responseTime) / 2
	}

	m.metrics.LastUpdated = time.Now()
}

// GetMetrics 獲取監控指標
func (m *Monitor) GetMetrics() *Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 深拷貝指標數據
	metricsCopy := &Metrics{
		RequestCount:    m.metrics.RequestCount,
		ErrorCount:      m.metrics.ErrorCount,
		ResponseTime:    m.metrics.ResponseTime,
		StatusCodes:     make(map[int]int64),
		EndpointMetrics: make(map[string]*EndpointMetrics),
		LastUpdated:     m.metrics.LastUpdated,
	}

	// 拷貝狀態碼統計
	for code, count := range m.metrics.StatusCodes {
		metricsCopy.StatusCodes[code] = count
	}

	// 拷貝端點指標
	for endpoint, metric := range m.metrics.EndpointMetrics {
		metricsCopy.EndpointMetrics[endpoint] = &EndpointMetrics{
			RequestCount: metric.RequestCount,
			ErrorCount:   metric.ErrorCount,
			ResponseTime: metric.ResponseTime,
			StatusCodes:  make(map[int]int64),
		}

		for code, count := range metric.StatusCodes {
			metricsCopy.EndpointMetrics[endpoint].StatusCodes[code] = count
		}
	}

	return metricsCopy
}

// GetHealthStatus 獲取健康狀態
func (m *Monitor) GetHealthStatus() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	errorRate := float64(0)
	if m.metrics.RequestCount > 0 {
		errorRate = float64(m.metrics.ErrorCount) / float64(m.metrics.RequestCount) * 100
	}

	return map[string]interface{}{
		"status":            "healthy",
		"request_count":     m.metrics.RequestCount,
		"error_rate":        errorRate,
		"avg_response_time": m.metrics.ResponseTime.Milliseconds(),
		"uptime":            time.Since(time.Now().Add(-time.Hour)), // 簡化的運行時間
	}
}

// metricsCollectionLoop 指標收集循環
func (m *Monitor) metricsCollectionLoop() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒收集一次指標
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectMetrics()
		case <-m.stopCh:
			return
		}
	}
}

// collectMetrics 收集指標
func (m *Monitor) collectMetrics() {
	metrics := m.GetMetrics()
	m.logger.Debug("Collected metrics",
		zap.Int64("request_count", metrics.RequestCount),
		zap.Int64("error_count", metrics.ErrorCount),
		zap.Duration("avg_response_time", metrics.ResponseTime))

	// 這裡可以將指標發送到外部監控系統
	// 例如：Prometheus, InfluxDB, etc.
}

// ResetMetrics 重置指標
func (m *Monitor) ResetMetrics() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics.RequestCount = 0
	m.metrics.ErrorCount = 0
	m.metrics.ResponseTime = 0
	m.metrics.StatusCodes = make(map[int]int64)
	m.metrics.EndpointMetrics = make(map[string]*EndpointMetrics)
	m.metrics.LastUpdated = time.Now()

	m.logger.Info("Metrics reset")
}

// EndpointStat 端點統計信息
type EndpointStat struct {
	Endpoint     string        `json:"endpoint"`
	RequestCount int64         `json:"request_count"`
	ErrorRate    float64       `json:"error_rate"`
	ResponseTime time.Duration `json:"avg_response_time"`
}

// GetTopEndpoints 獲取請求量最多的端點
func (m *Monitor) GetTopEndpoints(limit int) []EndpointStat {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var stats []EndpointStat
	for endpoint, metric := range m.metrics.EndpointMetrics {
		errorRate := float64(0)
		if metric.RequestCount > 0 {
			errorRate = float64(metric.ErrorCount) / float64(metric.RequestCount) * 100
		}

		stats = append(stats, EndpointStat{
			Endpoint:     endpoint,
			RequestCount: metric.RequestCount,
			ErrorRate:    errorRate,
			ResponseTime: metric.ResponseTime,
		})
	}

	// 按請求量排序
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].RequestCount < stats[j].RequestCount {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if len(stats) > limit {
		stats = stats[:limit]
	}

	return stats
}
