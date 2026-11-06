package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"back/domain"
	"back/domain/inputs"
	"back/domain/outputs"
	"back/services"
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
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, outputs.ToUserResponse(user))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input inputs.UserInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		writeError(w, r, domain.Validation("username, email and password are required"))
		return
	}
	if err := services.ValidatePasswordForUser(input.Username, input.Password); err != nil {
		writeError(w, r, domain.Validation(err.Error()))
		return
	}
	// Las cuentas creadas por un administrador deben cambiar la contraseña
	// inicial en el primer acceso.
	input.MustChangePassword = true

	created, err := h.deps.UserStore.Create(input)
	if err != nil {
		writeError(w, r, err)
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

	var input inputs.UserUpdateInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Password != nil && *input.Password != "" {
		username := ""
		if input.Username != nil {
			username = *input.Username
		} else if existing, err := h.deps.UserStore.FindByID(id); err == nil {
			username = existing.Username
		}
		if err := services.ValidatePasswordForUser(username, *input.Password); err != nil {
			writeError(w, r, domain.Validation(err.Error()))
			return
		}
	}

	if err := h.deps.UserStore.Update(id, input); err != nil {
		writeError(w, r, err)
		return
	}

	if h.deps.Config.JWTStoreEnabled &&
		(input.Permisos != nil || input.Activo != nil || (input.Password != nil && *input.Password != "")) {
		h.deps.JWTStore.RemoveUserTokens(id)
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
		writeError(w, r, err)
		return
	}

	if h.deps.Config.JWTStoreEnabled {
		h.deps.JWTStore.RemoveUserTokens(id)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
