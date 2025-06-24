package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/middleware/security"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestXSSMiddleware(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		Security: config.SecurityConfig{
			XSS: config.XSSConfig{
				Enabled: true,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 XSS 中間件
	xssMiddleware := security.NewXSSMiddleware(cfg, logger)

	tests := []struct {
		name           string
		method         string
		contentType    string
		body           interface{}
		queryParams    map[string]string
		expectedStatus int
		description    string
	}{
		{
			name:           "正常請求應該通過",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"name": "test", "email": "test@example.com"},
			expectedStatus: http.StatusOK,
			description:    "正常的 JSON 請求應該通過 XSS 檢查",
		},
		{
			name:           "包含腳本標籤的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"name": "<script>alert('xss')</script>"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 script 標籤的請求應該被阻擋",
		},
		{
			name:           "包含事件處理器的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"name": "<img src=x onerror=alert('xss')>"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含事件處理器的請求應該被阻擋",
		},
		{
			name:           "包含 JavaScript 協議的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"url": "javascript:alert('xss')"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 JavaScript 協議的請求應該被阻擋",
		},
		{
			name:           "包含編碼腳本的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"name": "%3Cscript%3Ealert('xss')%3C/script%3E"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含編碼腳本的請求應該被阻擋",
		},
		{
			name:           "查詢參數中的 XSS 應該被阻擋",
			method:         "GET",
			queryParams:    map[string]string{"q": "<script>alert('xss')</script>"},
			expectedStatus: http.StatusBadRequest,
			description:    "查詢參數中的 XSS 應該被阻擋",
		},
		{
			name:           "表單數據中的 XSS 應該被阻擋",
			method:         "POST",
			contentType:    "application/x-www-form-urlencoded",
			body:           "name=<script>alert('xss')</script>&email=test@example.com",
			expectedStatus: http.StatusBadRequest,
			description:    "表單數據中的 XSS 應該被阻擋",
		},
		{
			name:           "包含 iframe 的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"content": "<iframe src='javascript:alert(\"xss\")'></iframe>"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 iframe 的請求應該被阻擋",
		},
		{
			name:           "包含危險 CSS 的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"style": "expression(alert('xss'))"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含危險 CSS 的請求應該被阻擋",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 創建測試路由
			router := gin.New()
			router.Use(xssMiddleware.XSSProtection())
			router.Any("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// 創建請求
			var req *http.Request
			if tt.method == "GET" {
				req = httptest.NewRequest(tt.method, "/test", nil)
				// 添加查詢參數
				q := req.URL.Query()
				for key, value := range tt.queryParams {
					q.Add(key, value)
				}
				req.URL.RawQuery = q.Encode()
			} else {
				var body []byte
				var err error
				if tt.contentType == "application/json" {
					body, err = json.Marshal(tt.body)
					assert.NoError(t, err)
				} else {
					body = []byte(tt.body.(string))
				}
				req = httptest.NewRequest(tt.method, "/test", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", tt.contentType)
			}

			// 創建響應記錄器
			w := httptest.NewRecorder()

			// 執行請求
			router.ServeHTTP(w, req)

			// 驗證響應
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
		})
	}
}

func TestXSSMiddlewareDisabled(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置（禁用 XSS 防護）
	cfg := &config.Config{
		Security: config.SecurityConfig{
			XSS: config.XSSConfig{
				Enabled: false,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 XSS 中間件
	xssMiddleware := security.NewXSSMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(xssMiddleware.XSSProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建包含 XSS 的請求
	body := map[string]string{"name": "<script>alert('xss')</script>"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證響應（應該通過，因為 XSS 防護被禁用）
	assert.Equal(t, http.StatusOK, w.Code, "當 XSS 防護被禁用時，包含 XSS 的請求應該通過")
}

func TestXSSMiddlewareSanitizeHTML(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Security: config.SecurityConfig{
			XSS: config.XSSConfig{
				Enabled: true,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 XSS 中間件
	xssMiddleware := security.NewXSSMiddleware(cfg, logger)

	tests := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "<p>正常文本</p>",
			expected:    "<p>正常文本</p>",
			description: "正常的 HTML 應該保持不變",
		},
		{
			input:       "<script>alert('xss')</script><p>文本</p>",
			expected:    "<p>文本</p>",
			description: "腳本標籤應該被移除",
		},
		{
			input:       "<img src=x onerror=alert('xss')>",
			expected:    "<img src=x>",
			description: "事件處理器應該被移除",
		},
		{
			input:       "<a href=\"javascript:alert('xss')\">鏈接</a>",
			expected:    "<a href=\"\">鏈接</a>",
			description: "JavaScript 協議應該被移除",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := xssMiddleware.SanitizeHTML(tt.input)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}
