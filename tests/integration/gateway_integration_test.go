package integration

import (
	"net/http"
	"os"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get("http://localhost:8088/health")
	if err != nil {
		t.Fatalf("/health 請求失敗: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/health 回應狀態碼錯誤: %d", resp.StatusCode)
	}
}

func TestSystemStatusEndpoint(t *testing.T) {
	resp, err := http.Get("http://localhost:8088/api/v1/system/status")
	if err != nil {
		t.Fatalf("/api/v1/system/status 請求失敗: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/v1/system/status 回應狀態碼錯誤: %d", resp.StatusCode)
	}
}

func TestSystemMetricsEndpoint(t *testing.T) {
	resp, err := http.Get("http://localhost:8088/api/v1/system/metrics")
	if err != nil {
		t.Fatalf("/api/v1/system/metrics 請求失敗: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/v1/system/metrics 回應狀態碼錯誤: %d", resp.StatusCode)
	}
}

// 更多測試可依需求擴充，如 JWT 驗證、RateLimit、CORS、代理、管理端點等

func TestMain(m *testing.M) {
	// 可在這裡啟動 gateway server（如需）
	os.Exit(m.Run())
}
