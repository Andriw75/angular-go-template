package inputs

type BusInput struct {
	Placa               string  `json:"placa"`
	Nombre              string  `json:"nombre"`
	Marca               string  `json:"marca"`
	Modelo              string  `json:"modelo"`
	Anio                int     `json:"anio"`
	Capacidad           int     `json:"capacidad"`
	Tipo                string  `json:"tipo"`
	Activo              bool    `json:"activo"`
	FechaCompra         string  `json:"fecha_compra"`
	UltimoMantenimiento *string `json:"ultimo_mantenimiento,omitempty"`
	Precio              float64 `json:"precio"`
	Peso                float64 `json:"peso"`
	Color               string  `json:"color"`
	Descripcion         string  `json:"descripcion"`
}

type BusUpdateInput struct {
	Placa               *string  `json:"placa,omitempty"`
	Nombre              *string  `json:"nombre,omitempty"`
	Marca               *string  `json:"marca,omitempty"`
	Modelo              *string  `json:"modelo,omitempty"`
	Anio                *int     `json:"anio,omitempty"`
	Capacidad           *int     `json:"capacidad,omitempty"`
	Tipo                *string  `json:"tipo,omitempty"`
	Activo              *bool    `json:"activo,omitempty"`
	FechaCompra         *string  `json:"fecha_compra,omitempty"`
	UltimoMantenimiento **string `json:"ultimo_mantenimiento,omitempty"`
	Precio              *float64 `json:"precio,omitempty"`
	Peso                *float64 `json:"peso,omitempty"`
	Color               *string  `json:"color,omitempty"`
	Descripcion         *string  `json:"descripcion,omitempty"`
}
