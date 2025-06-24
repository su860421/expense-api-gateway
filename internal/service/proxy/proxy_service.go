package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/service/discovery"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProxyService 代理服務
type ProxyService struct {
	config          *config.Config
	logger          *zap.Logger
	routeParser     *RouteParser
	discovery       discovery.ServiceDiscovery
	httpClient      *http.Client
	maintenanceMode *bool // 指向維護模式狀態的指針
}

// ProxyRequest 代理請求
type ProxyRequest struct {
	Method      string
	Path        string
	Headers     map[string]string
	Body        io.Reader
	QueryParams map[string]string
	Timeout     time.Duration
}

// ProxyResponse 代理響應
type ProxyResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Duration   time.Duration
	Error      error
}

// NewProxyService 創建新的代理服務
func NewProxyService(cfg *config.Config, logger *zap.Logger, routeParser *RouteParser, discovery discovery.ServiceDiscovery) *ProxyService {
	maintenanceMode := false
	return &ProxyService{
		config:          cfg,
		logger:          logger,
		routeParser:     routeParser,
		discovery:       discovery,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		maintenanceMode: &maintenanceMode,
	}
}

// SetMaintenanceMode 設置維護模式
func (p *ProxyService) SetMaintenanceMode(enabled bool) {
	*p.maintenanceMode = enabled
}

// GetMaintenanceMode 獲取維護模式狀態
func (p *ProxyService) GetMaintenanceMode() bool {
	return *p.maintenanceMode
}

// ProxyRequest 代理請求到目標服務
func (p *ProxyService) ProxyRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	start := time.Now()

	// 匹配路由
	route, service, err := p.routeParser.MatchRoute(req.Method, req.Path)
	if err != nil {
		return &ProxyResponse{
			StatusCode: http.StatusNotFound,
			Error:      fmt.Errorf("no route found: %w", err),
			Duration:   time.Since(start),
		}, err
	}

	// 檢查認證要求
	if route.AuthRequired {
		// 這裡可以添加認證檢查邏輯
		// 目前由中間件處理
	}

	// 檢查角色權限
	if len(route.Roles) > 0 {
		// 這裡可以添加角色檢查邏輯
		// 目前由中間件處理
	}

	// 發現服務實例
	instances, err := p.discovery.Discover(route.Service)
	if err != nil || len(instances) == 0 {
		return &ProxyResponse{
			StatusCode: http.StatusServiceUnavailable,
			Error:      fmt.Errorf("service not available: %s", route.Service),
			Duration:   time.Since(start),
		}, err
	}

	// 選擇服務實例（簡單輪詢，可以擴展為負載均衡）
	instance := instances[0]

	// 構建目標 URL
	targetURL, err := p.buildTargetURL(instance, req.Path, route, service)
	if err != nil {
		return &ProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to build target URL: %w", err),
			Duration:   time.Since(start),
		}, err
	}

	// 創建 HTTP 請求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL.String(), req.Body)
	if err != nil {
		return &ProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
			Duration:   time.Since(start),
		}, err
	}

	// 設置請求頭
	p.setRequestHeaders(httpReq, req.Headers, route, service)

	// 設置查詢參數
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// 設置超時
	if req.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, req.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	} else if route.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, route.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	}

	// 執行請求
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return &ProxyResponse{
			StatusCode: http.StatusBadGateway,
			Error:      fmt.Errorf("request failed: %w", err),
			Duration:   time.Since(start),
		}, err
	}
	defer resp.Body.Close()

	// 讀取響應體
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to read response body: %w", err),
			Duration:   time.Since(start),
		}, err
	}

	// 構建響應頭
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	duration := time.Since(start)

	p.logger.Debug("Proxy request completed",
		zap.String("method", req.Method),
		zap.String("path", req.Path),
		zap.String("target", targetURL.String()),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration))

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
		Duration:   duration,
	}, nil
}

