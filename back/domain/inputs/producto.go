package inputs

type ProductoInput struct {
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio"`
	Stock       int     `json:"stock"`
	CategoriaID int64   `json:"categoria_id"`
	Activo      bool    `json:"activo"`
}

type ProductoUpdateInput struct {
	Nombre      *string  `json:"nombre,omitempty"`
	Descripcion *string  `json:"descripcion,omitempty"`
	Precio      *float64 `json:"precio,omitempty"`
	Stock       *int     `json:"stock,omitempty"`
	CategoriaID *int64   `json:"categoria_id,omitempty"`
	Activo      *bool    `json:"activo,omitempty"`
}
