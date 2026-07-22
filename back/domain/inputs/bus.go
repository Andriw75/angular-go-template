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
