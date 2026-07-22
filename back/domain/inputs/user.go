package inputs

type UserInput struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Activo   bool     `json:"activo"`
	Permisos []string `json:"permisos"`
}

type UserUpdateInput struct {
	Username *string   `json:"username,omitempty"`
	Email    *string   `json:"email,omitempty"`
	Password *string   `json:"password,omitempty"`
	Activo   *bool     `json:"activo,omitempty"`
	Permisos *[]string `json:"permisos,omitempty"`
}
