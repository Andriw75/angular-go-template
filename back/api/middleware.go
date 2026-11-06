package api

import (
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
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
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
