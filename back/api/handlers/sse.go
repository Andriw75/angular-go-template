package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEEvent es el sobre unico que viaja por el SSE global.
type SSEEvent struct {
	Module string      `json:"module"`
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
}

type sseChannel struct {
	Module string
	Name   string
}

type sseClient struct {
	ch       chan SSEEvent
	channels map[sseChannel]struct{}
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[*sseClient]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[*sseClient]struct{})}
}

// Register crea un cliente suscrito a los canales indicados.
func (h *SSEHub) Register(channels []sseChannel) *sseClient {
	h.mu.Lock()
	defer h.mu.Unlock()

	set := make(map[sseChannel]struct{}, len(channels))
	for _, c := range channels {
		set[c] = struct{}{}
	}

	client := &sseClient{
		ch:       make(chan SSEEvent, 64),
		channels: set,
	}
	h.clients[client] = struct{}{}
	return client
}

func (h *SSEHub) Unregister(client *sseClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	close(client.ch)
}

// Broadcast envia el evento solo a los clientes suscritos al canal.
func (h *SSEHub) Broadcast(module, name string, event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := sseChannel{Module: module, Name: name}
	for client := range h.clients {
		if _, ok := client.channels[key]; !ok {
			continue
		}
		select {
		case client.ch <- event:
		default:
			// cliente lento: se descarta y se resincroniza al reconectar
		}
	}
}

func sseWriteEvent(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
