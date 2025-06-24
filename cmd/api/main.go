package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/router"
	"expense-api-gateway/internal/service/discovery"
	"expense-api-gateway/internal/service/monitor"
	"expense-api-gateway/internal/service/proxy"
	"expense-api-gateway/pkg/healthcheck"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 設置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 載入配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日誌
	logger := initLogger(cfg)
	defer logger.Sync()

	logger.Info("Starting AI 智能報銷系統 API Gateway",
		zap.String("version", cfg.App.Version),
		zap.String("mode", cfg.App.Mode),
		zap.Int("port", cfg.App.Port))

	// 初始化服務發現
	serviceDiscovery := discovery.New(cfg, logger)

	// 初始化監控服務
	monitorService := monitor.New(cfg, logger)

	// 初始化健康檢查
	healthChecker := healthcheck.New()

	// 初始化路由解析器
	routeParser := proxy.NewRouteParser(cfg, logger)

	// 初始化代理服務
	proxyService := proxy.NewProxyService(cfg, logger, routeParser, serviceDiscovery)

	// 設置路由
	var r *gin.Engine
	if cfg.App.UseDynamicRouting {
		// 使用動態路由（基於 services.yaml）
		r = router.SetupWithProxy(cfg, logger, serviceDiscovery, monitorService, healthChecker, routeParser, proxyService)
	} else {
		// 使用靜態路由
		r = router.Setup(cfg, logger, serviceDiscovery, monitorService, healthChecker)
	}

	// 創建 HTTP 服務器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 啟動服務器
	go func() {
		logger.Info("Starting HTTP server", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 優雅關機
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// initLogger 初始化日誌
func initLogger(cfg *config.Config) *zap.Logger {
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = "stacktrace"

	if cfg.Log.Format == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		config.Encoding = "json"
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}
