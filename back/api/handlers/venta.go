package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"back/domain/inputs"
	"back/domain/outputs"
	"back/store"
)

type VentaHandler struct {
	deps *Dependencies
}

func NewVentaHandler(deps *Dependencies) *VentaHandler {
	return &VentaHandler{deps: deps}
}

func parseVentaFilters(r *http.Request) store.VentaFilters {
	q := r.URL.Query()
	return store.VentaFilters{
		Q:      strings.TrimSpace(q.Get("q")),
		Serie:  strings.TrimSpace(q.Get("serie")),
		Estado: strings.TrimSpace(q.Get("estado")),
		Desde:  strings.TrimSpace(q.Get("desde")),
		Hasta:  strings.TrimSpace(q.Get("hasta")),
	}
}

func (h *VentaHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 1000 {
		limit = 20
	}

	ventas, total, err := h.deps.VentaStore.List(offset, limit, parseVentaFilters(r))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al listar ventas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":   outputs.ToVentaResponses(ventas),
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (h *VentaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	venta, err := h.deps.VentaStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "venta not found")
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToVentaResponse(venta))
}

func (h *VentaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.VentaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}

	venta, err := h.deps.VentaStore.Create(input, userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, outputs.ToVentaResponse(&venta))
}

func (h *VentaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var input inputs.VentaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	venta, err := h.deps.VentaStore.Update(id, input)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToVentaResponse(&venta))
}

func (h *VentaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.deps.VentaStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "venta deleted"})
}

func (h *VentaHandler) Emit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	venta, err := h.deps.VentaStore.Emit(id)
	if err != nil {
		slog.Error("failed to emit venta", "error", err, "id", id)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToVentaResponse(&venta))
}

func (h *VentaHandler) Annul(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Motivo string `json:"motivo"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	claims := GetClaims(r.Context())
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}

	venta, err := h.deps.VentaStore.Annul(id, body.Motivo, userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToVentaResponse(&venta))
}
