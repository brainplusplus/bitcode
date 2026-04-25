package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type Claims struct {
	UserID         string   `json:"user_id"`
	Username       string   `json:"username"`
	Roles          []string `json:"roles"`
	Groups         []string `json:"groups"`
	ImpersonatedBy string   `json:"impersonated_by,omitempty"`
	Purpose        string   `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

type TokenOption func(*Claims)

func WithImpersonatedBy(adminID string) TokenOption {
	return func(c *Claims) {
		c.ImpersonatedBy = adminID
	}
}

func WithPurpose(purpose string) TokenOption {
	return func(c *Claims) {
		c.Purpose = purpose
	}
}

func WithExpiration(d time.Duration) TokenOption {
	return func(c *Claims) {
		c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(d))
	}
}

func GenerateToken(cfg JWTConfig, userID string, username string, roles []string, groups []string, opts ...TokenOption) (string, error) {
	if cfg.Expiration == 0 {
		cfg.Expiration = 24 * time.Hour
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Groups:   groups,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bitcode",
		},
	}

	for _, opt := range opts {
		opt(&claims)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}

func ValidateToken(cfg JWTConfig, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
