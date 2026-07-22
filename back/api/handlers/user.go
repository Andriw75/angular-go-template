package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"back/domain/inputs"
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

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	user, err := h.deps.UserStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, outputs.ToUserResponse(user))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	created, err := h.deps.UserStore.Create(input)
	if err != nil {
		slog.Error("failed to create user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, outputs.ToUserResponse(&created))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var input inputs.UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Username == "" || input.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "username and email are required")
		return
	}

	if err := h.deps.UserStore.Update(id, input); err != nil {
		slog.Error("failed to update user", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	updated, _ := h.deps.UserStore.FindByID(id)
	writeJSON(w, http.StatusOK, outputs.ToUserResponse(updated))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := GetClaims(r.Context())
	if claims != nil && claims.UserID == id {
		writeJSONError(w, http.StatusForbidden, "cannot delete yourself")
		return
	}

	if err := h.deps.UserStore.Delete(id); err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
