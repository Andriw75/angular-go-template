package handlers

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"back/domain/inputs"
	"back/domain/outputs"
	"back/infrastructure/auth"
	"back/services"
)

type AuthHandler struct {
	deps *Dependencies
}

func NewAuthHandler(deps *Dependencies) *AuthHandler {
	return &AuthHandler{deps: deps}
}

// setSessionHeaders expone al frontend cuánto le queda al token actual.
// El backend solo aporta el dato; el frontend decide si muestra el contador.
func (h *AuthHandler) setSessionHeaders(w http.ResponseWriter, claims *auth.Claims) {
	if claims == nil || claims.ExpiresAt == nil {
		return
	}

	remaining := int(time.Until(claims.ExpiresAt.Time).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("X-Session-Expires-In", strconv.Itoa(remaining))
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.deps.Config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Duration(h.deps.Config.JWTExpirationMin) * time.Minute),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.deps.Config.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// clientIP prioriza las cabeceras que pone nginx. Si el binario se expone
// directamente, un atacante podría falsificarlas para evadir el límite por IP,
// pero el límite por usuario sigue activo.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if i := strings.IndexByte(forwarded, ','); i >= 0 {
			forwarded = forwarded[:i]
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		writeJSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	key := auth.LoginKey(clientIP(r), username)
	if h.deps.LoginLimiter != nil {
		if allowed, retry := h.deps.LoginLimiter.Allowed(key); !allowed {
			seconds := int(retry.Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeJSONErrorCode(w, http.StatusTooManyRequests, "too_many_attempts", "demasiados intentos; intenta más tarde")
			return
		}
	}

	user, ok := h.deps.UserStore.IsValidPassword(h.deps.CryptManager, username, password)
	if !ok {
		if h.deps.LoginLimiter != nil {
			h.deps.LoginLimiter.Fail(key)
		}
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if h.deps.LoginLimiter != nil {
		h.deps.LoginLimiter.Reset(key)
	}

	token, err := h.deps.JWTManager.Generate(user.ID, user.Username, user.Email, user.Activo, user.Permisos, user.MustChangePassword)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.setSessionCookie(w, token)

	claims, _ := h.deps.JWTManager.Validate(token)
	h.setSessionHeaders(w, claims)

	writeJSON(w, http.StatusOK, outputs.ToUserResponse(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if h.deps.Config.JWTStoreEnabled && claims != nil && claims.ID != "" {
		h.deps.JWTStore.Remove(claims.ID)
	}

	h.clearSessionCookie(w)

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	h.setSessionHeaders(w, claims)

	writeJSON(w, http.StatusOK, outputs.UserResponse{
		ID:                 claims.UserID,
		Username:           claims.Username,
		Email:              claims.Email,
		Activo:             claims.Activo,
		Permisos:           claims.Permisos,
		MustChangePassword: claims.MustChangePassword,
	})
}

// ChangePassword permite al usuario autenticado cambiar su propia contraseña.
// Al terminar revoca todos sus tokens y limpia la cookie: debe volver a entrar.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var input inputs.ChangePasswordInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "current_password y new_password son obligatorios")
		return
	}

	user, err := h.deps.UserStore.FindByID(claims.UserID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}
	if !h.deps.CryptManager.Verify(user.Password, input.CurrentPassword) {
		writeJSONError(w, http.StatusBadRequest, "la contraseña actual es incorrecta")
		return
	}
	if input.CurrentPassword == input.NewPassword {
		writeJSONError(w, http.StatusBadRequest, "la contraseña nueva debe ser diferente a la actual")
		return
	}
	if err := services.ValidatePasswordForUser(user.Username, input.NewPassword); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := h.deps.CryptManager.Hash(input.NewPassword)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.deps.UserStore.UpdatePassword(user.ID, hash, false); err != nil {
		slog.Error("failed to update password", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if h.deps.Config.JWTStoreEnabled {
		h.deps.JWTStore.RemoveUserTokens(user.ID)
	}
	h.clearSessionCookie(w)

	writeJSON(w, http.StatusOK, map[string]string{"message": "contraseña actualizada; inicia sesión de nuevo"})
}

func (h *AuthHandler) AuthMiddleware(requiredPermission ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(h.deps.Config.CookieName)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			claims, err := h.deps.JWTManager.Validate(cookie.Value)
			if err != nil {
				h.clearSessionCookie(w)
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			renewed := false

			if h.deps.Config.JWTStoreEnabled && claims.ID != "" && !h.deps.JWTStore.Exists(claims.ID) {
				if h.deps.Config.JWTLoadUserOnRenew {
					user, uerr := h.deps.UserStore.FindByID(claims.UserID)
					if uerr == nil && user.Activo {
						newToken, terr := h.deps.JWTManager.Generate(user.ID, user.Username, user.Email, user.Activo, user.Permisos, user.MustChangePassword)
						if terr == nil {
							h.setSessionCookie(w, newToken)
							claims, _ = h.deps.JWTManager.Validate(newToken)
							renewed = true
						}
					}
				}
				if !renewed {
					h.clearSessionCookie(w)
					writeJSONError(w, http.StatusUnauthorized, "session expired")
					return
				}
			}

			if !renewed && h.deps.JWTManager.ShouldRenew(claims) {
				if h.deps.Config.JWTLoadUserOnRenew {
					user, uerr := h.deps.UserStore.FindByID(claims.UserID)
					if uerr == nil {
						newToken, terr := h.deps.JWTManager.Generate(user.ID, user.Username, user.Email, user.Activo, user.Permisos, user.MustChangePassword)
						if terr == nil {
							h.setSessionCookie(w, newToken)
							claims, _ = h.deps.JWTManager.Validate(newToken)
						}
					}
				} else {
					newToken, terr := h.deps.JWTManager.Generate(claims.UserID, claims.Username, claims.Email, claims.Activo, claims.Permisos, claims.MustChangePassword)
					if terr == nil {
						h.setSessionCookie(w, newToken)
					}
				}
			}

			h.setSessionHeaders(w, claims)

			// Bloquea todo salvo las rutas de auth hasta que el usuario cambie
			// la contraseña inicial.
			if claims.MustChangePassword && !strings.HasPrefix(r.URL.Path, "/auth/") {
				writeJSONErrorCode(w, http.StatusForbidden, "password_change_required", "debes cambiar la contraseña antes de continuar")
				return
			}

			if len(requiredPermission) > 0 {
				if !tienePermiso(claims.Permisos, requiredPermission[0]) {
					writeJSONError(w, http.StatusForbidden, "forbidden")
					return
				}
			}

			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *AuthHandler) OptionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(h.deps.Config.CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := h.deps.JWTManager.Validate(cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			h.setSessionHeaders(w, claims)

			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func tienePermiso(permisos []string, required string) bool {
	for _, p := range permisos {
		if p == required {
			return true
		}
	}
	return false
}
