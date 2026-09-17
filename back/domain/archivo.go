package domain

import "time"

type Visibilidad string

const (
	VisibilidadPublico     Visibilidad = "publico"
	VisibilidadAutenticado Visibilidad = "autenticado"
	VisibilidadPropietario Visibilidad = "propietario"
)

type Archivo struct {
	ID          int64        `json:"id"`
	FileName    string       `json:"file_name"`
	Nombre      string       `json:"nombre"`
	Tipo        string       `json:"tipo"`
	Tamano      int64        `json:"tamano"`
	Visibilidad Visibilidad  `json:"visibilidad"`
	UserID      int64        `json:"user_id"`
	CreadoEn    time.Time    `json:"creado_en"`
}

func (a *Archivo) SetID(id int64) { a.ID = id }

func IsValidVisibilidad(v Visibilidad) bool {
	return v == VisibilidadPublico || v == VisibilidadAutenticado || v == VisibilidadPropietario
}