// ProxyGinRequest 代理 Gin 請求
func (p *ProxyService) ProxyGinRequest(c *gin.Context) {
	start := time.Now()

	// 檢查維護模式
	if p.isMaintenanceMode(c) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Service is under maintenance",
		})
		return
	}

	// 匹配路由
	route, service, err := p.routeParser.MatchRoute(c.Request.Method, c.Request.URL.Path)
	if err != nil {
		p.logger.Warn("No route found",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Error(err))

		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Route not found",
		})
		return
	}

	// 發現服務實例
	instances, err := p.discovery.Discover(route.Service)
	if err != nil || len(instances) == 0 {
		p.logger.Error("Service not available",
			zap.String("service", route.Service),
			zap.Error(err))

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Service not available",
		})
		return
	}

	// 選擇服務實例
	instance := instances[0]

	// 構建目標 URL
	targetURL, err := p.buildTargetURL(instance, c.Request.URL.Path, route, service)
	if err != nil {
		p.logger.Error("Failed to build target URL",
			zap.Error(err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Internal server error",
		})
		return
	}

	// 創建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定義 Director 函數
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		p.customizeRequest(req, c, route, service)
	}

	// 自定義 ModifyResponse 函數
	proxy.ModifyResponse = func(resp *http.Response) error {
		p.customizeResponse(resp, c)
		return nil
	}

	// 自定義 ErrorHandler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("Proxy error",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err))

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"status":"error","message":"Bad Gateway"}`))
	}

	// 執行代理
	proxy.ServeHTTP(c.Writer, c.Request)

	// 記錄指標
	duration := time.Since(start)
	p.logger.Debug("Proxy request completed",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("target", targetURL.String()),
		zap.Int("status_code", c.Writer.Status()),
		zap.Duration("duration", duration))
}

// buildTargetURL 構建目標 URL
func (p *ProxyService) buildTargetURL(instance *discovery.ServiceInstance, path string, route *RouteConfig, service *ServiceConfig) (*url.URL, error) {
	// 構建基礎 URL
	targetURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", instance.Address, instance.Port),
	}

	// 處理路徑重寫
	if route.StripPrefix {
		// 移除前綴
		path = strings.TrimPrefix(path, route.Pattern)
	} else if route.RewritePath != "" {
		// 重寫路徑
		path = route.RewritePath
	}

	targetURL.Path = path

	return targetURL, nil
}

// setRequestHeaders 設置請求頭
func (p *ProxyService) setRequestHeaders(req *http.Request, headers map[string]string, route *RouteConfig, service *ServiceConfig) {
	// 設置自定義請求頭
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 設置路由級別的請求頭
	for key, value := range route.Headers {
		req.Header.Set(key, value)
	}

	// 設置服務級別的請求頭
	for key, value := range service.Headers {
		req.Header.Set(key, value)
	}

	// 設置代理相關的請求頭
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
}

// customizeRequest 自定義請求
func (p *ProxyService) customizeRequest(req *http.Request, c *gin.Context, route *RouteConfig, service *ServiceConfig) {
	// 設置請求頭
	p.setRequestHeaders(req, make(map[string]string), route, service)

	// 設置用戶信息（如果存在）
	if userID, exists := c.Get("user_id"); exists {
		req.Header.Set("X-User-ID", userID.(string))
	}

	if companyID, exists := c.Get("company_id"); exists {
		req.Header.Set("X-Company-ID", companyID.(string))
	}

	// 設置請求體大小限制
	if route.MaxBodySize > 0 {
		req.ContentLength = route.MaxBodySize
	} else if service.MaxBodySize > 0 {
		req.ContentLength = service.MaxBodySize
	}
}

// customizeResponse 自定義響應
func (p *ProxyService) customizeResponse(resp *http.Response, c *gin.Context) {
	// 設置響應頭
	resp.Header.Set("X-Proxy-By", "expense-api-gateway")
	resp.Header.Set("X-Proxy-Time", time.Now().Format(time.RFC3339))
}

// isMaintenanceMode 檢查是否處於維護模式
func (p *ProxyService) isMaintenanceMode(c *gin.Context) bool {
	return *p.maintenanceMode
}

// GetServiceHealth 獲取服務健康狀態
func (p *ProxyService) GetServiceHealth(serviceName string) (bool, error) {
	instances, err := p.discovery.Discover(serviceName)
	if err != nil {
		return false, err
	}

	if len(instances) == 0 {
		return false, fmt.Errorf("no instances found for service: %s", serviceName)
	}

	// 檢查第一個實例的健康狀態
	instance := instances[0]
	return instance.Health == discovery.HealthStatusHealthy, nil
}

// GetProxyStats 獲取代理統計信息
func (p *ProxyService) GetProxyStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 獲取所有服務
	services := p.routeParser.GetAllServices()
	stats["total_services"] = len(services)

	// 獲取所有路由
	routes := p.routeParser.GetAllRoutes()
	stats["total_routes"] = len(routes)

	// 獲取最後重載時間
	stats["last_reload"] = p.routeParser.GetLastReloadTime()

	return stats
}
