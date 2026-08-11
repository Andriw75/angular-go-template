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

func (u *User) SetID(id int64) { u.ID = id }

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

func (b *Bus) SetID(id int64) { b.ID = id }

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

type MensajePendiente struct {
	ID                  int64      `json:"id"`
	Telefono            string     `json:"telefono"`
	HoraSolicitada      time.Time  `json:"hora_solicitada"`
	HoraDesactivacion   time.Time  `json:"hora_desactivacion"`
	UsuarioAcargo       *string    `json:"usuario_acargo"`
	HoraUsuarioAsignado *time.Time `json:"hora_usuario_asignado"`
	Estado              string     `json:"estado"`
	CreadoEn            time.Time  `json:"creado_en"`
	ActualizadoEn       time.Time  `json:"actualizado_en"`
}

func (m *MensajePendiente) SetID(id int64) { m.ID = id }

type Categoria struct {
	ID             int64     `json:"id"`
	Nombre         string    `json:"nombre"`
	Descripcion    string    `json:"descripcion"`
	Activo         bool      `json:"activo"`
	CreadoEn       time.Time `json:"creado_en"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

func (c *Categoria) SetID(id int64) { c.ID = id }

type Producto struct {
	ID            int64     `json:"id"`
	Nombre        string    `json:"nombre"`
	Descripcion   string    `json:"descripcion"`
	Precio        float64   `json:"precio"`
	Stock         int       `json:"stock"`
	CategoriaID   int64     `json:"categoria_id"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

func (p *Producto) SetID(id int64) { p.ID = id }

type Imagen struct {
	ID        int64     `json:"id"`
	Tipo      string    `json:"tipo"`
	EntidadID int64     `json:"entidad_id"`
	FileName  string    `json:"file_name"`
	Orden     int       `json:"orden"`
	CreadoEn  time.Time `json:"creado_en"`
}

func (i *Imagen) SetID(id int64) { i.ID = id }
