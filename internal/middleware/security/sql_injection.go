package security

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"expense-api-gateway/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SQLInjectionMiddleware SQL 注入防護中間件
type SQLInjectionMiddleware struct {
	config *config.Config
	logger *zap.Logger
	// SQL 注入攻擊模式
	sqlPatterns []*regexp.Regexp
	// 危險的 SQL 關鍵字
	dangerousKeywords map[string]bool
	// 危險的函數
	dangerousFunctions map[string]bool
}

// NewSQLInjectionMiddleware 創建新的 SQL 注入防護中間件
func NewSQLInjectionMiddleware(cfg *config.Config, logger *zap.Logger) *SQLInjectionMiddleware {
	middleware := &SQLInjectionMiddleware{
		config: cfg,
		logger: logger,
		sqlPatterns: []*regexp.Regexp{
			// 基本的 SQL 注入模式
			regexp.MustCompile(`(?i)(union\s+select|select\s+union)`),
			regexp.MustCompile(`(?i)(insert\s+into|update\s+set|delete\s+from|drop\s+table|create\s+table)`),
			regexp.MustCompile(`(?i)(alter\s+table|truncate\s+table|backup\s+database)`),
			// 註釋
			regexp.MustCompile(`(?i)(--|\#|\/\*|\*\/)`),
			// 分號
			regexp.MustCompile(`(?i);\s*$`),
			// 布爾運算符
			regexp.MustCompile(`(?i)(\bor\b|\band\b|\bnot\b)`),
			// 危險函數
			regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_executesql\s*\(|xp_cmdshell\s*\()`),
			regexp.MustCompile(`(?i)(load_file\s*\(|into\s+outfile|into\s+dumpfile)`),
			// 編碼的攻擊
			regexp.MustCompile(`(?i)(%27|%22|%3B|%2D%2D)`),
			// 雙編碼
			regexp.MustCompile(`(?i)(%2527|%2522|%253B|%252D%252D)`),
			// 十六進制編碼
			regexp.MustCompile(`(?i)(0x[0-9a-fA-F]+)`),
			// 時間盲注
			regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\s*\(|waitfor\s+delay)`),
			// 堆疊查詢
			regexp.MustCompile(`(?i)(;\s*select|;\s*insert|;\s*update|;\s*delete|;\s*drop)`),
		},
		dangerousKeywords: map[string]bool{
			"union": true, "select": true, "insert": true, "update": true, "delete": true,
			"drop": true, "create": true, "alter": true, "truncate": true, "backup": true,
			"restore": true, "grant": true, "revoke": true, "deny": true,
		},
		dangerousFunctions: map[string]bool{
			"exec": true, "execute": true, "sp_executesql": true, "xp_cmdshell": true,
			"load_file": true, "into outfile": true, "into dumpfile": true,
			"sleep": true, "benchmark": true, "waitfor": true,
		},
	}

	return middleware
}

// SQLInjectionProtection SQL 注入防護中間件
func (m *SQLInjectionMiddleware) SQLInjectionProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 檢查是否啟用 SQL 注入防護
		if !m.config.Security.SQLInjection.Enabled {
			c.Next()
			return
		}

		// 檢查請求方法
		if c.Request.Method == http.MethodGet {
			// 檢查查詢參數
			if err := m.checkQueryParams(c); err != nil {
				m.logger.Warn("SQL injection attack detected in query params",
					zap.String("ip", c.ClientIP()),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
					"message": "SQL 注入攻擊檢測到，請勿提交惡意內容",
					"code":    "SQL_INJECTION_DETECTED",
				})
				c.Abort()
				return
			}
		} else if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			// 檢查請求體
			if err := m.checkRequestBody(c); err != nil {
				m.logger.Warn("SQL injection attack detected in request body",
					zap.String("ip", c.ClientIP()),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
					"message": "SQL 注入攻擊檢測到，請勿提交惡意內容",
					"code":    "SQL_INJECTION_DETECTED",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// checkQueryParams 檢查查詢參數
func (m *SQLInjectionMiddleware) checkQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if m.isSQLInjection(value) {
				return fmt.Errorf("SQL injection detected in query param %s: %s", key, value)
			}
		}
	}
	return nil
}

// checkRequestBody 檢查請求體
func (m *SQLInjectionMiddleware) checkRequestBody(c *gin.Context) error {
	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "application/json") {
		return m.checkJSONBody(c)
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		return m.checkFormBody(c)
	} else if strings.Contains(contentType, "multipart/form-data") {
		return m.checkMultipartBody(c)
	}

	return nil
}

