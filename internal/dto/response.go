package dto

import (
	"time"
)

// APIResponse 統一API響應格式
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorInfo 錯誤信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	Field   string `json:"field,omitempty"`
}

// Meta 元數據信息
type Meta struct {
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	Total      int64  `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	Version    string `json:"version,omitempty"`
}

// PaginationRequest 分頁請求
type PaginationRequest struct {
	Page     int    `form:"page" json:"page" binding:"min=1"`
	PageSize int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`
	Sort     string `form:"sort" json:"sort"`
	Order    string `form:"order" json:"order" binding:"oneof=asc desc"`
}

// PaginationResponse 分頁響應
type PaginationResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// SuccessResponse 成功響應構造器
func SuccessResponse(data interface{}, message ...string) *APIResponse {
	msg := "Success"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	return &APIResponse{
		Success:   true,
		Message:   msg,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// ErrorResponse 錯誤響應構造器
func ErrorResponse(code, message string, detail ...string) *APIResponse {
	errorInfo := &ErrorInfo{
		Code:    code,
		Message: message,
	}

	if len(detail) > 0 && detail[0] != "" {
		errorInfo.Detail = detail[0]
	}

	return &APIResponse{
		Success:   false,
		Error:     errorInfo,
		Timestamp: time.Now().Unix(),
	}
}

// ValidationErrorResponse 驗證錯誤響應
func ValidationErrorResponse(field, message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Field:   field,
		},
		Timestamp: time.Now().Unix(),
	}
}

// PaginatedResponse 分頁響應構造器
func PaginatedResponse(items interface{}, pagination *PaginationResponse) *APIResponse {
	return &APIResponse{
		Success:   true,
		Data:      pagination,
		Timestamp: time.Now().Unix(),
	}
}

// WithRequestID 添加請求ID
func (r *APIResponse) WithRequestID(requestID string) *APIResponse {
	r.RequestID = requestID
	return r
}

// WithMeta 添加元數據
func (r *APIResponse) WithMeta(meta *Meta) *APIResponse {
	r.Meta = meta
	return r
}

// NewPagination 創建分頁信息
func NewPagination(page, pageSize int, total int64) *PaginationResponse {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// SetItems 設置分頁項目
func (p *PaginationResponse) SetItems(items interface{}) *PaginationResponse {
	p.Items = items
	return p
}

// DefaultPagination 默認分頁參數
func DefaultPagination() *PaginationRequest {
	return &PaginationRequest{
		Page:     1,
		PageSize: 20,
		Sort:     "created_at",
		Order:    "desc",
	}
}

// GetOffset 獲取分頁偏移量
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 獲取分頁限制
func (p *PaginationRequest) GetLimit() int {
	return p.PageSize
}

// Validate 驗證分頁參數
func (p *PaginationRequest) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
	return nil
}

// HealthCheckResponse 健康檢查響應
type HealthCheckResponse struct {
	Status    string                   `json:"status"`
	Timestamp int64                    `json:"timestamp"`
	Uptime    string                   `json:"uptime"`
	Version   string                   `json:"version"`
	Services  map[string]ServiceStatus `json:"services"`
}

// ServiceStatus 服務狀態
type ServiceStatus struct {
	Status       string `json:"status"`
	LastCheck    int64  `json:"last_check"`
	ResponseTime string `json:"response_time"`
	Error        string `json:"error,omitempty"`
}

// MetricsResponse 監控指標響應
type MetricsResponse struct {
	RequestCount    int64                  `json:"request_count"`
	ErrorCount      int64                  `json:"error_count"`
	AverageResponse float64                `json:"average_response_time"`
	StatusCodes     map[string]int64       `json:"status_codes"`
	EndpointMetrics map[string]interface{} `json:"endpoint_metrics"`
	SystemMetrics   *SystemMetrics         `json:"system_metrics"`
	LastUpdated     int64                  `json:"last_updated"`
}

// SystemMetrics 系統指標
type SystemMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	DiskUsage      float64 `json:"disk_usage"`
	GoroutineCount int     `json:"goroutine_count"`
}

// RouteInfo 路由信息
type RouteInfo struct {
	Method       string   `json:"method"`
	Path         string   `json:"path"`
	ServiceName  string   `json:"service_name"`
	AuthRequired bool     `json:"auth_required"`
	Roles        []string `json:"roles,omitempty"`
	Description  string   `json:"description,omitempty"`
}

// ServiceInfo 服務信息
type ServiceInfo struct {
	Name      string   `json:"name"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Status    string   `json:"status"`
	Instances []string `json:"instances"`
	Health    string   `json:"health"`
	LastCheck int64    `json:"last_check"`
}
