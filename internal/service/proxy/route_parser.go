package proxy

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// RouteConfig 路由配置
// 注意：service 對應 services.yaml 的 key
// pattern 支援 /xxx/*
type RouteConfig struct {
	ID           string            `yaml:"id"`
	Pattern      string            `yaml:"pattern"`
	Service      string            `yaml:"service"`
	Methods      []string          `yaml:"methods"`
	AuthRequired bool              `yaml:"auth_required"`
	Roles        []string          `yaml:"roles"`
	Timeout      time.Duration     `yaml:"timeout"`
	MaxBodySize  int64             `yaml:"max_body_size"`
	Streaming    bool              `yaml:"streaming"`
	Headers      map[string]string `yaml:"headers"`
	StripPrefix  bool              `yaml:"strip_prefix"`
	RewritePath  string            `yaml:"rewrite_path"`
}

// ServiceConfig 服務配置
// 注意：key 為 service 名稱
// hosts 支援多 host
// port、health_check、headers、max_body_size
// ...
type ServiceConfig struct {
	Hosts       []string          `yaml:"hosts"`
	Port        int               `yaml:"port"`
	HealthCheck string            `yaml:"health_check"`
	Timeout     time.Duration     `yaml:"timeout"`
	MaxBodySize int64             `yaml:"max_body_size"`
	Headers     map[string]string `yaml:"headers"`
}

// RouteGroup 路由組配置
type RouteGroup struct {
	Name       string        `yaml:"name"`
	Prefix     string        `yaml:"prefix"`
	Middleware []string      `yaml:"middleware"`
	Routes     []RouteConfig `yaml:"routes"`
}

// ServicesConfig services.yaml 結構
type ServicesConfig struct {
	Groups   []RouteGroup             `yaml:"groups"`
	Routes   []RouteConfig            `yaml:"routes"`
	Services map[string]ServiceConfig `yaml:"services"`
	Version  string                   `yaml:"version"`
}

// RouteParser 路由解析器
type RouteParser struct {
	logger     *zap.Logger
	routes     []*RouteConfig
	services   map[string]*ServiceConfig
	groups     []*RouteGroup
	mutex      sync.RWMutex
	lastReload time.Time
	filePath   string
}

// NewRouteParser 創建新的路由解析器
func NewRouteParser(_ interface{}, logger *zap.Logger) *RouteParser {
	filePath := "configs/services.yaml"
	return &RouteParser{
		logger:   logger,
		routes:   make([]*RouteConfig, 0),
		services: make(map[string]*ServiceConfig),
		groups:   make([]*RouteGroup, 0),
		filePath: filePath,
	}
}

// LoadConfig 載入路由配置
func (p *RouteParser) LoadConfig() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var servicesConfig ServicesConfig
	if err := yaml.Unmarshal(data, &servicesConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	p.routes = make([]*RouteConfig, 0)
	p.services = make(map[string]*ServiceConfig)
	p.groups = make([]*RouteGroup, 0)

	for name, service := range servicesConfig.Services {
		serviceCopy := service
		p.services[name] = &serviceCopy
	}
	for _, route := range servicesConfig.Routes {
		routeCopy := route
		p.routes = append(p.routes, &routeCopy)
	}
	for _, group := range servicesConfig.Groups {
		groupCopy := group
		p.groups = append(p.groups, &groupCopy)
	}

	p.lastReload = time.Now()
	if p.logger != nil {
		p.logger.Info("Route configuration loaded successfully",
			zap.Int("routes", len(p.routes)),
			zap.Int("services", len(p.services)),
			zap.Int("groups", len(p.groups)),
			zap.String("version", servicesConfig.Version))
	}
	return nil
}

// MatchRoute 匹配路由
func (p *RouteParser) MatchRoute(method, path string) (*RouteConfig, *ServiceConfig, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 先檢查 group
	for _, group := range p.groups {
		if strings.HasPrefix(path, group.Prefix) {
			for _, route := range group.Routes {
				if matchRoutePattern(route.Pattern, path) && matchMethod(route.Methods, method) {
					service, exists := p.services[route.Service]
					if !exists {
						return nil, nil, fmt.Errorf("service not found: %s", route.Service)
					}
					return &route, service, nil
				}
			}
		}
	}
	// 再檢查全局 routes
	for _, route := range p.routes {
		if matchRoutePattern(route.Pattern, path) && matchMethod(route.Methods, method) {
			service, exists := p.services[route.Service]
			if !exists {
				return nil, nil, fmt.Errorf("service not found: %s", route.Service)
			}
			return route, service, nil
		}
	}
	return nil, nil, fmt.Errorf("no route found for %s %s", method, path)
}

func matchRoutePattern(pattern, path string) bool {
	regexPattern := patternToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, path)
	if err != nil {
		return false
	}
	return matched
}

func patternToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	pattern = regexp.MustCompile(`\\/:([^/]+)`).ReplaceAllString(pattern, "/[^/]+")
	if !strings.HasSuffix(pattern, "$") {
		pattern += "$"
	}
	return pattern
}

func matchMethod(allowedMethods []string, requestMethod string) bool {
	if len(allowedMethods) == 0 {
		return true
	}
	for _, m := range allowedMethods {
		if strings.EqualFold(m, requestMethod) {
			return true
		}
	}
	return false
}

// GetAllRoutes 取得所有路由
func (p *RouteParser) GetAllRoutes() []*RouteConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.routes
}

// GetAllGroups 取得所有分組
func (p *RouteParser) GetAllGroups() []*RouteGroup {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.groups
}

// GetAllServices 取得所有服務
func (p *RouteParser) GetAllServices() map[string]*ServiceConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.services
}

// GetLastReloadTime 取得最後重載時間
func (p *RouteParser) GetLastReloadTime() time.Time {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.lastReload
}
