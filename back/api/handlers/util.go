package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"back/domain/outputs"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, outputs.ErrorResponse{Error: msg})
}

func writeJSONErrorCode(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, outputs.ErrorResponse{Error: msg, Code: code})
}

// safeFileName rechaza rutas, separadores y bytes nulos. Los nombres nuevos se
// generan en el servidor, pero esto protege también filas antiguas en la BD.
func safeFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return false
	}
	return name == filepath.Base(name)
}

func isImageFileName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	}
	return false
}
