package inputs

type CategoriaInput struct {
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	Activo      bool   `json:"activo"`
}
