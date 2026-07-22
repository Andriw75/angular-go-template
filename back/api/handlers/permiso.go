package handlers

import "net/http"

type PermisoHandler struct {
	deps *Dependencies
}

func NewPermisoHandler(deps *Dependencies) *PermisoHandler {
	return &PermisoHandler{deps: deps}
}

func (h *PermisoHandler) List(w http.ResponseWriter, r *http.Request) {
	permisos := h.deps.PermisoStore.FindAll()
	writeJSON(w, http.StatusOK, permisos)
}
