package domain

import "time"

type User struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Password       string    `json:"-"`
	Activo         bool      `json:"activo"`
	Permisos       []string  `json:"permisos"`
	CreadoEn       time.Time `json:"creado_en"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

type Bus struct {
	ID                  int64      `json:"id"`
	Placa               string     `json:"placa"`
	Nombre              string     `json:"nombre"`
	Marca               string     `json:"marca"`
	Modelo              string     `json:"modelo"`
	Anio                int        `json:"anio"`
	Capacidad           int        `json:"capacidad"`
	Tipo                string     `json:"tipo"`
	Activo              bool       `json:"activo"`
	FechaCompra         time.Time  `json:"fecha_compra"`
	UltimoMantenimiento *time.Time `json:"ultimo_mantenimiento,omitempty"`
	Precio              float64    `json:"precio"`
	Peso                float64    `json:"peso"`
	Color               string     `json:"color"`
	Descripcion         string     `json:"descripcion"`
	CreadoEn            time.Time  `json:"creado_en"`
	ActualizadoEn       time.Time  `json:"actualizado_en"`
}

type Permission struct {
	ID          int64  `json:"id"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
}

type UserPermission struct {
	UserID       int64  `json:"usuario_id"`
	PermissionID int64  `json:"permiso_id"`
	Permission   string `json:"permiso"`
}
