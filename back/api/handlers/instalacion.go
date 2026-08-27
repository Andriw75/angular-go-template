package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/inputs"
)

type InstalacionHandler struct {
	deps *Dependencies
}

func NewInstalacionHandler(deps *Dependencies) *InstalacionHandler {
	return &InstalacionHandler{deps: deps}
}

func (h *InstalacionHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.deps.InstalacionStore.List())
}

func (h *InstalacionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.InstalacionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Tipo == "" || input.Nombre == "" {
		writeJSONError(w, http.StatusBadRequest, "tipo and nombre are required")
		return
	}
	if input.Estado == "" {
		input.Estado = "PENDIENTE_INSTALACION"
	}
	if input.Direccion == "" {
		input.Direccion = "Sin dirección"
	}
	if input.Zona == "" {
		input.Zona = "CENTRO"
	}
	if input.FechaInstalacion == "" {
		input.FechaInstalacion = time.Now().Format("2006-01-02")
	}
	// Sin serial no es bloqueante: se asigna uno automático.
	if input.Serial == "" {
		input.Serial = "SN-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	}

	inst := domain.Instalacion{
		Tipo:             input.Tipo,
		Nombre:           input.Nombre,
		Estado:           input.Estado,
		Lat:              input.Lat,
		Lng:              input.Lng,
		Direccion:        input.Direccion,
		Zona:             input.Zona,
		FechaInstalacion: input.FechaInstalacion,
		Serial:           input.Serial,
		Descripcion:      input.Descripcion,
		Fabricante:       input.Fabricante,
		Modelo:           input.Modelo,
		Instalador:       input.Instalador,
		Proveedor:        input.Proveedor,
		UltimaConexion:   input.UltimaConexion,
		Firmware:         input.Firmware,
		IP:               input.IP,
		MAC:              input.MAC,
		Resolucion:       input.Resolucion,
		Contenido:        input.Contenido,
		Senal:            input.Senal,
		Bateria:          input.Bateria,
		Potencia:         input.Potencia,
		Contacto:         input.Contacto,
	}

	created, err := h.deps.InstalacionStore.Create(inst)
	if err != nil {
		slog.Error("failed to create instalacion", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create instalacion")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *InstalacionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.InstalacionUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.deps.InstalacionStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "instalacion not found")
		return
	}

	if input.Tipo != nil {
		existing.Tipo = *input.Tipo
	}
	if input.Nombre != nil {
		existing.Nombre = *input.Nombre
	}
	if input.Estado != nil {
		existing.Estado = *input.Estado
	}
	if input.Lat != nil {
		existing.Lat = *input.Lat
	}
	if input.Lng != nil {
		existing.Lng = *input.Lng
	}
	if input.Direccion != nil {
		existing.Direccion = *input.Direccion
	}
	if input.Zona != nil {
		existing.Zona = *input.Zona
	}
	if input.FechaInstalacion != nil {
		existing.FechaInstalacion = *input.FechaInstalacion
	}
	if input.Serial != nil {
		existing.Serial = *input.Serial
	}
	applyOptionalString(&existing.Descripcion, input.Descripcion)
	applyOptionalString(&existing.Fabricante, input.Fabricante)
	applyOptionalString(&existing.Modelo, input.Modelo)
	applyOptionalString(&existing.Instalador, input.Instalador)
	applyOptionalString(&existing.Proveedor, input.Proveedor)
	applyOptionalString(&existing.Firmware, input.Firmware)
	applyOptionalString(&existing.IP, input.IP)
	applyOptionalString(&existing.MAC, input.MAC)
	applyOptionalString(&existing.Resolucion, input.Resolucion)
	applyOptionalString(&existing.Contenido, input.Contenido)
	applyOptionalNumber(&existing.Senal, input.Senal)
	applyOptionalNumber(&existing.Bateria, input.Bateria)
	applyOptionalNumber(&existing.Potencia, input.Potencia)
	applyOptionalString(&existing.Contacto, input.Contacto)

	// Permite limpiar ultima_conexion enviando null.
	if input.UltimaConexion != nil {
		existing.UltimaConexion = *input.UltimaConexion
	}

	if err := h.deps.InstalacionStore.Update(id, *existing); err != nil {
		slog.Error("failed to update instalacion", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update instalacion")
		return
	}

	writeJSON(w, http.StatusOK, *existing)
}

func (h *InstalacionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.deps.InstalacionStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "instalacion not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "instalacion deleted"})
}

// applyOptionalString aplica un puntero opcional; si llega con string vacío, limpia el campo.
func applyOptionalString(dst **string, src *string) {
	if src == nil {
		return
	}
	if *src == "" {
		*dst = nil
		return
	}
	*dst = src
}

func applyOptionalNumber(dst **float64, src **float64) {
	if src == nil {
		return
	}
	if *src == nil {
		*dst = nil
		return
	}
	*dst = *src
}
