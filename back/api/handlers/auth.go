package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"back/domain/outputs"
)

type AuthHandler struct {
	deps *Dependencies
}

func NewAuthHandler(deps *Dependencies) *AuthHandler {
	return &AuthHandler{deps: deps}
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		writeJSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, ok := h.deps.UserStore.IsValidPassword(h.deps.CryptManager, username, password)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.deps.JWTManager.Generate(user.ID, user.Username, user.Permisos)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.deps.Config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(time.Duration(h.deps.Config.JWTExpirationMin) * time.Minute),
	})

	writeJSON(w, http.StatusOK, outputs.ToUserResponse(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.deps.Config.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.Config.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.deps.UserStore.FindByID(claims.UserID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, outputs.ToUserResponse(user))
}

func (h *AuthHandler) RenewMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.deps.Config.CookieName)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}

		claims, err := h.deps.JWTManager.Validate(cookie.Value)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     h.deps.Config.CookieName,
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   h.deps.Config.CookieSecure,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1,
			})
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if h.deps.JWTManager.ShouldRenew(claims) {
			newToken, err := h.deps.JWTManager.Generate(claims.UserID, claims.Username, claims.Permisos)
			if err == nil {
				http.SetCookie(w, &http.Cookie{
					Name:     h.deps.Config.CookieName,
					Value:    newToken,
					Path:     "/",
					HttpOnly: true,
					Secure:   h.deps.Config.CookieSecure,
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Duration(h.deps.Config.JWTExpirationMin) * time.Minute),
				})
			}
		}

		ctx := WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
