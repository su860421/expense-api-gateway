package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 配置結構
type Config struct {
	App       AppConfig       `yaml:"app"`
	Log       LogConfig       `yaml:"log"`
	JWT       JWTConfig       `yaml:"jwt"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Monitor   MonitorConfig   `yaml:"monitor"`
	CORS      CORSConfig      `yaml:"cors"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Security  SecurityConfig  `yaml:"security"`
}

// AppConfig 應用配置
type AppConfig struct {
	Name              string        `yaml:"name"`
	Version           string        `yaml:"version"`
	Port              int           `yaml:"port"`
	Mode              string        `yaml:"mode"`
	StartTime         time.Time     `yaml:"-"`
	UseDynamicRouting bool          `yaml:"use_dynamic_routing"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
}

// LogConfig 日誌配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret            string        `yaml:"secret"`
	Expiration        time.Duration `yaml:"expiration"`
	RefreshExpiration time.Duration `yaml:"refresh_expiration"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled     bool                     `yaml:"enabled"`
	GlobalLimit int                      `yaml:"global_limit"`
	IPLimit     RateLimitRule            `yaml:"ip_limit"`
	UserLimit   RateLimitRule            `yaml:"user_limit"`
	APILimit    map[string]RateLimitRule `yaml:"api_limit"`
}

// RateLimitRule 限流規則
type RateLimitRule struct {
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// MonitorConfig 監控配置
type MonitorConfig struct {
	Enabled           bool   `yaml:"enabled"`
	MetricsPath       string `yaml:"metrics_path"`
	PrometheusEnabled bool   `yaml:"prometheus_enabled"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAge         int      `yaml:"max_age"`
}

// DiscoveryConfig 服務發現配置
type DiscoveryConfig struct {
	Type     string                   `yaml:"type"`
	Interval time.Duration            `yaml:"interval"`
	Timeout  time.Duration            `yaml:"timeout"`
	Services map[string]ServiceConfig `yaml:"services"`
}

// ServiceConfig 服務配置
type ServiceConfig struct {
	Hosts       []string          `yaml:"hosts"`
	Port        int               `yaml:"port"`
	HealthCheck string            `yaml:"health_check"`
	Headers     map[string]string `yaml:"headers"`
	MaxBodySize int64             `yaml:"max_body_size"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	XSS          XSSConfig          `yaml:"xss"`
	SQLInjection SQLInjectionConfig `yaml:"sql_injection"`
}

// XSSConfig XSS 防護配置
type XSSConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SQLInjectionConfig SQL 注入防護配置
type SQLInjectionConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Load 載入配置
func Load(configPath string) (*Config, error) {
	// 如果沒有指定配置文件路徑，使用默認路徑
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// 讀取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 設置默認值
	config.setDefaults()

	// 驗證配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 設置啟動時間
	config.App.StartTime = time.Now()

	return &config, nil
}

// setDefaults 設置默認值
func (c *Config) setDefaults() {
	// App 配置默認值
	if c.App.Name == "" {
		c.App.Name = "expense-api-gateway"
	}
	if c.App.Version == "" {
		c.App.Version = "1.0.0"
	}
	if c.App.Port == 0 {
		c.App.Port = 8080
	}
	if c.App.Mode == "" {
		c.App.Mode = "release"
	}
	if c.App.ReadTimeout == 0 {
		c.App.ReadTimeout = 30 * time.Second
	}
	if c.App.WriteTimeout == 0 {
		c.App.WriteTimeout = 30 * time.Second
	}
	if c.App.IdleTimeout == 0 {
		c.App.IdleTimeout = 120 * time.Second
	}

	// 日誌配置默認值
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}

	// JWT 配置默認值
	if c.JWT.Secret == "" {
		c.JWT.Secret = "your-secret-key"
	}
	if c.JWT.Expiration == 0 {
		c.JWT.Expiration = 24 * time.Hour
	}
	if c.JWT.RefreshExpiration == 0 {
		c.JWT.RefreshExpiration = 7 * 24 * time.Hour
	}

	// 限流配置默認值
	if c.RateLimit.GlobalLimit == 0 {
		c.RateLimit.GlobalLimit = 1000
	}
	if c.RateLimit.IPLimit.Requests == 0 {
		c.RateLimit.IPLimit.Requests = 100
	}
	if c.RateLimit.IPLimit.Window == 0 {
		c.RateLimit.IPLimit.Window = time.Minute
	}
	if c.RateLimit.UserLimit.Requests == 0 {
		c.RateLimit.UserLimit.Requests = 200
	}
	if c.RateLimit.UserLimit.Window == 0 {
		c.RateLimit.UserLimit.Window = time.Minute
	}

	// 監控配置默認值
	if c.Monitor.MetricsPath == "" {
		c.Monitor.MetricsPath = "/metrics"
	}

	// CORS 配置默認值
	if len(c.CORS.AllowedOrigins) == 0 {
		c.CORS.AllowedOrigins = []string{"*"}
	}
	if len(c.CORS.AllowedMethods) == 0 {
		c.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.CORS.AllowedHeaders) == 0 {
		c.CORS.AllowedHeaders = []string{"*"}
	}
	if c.CORS.MaxAge == 0 {
		c.CORS.MaxAge = 86400
	}

	// 服務發現配置默認值
	if c.Discovery.Type == "" {
		c.Discovery.Type = "static"
	}
	if c.Discovery.Interval == 0 {
		c.Discovery.Interval = 30 * time.Second
	}
	if c.Discovery.Timeout == 0 {
		c.Discovery.Timeout = 5 * time.Second
	}

	// 安全配置默認值
	if !c.Security.XSS.Enabled {
		c.Security.XSS.Enabled = true // 默認啟用 XSS 防護
	}
	if !c.Security.SQLInjection.Enabled {
		c.Security.SQLInjection.Enabled = true // 默認啟用 SQL 注入防護
	}
}

// validate 驗證配置
func (c *Config) validate() error {
	// 驗證端口
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.App.Port)
	}

	// 驗證 JWT 密鑰
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// 驗證限流配置
	if c.RateLimit.Enabled {
		if c.RateLimit.GlobalLimit <= 0 {
			return fmt.Errorf("global rate limit must be positive")
		}
		if c.RateLimit.IPLimit.Requests < 0 {
			return fmt.Errorf("IP rate limit requests cannot be negative")
		}
		if c.RateLimit.UserLimit.Requests < 0 {
			return fmt.Errorf("user rate limit requests cannot be negative")
		}
	}

	return nil
}

// GetServiceConfig 獲取服務配置
func (c *Config) GetServiceConfig(serviceName string) (*ServiceConfig, bool) {
	config, exists := c.Discovery.Services[serviceName]
	return &config, exists
}

// IsServiceEnabled 檢查服務是否啟用
func (c *Config) IsServiceEnabled(serviceName string) bool {
	_, exists := c.Discovery.Services[serviceName]
	return exists
}
