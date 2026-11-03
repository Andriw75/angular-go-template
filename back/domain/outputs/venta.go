package outputs

import (
	"time"

	"back/domain"
)

type VentaItemResponse struct {
	ID                 int64   `json:"id"`
	ProductoID         *int64  `json:"producto_id,omitempty"`
	Descripcion        string  `json:"descripcion"`
	Cantidad           float64 `json:"cantidad"`
	PrecioUnitario     float64 `json:"precio_unitario"`
	Descuento          float64 `json:"descuento"`
	ImpuestoPorcentaje float64 `json:"impuesto_porcentaje"`
	Subtotal           float64 `json:"subtotal"`
	Impuesto           float64 `json:"impuesto"`
	Total              float64 `json:"total"`
}

type VentaResponse struct {
	ID               int64               `json:"id"`
	Serie            string              `json:"serie"`
	Correlativo      int64               `json:"correlativo"`
	Codigo           string              `json:"codigo"`
	ClienteNombre    string              `json:"cliente_nombre"`
	ClienteDocumento string              `json:"cliente_documento"`
	ClienteDireccion string              `json:"cliente_direccion"`
	Fecha            string              `json:"fecha"`
	Moneda           string              `json:"moneda"`
	Subtotal         float64             `json:"subtotal"`
	Descuento        float64             `json:"descuento"`
	Impuesto         float64             `json:"impuesto"`
	Total            float64             `json:"total"`
	Estado           string              `json:"estado"`
	Observaciones    string              `json:"observaciones"`
	CreadoEn         string              `json:"creado_en"`
	ActualizadoEn    string              `json:"actualizado_en"`
	Items            []VentaItemResponse `json:"items"`
}

func ToVentaResponse(v *domain.Venta) VentaResponse {
	items := make([]VentaItemResponse, 0, len(v.Items))
	for _, i := range v.Items {
		items = append(items, VentaItemResponse{
			ID:                 i.ID,
			ProductoID:         i.ProductoID,
			Descripcion:        i.Descripcion,
			Cantidad:           i.Cantidad,
			PrecioUnitario:     i.PrecioUnitario,
			Descuento:          i.Descuento,
			ImpuestoPorcentaje: i.ImpuestoPorcentaje,
			Subtotal:           i.Subtotal,
			Impuesto:           i.Impuesto,
			Total:              i.Total,
		})
	}
	return VentaResponse{
		ID:               v.ID,
		Serie:            v.Serie,
		Correlativo:      v.Correlativo,
		Codigo:           v.Codigo,
		ClienteNombre:    v.ClienteNombre,
		ClienteDocumento: v.ClienteDocumento,
		ClienteDireccion: v.ClienteDireccion,
		Fecha:            v.Fecha.Format("2006-01-02"),
		Moneda:           v.Moneda,
		Subtotal:         v.Subtotal,
		Descuento:        v.Descuento,
		Impuesto:         v.Impuesto,
		Total:            v.Total,
		Estado:           v.Estado,
		Observaciones:    v.Observaciones,
		CreadoEn:         v.CreadoEn.Format(time.RFC3339),
		ActualizadoEn:    v.ActualizadoEn.Format(time.RFC3339),
		Items:            items,
	}
}

func ToVentaResponses(ventas []domain.Venta) []VentaResponse {
	res := make([]VentaResponse, 0, len(ventas))
	for i := range ventas {
		res = append(res, ToVentaResponse(&ventas[i]))
	}
	return res
}
