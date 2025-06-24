package jwt

import (
	"expense-api-gateway/internal/config"
	"expense-api-gateway/internal/domain"
	"expense-api-gateway/pkg/auth"
	"time"

	"go.uber.org/zap"
)

// JWTService JWT 服務實現
type JWTService struct {
	manager *auth.JWTManager
	config  *config.Config
	logger  *zap.Logger
}

// NewJWTService 創建新的 JWT 服務
func NewJWTService(cfg *config.Config, logger *zap.Logger) *JWTService {
	manager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.Expiration,
		cfg.JWT.RefreshExpiration,
		"expense-api-gateway",
	)

	return &JWTService{
		manager: manager,
		config:  cfg,
		logger:  logger,
	}
}

// ValidateToken 驗證 JWT Token
func (j *JWTService) ValidateToken(tokenString string) (*domain.AuthResult, error) {
	// 驗證 Token 格式
	if err := auth.ValidateTokenString(tokenString); err != nil {
		j.logger.Debug("Invalid token format", zap.Error(err))
		return &domain.AuthResult{
			Success: false,
			Error:   domain.ErrInvalidToken,
			Message: "Invalid token format",
		}, err
	}

	// 解析和驗證 Token
	claims, err := j.manager.ValidateToken(tokenString)
	if err != nil {
		j.logger.Debug("Token validation failed", zap.Error(err))

		return &domain.AuthResult{
			Success: false,
			Error:   domain.ErrInvalidToken,
			Message: "Token validation failed",
		}, err
	}

	// 檢查 Token 是否過期
	if auth.IsTokenExpired(claims) {
		j.logger.Debug("Token expired", zap.String("user_id", claims.UserID))
		return &domain.AuthResult{
			Success: false,
			Error:   domain.ErrTokenExpired,
			Message: "Token has expired",
		}, domain.ErrTokenExpired
	}

	// 轉換為 AuthUser
	user := claims.ToAuthUser()

	j.logger.Debug("Token validated successfully",
		zap.String("user_id", user.ID),
		zap.String("company_id", user.CompanyID),
		zap.String("role", user.Role))

	return &domain.AuthResult{
		Success: true,
		User:    user,
		Message: "Token validated successfully",
	}, nil
}

// GenerateToken 生成 JWT Token
func (j *JWTService) GenerateToken(user *domain.AuthUser) (*auth.TokenPair, error) {
	tokenPair, err := j.manager.GenerateTokenPair(user)
	if err != nil {
		j.logger.Error("Failed to generate token",
			zap.String("user_id", user.ID),
			zap.Error(err))
		return nil, err
	}

	j.logger.Info("Token generated successfully",
		zap.String("user_id", user.ID),
		zap.String("company_id", user.CompanyID))

	return tokenPair, nil
}

// RefreshToken 刷新 Token
func (j *JWTService) RefreshToken(refreshToken string, user *domain.AuthUser) (*auth.TokenPair, error) {
	accessToken, newRefreshToken, err := j.manager.RefreshToken(refreshToken, user)
	if err != nil {
		j.logger.Error("Failed to refresh token",
			zap.String("user_id", user.ID),
			zap.Error(err))
		return nil, err
	}

	// 計算過期時間
	expiryDuration := j.manager.GetTokenExpiry()
	expiresAt := time.Now().Add(expiryDuration)

	tokenPair := &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(expiryDuration.Seconds()),
		ExpiresAt:    expiresAt,
	}

	j.logger.Info("Token refreshed successfully",
		zap.String("user_id", user.ID))

	return tokenPair, nil
}

// ExtractTokenFromAuthHeader 從認證頭提取 Token
func (j *JWTService) ExtractTokenFromAuthHeader(authHeader string) (string, error) {
	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		j.logger.Debug("Failed to extract token from header", zap.Error(err))
		return "", err
	}
	return token, nil
}

// GetTokenInfo 獲取 Token 信息
func (j *JWTService) GetTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := j.manager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &TokenInfo{
		UserID:    claims.UserID,
		CompanyID: claims.CompanyID,
		Role:      claims.Role,
		Email:     claims.Email,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
		IsExpired: auth.IsTokenExpired(claims),
		Remaining: auth.GetRemainingTime(claims),
	}, nil
}

// TokenInfo Token 信息
type TokenInfo struct {
	UserID    string        `json:"user_id"`
	CompanyID string        `json:"company_id"`
	Role      string        `json:"role"`
	Email     string        `json:"email"`
	IssuedAt  int64         `json:"issued_at"`
	ExpiresAt int64         `json:"expires_at"`
	IsExpired bool          `json:"is_expired"`
	Remaining time.Duration `json:"remaining"`
}

// CreateAuthHeaders 創建認證 Headers
func (j *JWTService) CreateAuthHeaders(user *domain.AuthUser) map[string]string {
	headers := map[string]string{
		"X-User-ID":    user.ID,
		"X-Company-ID": user.CompanyID,
		"X-User-Role":  user.Role,
		"X-User-Email": user.Email,
	}

	j.logger.Debug("Created auth headers",
		zap.String("user_id", user.ID),
		zap.String("company_id", user.CompanyID))

	return headers
}

// ValidateRole 驗證用戶角色
func (j *JWTService) ValidateRole(user *domain.AuthUser, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true // 沒有角色要求
	}

	for _, role := range requiredRoles {
		if user.HasRole(role) {
			return true
		}
	}

	j.logger.Debug("Role validation failed",
		zap.String("user_id", user.ID),
		zap.String("user_role", user.Role),
		zap.Strings("required_roles", requiredRoles))

	return false
}

// ValidateCompany 驗證用戶公司
func (j *JWTService) ValidateCompany(user *domain.AuthUser, requiredCompanyID string) bool {
	if requiredCompanyID == "" {
		return true // 沒有公司要求
	}

	isValid := user.BelongsToCompany(requiredCompanyID)
	if !isValid {
		j.logger.Debug("Company validation failed",
			zap.String("user_id", user.ID),
			zap.String("user_company", user.CompanyID),
			zap.String("required_company", requiredCompanyID))
	}

	return isValid
}

// GetConfig 獲取 JWT 配置
func (j *JWTService) GetConfig() *domain.JWTConfig {
	return &domain.JWTConfig{
		Secret:            j.config.JWT.Secret,
		Expiration:        j.config.JWT.Expiration,
		RefreshExpiration: j.config.JWT.RefreshExpiration,
		Issuer:            "expense-api-gateway",
	}
}
