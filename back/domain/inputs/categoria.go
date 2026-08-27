package inputs

type CategoriaInput struct {
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	Activo      bool   `json:"activo"`
}

type CategoriaUpdateInput struct {
	Nombre      *string `json:"nombre,omitempty"`
	Descripcion *string `json:"descripcion,omitempty"`
	Activo      *bool   `json:"activo,omitempty"`
}
