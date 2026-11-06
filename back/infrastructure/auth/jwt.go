package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// maxTokenBytes acota el token antes de parsearlo. Un JWT normal ronda 1 KiB;
// este límite evita amplificación de memoria con cabeceras manipuladas.
const maxTokenBytes = 4 << 10

type JWTManager struct {
	secret     string
	duration   time.Duration
	renewAfter time.Duration
	Store      *JWTStore
}

type Claims struct {
	UserID             int64    `json:"user_id"`
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	Activo             bool     `json:"activo"`
	Permisos           []string `json:"permisos"`
	MustChangePassword bool     `json:"must_change_password"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, expirationMinutes, renewMinutes int) *JWTManager {
	renewAfter := time.Duration(expirationMinutes-5) * time.Minute
	if renewMinutes > 0 {
		renewAfter = time.Duration(renewMinutes) * time.Minute
	}
	return &JWTManager{
		secret:     secret,
		duration:   time.Duration(expirationMinutes) * time.Minute,
		renewAfter: renewAfter,
	}
}

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *JWTManager) Generate(userID int64, username, email string, activo bool, permisos []string, mustChangePassword bool) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.duration)
	claims := &Claims{
		UserID:             userID,
		Username:           username,
		Email:              email,
		Activo:             activo,
		Permisos:           permisos,
		MustChangePassword: mustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	if m.Store != nil {
		jti, err := generateJTI()
		if err != nil {
			return "", fmt.Errorf("failed to generate jti: %w", err)
		}
		claims.ID = jti
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", err
	}

	if m.Store != nil && claims.ID != "" {
		m.Store.Add(claims.ID, userID, expiresAt)
	}

	return tokenStr, nil
}

func (m *JWTManager) Validate(tokenStr string) (*Claims, error) {
	if tokenStr == "" || len(tokenStr) > maxTokenBytes {
		return nil, fmt.Errorf("invalid token size")
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			// Solo HS256: descarta algoritmos distintos y variantes HMAC.
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
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
