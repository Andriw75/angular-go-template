package inputs

type UserInput struct {
	Username           string   `json:"username"`
	Email              string   `json:"email"`
	Password           string   `json:"password"`
	Activo             bool     `json:"activo"`
	Permisos           []string `json:"permisos"`
	MustChangePassword bool     `json:"must_change_password"`
}

type UserUpdateInput struct {
	Username           *string   `json:"username,omitempty"`
	Email              *string   `json:"email,omitempty"`
	Password           *string   `json:"password,omitempty"`
	Activo             *bool     `json:"activo,omitempty"`
	Permisos           *[]string `json:"permisos,omitempty"`
	MustChangePassword *bool     `json:"must_change_password,omitempty"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
