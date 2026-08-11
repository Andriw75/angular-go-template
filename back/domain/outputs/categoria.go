package outputs

import (
	"time"

	"back/domain"
)

type CategoriaResponse struct {
	ID            int64  `json:"id"`
	Nombre        string `json:"nombre"`
	Descripcion   string `json:"descripcion"`
	Activo        bool   `json:"activo"`
	CreadoEn      string `json:"creado_en"`
	ActualizadoEn string `json:"actualizado_en"`
}

func ToCategoriaResponse(c *domain.Categoria) CategoriaResponse {
	return CategoriaResponse{
		ID:            c.ID,
		Nombre:        c.Nombre,
		Descripcion:   c.Descripcion,
		Activo:        c.Activo,
		CreadoEn:      c.CreadoEn.Format(time.RFC3339),
		ActualizadoEn: c.ActualizadoEn.Format(time.RFC3339),
	}
}
