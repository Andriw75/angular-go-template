package inputs

type UserInput struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Activo   bool     `json:"activo"`
	Permisos []string `json:"permisos"`
}
