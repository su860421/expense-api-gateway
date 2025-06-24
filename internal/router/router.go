package router

import (
	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/handler"
	"expense-api-gateway/internal/middleware/auth"
	"expense-api-gateway/internal/middleware/cors"
	"expense-api-gateway/internal/middleware/logging"
	"expense-api-gateway/internal/middleware/ratelimit"
	"expense-api-gateway/internal/middleware/security"
	"expense-api-gateway/internal/service/discovery"
	"expense-api-gateway/internal/service/monitor"
	"expense-api-gateway/internal/service/proxy"
	"expense-api-gateway/pkg/healthcheck"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Setup 設置路由
func Setup(
	cfg *config.Config,
	logger *zap.Logger,
	serviceDiscovery discovery.ServiceDiscovery,
	monitorService *monitor.Monitor,
	healthChecker *healthcheck.HealthChecker,
) *gin.Engine {
	// 創建 Gin 引擎

	r := gin.New()

	// 初始化中間件
	jwtMiddleware := auth.NewJWTMiddleware(cfg, logger)
	rateLimitMiddleware := ratelimit.NewRateLimitMiddleware(cfg, logger)
	xssMiddleware := security.NewXSSMiddleware(cfg, logger)
	sqlInjectionMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

	// 添加全局中間件
	r.Use(logging.Middleware(logger, monitorService))
	r.Use(cors.Middleware(cfg))
	r.Use(gin.Recovery())
	r.Use(xssMiddleware.XSSProtection())
	r.Use(sqlInjectionMiddleware.SQLInjectionProtection())

	// 添加限流中間件
	if cfg.RateLimit.Enabled {
		r.Use(rateLimitMiddleware.GlobalRateLimit())
		r.Use(rateLimitMiddleware.IPRateLimit())
	}

	// 創建處理器
	h := handler.New(cfg, logger, serviceDiscovery, monitorService)

	// 健康檢查路由
	r.GET("/health", healthChecker.Handler())

	// API 版本管理
	v1 := r.Group("/api/v1")
	{
		// 系統管理路由
		system := v1.Group("/system")
		{
			system.GET("/status", h.GetSystemStatus)
			system.GET("/metrics", h.GetMetrics)
			system.POST("/metrics/reset", h.ResetMetrics)
		}

		// 服務發現路由
		services := v1.Group("/services")
		{
			services.GET("", h.ListServices)
			services.GET("/:name", h.GetService)
			services.POST("/:name/register", h.RegisterService)
			services.DELETE("/:name/:id", h.DeregisterService)
		}

		// 通用代理路由（用於其他服務）
		proxy := v1.Group("/proxy")
		proxy.Use(jwtMiddleware.OptionalAuth()) // 可選認證
		{
			proxy.Any("/*path", h.ProxyHandler)
		}
	}

	// 管理端點
	admin := r.Group("/admin")
	admin.Use(jwtMiddleware.Authenticate())
	admin.Use(jwtMiddleware.RequireRoles("admin"))
	{
		admin.GET("/config", h.GetConfig)
		admin.POST("/config/reload", h.ReloadConfig)
		admin.GET("/routes", h.GetRoutes)
		admin.POST("/maintenance", h.ToggleMaintenanceMode)
		admin.GET("/rate-limit/stats", h.GetRateLimitStats)
		admin.POST("/rate-limit/reset", h.ResetRateLimit)
	}

	// 監控端點
	if cfg.Monitor.Enabled {
		r.GET(cfg.Monitor.MetricsPath, h.GetPrometheusMetrics)
	}

	return r
}

// SetupWithProxy 設置路由（使用新的代理服務）
func SetupWithProxy(
	cfg *config.Config,
	logger *zap.Logger,
	serviceDiscovery discovery.ServiceDiscovery,
	monitorService *monitor.Monitor,
	healthChecker *healthcheck.HealthChecker,
	routeParser *proxy.RouteParser,
	proxyService *proxy.ProxyService,
) *gin.Engine {
	// 創建 Gin 引擎
	r := gin.New()

	// 初始化中間件
	jwtMiddleware := auth.NewJWTMiddleware(cfg, logger)
	rateLimitMiddleware := ratelimit.NewRateLimitMiddleware(cfg, logger)
	xssMiddleware := security.NewXSSMiddleware(cfg, logger)
	sqlInjectionMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

	// 添加全局中間件
	r.Use(logging.Middleware(logger, monitorService))
	r.Use(cors.Middleware(cfg))
	r.Use(gin.Recovery())
	r.Use(xssMiddleware.XSSProtection())
	r.Use(sqlInjectionMiddleware.SQLInjectionProtection())

	// 添加限流中間件
	if cfg.RateLimit.Enabled {
		r.Use(rateLimitMiddleware.GlobalRateLimit())
		r.Use(rateLimitMiddleware.IPRateLimit())
		r.Use(rateLimitMiddleware.UserRateLimit())
		r.Use(rateLimitMiddleware.APIRateLimit())
	}

	// 創建處理器
	h := handler.NewWithProxy(cfg, logger, serviceDiscovery, monitorService, proxyService)

	// 健康檢查路由
	r.GET("/health", healthChecker.Handler())

	// 系統端點（由 Gateway 自己處理）
	v1 := r.Group("/api/v1")
	{
		// 系統狀態端點
		v1.GET("/system/status", h.GetSystemStatus)
		v1.GET("/system/metrics", h.GetMetrics)
		v1.POST("/system/metrics/reset", h.ResetMetrics)

		// 服務管理端點
		v1.GET("/services", h.ListServices)
		v1.GET("/services/:service", h.GetService)
		v1.POST("/services", h.RegisterService)
		v1.DELETE("/services/:service", h.DeregisterService)

		// 通用代理路由（用於其他服務）
		proxy := v1.Group("/proxy")
		proxy.Use(jwtMiddleware.OptionalAuth()) // 可選認證
		{
			proxy.Any("/*path", h.ProxyHandler)
		}
	}

	// 動態路由（基於 services.yaml 配置）
	setupDynamicRoutes(r, jwtMiddleware, h, routeParser)

	// 管理端點
	admin := r.Group("/admin")
	admin.Use(jwtMiddleware.Authenticate())
	admin.Use(jwtMiddleware.RequireRoles("admin"))
	{
		admin.GET("/config", h.GetConfig)
		admin.POST("/config/reload", h.ReloadConfig)
		admin.GET("/routes", h.GetRoutes)
		admin.POST("/maintenance", h.ToggleMaintenanceMode)
		admin.GET("/rate-limit/stats", h.GetRateLimitStats)
		admin.POST("/rate-limit/reset", h.ResetRateLimit)
		admin.GET("/proxy/stats", h.GetProxyStats)
	}

	// 監控端點
	if cfg.Monitor.Enabled {
		r.GET(cfg.Monitor.MetricsPath, h.GetPrometheusMetrics)
	}

	return r
}

// setupDynamicRoutes 設置動態路由
func setupDynamicRoutes(
	r *gin.Engine,
	jwtMiddleware *auth.JWTMiddleware,
	h *handler.Handler,
	routeParser *proxy.RouteParser,
) {
	// 載入路由配置
	if err := routeParser.LoadConfig(); err != nil {
		// 如果載入失敗，使用靜態路由
		return
	}

	// 獲取所有路由配置
	routes := routeParser.GetAllRoutes()
	groups := routeParser.GetAllGroups()

	// 設置路由組
	for _, group := range groups {
		groupRouter := r.Group(group.Prefix)

		// 添加組級別中間件
		for _, middlewareName := range group.Middleware {
			switch middlewareName {
			case "auth":
				groupRouter.Use(jwtMiddleware.Authenticate())
			case "cors":
				// CORS 已在全局設置
			case "ratelimit":
				// 限流已在全局設置
			}
		}

		// 設置組內路由
		for _, route := range group.Routes {
			setupRoute(groupRouter, &route, jwtMiddleware, h)
		}
	}

	// 設置全局路由
	for _, route := range routes {
		// 為全局路由創建一個組
		routeGroup := r.Group("")
		setupRoute(routeGroup, route, jwtMiddleware, h)
	}
}

// setupRoute 設置單個路由
func setupRoute(
	router interface{},
	route *proxy.RouteConfig,
	jwtMiddleware *auth.JWTMiddleware,
	h *handler.Handler,
) {
	// 確定 HTTP 方法
	methods := route.Methods
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}

	// 設置中間件
	var handlers []gin.HandlerFunc

	// 添加認證中間件
	if route.AuthRequired {
		handlers = append(handlers, jwtMiddleware.Authenticate())

		// 添加角色檢查
		if len(route.Roles) > 0 {
			handlers = append(handlers, jwtMiddleware.RequireRoles(route.Roles...))
		}
	}

	// 添加代理處理器
	handlers = append(handlers, h.ProxyHandler)

	// 根據路由器類型註冊路由
	switch r := router.(type) {
	case *gin.Engine:
		for _, method := range methods {
			switch method {
			case "GET":
				r.GET(route.Pattern, handlers...)
			case "POST":
				r.POST(route.Pattern, handlers...)
			case "PUT":
				r.PUT(route.Pattern, handlers...)
			case "DELETE":
				r.DELETE(route.Pattern, handlers...)
			case "PATCH":
				r.PATCH(route.Pattern, handlers...)
			case "OPTIONS":
				r.OPTIONS(route.Pattern, handlers...)
			case "HEAD":
				r.HEAD(route.Pattern, handlers...)
			}
		}
	case *gin.RouterGroup:
		for _, method := range methods {
			switch method {
			case "GET":
				r.GET(route.Pattern, handlers...)
			case "POST":
				r.POST(route.Pattern, handlers...)
			case "PUT":
				r.PUT(route.Pattern, handlers...)
			case "DELETE":
				r.DELETE(route.Pattern, handlers...)
			case "PATCH":
				r.PATCH(route.Pattern, handlers...)
			case "OPTIONS":
				r.OPTIONS(route.Pattern, handlers...)
			case "HEAD":
				r.HEAD(route.Pattern, handlers...)
			}
		}
	}
}
