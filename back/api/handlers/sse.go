package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type SSEHub struct {
	mu         sync.RWMutex
	clients    map[chan SSEEvent]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan SSEEvent]struct{})}
}

func (h *SSEHub) Register() chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan SSEEvent, 64)
	h.clients[ch] = struct{}{}
	return ch
}

func (h *SSEHub) Unregister(ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

func (h *SSEHub) Broadcast(event SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			// client too slow, skip
		}
	}
}

func sseWrite(w http.ResponseWriter, flusher http.Flusher, ch chan SSEEvent, ctx <-chan struct{}) {
	for {
		select {
		case <-ctx:
			return
		case event := <-ch:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
