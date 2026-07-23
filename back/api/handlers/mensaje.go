package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain/inputs"
	"back/domain/outputs"
)

type MensajeHandler struct {
	deps *Dependencies
}

func NewMensajeHandler(deps *Dependencies) *MensajeHandler {
	return &MensajeHandler{deps: deps}
}

func (h *MensajeHandler) Events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send current active state immediately on connect
	activos := h.deps.ActivosStore.GetAll()
	currentResp := make([]outputs.MensajeResponse, 0, len(activos))
	for i := range activos {
		currentResp = append(currentResp, outputs.ToMensajeResponse(&activos[i]))
	}
	currentEvent, _ := json.Marshal(SSEEvent{Type: "current", Data: currentResp})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", currentEvent)
	flusher.Flush()

	ch := h.deps.SSEHub.Register()
	defer h.deps.SSEHub.Unregister(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *MensajeHandler) List(w http.ResponseWriter, r *http.Request) {
	activos := h.deps.ActivosStore.GetAll()
	result := make([]outputs.MensajeResponse, 0, len(activos))
	for i := range activos {
		result = append(result, outputs.ToMensajeResponse(&activos[i]))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *MensajeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if m, ok := h.deps.ActivosStore.GetByID(id); ok {
		writeJSON(w, http.StatusOK, outputs.ToMensajeResponse(&m))
		return
	}

	m, err := h.deps.MensajeStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, outputs.ToMensajeResponse(m))
}

func (h *MensajeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.MensajeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Telefono == "" || input.HoraDesactivacion == "" {
		writeJSONError(w, http.StatusBadRequest, "telefono and hora_desactivacion are required")
		return
	}

	now := time.Now()
	created, err := h.deps.MensajeStore.Create(input, now)
	if err != nil {
		slog.Error("failed to create mensaje", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create mensaje")
		return
	}

	h.deps.ActivosStore.Set(created)

	resp := outputs.ToMensajeResponse(&created)
	h.deps.SSEHub.Broadcast(SSEEvent{Type: "create", Data: resp})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *MensajeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.MensajeUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims := GetClaims(r.Context())
	username := ""
	if claims != nil {
		username = claims.Username
	}

	now := time.Now()
	updated, err := h.deps.MensajeStore.Update(id, input, now, username)
	if err != nil {
		slog.Error("failed to update mensaje", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update mensaje")
		return
	}

	resp := outputs.ToMensajeResponse(&updated)

	if updated.Estado == "finalizado" {
		h.deps.ActivosStore.Delete(id)
		h.deps.SSEHub.Broadcast(SSEEvent{Type: "delete", Data: map[string]int64{"id": id}})
	} else {
		h.deps.ActivosStore.Set(updated)
		h.deps.SSEHub.Broadcast(SSEEvent{Type: "update", Data: resp})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *MensajeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.deps.MensajeStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}

	h.deps.ActivosStore.Delete(id)
	h.deps.SSEHub.Broadcast(SSEEvent{Type: "delete", Data: map[string]int64{"id": id}})
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *MensajeHandler) ListPasadas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	pasadas, total := h.deps.MensajeStore.ListPasadas(offset, limit)
	result := make([]outputs.MensajeResponse, 0, len(pasadas))
	for i := range pasadas {
		result = append(result, outputs.ToMensajeResponse(&pasadas[i]))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   result,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (h *MensajeHandler) CountPasadas(w http.ResponseWriter, r *http.Request) {
	count := h.deps.MensajeStore.CountPasadas()
	writeJSON(w, http.StatusOK, count)
}
