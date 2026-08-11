package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/inputs"
	"back/domain/outputs"
	"back/mock"
)

type ProductoHandler struct {
	deps *Dependencies
}

func NewProductoHandler(deps *Dependencies) *ProductoHandler {
	return &ProductoHandler{deps: deps}
}

func parseProductoFilters(r *http.Request) mock.ProductoFilters {
	q := r.URL.Query()
	f := mock.ProductoFilters{
		Q: strings.TrimSpace(q.Get("q")),
	}
	if s := q.Get("categoria_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			f.CategoriaID = v
		}
	}
	if s := q.Get("activo"); s != "" {
		v, err := strconv.ParseBool(s)
		if err == nil {
			f.Activo = &v
		}
	}
	return f
}

func (h *ProductoHandler) Count(w http.ResponseWriter, r *http.Request) {
	count := h.deps.ProductoStore.Count(parseProductoFilters(r))
	writeJSON(w, http.StatusOK, count)
}

func (h *ProductoHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	result := h.deps.ProductoStore.List(offset, limit, parseProductoFilters(r))
	for i := range result.Data {
		result.Data[i].Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("producto", result.Data[i].ID))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ProductoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	producto, err := h.deps.ProductoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}

	resp := outputs.ToProductoResponse(producto)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("producto", resp.ID))
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProductoHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.ProductoStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}
	uploadImagesForEntity(h.deps, "producto", id, w, r)
}

func (h *ProductoHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id, imagenID, ok := parseImagenParams(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.ProductoStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}
	deleteImageForEntity(h.deps, "producto", id, imagenID, w)
}

func (h *ProductoHandler) ReorderImages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.ProductoStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}
	reorderImagesForEntity(h.deps, "producto", id, w, r)
}

func (h *ProductoHandler) validateCategoria(id int64) bool {
	_, err := h.deps.CategoriaStore.FindByID(id)
	return err == nil
}

func (h *ProductoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.ProductoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Nombre == "" {
		writeJSONError(w, http.StatusBadRequest, "nombre is required")
		return
	}
	if input.CategoriaID <= 0 || !h.validateCategoria(input.CategoriaID) {
		writeJSONError(w, http.StatusBadRequest, "invalid categoria_id")
		return
	}

	producto := domain.Producto{
		Nombre:      input.Nombre,
		Descripcion: input.Descripcion,
		Precio:      input.Precio,
		Stock:       input.Stock,
		CategoriaID: input.CategoriaID,
		Activo:      input.Activo,
	}

	created, err := h.deps.ProductoStore.Create(producto)
	if err != nil {
		slog.Error("failed to create producto", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create producto")
		return
	}

	resp := outputs.ToProductoResponse(&created)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("producto", resp.ID))
	writeJSON(w, http.StatusCreated, resp)
}

func (h *ProductoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.ProductoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.deps.ProductoStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}

	if input.CategoriaID <= 0 || !h.validateCategoria(input.CategoriaID) {
		writeJSONError(w, http.StatusBadRequest, "invalid categoria_id")
		return
	}

	existing.Nombre = input.Nombre
	existing.Descripcion = input.Descripcion
	existing.Precio = input.Precio
	existing.Stock = input.Stock
	existing.CategoriaID = input.CategoriaID
	existing.Activo = input.Activo

	if err := h.deps.ProductoStore.Update(id, *existing); err != nil {
		slog.Error("failed to update producto", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update producto")
		return
	}

	resp := outputs.ToProductoResponse(existing)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("producto", resp.ID))
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProductoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var filesToRemove []string
	for _, img := range h.deps.ImagenStore.ListByEntidad("producto", id) {
		filesToRemove = append(filesToRemove, img.FileName)
	}

	// Baja en el store primero; si falla, no se toca nada.
	if err := h.deps.ProductoStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "producto not found")
		return
	}
	h.deps.ImagenStore.DeleteByEntidad("producto", id)

	// Solo tras el éxito: limpiar archivos en disco.
	h.deps.Storage.RemoveImages(filesToRemove...)

	writeJSON(w, http.StatusOK, map[string]string{"message": "producto deleted"})
}
