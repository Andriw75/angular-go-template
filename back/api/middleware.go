package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	defaultMaxBodyBytes int64 = 2 << 20   // 2 MiB
	uploadMaxBodyBytes  int64 = 120 << 20 // 120 MiB
)

// securityHeaders agrega cabeceras de defensa en profundidad a la API.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// limitBody acota el cuerpo de la petición. Las subidas usan un límite mayor
// porque sus handlers ya validan tamaño por archivo.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := defaultMaxBodyBytes
		if isUploadRequest(r) {
			limit = uploadMaxBodyBytes
		}

		if r.ContentLength > limit {
			writeBodyTooLarge(w)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

func isUploadRequest(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch {
	case r.URL.Path == "/archivos":
		return true
	case strings.HasPrefix(r.URL.Path, "/categorias/") && strings.HasSuffix(r.URL.Path, "/imagenes"):
		return true
	case strings.HasPrefix(r.URL.Path, "/productos/") && strings.HasSuffix(r.URL.Path, "/imagenes"):
		return true
	default:
		return false
	}
}

func writeBodyTooLarge(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{Error: "el cuerpo de la petición es demasiado grande", Code: "request_too_large"})
}

// corsMiddleware aplica CORS solo a los orígenes permitidos. CORS_ORIGIN admite
// una lista separada por comas.
func corsMiddleware(origins string) func(http.Handler) http.Handler {
	allowed := parseOrigins(origins)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" && allowed[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-CSRF-Token")
				h.Set("Access-Control-Expose-Headers", "X-Session-Expires-In")
				h.Set("Access-Control-Max-Age", "600")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// csrfHeader es la cabecera que envía el cliente con el valor de la cookie CSRF.
const csrfHeader = "X-CSRF-Token"

// csrfProtection aplica el patrón double-submit: en métodos que mutan estado
// exige que la cabecera coincida con la cookie CSRF (que el JS del cliente
// puede leer). El login queda exento porque es quien emite la cookie.
func csrfProtection(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/auth/token" {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cookieName)
			header := r.Header.Get(csrfHeader)
			if err != nil || cookie.Value == "" || header == "" ||
				subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}{Error: "token CSRF ausente o inválido", Code: "csrf_failed"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseOrigins(raw string) map[string]bool {
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}
	return allowed
}
