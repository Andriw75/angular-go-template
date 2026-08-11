package outputs

import (
	"time"

	"back/domain"
)

type ProductoResponse struct {
	ID            int64   `json:"id"`
	Nombre        string  `json:"nombre"`
	Descripcion   string  `json:"descripcion"`
	Precio        float64 `json:"precio"`
	Stock         int     `json:"stock"`
	CategoriaID   int64   `json:"categoria_id"`
	Activo        bool    `json:"activo"`
	CreadoEn      string  `json:"creado_en"`
	ActualizadoEn string  `json:"actualizado_en"`
}

func ToProductoResponse(p *domain.Producto) ProductoResponse {
	return ProductoResponse{
		ID:            p.ID,
		Nombre:        p.Nombre,
		Descripcion:   p.Descripcion,
		Precio:        p.Precio,
		Stock:         p.Stock,
		CategoriaID:   p.CategoriaID,
		Activo:        p.Activo,
		CreadoEn:      p.CreadoEn.Format(time.RFC3339),
		ActualizadoEn: p.ActualizadoEn.Format(time.RFC3339),
	}
}
