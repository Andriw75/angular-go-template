package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// newJWTTestSecret genera el secreto en runtime para no versionar literales que
// Gitleaks (correctamente) confundiría con una credencial real.
func newJWTTestSecret(t *testing.T) string {
	t.Helper()

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("generar secreto de test: %v", err)
	}
	return hex.EncodeToString(b)
}

func TestValidateAcceptsHS256(t *testing.T) {
	manager := NewJWTManager(newJWTTestSecret(t), 15, 5)

	token, err := manager.Generate(1, "user", "user@example.com", true, nil, false)
	if err != nil {
		t.Fatalf("generar token: %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("validar token: %v", err)
	}
	if claims.UserID != 1 {
		t.Fatalf("user_id = %d, se esperaba 1", claims.UserID)
	}
}

func TestValidateRejectsHS512(t *testing.T) {
	secret := newJWTTestSecret(t)
	manager := NewJWTManager(secret, 15, 5)

	claims := &Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("firmar token HS512: %v", err)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("un token HS512 no debe aceptarse")
	}
}

func TestValidateRejectsAlgNone(t *testing.T) {
	manager := NewJWTManager(newJWTTestSecret(t), 15, 5)

	claims := &Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("firmar token alg=none: %v", err)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("un token alg=none no debe aceptarse")
	}
}

func TestValidateRejectsTokenWithoutExpiration(t *testing.T) {
	secret := newJWTTestSecret(t)
	manager := NewJWTManager(secret, 15, 5)

	claims := &Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("firmar token sin exp: %v", err)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("un token sin exp no debe aceptarse")
	}
}

func TestValidateRejectsOversizedToken(t *testing.T) {
	manager := NewJWTManager(newJWTTestSecret(t), 15, 5)

	if _, err := manager.Validate(strings.Repeat("a", maxTokenBytes+1)); err == nil {
		t.Fatal("un token sobredimensionado no debe aceptarse")
	}
}
