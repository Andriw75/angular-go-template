package handlers

import (
	"net/http"
	"sort"

	"back/domain/outputs"
)

// sseChannels declara los canales que viajan por el SSE global. Cada canal
// (module, name) tiene su propio permiso; si un modulo emite varias cosas,
// se agregan mas filas con el mismo module y distinto name.
var sseChannels = []struct {
	Module     string
	Name       string
	Permission string
}{
	{Module: "mensajes_pendientes", Name: "activos", Permission: "mensajes_pendientes"},
}

type SSEHandler struct {
	deps *Dependencies
}

func NewSSEHandler(deps *Dependencies) *SSEHandler {
	return &SSEHandler{deps: deps}
}

func (h *SSEHandler) Events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	claims := GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	channels := allowedSSEChannels(claims.Permisos)
	if len(channels) == 0 {
		writeJSONError(w, http.StatusForbidden, "sin permisos para tiempo real")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Registrar antes de enviar el snapshot evita perder eventos que ocurran
	// justo al conectar. El cliente deduplica los create por id.
	client := h.deps.SSEHub.Register(channels)
	defer h.deps.SSEHub.Unregister(client)

	for _, ch := range channels {
		h.writeCurrent(w, flusher, ch)
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-client.ch:
			if !ok {
				return
			}
			sseWriteEvent(w, flusher, event)
		}
	}
}

func allowedSSEChannels(permisos []string) []sseChannel {
	allowed := make([]sseChannel, 0, len(sseChannels))
	seen := make(map[sseChannel]struct{}, len(sseChannels))

	for _, ch := range sseChannels {
		if !tienePermiso(permisos, ch.Permission) {
			continue
		}
		key := sseChannel{Module: ch.Module, Name: ch.Name}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		allowed = append(allowed, key)
	}

	sort.Slice(allowed, func(i, j int) bool {
		if allowed[i].Module == allowed[j].Module {
			return allowed[i].Name < allowed[j].Name
		}
		return allowed[i].Module < allowed[j].Module
	})
	return allowed
}

func (h *SSEHandler) writeCurrent(w http.ResponseWriter, flusher http.Flusher, ch sseChannel) {
	switch ch.Module {
	case "mensajes_pendientes":
		activos := h.deps.ActivosStore.GetAll()
		resp := make([]outputs.MensajeResponse, 0, len(activos))
		for i := range activos {
			resp = append(resp, outputs.ToMensajeResponse(&activos[i]))
		}
		sseWriteEvent(w, flusher, SSEEvent{
			Module: ch.Module,
			Name:   ch.Name,
			Type:   "current",
			Data:   resp,
		})
	}
}
