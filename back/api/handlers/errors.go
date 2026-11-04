package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"back/domain"
)

// writeError centraliza el mapeo de errores de negocio a HTTP. Los errores
// desconocidos se registran y se responden como 500 sin exponer detalles.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	var appErr *domain.Error
	if errors.As(err, &appErr) {
		status := statusForKind(appErr.Kind)
		if status >= http.StatusInternalServerError {
			logRequestError(r, err)
		}
		code := appErr.Code
		if code == "" {
			code = "internal_error"
		}
		writeJSONErrorCode(w, status, code, appErr.Message)
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		writeJSONErrorCode(w, http.StatusNotFound, "not_found", "recurso no encontrado")
		return
	}

	logRequestError(r, err)
	writeJSONErrorCode(w, http.StatusInternalServerError, "internal_error", "error interno")
}

func statusForKind(kind domain.ErrorKind) int {
	switch kind {
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindConflict:
		return http.StatusConflict
	case domain.KindValidation:
		return http.StatusUnprocessableEntity
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	case domain.KindForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func logRequestError(r *http.Request, err error) {
	slog.Error("request failed",
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", middleware.GetReqID(r.Context()),
	)
}
