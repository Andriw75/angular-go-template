package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/outputs"
)

const (
	maxArchivoUploadSize = 100 << 20 // límite total por petición
	maxArchivoFileBytes  = 100 << 20 // límite por archivo
	maxArchivoFiles      = 20
)

var allowedArchivoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".txt": true, ".csv": true, ".zip": true, ".rar": true,
	".mp4": true, ".mp3": true, ".webm": true,
}

// allowedArchivoMimes valida el contenido real (magic bytes), no solo la
// extensión. Los formatos legacy de Office son OLE/octet-stream; docx/xlsx son
// ZIP por dentro.
var allowedArchivoMimes = map[string]map[string]bool{
	".jpg":  {"image/jpeg": true},
	".jpeg": {"image/jpeg": true},
	".png":  {"image/png": true},
	".webp": {"image/webp": true},
	".gif":  {"image/gif": true},
	".pdf":  {"application/pdf": true},
	".doc":  {"application/x-ole-storage": true, "application/octet-stream": true, "application/vnd.ms-office": true},
	".docx": {"application/zip": true},
	".xls":  {"application/x-ole-storage": true, "application/octet-stream": true, "application/vnd.ms-office": true},
	".xlsx": {"application/zip": true},
	".txt":  {"text/plain": true, "application/octet-stream": true},
	".csv":  {"text/plain": true, "text/csv": true, "application/octet-stream": true},
	".zip":  {"application/zip": true},
	".rar":  {"application/vnd.rar": true, "application/x-rar-compressed": true, "application/octet-stream": true},
	".mp4":  {"video/mp4": true},
	".mp3":  {"audio/mpeg": true, "audio/mp3": true},
	".webm": {"video/webm": true},
}

func secureFileName(prefix, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(b), ext), nil
}

func validateUploadContent(fh *multipart.FileHeader, ext string) error {
	if fh.Size > maxArchivoFileBytes {
		return fmt.Errorf("el archivo supera el límite de %d MB", maxArchivoFileBytes>>20)
	}
	allowed := allowedArchivoMimes[ext]
	if len(allowed) == 0 {
		return fmt.Errorf("extensión %q no permitida", ext)
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	detected := http.DetectContentType(head[:n])
	if !allowed[detected] {
		return fmt.Errorf("el contenido del archivo no coincide con la extensión %q", ext)
	}
	return nil
}

func cleanOriginalName(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if len(name) > 200 {
		name = name[:200]
	}
	if name == "" || name == "." {
		name = "archivo"
	}
	return name
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
	// Límite duro aunque nginx ya limite el body: protege si Go se expone directo.
	r.Body = http.MaxBytesReader(w, r.Body, maxArchivoUploadSize+(10<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "error al parsear formulario o archivo demasiado grande")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeJSONError(w, http.StatusBadRequest, "files requerido")
		return
	}
	if len(files) > maxArchivoFiles {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("máximo %d archivos por carga", maxArchivoFiles))
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

	filesDir := h.deps.Config.PrivateFilesDir()
	if vis == domain.VisibilidadPublico {
		filesDir = h.deps.Config.PublicFilesDir()
	}
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al preparar carpeta")
		return
	}

	var savedNames []string
	rollback := func() {
		for _, name := range savedNames {
			_ = os.Remove(filepath.Join(filesDir, name))
		}
	}

	var created []domain.Archivo
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedArchivoExts[ext] {
			rollback()
			writeJSONError(w, http.StatusBadRequest, "extension "+ext+" no permitida")
			return
		}
		if err := validateUploadContent(fh, ext); err != nil {
			rollback()
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Nombre generado por el servidor: nunca se usa el nombre original como
		// ruta en disco.
		name, err := secureFileName("archivo", ext)
		if err != nil {
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al generar nombre de archivo")
			return
		}

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

		written, copyErr := io.Copy(dst, io.LimitReader(src, maxArchivoFileBytes+1))
		closeErr := dst.Close()
		src.Close()
		if copyErr != nil || closeErr != nil || written > maxArchivoFileBytes {
			_ = os.Remove(filepath.Join(filesDir, name))
			rollback()
			writeJSONError(w, http.StatusBadRequest, "archivo demasiado grande o error al guardar")
			return
		}

		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		archivo, err := h.deps.ArchivoStore.Create(name, cleanOriginalName(fh.Filename), mimeType, written, vis, userID)
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
	if !safeFileName(filename) {
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

	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !isImageFileName(filename) {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": archivo.Nombre}))
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

	dir := h.deps.Config.PrivateFilesDir()
	other := h.deps.Config.PublicFilesDir()
	if archivo.Visibilidad == domain.VisibilidadPublico {
		dir, other = other, dir
	}
	// Defensa: si una transición falló y el archivo quedó en la otra carpeta,
	// moverlo a la que corresponde.
	if _, err := os.Stat(filepath.Join(other, filename)); err == nil {
		_ = h.deps.Storage.MoveFileIfMissing(other, dir, filename)
	}
	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		filePath = filepath.Join(other, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
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

	oldPublic := archivo.Visibilidad == domain.VisibilidadPublico
	newPublic := body.Visibilidad == domain.VisibilidadPublico
	src := h.deps.Config.PrivateFilesDir()
	dst := h.deps.Config.PublicFilesDir()
	if oldPublic {
		src, dst = h.deps.Config.PublicFilesDir(), h.deps.Config.PrivateFilesDir()
	}
	if oldPublic != newPublic {
		if err := h.deps.Storage.MoveFile(src, dst, archivo.FileName); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "error al mover archivo")
			return
		}
	}

	if err := h.deps.ArchivoStore.UpdateVisibilidad(id, body.Visibilidad); err != nil {
		if oldPublic != newPublic {
			_ = h.deps.Storage.MoveFile(dst, src, archivo.FileName)
		}
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

	h.deps.Storage.RemoveIn(h.deps.Config.PublicFilesDir(), archivo.FileName)
	h.deps.Storage.RemoveIn(h.deps.Config.PrivateFilesDir(), archivo.FileName)

	slog.Info("archivo eliminado", "id", id, "file", archivo.FileName)
	writeJSON(w, http.StatusOK, map[string]string{"message": "archivo deleted"})
}
