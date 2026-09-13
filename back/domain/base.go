package domain

import "time"

type User struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Password      string    `json:"-"`
	Activo        bool      `json:"activo"`
	Permisos      []string  `json:"permisos"`
	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

func (u *User) SetID(id int64) { u.ID = id }

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
