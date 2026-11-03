package domain

import "time"

type VentaItem struct {
	ID                 int64
	VentaID            int64
	ProductoID         *int64
	Descripcion        string
	Cantidad           float64
	PrecioUnitario     float64
	Descuento          float64
	ImpuestoPorcentaje float64
	Subtotal           float64
	Impuesto           float64
	Total              float64
}

type Venta struct {
	ID               int64
	Serie            string
	Correlativo      int64
	Codigo           string
	ClienteNombre    string
	ClienteDocumento string
	ClienteDireccion string
	Fecha            time.Time
	Moneda           string
	Subtotal         float64
	Descuento        float64
	Impuesto         float64
	Total            float64
	Estado           string
	Observaciones    string
	UserID           int64
	CreadoEn         time.Time
	ActualizadoEn    time.Time
	Items            []VentaItem
}
