package domain

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorCode 錯誤碼類型
type ErrorCode string

const (
	// 認證相關錯誤
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeInsufficientRole ErrorCode = "INSUFFICIENT_ROLE"

	// 路由相關錯誤
	ErrCodeRouteNotFound   ErrorCode = "ROUTE_NOT_FOUND"
	ErrCodeServiceNotFound ErrorCode = "SERVICE_NOT_FOUND"
	ErrCodeServiceDown     ErrorCode = "SERVICE_DOWN"
	ErrCodeInvalidRoute    ErrorCode = "INVALID_ROUTE"

	// 請求相關錯誤
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrCodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodePayloadTooLarge  ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrCodeTimeout          ErrorCode = "TIMEOUT"

	// 系統相關錯誤
	ErrCodeInternalError ErrorCode = "INTERNAL_ERROR"
	ErrCodeConfigError   ErrorCode = "CONFIG_ERROR"
	ErrCodeNetworkError  ErrorCode = "NETWORK_ERROR"

	// 限流相關錯誤
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeTooManyRequests   ErrorCode = "TOO_MANY_REQUESTS"
)

// GatewayError 網關錯誤
type GatewayError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	StatusCode int       `json:"status_code"`
	Cause      error     `json:"-"`
}

// Error 實現 error 接口
func (e *GatewayError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 實現 error unwrapping
func (e *GatewayError) Unwrap() error {
	return e.Cause
}

// NewGatewayError 創建新的網關錯誤
func NewGatewayError(code ErrorCode, message string, statusCode int) *GatewayError {
	return &GatewayError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithDetail 添加錯誤詳情
func (e *GatewayError) WithDetail(detail string) *GatewayError {
	e.Detail = detail
	return e
}

// WithCause 添加原因錯誤
func (e *GatewayError) WithCause(cause error) *GatewayError {
	e.Cause = cause
	return e
}

// 預定義錯誤
var (
	ErrUnauthorized = NewGatewayError(
		ErrCodeUnauthorized,
		"Authentication required",
		http.StatusUnauthorized,
	)

	ErrInvalidToken = NewGatewayError(
		ErrCodeInvalidToken,
		"Invalid token",
		http.StatusUnauthorized,
	)

	ErrTokenExpired = NewGatewayError(
		ErrCodeTokenExpired,
		"Token has expired",
		http.StatusUnauthorized,
	)

	ErrForbidden = NewGatewayError(
		ErrCodeForbidden,
		"Access denied",
		http.StatusForbidden,
	)

	ErrInsufficientRole = NewGatewayError(
		ErrCodeInsufficientRole,
		"Insufficient role permissions",
		http.StatusForbidden,
	)

	ErrRouteNotFound = NewGatewayError(
		ErrCodeRouteNotFound,
		"Route not found",
		http.StatusNotFound,
	)

	ErrServiceNotFound = NewGatewayError(
		ErrCodeServiceNotFound,
		"Target service not found",
		http.StatusNotFound,
	)

	ErrServiceDown = NewGatewayError(
		ErrCodeServiceDown,
		"Target service is unavailable",
		http.StatusServiceUnavailable,
	)

	ErrBadRequest = NewGatewayError(
		ErrCodeBadRequest,
		"Bad request",
		http.StatusBadRequest,
	)

	ErrMethodNotAllowed = NewGatewayError(
		ErrCodeMethodNotAllowed,
		"HTTP method not allowed",
		http.StatusMethodNotAllowed,
	)

	ErrPayloadTooLarge = NewGatewayError(
		ErrCodePayloadTooLarge,
		"Request payload too large",
		http.StatusRequestEntityTooLarge,
	)

	ErrTimeout = NewGatewayError(
		ErrCodeTimeout,
		"Request timeout",
		http.StatusGatewayTimeout,
	)

	ErrInternalError = NewGatewayError(
		ErrCodeInternalError,
		"Internal server error",
		http.StatusInternalServerError,
	)

	ErrRateLimitExceeded = NewGatewayError(
		ErrCodeRateLimitExceeded,
		"Rate limit exceeded",
		http.StatusTooManyRequests,
	)
)

// IsGatewayError 檢查是否為網關錯誤
func IsGatewayError(err error) (*GatewayError, bool) {
	if ge, ok := err.(*GatewayError); ok {
		return ge, true
	}
	return nil, false
}

// ErrorResponse 錯誤響應結構
type ErrorResponse struct {
	Success   bool          `json:"success"`
	Error     *GatewayError `json:"error"`
	Timestamp string        `json:"timestamp"`
	RequestID string        `json:"request_id,omitempty"`
	Path      string        `json:"path,omitempty"`
}

// NewErrorResponse 創建錯誤響應
func NewErrorResponse(err *GatewayError, requestID, path string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     err,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
		RequestID: requestID,
		Path:      path,
	}
}
