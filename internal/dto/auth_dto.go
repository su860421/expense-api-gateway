package dto

import "time"

// LoginRequest 登入請求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse 登入響應
type LoginResponse struct {
	Success      bool      `json:"success"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// UserInfo 用戶信息
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CompanyID string `json:"company_id"`
	Role      string `json:"role"`
	Avatar    string `json:"avatar,omitempty"`
	IsActive  bool   `json:"is_active"`
}

// RefreshTokenRequest 刷新 Token 請求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthHeaders 認證 Headers
type AuthHeaders struct {
	UserID    string `header:"X-User-ID"`
	CompanyID string `header:"X-Company-ID"`
	Role      string `header:"X-User-Role"`
	Email     string `header:"X-User-Email"`
}

// ValidateTokenRequest 驗證 Token 請求
type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// ValidateTokenResponse 驗證 Token 響應
type ValidateTokenResponse struct {
	Valid  bool                   `json:"valid"`
	User   *UserInfo              `json:"user,omitempty"`
	Claims map[string]interface{} `json:"claims,omitempty"`
}

// PermissionCheck 權限檢查
type PermissionCheck struct {
	Resource      string   `json:"resource"`
	Action        string   `json:"action"`
	RequiredRoles []string `json:"required_roles"`
	CompanyID     string   `json:"company_id"`
}

// AuthMiddlewareConfig 認證中間件配置
type AuthMiddlewareConfig struct {
	SkipPaths      []string      `json:"skip_paths"`
	RequiredClaims []string      `json:"required_claims"`
	TokenExpiry    time.Duration `json:"token_expiry"`
	RefreshExpiry  time.Duration `json:"refresh_expiry"`
}

// TokenValidationResult Token 驗證結果
type TokenValidationResult struct {
	IsValid   bool                   `json:"is_valid"`
	UserID    string                 `json:"user_id"`
	CompanyID string                 `json:"company_id"`
	Role      string                 `json:"role"`
	Email     string                 `json:"email"`
	Claims    map[string]interface{} `json:"claims"`
	Error     string                 `json:"error,omitempty"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// AuthErrorResponse 認證錯誤響應
type AuthErrorResponse struct {
	Success   bool   `json:"success"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// LogoutRequest 登出請求
type LogoutRequest struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	LogoutAll    bool   `json:"logout_all"` // 是否登出所有設備
}

// LogoutResponse 登出響應
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ChangePasswordRequest 修改密碼請求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// ForgotPasswordRequest 忘記密碼請求
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密碼請求
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// JWTPayload JWT 負載
type JWTPayload struct {
	UserID    string    `json:"user_id"`
	CompanyID string    `json:"company_id"`
	Role      string    `json:"role"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// ToAuthHeaders 轉換為認證 Headers
func (u *UserInfo) ToAuthHeaders() map[string]string {
	return map[string]string{
		"X-User-ID":    u.ID,
		"X-Company-ID": u.CompanyID,
		"X-User-Role":  u.Role,
		"X-User-Email": u.Email,
	}
}

// IsValidRole 檢查角色是否有效
func (u *UserInfo) IsValidRole(allowedRoles []string) bool {
	for _, role := range allowedRoles {
		if u.Role == role {
			return true
		}
	}
	return false
}
