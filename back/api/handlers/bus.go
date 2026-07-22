package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/inputs"
	"back/domain/outputs"
	"back/mock"
)

type BusHandler struct {
	deps *Dependencies
}

func NewBusHandler(deps *Dependencies) *BusHandler {
	return &BusHandler{deps: deps}
}

func parseBusFilters(r *http.Request) mock.BusFilters {
	q := r.URL.Query()
	f := mock.BusFilters{
		Q:    strings.TrimSpace(q.Get("q")),
		Tipo: strings.TrimSpace(q.Get("tipo")),
	}
	if s := q.Get("activo"); s != "" {
		v, err := strconv.ParseBool(s)
		if err == nil {
			f.Activo = &v
		}
	}
	return f
}

func (h *BusHandler) Count(w http.ResponseWriter, r *http.Request) {
	filters := parseBusFilters(r)
	count := h.deps.BusStore.Count(filters)
	writeJSON(w, http.StatusOK, count)
}

func (h *BusHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	result := h.deps.BusStore.List(offset, limit, parseBusFilters(r))
	writeJSON(w, http.StatusOK, result)
}

func (h *BusHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	bus, err := h.deps.BusStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "bus not found")
		return
	}

	writeJSON(w, http.StatusOK, outputs.ToBusResponse(bus))
}

func (h *BusHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.BusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fechaCompra, err := time.Parse("2006-01-02", input.FechaCompra)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid fecha_compra, use YYYY-MM-DD")
		return
	}

	var ultimoMantenimiento *time.Time
	if input.UltimoMantenimiento != nil {
		t, err := time.Parse("2006-01-02", *input.UltimoMantenimiento)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid ultimo_mantenimiento, use YYYY-MM-DD")
			return
		}
		ultimoMantenimiento = &t
	}

	bus := domain.Bus{
		Placa:               input.Placa,
		Nombre:              input.Nombre,
		Marca:               input.Marca,
		Modelo:              input.Modelo,
		Anio:                input.Anio,
		Capacidad:           input.Capacidad,
		Tipo:                input.Tipo,
		Activo:              input.Activo,
		FechaCompra:         fechaCompra,
		UltimoMantenimiento: ultimoMantenimiento,
		Precio:              input.Precio,
		Peso:                input.Peso,
		Color:               input.Color,
		Descripcion:         input.Descripcion,
	}

	created, err := h.deps.BusStore.Create(bus)
	if err != nil {
		slog.Error("failed to create bus", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create bus")
		return
	}

	writeJSON(w, http.StatusCreated, outputs.ToBusResponse(&created))
}

func (h *BusHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.BusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fechaCompra, err := time.Parse("2006-01-02", input.FechaCompra)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid fecha_compra, use YYYY-MM-DD")
		return
	}

	var ultimoMantenimiento *time.Time
	if input.UltimoMantenimiento != nil {
		t, err := time.Parse("2006-01-02", *input.UltimoMantenimiento)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid ultimo_mantenimiento, use YYYY-MM-DD")
			return
		}
		ultimoMantenimiento = &t
	}

	existing, err := h.deps.BusStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "bus not found")
		return
	}

	existing.Placa = input.Placa
	existing.Nombre = input.Nombre
	existing.Marca = input.Marca
	existing.Modelo = input.Modelo
	existing.Anio = input.Anio
	existing.Capacidad = input.Capacidad
	existing.Tipo = input.Tipo
	existing.Activo = input.Activo
	existing.FechaCompra = fechaCompra
	existing.UltimoMantenimiento = ultimoMantenimiento
	existing.Precio = input.Precio
	existing.Peso = input.Peso
	existing.Color = input.Color
	existing.Descripcion = input.Descripcion

	if err := h.deps.BusStore.Update(id, *existing); err != nil {
		slog.Error("failed to update bus", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update bus")
		return
	}

	writeJSON(w, http.StatusOK, outputs.ToBusResponse(existing))
}

func (h *BusHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.deps.BusStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "bus not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "bus deleted"})
}
