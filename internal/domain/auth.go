package domain

import (
	"errors"
	"time"
)

// AuthUser 認證用戶信息（用於 API Gateway 轉發）
type AuthUser struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	CompanyID string   `json:"company_id"`
	Role      string   `json:"role"`
	Roles     []string `json:"roles"`
}

// HasRole 檢查用戶是否具有指定角色
func (u *AuthUser) HasRole(role string) bool {
	if u.Role == role {
		return true
	}
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// BelongsToCompany 檢查用戶是否屬於指定公司
func (u *AuthUser) BelongsToCompany(companyID string) bool {
	return u.CompanyID == companyID
}

// JWTClaims JWT 聲明結構
type JWTClaims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	CompanyID string   `json:"company_id"`
	Role      string   `json:"role"`
	Roles     []string `json:"roles"`
	Exp       int64    `json:"exp"`
	Iat       int64    `json:"iat"`
	IssuedAt  int64    `json:"issued_at"`
	ExpiresAt int64    `json:"expires_at"`
}

// ToAuthUser 將 JWTClaims 轉換為 AuthUser
func (c *JWTClaims) ToAuthUser() *AuthUser {
	return &AuthUser{
		ID:        c.UserID,
		Username:  c.Username,
		Email:     c.Email,
		CompanyID: c.CompanyID,
		Role:      c.Role,
		Roles:     c.Roles,
	}
}

// IsValid 檢查 JWT Claims 是否有效
func (c *JWTClaims) IsValid() bool {
	now := time.Now().Unix()
	return c.ExpiresAt > now && c.IssuedAt <= now
}

// Valid 實作 jwt.Claims 介面
func (c *JWTClaims) Valid() error {
	now := time.Now().Unix()
	if c.ExpiresAt < now {
		return errors.New("token is expired")
	}
	return nil
}

// AuthResult 認證結果
type AuthResult struct {
	Success bool          `json:"success"`
	User    *AuthUser     `json:"user,omitempty"`
	Error   *GatewayError `json:"error,omitempty"`
	Message string        `json:"message"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret            string        `json:"secret"`
	Expiration        time.Duration `json:"expiration"`
	RefreshExpiration time.Duration `json:"refresh_expiration"`
	Issuer            string        `json:"issuer"`
}
