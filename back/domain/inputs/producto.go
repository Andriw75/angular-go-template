package inputs

type ProductoInput struct {
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio"`
	Stock       int     `json:"stock"`
	CategoriaID int64   `json:"categoria_id"`
	Activo      bool    `json:"activo"`
}
