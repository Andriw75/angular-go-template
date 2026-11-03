package handlers

import (
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

	"github.com/go-chi/chi/v5"

	"back/domain/outputs"
)

var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

const (
	maxUploadSize     = 64 << 20
	maxImageFileBytes = 32 << 20
)

func validateImageContent(fh *multipart.FileHeader) error {
	if fh.Size > maxImageFileBytes {
		return fmt.Errorf("la imagen supera el límite de %d MB", maxImageFileBytes>>20)
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

	switch http.DetectContentType(head[:n]) {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return nil
	default:
		return fmt.Errorf("el contenido de la imagen no es válido")
	}
}

// ImageIsPublic indica si la imagen debe servirse por nginx. Para productos
// exige producto activo y categoría activa; para categorías, categoría activa.
func ImageIsPublic(deps *Dependencies, tipo string, entidadID int64) (bool, error) {
	switch tipo {
	case "producto":
		producto, err := deps.ProductoStore.FindByID(entidadID)
		if err != nil {
			return false, err
		}
		if !producto.Activo {
			return false, nil
		}
		categoria, err := deps.CategoriaStore.FindByID(producto.CategoriaID)
		if err != nil {
			return false, err
		}
		return categoria.Activo, nil

	case "categoria":
		categoria, err := deps.CategoriaStore.FindByID(entidadID)
		if err != nil {
			return false, err
		}
		return categoria.Activo, nil

	default:
		return false, fmt.Errorf("tipo de imagen desconocido: %s", tipo)
	}
}

func imageDirs(deps *Dependencies, public bool) (dir, other string) {
	if public {
		return deps.Config.PublicImagesDir(), deps.Config.PrivateImagesDir()
	}
	return deps.Config.PrivateImagesDir(), deps.Config.PublicImagesDir()
}

// EnsureImagePlacement mueve las imágenes de una entidad a la carpeta que
// corresponde según su estado actual (pública o privada).
func EnsureImagePlacement(deps *Dependencies, tipo string, entidadID int64) {
	public, err := ImageIsPublic(deps, tipo, entidadID)
	if err != nil {
		return
	}
	dir, other := imageDirs(deps, public)
	for _, img := range deps.ImagenStore.ListByEntidad(tipo, entidadID) {
		if err := deps.Storage.MoveFileIfMissing(other, dir, img.FileName); err != nil {
			slog.Warn("no se pudo mover imagen", "tipo", tipo, "entidad", entidadID, "file", img.FileName, "error", err)
		}
	}
}

// EnsureCategoryImagesPlacement mueve las imágenes de la categoría y de todos
// sus productos, porque el estado de la categoría afecta a ambos.
func EnsureCategoryImagesPlacement(deps *Dependencies, categoriaID int64) {
	EnsureImagePlacement(deps, "categoria", categoriaID)
	for _, producto := range deps.ProductoStore.ListByCategoria(categoriaID) {
		EnsureImagePlacement(deps, "producto", producto.ID)
	}
}

// RemoveImageFiles borra los archivos físicos de una entidad de ambas carpetas
// (pública y privada), para cubrir cualquier transición previa.
func RemoveImageFiles(deps *Dependencies, tipo string, entidadID int64) {
	var filenames []string
	for _, img := range deps.ImagenStore.ListByEntidad(tipo, entidadID) {
		filenames = append(filenames, img.FileName)
	}
	RemoveImageFileNames(deps, filenames)
}

// RemoveImageFileNames borra archivos de imágenes de ambas carpetas.
func RemoveImageFileNames(deps *Dependencies, filenames []string) {
	deps.Storage.RemoveIn(deps.Config.PublicImagesDir(), filenames...)
	deps.Storage.RemoveIn(deps.Config.PrivateImagesDir(), filenames...)
}

func uploadImagesForEntity(deps *Dependencies, tipo string, entidadID int64, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+(10<<20))
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "error al parsear formulario o archivo demasiado grande")
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		writeJSONError(w, http.StatusBadRequest, "images requerido")
		return
	}
	if len(files) > maxArchivoFiles {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("máximo %d imágenes por carga", maxArchivoFiles))
		return
	}
	for _, fh := range files {
		if err := validateImageContent(fh); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	public, err := ImageIsPublic(deps, tipo, entidadID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "entidad no encontrada")
		return
	}

	dir, _ := imageDirs(deps, public)
	prefix := fmt.Sprintf("%s_%d", tipo, entidadID)
	filenames, err := deps.Storage.SaveFilesIn(dir, files, prefix, imageExts)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var created []int64
	rollback := func() {
		for _, cid := range created {
			if _, ferr := deps.ImagenStore.FindByID(cid); ferr == nil {
				_ = deps.ImagenStore.Delete(cid)
			}
		}
		deps.Storage.RemoveIn(dir, filenames...)
	}

	for _, fn := range filenames {
		img, err := deps.ImagenStore.Add(tipo, entidadID, fn)
		if err != nil {
			rollback()
			writeJSONError(w, http.StatusInternalServerError, "error al registrar imagen")
			return
		}
		created = append(created, img.ID)
	}

	writeJSON(w, http.StatusCreated, outputs.ToImagenResponses(deps.ImagenStore.ListByEntidad(tipo, entidadID), public))
}

