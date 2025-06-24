package domain

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Route 路由規則
type Route struct {
	ID           string            `json:"id"`
	Pattern      string            `json:"pattern"`
	ServiceName  string            `json:"service_name"`
	Methods      []string          `json:"methods"`
	AuthRequired bool              `json:"auth_required"`
	Roles        []string          `json:"roles"`
	Timeout      time.Duration     `json:"timeout"`
	MaxBodySize  int64             `json:"max_body_size"`
	Streaming    bool              `json:"streaming"`
	Metadata     map[string]string `json:"metadata"`
}

// RouteMatch 路由匹配結果
type RouteMatch struct {
	Route       *Route
	PathParams  map[string]string
	QueryParams map[string]string
	IsMatch     bool
}

// RouteGroup 路由組
type RouteGroup struct {
	Name       string            `json:"name"`
	Prefix     string            `json:"prefix"`
	Middleware []string          `json:"middleware"`
	Routes     []*Route          `json:"routes"`
	Metadata   map[string]string `json:"metadata"`
}

// ServiceRoute 服務路由配置
type ServiceRoute struct {
	ServiceName string        `json:"service_name"`
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	HealthPath  string        `json:"health_path"`
	Timeout     time.Duration `json:"timeout"`
	MaxBodySize int64         `json:"max_body_size"`
	Streaming   bool          `json:"streaming"`
}

// RouteConfig 路由配置集合
type RouteConfig struct {
	Routes   []*Route        `json:"routes"`
	Groups   []*RouteGroup   `json:"groups"`
	Services []*ServiceRoute `json:"services"`
	Version  string          `json:"version"`
}

// MatchesPattern 檢查路由是否匹配指定模式
func (r *Route) MatchesPattern(path string) bool {
	// 移除尾部斜線進行比較
	pattern := strings.TrimSuffix(r.Pattern, "/")
	path = strings.TrimSuffix(path, "/")

	// 處理萬用字元 /*
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}

	// 精確匹配
	return pattern == path
}

// MatchesMethod 檢查路由是否匹配指定方法
func (r *Route) MatchesMethod(method string) bool {
	if len(r.Methods) == 0 {
		return true // 如果沒有指定方法，匹配所有方法
	}

	for _, m := range r.Methods {
		if strings.EqualFold(m, method) || m == "*" {
			return true
		}
	}
	return false
}

// GetTargetURL 獲取目標服務 URL
func (sr *ServiceRoute) GetTargetURL() string {
	return fmt.Sprintf("http://%s:%d", sr.Host, sr.Port)
}

// IsHealthy 檢查服務是否健康
func (sr *ServiceRoute) IsHealthy() bool {
	if sr.HealthPath == "" {
		return true // 如果沒有健康檢查路徑，假設健康
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := sr.GetTargetURL() + sr.HealthPath
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// FindRoute 從路由配置中查找匹配的路由
func (rc *RouteConfig) FindRoute(path, method string) *RouteMatch {
	for _, route := range rc.Routes {
		if route.MatchesPattern(path) && route.MatchesMethod(method) {
			return &RouteMatch{
				Route:       route,
				PathParams:  extractPathParams(route.Pattern, path),
				QueryParams: make(map[string]string),
				IsMatch:     true,
			}
		}
	}

	// 檢查路由組
	for _, group := range rc.Groups {
		if strings.HasPrefix(path, group.Prefix) {
			for _, route := range group.Routes {
				fullPattern := group.Prefix + route.Pattern
				if matchPattern(fullPattern, path) && route.MatchesMethod(method) {
					return &RouteMatch{
						Route:       route,
						PathParams:  extractPathParams(fullPattern, path),
						QueryParams: make(map[string]string),
						IsMatch:     true,
					}
				}
			}
		}
	}

	return &RouteMatch{IsMatch: false}
}

// extractPathParams 提取路徑參數
func extractPathParams(pattern, path string) map[string]string {
	params := make(map[string]string)
	// 這裡可以實現更複雜的路徑參數提取邏輯
	// 目前返回空 map
	return params
}

// matchPattern 檢查路徑是否匹配模式
func matchPattern(pattern, path string) bool {
	// 簡化的模式匹配，可以根據需要擴展
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}
