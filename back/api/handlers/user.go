package handlers

import (
	"net/http"

	"back/domain/outputs"
)

type UserHandler struct {
	deps *Dependencies
}

func NewUserHandler(deps *Dependencies) *UserHandler {
	return &UserHandler{deps: deps}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users := h.deps.UserStore.FindAll()
	result := make([]outputs.UserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, outputs.ToUserResponse(&u))
	}
	writeJSON(w, http.StatusOK, result)
}