func deleteImageForEntity(deps *Dependencies, tipo string, entidadID, imagenID int64, w http.ResponseWriter) {
	img, err := deps.ImagenStore.FindByID(imagenID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "imagen not found")
		return
	}
	if img.Tipo != tipo || img.EntidadID != entidadID {
		writeJSONError(w, http.StatusNotFound, "imagen not found")
		return
	}

	// Primero la baja en el store; solo tras éxito se borran los archivos.
	if err := deps.ImagenStore.Delete(imagenID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete imagen")
		return
	}
	deps.Storage.RemoveIn(deps.Config.PublicImagesDir(), img.FileName)
	deps.Storage.RemoveIn(deps.Config.PrivateImagesDir(), img.FileName)

	writeJSON(w, http.StatusOK, map[string]string{"message": "imagen deleted"})
}

func parseImagenParams(r *http.Request) (int64, int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	imagenID, err := strconv.ParseInt(chi.URLParam(r, "imagenID"), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return id, imagenID, true
}

func reorderImagesForEntity(deps *Dependencies, tipo string, entidadID int64, w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := deps.ImagenStore.Reorder(tipo, entidadID, body.IDs); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	public, err := ImageIsPublic(deps, tipo, entidadID)
	if err != nil {
		public = false
	}
	writeJSON(w, http.StatusOK, outputs.ToImagenResponses(deps.ImagenStore.ListByEntidad(tipo, entidadID), public))
}

type ImagenHandler struct {
	deps *Dependencies
}

func NewImagenHandler(deps *Dependencies) *ImagenHandler {
	return &ImagenHandler{deps: deps}
}

// Serve entrega imágenes restringidas. Las públicas las sirve nginx; si llega
// una petición pública aquí, se responde desde la carpeta pública como respaldo.
func (h *ImagenHandler) Serve(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if !safeFileName(filename) {
		http.NotFound(w, r)
		return
	}

	img, err := h.deps.ImagenStore.FindByFileName(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	public, err := ImageIsPublic(h.deps, img.Tipo, img.EntidadID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dir, other := imageDirs(h.deps, public)
	if !public {
		claims := GetClaims(r.Context())
		if claims == nil {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if !tienePermiso(claims.Permisos, "shop") {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	// Defensa: si una transición falló y el archivo quedó en la otra carpeta,
	// moverlo a la que corresponde.
	if _, err := os.Stat(filepath.Join(other, filename)); err == nil {
		_ = h.deps.Storage.MoveFileIfMissing(other, dir, filename)
	}

	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Respaldo: probar la otra carpeta.
		filePath = filepath.Join(other, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
	}

	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if public {
		w.Header().Set("Cache-Control", "public, max-age=60")
	} else {
		w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	}
	http.ServeFile(w, r, filePath)
}