// checkJSONBody 檢查 JSON 請求體
func (m *SQLInjectionMiddleware) checkJSONBody(c *gin.Context) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	// 恢復請求體
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 檢查 JSON 字符串中的 SQL 注入
	bodyStr := string(body)
	if m.isSQLInjection(bodyStr) {
		return fmt.Errorf("SQL injection detected in JSON body")
	}

	return nil
}

// checkFormBody 檢查表單請求體
func (m *SQLInjectionMiddleware) checkFormBody(c *gin.Context) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	for key, values := range c.Request.PostForm {
		for _, value := range values {
			if m.isSQLInjection(value) {
				return fmt.Errorf("SQL injection detected in form field %s: %s", key, value)
			}
		}
	}

	return nil
}

// checkMultipartBody 檢查多部分表單請求體
func (m *SQLInjectionMiddleware) checkMultipartBody(c *gin.Context) error {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return err
	}

	// 檢查表單字段
	for key, values := range c.Request.MultipartForm.Value {
		for _, value := range values {
			if m.isSQLInjection(value) {
				return fmt.Errorf("SQL injection detected in multipart form field %s: %s", key, value)
			}
		}
	}

	return nil
}

// isSQLInjection 檢查是否為 SQL 注入攻擊
func (m *SQLInjectionMiddleware) isSQLInjection(input string) bool {
	// 檢查 SQL 注入模式
	for _, pattern := range m.sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// 檢查編碼的攻擊向量
	decoded := m.decodeURL(input)
	for _, pattern := range m.sqlPatterns {
		if pattern.MatchString(decoded) {
			return true
		}
	}

	// 檢查危險關鍵字組合
	if m.checkDangerousCombinations(input) {
		return true
	}

	return false
}

// checkDangerousCombinations 檢查危險的關鍵字組合
func (m *SQLInjectionMiddleware) checkDangerousCombinations(input string) bool {
	input = strings.ToLower(input)

	// 檢查多個危險關鍵字的組合
	dangerousCount := 0
	for keyword := range m.dangerousKeywords {
		if strings.Contains(input, keyword) {
			dangerousCount++
		}
	}

	// 如果包含多個危險關鍵字，可能是攻擊
	if dangerousCount >= 2 {
		return true
	}

	// 檢查危險函數
	for function := range m.dangerousFunctions {
		if strings.Contains(input, function) {
			return true
		}
	}

	return false
}

// decodeURL 簡單的 URL 解碼
func (m *SQLInjectionMiddleware) decodeURL(input string) string {
	// 這裡可以實現更複雜的解碼邏輯
	// 目前只是簡單的字符串替換
	decoded := strings.ReplaceAll(input, "%27", "'")
	decoded = strings.ReplaceAll(decoded, "%22", "\"")
	decoded = strings.ReplaceAll(decoded, "%3B", ";")
	decoded = strings.ReplaceAll(decoded, "%2D%2D", "--")
	decoded = strings.ReplaceAll(decoded, "%23", "#")
	decoded = strings.ReplaceAll(decoded, "%2F%2A", "/*")
	decoded = strings.ReplaceAll(decoded, "%2A%2F", "*/")
	return decoded
}

// SanitizeSQL 清理 SQL 輸入
func (m *SQLInjectionMiddleware) SanitizeSQL(input string) string {
	// 移除 -- 或 # 之後到行尾
	input = regexp.MustCompile(`(?i)(--|#).*?$`).ReplaceAllString(input, "")
	// 移除 /* ... */ 區塊
	input = regexp.MustCompile(`(?is)/\*.*?\*/`).ReplaceAllString(input, "")
	// 移除所有分號
	input = strings.ReplaceAll(input, ";", "")
	// 危險函數
	for function := range m.dangerousFunctions {
		input = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(function)+`\s*\(`).ReplaceAllString(input, "(")
	}
	// 移除編碼的攻擊向量
	input = regexp.MustCompile(`(?i)(%27|%22|%3B|%2D%2D|%23|%2F%2A|%2A%2F)`).ReplaceAllString(input, "")
	// 清理多餘的空格
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	input = strings.TrimSpace(input)
	return input
}

// ValidateSQLInput 驗證 SQL 輸入
func (m *SQLInjectionMiddleware) ValidateSQLInput(input string) error {
	if m.isSQLInjection(input) {
		return fmt.Errorf("invalid SQL input detected")
	}
	return nil
}
