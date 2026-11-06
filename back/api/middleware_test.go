package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("falta Content-Security-Policy")
	}
}

func TestLimitBodyRejectsLargeContentLength(t *testing.T) {
	called := false
	handler := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{}"))
	req.ContentLength = defaultMaxBodyBytes + 1

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, se esperaba 413", rec.Code)
	}
	if called {
		t.Fatal("el handler no debía ejecutarse")
	}
}

func TestLimitBodyAllowsUploadPath(t *testing.T) {
	called := false
	handler := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = io.Copy(io.Discard, r.Body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/archivos", strings.NewReader("contenido"))
	req.ContentLength = defaultMaxBodyBytes + 1

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("una subida válida no debía bloquearse por el límite JSON")
	}
}

func TestLimitBodyCutsStreamWithoutContentLength(t *testing.T) {
	var readErr error
	handler := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.Copy(io.Discard, r.Body)
	}))

	body := strings.Repeat("a", int(defaultMaxBodyBytes)+1024)
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if readErr == nil {
		t.Fatal("se esperaba que MaxBytesReader cortara el body")
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	handler := corsMiddleware("http://localhost:4200, https://app.example.com")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Origin") {
		t.Fatal("falta Vary: Origin")
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := corsMiddleware("http://localhost:4200")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("un origen no permitido no debe recibir CORS: %q", got)
	}
}
