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

type CategoriaHandler struct {
	deps *Dependencies
}

func NewCategoriaHandler(deps *Dependencies) *CategoriaHandler {
	return &CategoriaHandler{deps: deps}
}

func parseCategoriaFilters(r *http.Request) mock.CategoriaFilters {
	q := r.URL.Query()
	f := mock.CategoriaFilters{
		Q: strings.TrimSpace(q.Get("q")),
	}
	if s := q.Get("activo"); s != "" {
		v, err := strconv.ParseBool(s)
		if err == nil {
			f.Activo = &v
		}
	}
	return f
}

func (h *CategoriaHandler) Count(w http.ResponseWriter, r *http.Request) {
	count := h.deps.CategoriaStore.Count(parseCategoriaFilters(r))
	writeJSON(w, http.StatusOK, count)
}

func (h *CategoriaHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	result := h.deps.CategoriaStore.List(offset, limit, parseCategoriaFilters(r))
	for i := range result.Data {
		result.Data[i].Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("categoria", result.Data[i].ID))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CategoriaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	categoria, err := h.deps.CategoriaStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}

	resp := outputs.ToCategoriaResponse(categoria)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("categoria", resp.ID))
	writeJSON(w, http.StatusOK, resp)
}

func (h *CategoriaHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.CategoriaStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}
	uploadImagesForEntity(h.deps, "categoria", id, w, r)
}

func (h *CategoriaHandler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id, imagenID, ok := parseImagenParams(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.CategoriaStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}
	deleteImageForEntity(h.deps, "categoria", id, imagenID, w)
}

func (h *CategoriaHandler) ReorderImages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.deps.CategoriaStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}
	reorderImagesForEntity(h.deps, "categoria", id, w, r)
}

func (h *CategoriaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.CategoriaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Nombre == "" {
		writeJSONError(w, http.StatusBadRequest, "nombre is required")
		return
	}

	categoria := domain.Categoria{
		Nombre:      input.Nombre,
		Descripcion: input.Descripcion,
		Activo:      input.Activo,
	}

	created, err := h.deps.CategoriaStore.Create(categoria)
	if err != nil {
		slog.Error("failed to create categoria", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create categoria")
		return
	}

	resp := outputs.ToCategoriaResponse(&created)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("categoria", resp.ID))
	writeJSON(w, http.StatusCreated, resp)
}

func (h *CategoriaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.CategoriaUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.deps.CategoriaStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}

	if input.Nombre != nil {
		existing.Nombre = *input.Nombre
	}
	if input.Descripcion != nil {
		existing.Descripcion = *input.Descripcion
	}
	if input.Activo != nil {
		existing.Activo = *input.Activo
	}

	if err := h.deps.CategoriaStore.Update(id, *existing); err != nil {
		slog.Error("failed to update categoria", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update categoria")
		return
	}

	resp := outputs.ToCategoriaResponse(existing)
	resp.Imagenes = outputs.ToImagenResponses(h.deps.ImagenStore.ListByEntidad("categoria", resp.ID))
	writeJSON(w, http.StatusOK, resp)
}

func (h *CategoriaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if _, err := h.deps.CategoriaStore.FindByID(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}

	cascade, _ := strconv.ParseBool(r.URL.Query().Get("cascade"))

	products := h.deps.ProductoStore.ListByCategoria(id)
	if len(products) > 0 && !cascade {
		writeJSONError(w, http.StatusConflict, "categoria has products, use cascade=true to delete them too")
		return
	}

	// Recolectar archivos a eliminar (se borran solo tras el éxito en el store).
	var filesToRemove []string
	if cascade {
		for _, p := range products {
			for _, img := range h.deps.ImagenStore.ListByEntidad("producto", p.ID) {
				filesToRemove = append(filesToRemove, img.FileName)
			}
		}
	}
	for _, img := range h.deps.ImagenStore.ListByEntidad("categoria", id) {
		filesToRemove = append(filesToRemove, img.FileName)
	}

	// Bajas en el store (negocio) primero.
	if cascade {
		for _, p := range products {
			h.deps.ImagenStore.DeleteByEntidad("producto", p.ID)
			_ = h.deps.ProductoStore.Delete(p.ID)
		}
	}
	h.deps.ImagenStore.DeleteByEntidad("categoria", id)
	if err := h.deps.CategoriaStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "categoria not found")
		return
	}

	// Solo tras el éxito: limpiar archivos en disco.
	h.deps.Storage.RemoveImages(filesToRemove...)

	writeJSON(w, http.StatusOK, map[string]string{"message": "categoria deleted"})
}
