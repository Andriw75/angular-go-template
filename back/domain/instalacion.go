package domain

import "time"

type Instalacion struct {
	ID               int64     `json:"id"`
	Tipo             string    `json:"tipo"`
	Nombre           string    `json:"nombre"`
	Estado           string    `json:"estado"`
	Lat              float64   `json:"lat"`
	Lng              float64   `json:"lng"`
	Direccion        string    `json:"direccion"`
	Zona             string    `json:"zona"`
	FechaInstalacion string    `json:"fecha_instalacion"`
	Serial           string    `json:"serial"`
	Descripcion      *string   `json:"descripcion,omitempty"`
	Fabricante       *string   `json:"fabricante,omitempty"`
	Modelo           *string   `json:"modelo,omitempty"`
	Instalador       *string   `json:"instalador,omitempty"`
	Proveedor        *string   `json:"proveedor,omitempty"`
	UltimaConexion   *string   `json:"ultima_conexion,omitempty"`
	Firmware         *string   `json:"firmware,omitempty"`
	IP               *string   `json:"ip,omitempty"`
	MAC              *string   `json:"mac,omitempty"`
	Resolucion       *string   `json:"resolucion,omitempty"`
	Contenido        *string   `json:"contenido,omitempty"`
	Senal            *float64  `json:"senal,omitempty"`
	Bateria          *float64  `json:"bateria,omitempty"`
	Potencia         *float64  `json:"potencia,omitempty"`
	Contacto         *string   `json:"contacto,omitempty"`
	CreadoEn         time.Time `json:"creado_en"`
	ActualizadoEn    time.Time `json:"actualizado_en"`
}

func (i *Instalacion) SetID(id int64) { i.ID = id }
