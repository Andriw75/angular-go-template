package handlers

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/outputs"
)

const maxArchivoUploadSize = 100 << 20

var allowedArchivoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".txt": true, ".csv": true, ".zip": true, ".rar": true,
	".mp4": true, ".mp3": true, ".webm": true,
}

type ArchivoHandler struct {
	deps *Dependencies
}

func NewArchivoHandler(deps *Dependencies) *ArchivoHandler {
	return &ArchivoHandler{deps: deps}
}

func (h *ArchivoHandler) List(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	q := r.URL.Query().Get("q")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	archivos, total, err := h.deps.ArchivoStore.List(offset, limit, q)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al listar archivos")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  outputs.ToArchivoResponses(archivos),
		"total": total,
	})
}

func (h *ArchivoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxArchivoUploadSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "error al parsear formulario")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeJSONError(w, http.StatusBadRequest, "files requerido")
		return
	}

	vis := domain.Visibilidad(r.FormValue("visibilidad"))
	if !domain.IsValidVisibilidad(vis) {
		vis = domain.VisibilidadPublico
	}

	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}

	filesDir := h.deps.Config.FilesDir()

	var savedNames []string
	rollback := func() {
		for _, name := range savedNames {
			_ = os.Remove(filepath.Join(filesDir, name))
		}
	}

	var created []domain.Archivo
	for i, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedArchivoExts[ext] {
			rollback()
			writeJSONError(w, http.StatusBadRequest, "extension "+ext+" no permitida")
			return
		}

		name := "archivo_" + strconv.FormatInt(int64(i), 10) + "_" + fh.Filename
		src, err := fh.Open()
		if err != nil {
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al abrir archivo")
			return
		}

		dst, err := os.Create(filepath.Join(filesDir, name))
		if err != nil {
			src.Close()
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al guardar archivo")
			return
		}

		buf := make([]byte, 32*1024)
		for {
			nr, rerr := src.Read(buf)
			if nr > 0 {
				if _, werr := dst.Write(buf[:nr]); werr != nil {
					dst.Close()
					src.Close()
					rollback()
					writeJSONError(w, http.StatusInternalServerError, "error al escribir archivo")
					return
				}
			}
			if rerr != nil {
				break
			}
		}
		dst.Close()
		src.Close()

		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		archivo, err := h.deps.ArchivoStore.Create(name, fh.Filename, mimeType, fh.Size, vis, userID)
		if err != nil {
			_ = os.Remove(filepath.Join(filesDir, name))
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al registrar archivo")
			return
		}
		savedNames = append(savedNames, name)
		created = append(created, archivo)
	}

	writeJSON(w, http.StatusCreated, outputs.ToArchivoResponses(created))
}

func (h *ArchivoHandler) Serve(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	archivo, err := h.deps.ArchivoStore.FindByFileName(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch archivo.Visibilidad {
	case domain.VisibilidadPublico:
	case domain.VisibilidadAutenticado:
		claims := GetClaims(r.Context())
		if claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
	case domain.VisibilidadPropietario:
		claims := GetClaims(r.Context())
		if claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if claims.UserID != archivo.UserID {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
	default:
		http.NotFound(w, r)
		return
	}

	prefix := h.deps.Config.XAccelRedirectPrefix
	if prefix != "" {
		w.Header().Set("X-Accel-Redirect", prefix+filename)
		if archivo.Tipo != "" {
			w.Header().Set("Content-Type", archivo.Tipo)
		}
		w.Header().Set("Content-Length", strconv.FormatInt(archivo.Tamano, 10))
		w.WriteHeader(http.StatusOK)
		return
	}

	filePath := filepath.Join(h.deps.Config.FilesDir(), filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if archivo.Tipo != "" {
		w.Header().Set("Content-Type", archivo.Tipo)
	}
	if archivo.Visibilidad == domain.VisibilidadPublico {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	}
	http.ServeFile(w, r, filePath)
}

func (h *ArchivoHandler) UpdateVisibilidad(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Visibilidad domain.Visibilidad `json:"visibilidad"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !domain.IsValidVisibilidad(body.Visibilidad) {
		writeJSONError(w, http.StatusBadRequest, "visibilidad invalida")
		return
	}

	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	archivo, err := h.deps.ArchivoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo not found")
		return
	}

	if archivo.UserID != claims.UserID {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.deps.ArchivoStore.UpdateVisibilidad(id, body.Visibilidad); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al actualizar visibilidad")
		return
	}

	updated, _ := h.deps.ArchivoStore.FindByID(id)
	writeJSON(w, http.StatusOK, outputs.ToArchivoResponse(updated))
}

func (h *ArchivoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	archivo, err := h.deps.ArchivoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo not found")
		return
	}

	if archivo.UserID != claims.UserID {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.deps.ArchivoStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al eliminar archivo")
		return
	}

	filePath := filepath.Join(h.deps.Config.FilesDir(), archivo.FileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("no se pudo eliminar archivo del disco", "file", archivo.FileName, "error", err)
	}

	slog.Info("archivo eliminado", "id", id, "file", archivo.FileName)
	writeJSON(w, http.StatusOK, map[string]string{"message": "archivo deleted"})
}
