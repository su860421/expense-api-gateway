package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"expense-api-gateway/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// XSSMiddleware XSS 防護中間件
type XSSMiddleware struct {
	config *config.Config
	logger *zap.Logger
	// XSS 攻擊模式
	xssPatterns []*regexp.Regexp
	// 允許的 HTML 標籤
	allowedTags map[string]bool
	// 允許的屬性
	allowedAttributes map[string]bool
}

// NewXSSMiddleware 創建新的 XSS 防護中間件
func NewXSSMiddleware(cfg *config.Config, logger *zap.Logger) *XSSMiddleware {
	middleware := &XSSMiddleware{
		config: cfg,
		logger: logger,
		xssPatterns: []*regexp.Regexp{
			// 腳本標籤
			regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
			// 事件處理器
			regexp.MustCompile(`(?i)on\w+\s*=`),
			// JavaScript 協議
			regexp.MustCompile(`(?i)javascript:`),
			// 數據協議
			regexp.MustCompile(`(?i)data:`),
			// VBScript
			regexp.MustCompile(`(?i)vbscript:`),
			// 表達式
			regexp.MustCompile(`(?i)expression\s*\(`),
			// 編碼的腳本
			regexp.MustCompile(`(?i)%3Cscript|%3cscript`),
			// 雙編碼
			regexp.MustCompile(`(?i)%253Cscript|%253cscript`),
			// 常見的 XSS 向量
			regexp.MustCompile(`(?i)<iframe|<object|<embed|<form`),
			// 危險的 CSS
			regexp.MustCompile(`(?i)expression\s*\(|url\s*\(.*javascript:`),
		},
		allowedTags: map[string]bool{
			"p": true, "div": true, "span": true, "br": true, "hr": true,
			"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
			"ul": true, "ol": true, "li": true, "table": true, "tr": true, "td": true, "th": true,
			"strong": true, "em": true, "b": true, "i": true, "u": true, "code": true, "pre": true,
			"a": true, "img": true, "blockquote": true, "cite": true,
		},
		allowedAttributes: map[string]bool{
			"href": true, "src": true, "alt": true, "title": true, "class": true,
			"id": true, "style": true, "width": true, "height": true,
		},
	}

	return middleware
}

// XSSProtection XSS 防護中間件
func (m *XSSMiddleware) XSSProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 檢查是否啟用 XSS 防護
		if !m.config.Security.XSS.Enabled {
			c.Next()
			return
		}

		// 設置安全標頭
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")

		// 檢查請求方法
		if c.Request.Method == http.MethodGet {
			// 檢查查詢參數
			if err := m.checkQueryParams(c); err != nil {
				m.logger.Warn("XSS attack detected in query params",
					zap.String("ip", c.ClientIP()),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
					"message": "XSS 攻擊檢測到，請勿提交惡意內容",
					"code":    "XSS_ATTACK_DETECTED",
				})
				c.Abort()
				return
			}
		} else if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			// 檢查請求體
			if err := m.checkRequestBody(c); err != nil {
				m.logger.Warn("XSS attack detected in request body",
					zap.String("ip", c.ClientIP()),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "error",
					"message": "XSS 攻擊檢測到，請勿提交惡意內容",
					"code":    "XSS_ATTACK_DETECTED",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// checkQueryParams 檢查查詢參數
func (m *XSSMiddleware) checkQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if m.isXSSAttack(value) {
				return fmt.Errorf("XSS attack detected in query param %s: %s", key, value)
			}
		}
	}
	return nil
}

// checkRequestBody 檢查請求體
func (m *XSSMiddleware) checkRequestBody(c *gin.Context) error {
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
func (m *XSSMiddleware) checkJSONBody(c *gin.Context) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	// 恢復請求體
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 檢查 JSON 字符串中的 XSS
	bodyStr := string(body)
	if m.isXSSAttack(bodyStr) {
		return fmt.Errorf("XSS attack detected in JSON body")
	}

	// 額外檢查：解析 JSON 並檢查每個字段
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		for _, value := range jsonData {
			if strValue, ok := value.(string); ok {
				if m.isXSSAttack(strValue) {
					return fmt.Errorf("XSS attack detected in JSON field")
				}
			}
		}
	}

	return nil
}

// checkFormBody 檢查表單請求體
func (m *XSSMiddleware) checkFormBody(c *gin.Context) error {
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	for key, values := range c.Request.PostForm {
		for _, value := range values {
			if m.isXSSAttack(value) {
				return fmt.Errorf("XSS attack detected in form field %s: %s", key, value)
			}
		}
	}

	return nil
}

// checkMultipartBody 檢查多部分表單請求體
func (m *XSSMiddleware) checkMultipartBody(c *gin.Context) error {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return err
	}

	// 檢查表單字段
	for key, values := range c.Request.MultipartForm.Value {
		for _, value := range values {
			if m.isXSSAttack(value) {
				return fmt.Errorf("XSS attack detected in multipart form field %s: %s", key, value)
			}
		}
	}

	return nil
}

// isXSSAttack 檢查是否為 XSS 攻擊
func (m *XSSMiddleware) isXSSAttack(input string) bool {
	// 檢查 XSS 模式
	for _, pattern := range m.xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// 檢查編碼的攻擊向量
	decoded := m.decodeURL(input)
	for _, pattern := range m.xssPatterns {
		if pattern.MatchString(decoded) {
			return true
		}
	}

	// 額外檢查腳本標籤（更寬鬆的匹配）
	if regexp.MustCompile(`(?i)<script`).MatchString(input) {
		return true
	}

	return false
}

// decodeURL 簡單的 URL 解碼
func (m *XSSMiddleware) decodeURL(input string) string {
	// 這裡可以實現更複雜的解碼邏輯
	// 目前只是簡單的字符串替換
	decoded := strings.ReplaceAll(input, "%3C", "<")
	decoded = strings.ReplaceAll(decoded, "%3E", ">")
	decoded = strings.ReplaceAll(decoded, "%22", "\"")
	decoded = strings.ReplaceAll(decoded, "%27", "'")
	decoded = strings.ReplaceAll(decoded, "%2F", "/")
	decoded = strings.ReplaceAll(decoded, "%3B", ";")
	decoded = strings.ReplaceAll(decoded, "%3D", "=")
	return decoded
}

// SanitizeHTML 清理 HTML 內容
func (m *XSSMiddleware) SanitizeHTML(input string) string {
	// 移除所有腳本標籤
	input = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(input, "")
	// 移除所有 onxxx 屬性（單引號、雙引號、無引號都能處理）
	input = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(['\"][^'\"]*['\"]|[^\s>]+)`).ReplaceAllString(input, "")
	// 將 href/src 等屬性中的 javascript: 前綴及內容全部移除
	input = regexp.MustCompile(`(?i)(href|src)\s*=\s*['\"]?javascript:[^'\">]*['\"]?`).ReplaceAllString(input, "$1=\"\"")
	// 再將 href/src 屬性後多餘的內容（如 xss')）移除
	input = regexp.MustCompile(`(?i)(href|src)=\"\"[^\s>]*`).ReplaceAllString(input, "$1=\"\"")
	// 其他協議
	input = regexp.MustCompile(`(?i)vbscript:`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)data:`).ReplaceAllString(input, "")
	// 移除 iframe 標籤
	input = regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`).ReplaceAllString(input, "")
	// 移除危險的 CSS 表達式
	input = regexp.MustCompile(`(?i)expression\s*\(`).ReplaceAllString(input, "")
	// 清理多餘的空格
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	input = strings.TrimSpace(input)
	return input
}
