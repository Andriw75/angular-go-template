package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"back/infrastructure/auth"
	"back/services"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func newTestAuthHandler(t *testing.T) (*AuthHandler, *auth.JWTManager, *auth.JWTStore) {
	t.Helper()

	store := auth.NewJWTStore()
	manager := auth.NewJWTManager(testJWTSecret, 15, 5)
	manager.Store = store

	deps := &Dependencies{
		Config: &services.Config{
			CookieName:       "test_token",
			JWTStoreEnabled:  true,
			JWTExpirationMin: 15,
		},
		JWTManager: manager,
		JWTStore:   store,
	}
	return NewAuthHandler(deps), manager, store
}

func TestAuthMiddlewareAcceptsValidJTI(t *testing.T) {
	handler, manager, _ := newTestAuthHandler(t)

	token, err := manager.Generate(1, "user", "user@example.com", true, nil, false)
	if err != nil {
		t.Fatalf("generar token: %v", err)
	}

	called := false
	protected := handler.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_token", Value: token})
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rec.Code)
	}
	if !called {
		t.Fatal("el handler protegido debía ejecutarse")
	}
}

func TestAuthMiddlewareRejectsRevokedJTI(t *testing.T) {
	handler, manager, store := newTestAuthHandler(t)

	token, err := manager.Generate(1, "user", "user@example.com", true, nil, false)
	if err != nil {
		t.Fatalf("generar token: %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("validar token: %v", err)
	}
	store.Remove(claims.ID)

	called := false
	protected := handler.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{Name: "test_token", Value: token})
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", rec.Code)
	}
	if called {
		t.Fatal("el handler protegido no debía ejecutarse")
	}
	if !strings.Contains(rec.Body.String(), "session_expired") {
		t.Fatalf("respuesta = %s, se esperaba session_expired", rec.Body.String())
	}
}
