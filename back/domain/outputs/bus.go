package outputs

import (
	"time"

	"back/domain"
)

type BusResponse struct {
	ID                  int64      `json:"id"`
	Placa               string     `json:"placa"`
	Nombre              string     `json:"nombre"`
	Marca               string     `json:"marca"`
	Modelo              string     `json:"modelo"`
	Anio                int        `json:"anio"`
	Capacidad           int        `json:"capacidad"`
	Tipo                string     `json:"tipo"`
	Activo              bool       `json:"activo"`
	FechaCompra         string     `json:"fecha_compra"`
	UltimoMantenimiento *string    `json:"ultimo_mantenimiento,omitempty"`
	Precio              float64    `json:"precio"`
	Peso                float64    `json:"peso"`
	Color               string     `json:"color"`
	Descripcion         string     `json:"descripcion"`
	CreadoEn            string     `json:"creado_en"`
	ActualizadoEn       string     `json:"actualizado_en"`
}

func ToBusResponse(b *domain.Bus) BusResponse {
	r := BusResponse{
		ID:            b.ID,
		Placa:         b.Placa,
		Nombre:        b.Nombre,
		Marca:         b.Marca,
		Modelo:        b.Modelo,
		Anio:          b.Anio,
		Capacidad:     b.Capacidad,
		Tipo:          b.Tipo,
		Activo:        b.Activo,
		FechaCompra:   b.FechaCompra.Format("2006-01-02"),
		Precio:        b.Precio,
		Peso:          b.Peso,
		Color:         b.Color,
		Descripcion:   b.Descripcion,
		CreadoEn:      b.CreadoEn.Format(time.RFC3339),
		ActualizadoEn: b.ActualizadoEn.Format(time.RFC3339),
	}
	if b.UltimoMantenimiento != nil {
		s := b.UltimoMantenimiento.Format("2006-01-02")
		r.UltimoMantenimiento = &s
	}
	return r
}
