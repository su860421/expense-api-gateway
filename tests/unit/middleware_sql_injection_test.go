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

func TestSQLInjectionMiddleware(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置
	cfg := &config.Config{
		Security: config.SecurityConfig{
			SQLInjection: config.SQLInjectionConfig{
				Enabled: true,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 SQL 注入中間件
	sqlMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

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
			description:    "正常的 JSON 請求應該通過 SQL 注入檢查",
		},
		{
			name:           "包含 UNION SELECT 的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"id": "1 UNION SELECT * FROM users"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 UNION SELECT 的請求應該被阻擋",
		},
		{
			name:           "包含 DROP TABLE 的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"table": "DROP TABLE users"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 DROP TABLE 的請求應該被阻擋",
		},
		{
			name:           "包含註釋的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"comment": "test -- comment"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含註釋的請求應該被阻擋",
		},
		{
			name:           "包含分號的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"query": "SELECT * FROM users; DROP TABLE users"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含分號的請求應該被阻擋",
		},
		{
			name:           "包含 OR 運算符的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"condition": "1 OR 1=1"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含 OR 運算符的請求應該被阻擋",
		},
		{
			name:           "包含危險函數的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"function": "exec('cmd')"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含危險函數的請求應該被阻擋",
		},
		{
			name:           "包含編碼攻擊的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"encoded": "%27 UNION SELECT %2A FROM users"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含編碼攻擊的請求應該被阻擋",
		},
		{
			name:           "包含十六進制編碼的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"hex": "0x73656C656374"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含十六進制編碼的請求應該被阻擋",
		},
		{
			name:           "包含時間盲注的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"sleep": "sleep(5)"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含時間盲注的請求應該被阻擋",
		},
		{
			name:           "包含堆疊查詢的請求應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"stack": "; SELECT * FROM users"},
			expectedStatus: http.StatusBadRequest,
			description:    "包含堆疊查詢的請求應該被阻擋",
		},
		{
			name:           "查詢參數中的 SQL 注入應該被阻擋",
			method:         "GET",
			queryParams:    map[string]string{"id": "1' OR '1'='1"},
			expectedStatus: http.StatusBadRequest,
			description:    "查詢參數中的 SQL 注入應該被阻擋",
		},
		{
			name:           "表單數據中的 SQL 注入應該被阻擋",
			method:         "POST",
			contentType:    "application/x-www-form-urlencoded",
			body:           "username=admin'--&password=test",
			expectedStatus: http.StatusBadRequest,
			description:    "表單數據中的 SQL 注入應該被阻擋",
		},
		{
			name:           "多個危險關鍵字組合應該被阻擋",
			method:         "POST",
			contentType:    "application/json",
			body:           map[string]string{"query": "SELECT * FROM users WHERE id = 1 AND name = 'test'"},
			expectedStatus: http.StatusBadRequest,
			description:    "多個危險關鍵字組合應該被阻擋",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 創建測試路由
			router := gin.New()
			router.Use(sqlMiddleware.SQLInjectionProtection())
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

func TestSQLInjectionMiddlewareDisabled(t *testing.T) {
	// 設置測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試配置（禁用 SQL 注入防護）
	cfg := &config.Config{
		Security: config.SecurityConfig{
			SQLInjection: config.SQLInjectionConfig{
				Enabled: false,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 SQL 注入中間件
	sqlMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

	// 創建測試路由
	router := gin.New()
	router.Use(sqlMiddleware.SQLInjectionProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 創建包含 SQL 注入的請求
	body := map[string]string{"id": "1' OR '1'='1"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證響應（應該通過，因為 SQL 注入防護被禁用）
	assert.Equal(t, http.StatusOK, w.Code, "當 SQL 注入防護被禁用時，包含 SQL 注入的請求應該通過")
}

func TestSQLInjectionMiddlewareSanitizeSQL(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Security: config.SecurityConfig{
			SQLInjection: config.SQLInjectionConfig{
				Enabled: true,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 SQL 注入中間件
	sqlMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

	tests := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "SELECT * FROM users WHERE name = 'test'",
			expected:    "SELECT * FROM users WHERE name = 'test'",
			description: "正常的 SQL 應該保持不變",
		},
		{
			input:       "SELECT * FROM users -- comment",
			expected:    "SELECT * FROM users",
			description: "註釋應該被移除",
		},
		{
			input:       "SELECT * FROM users; DROP TABLE users",
			expected:    "SELECT * FROM users DROP TABLE users",
			description: "分號應該被移除",
		},
		{
			input:       "SELECT * FROM users WHERE id = 1 OR 1=1",
			expected:    "SELECT * FROM users WHERE id = 1 1=1",
			description: "OR 運算符應該被移除",
		},
		{
			input:       "exec('cmd')",
			expected:    "('cmd')",
			description: "危險函數應該被移除",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := sqlMiddleware.SanitizeSQL(tt.input)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestSQLInjectionMiddlewareValidateSQLInput(t *testing.T) {
	// 創建測試配置
	cfg := &config.Config{
		Security: config.SecurityConfig{
			SQLInjection: config.SQLInjectionConfig{
				Enabled: true,
			},
		},
	}

	// 創建日誌器
	logger := zap.NewNop()

	// 創建 SQL 注入中間件
	sqlMiddleware := security.NewSQLInjectionMiddleware(cfg, logger)

	tests := []struct {
		input       string
		shouldError bool
		description string
	}{
		{
			input:       "normal input",
			shouldError: false,
			description: "正常輸入應該通過驗證",
		},
		{
			input:       "1' OR '1'='1",
			shouldError: true,
			description: "SQL 注入輸入應該失敗驗證",
		},
		{
			input:       "UNION SELECT * FROM users",
			shouldError: true,
			description: "包含 UNION SELECT 的輸入應該失敗驗證",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := sqlMiddleware.ValidateSQLInput(tt.input)
			if tt.shouldError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}
