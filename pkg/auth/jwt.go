package auth

import (
	"errors"
	"time"

	"expense-api-gateway/internal/domain"

	"github.com/golang-jwt/jwt/v4"
)

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey     string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTManager 創建新的 JWT 管理器
func NewJWTManager(secretKey string, tokenExpiry, refreshExpiry time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
	}
}

// GenerateToken 生成 JWT Token
func (j *JWTManager) GenerateToken(user *domain.AuthUser) (string, error) {
	now := time.Now()
	claims := &domain.JWTClaims{
		UserID:    user.ID,
		CompanyID: user.CompanyID,
		Role:      user.Role,
		Email:     user.Email,
		Username:  user.Username,
		Roles:     user.Roles,
		Exp:       now.Add(j.tokenExpiry).Unix(),
		Iat:       now.Unix(),
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(j.tokenExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// GenerateRefreshToken 生成刷新 Token
func (j *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	claims := &jwt.StandardClaims{
		ExpiresAt: now.Add(j.refreshExpiry).Unix(),
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		Issuer:    j.issuer,
		Subject:   userID,
		Id:        generateTokenID(), // 可以用於 token 撤銷
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken 驗證 JWT Token
func (j *JWTManager) ValidateToken(tokenString string) (*domain.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &domain.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 檢查簽名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*domain.JWTClaims); ok && token.Valid {
		// 額外驗證
		if !claims.IsValid() {
			return nil, errors.New("invalid token claims")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken 驗證刷新 Token
func (j *JWTManager) ValidateRefreshToken(tokenString string) (*jwt.StandardClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.StandardClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// RefreshToken 刷新 Token
func (j *JWTManager) RefreshToken(refreshTokenString string, user *domain.AuthUser) (string, string, error) {
	// 驗證刷新 Token
	refreshClaims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", err
	}

	// 檢查用戶ID是否匹配
	if refreshClaims.Subject != user.ID {
		return "", "", errors.New("token user mismatch")
	}

	// 生成新的 Token 對
	newToken, err := j.GenerateToken(user)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := j.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	return newToken, newRefreshToken, nil
}

// ExtractTokenFromHeader 從 Authorization Header 提取 Token
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// 檢查 Bearer 前綴
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", errors.New("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with Bearer")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is required")
	}

	return token, nil
}

// GetTokenExpiry 獲取 Token 過期時間
func (j *JWTManager) GetTokenExpiry() time.Duration {
	return j.tokenExpiry
}

// GetRefreshExpiry 獲取刷新 Token 過期時間
func (j *JWTManager) GetRefreshExpiry() time.Duration {
	return j.refreshExpiry
}

// IsTokenExpired 檢查 Token 是否過期
func IsTokenExpired(claims *domain.JWTClaims) bool {
	return time.Now().Unix() > claims.ExpiresAt
}

// GetRemainingTime 獲取 Token 剩餘時間
func GetRemainingTime(claims *domain.JWTClaims) time.Duration {
	expiry := time.Unix(claims.ExpiresAt, 0)
	remaining := time.Until(expiry)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// generateTokenID 生成 Token ID（用於撤銷等功能）
func generateTokenID() string {
	// 這裡可以使用 UUID 或其他唯一ID生成方法
	// 暫時使用時間戳
	return string(rune(time.Now().UnixNano()))
}

// TokenPair Token 對
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// GenerateTokenPair 生成 Token 對
func (j *JWTManager) GenerateTokenPair(user *domain.AuthUser) (*TokenPair, error) {
	accessToken, err := j.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := j.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// 計算過期時間
	expiryDuration := j.GetTokenExpiry()
	expiresAt := time.Now().Add(expiryDuration)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(expiryDuration.Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateTokenString 驗證 Token 字串格式
func ValidateTokenString(token string) error {
	if token == "" {
		return errors.New("token is empty")
	}
	// 這裡可以添加更多的格式驗證
	return nil
}
