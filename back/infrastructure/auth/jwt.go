package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret     string
	duration   time.Duration
	renewAfter time.Duration
}

type Claims struct {
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Permisos []string `json:"permisos"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, expirationMinutes int) *JWTManager {
	return &JWTManager{
		secret:     secret,
		duration:   time.Duration(expirationMinutes) * time.Minute,
		renewAfter: time.Duration(expirationMinutes-5) * time.Minute,
	}
}

func (m *JWTManager) Generate(userID int64, username string, permisos []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Permisos: permisos,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

func (m *JWTManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (m *JWTManager) ShouldRenew(claims *Claims) bool {
	if claims == nil || claims.IssuedAt == nil {
		return false
	}
	elapsed := time.Since(claims.IssuedAt.Time)
	return elapsed >= m.renewAfter
}
