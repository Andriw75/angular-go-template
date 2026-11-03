package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"back/store"
)

type LabHandler struct {
	deps *Dependencies
}

func NewLabHandler(deps *Dependencies) *LabHandler {
	return &LabHandler{deps: deps}
}

type slowQueryRequest struct {
	Modo     string `json:"modo"`
	Segundos int    `json:"segundos"`
	Filas    int64  `json:"filas"`
}

// Start crea una tarea en segundo plano y devuelve su estado inicial.
func (h *LabHandler) Start(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	input, ok := decodeSlowQuery(w, r)
	if !ok {
		return
	}

	info, err := h.deps.SlowQueryStore.Start(r.Context(), claims.UserID, input)
	if err != nil {
		writeSlowQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

// Run ejecuta la consulta dentro de la peticion HTTP. Angular puede abortarla
// con unsubscribe(); el contexto de Go cancela la consulta en PostgreSQL.
func (h *LabHandler) Run(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	input, ok := decodeSlowQuery(w, r)
	if !ok {
		return
	}

	info, err := h.deps.SlowQueryStore.RunSync(r.Context(), claims.UserID, input)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		writeSlowQueryError(w, err)
		return
	}
	if r.Context().Err() != nil {
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *LabHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, h.deps.SlowQueryStore.List())
}

func (h *LabHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	info, err := h.deps.SlowQueryStore.Cancel(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeSlowQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func decodeSlowQuery(w http.ResponseWriter, r *http.Request) (store.SlowQueryInput, bool) {
	var req slowQueryRequest
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return store.SlowQueryInput{}, false
	}
	if req.Modo == "" {
		req.Modo = string(store.SlowQuerySleep)
	}
	return store.SlowQueryInput{
		Modo:     store.SlowQueryModo(req.Modo),
		Segundos: req.Segundos,
		Filas:    req.Filas,
	}, true
}

func writeSlowQueryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrSlowQueryNotFound):
		writeJSONError(w, http.StatusNotFound, "consulta no encontrada")
	case errors.Is(err, store.ErrSlowQueryUnsupported):
		writeJSONError(w, http.StatusNotImplemented, err.Error())
	case errors.Is(err, store.ErrSlowQueryLimit):
		writeJSONError(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, store.ErrSlowQueryInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "no se pudo ejecutar la consulta")
	}
}

type labNotifier struct {
	hub *SSEHub
}

func (n labNotifier) Notify(event string, info store.SlowQueryInfo) {
	n.hub.Broadcast("laboratorio", "tareas", SSEEvent{
		Module: "laboratorio",
		Name:   "tareas",
		Type:   event,
		Data:   info,
	})
}

func NewLabNotifier(hub *SSEHub) store.SlowQueryNotifier {
	return labNotifier{hub: hub}
}
