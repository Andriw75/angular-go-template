package inputs

type VentaItemInput struct {
	ProductoID         *int64  `json:"producto_id,omitempty"`
	Descripcion        string  `json:"descripcion"`
	Cantidad           float64 `json:"cantidad"`
	PrecioUnitario     float64 `json:"precio_unitario"`
	Descuento          float64 `json:"descuento"`
	ImpuestoPorcentaje float64 `json:"impuesto_porcentaje"`
}

type VentaInput struct {
	Serie            string           `json:"serie"`
	ClienteNombre    string           `json:"cliente_nombre"`
	ClienteDocumento string           `json:"cliente_documento"`
	ClienteDireccion string           `json:"cliente_direccion"`
	Fecha            string           `json:"fecha"`
	Moneda           string           `json:"moneda"`
	Observaciones    string           `json:"observaciones"`
	Items            []VentaItemInput `json:"items"`
}
