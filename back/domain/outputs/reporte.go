package outputs

type ReporteBusesResumen struct {
	Total             int     `json:"total"`
	Activos           int     `json:"activos"`
	Inactivos         int     `json:"inactivos"`
	PrecioTotal       float64 `json:"precio_total"`
	PesoTotal         float64 `json:"peso_total"`
	CapacidadPromedio float64 `json:"capacidad_promedio"`
}

type ReporteBusesTipo struct {
	Tipo              string  `json:"tipo"`
	Cantidad          int     `json:"cantidad"`
	PrecioTotal       float64 `json:"precio_total"`
	CapacidadPromedio float64 `json:"capacidad_promedio"`
}

type ReporteBuses struct {
	GeneradoEn string              `json:"generado_en"`
	Filtros    map[string]any      `json:"filtros"`
	Resumen    ReporteBusesResumen `json:"resumen"`
	PorTipo    []ReporteBusesTipo  `json:"por_tipo"`
	Data       []BusResponse       `json:"data"`
}

type ReporteVentasResumen struct {
	Total         int     `json:"total"`
	Emitidas      int     `json:"emitidas"`
	Anuladas      int     `json:"anuladas"`
	MontoTotal    float64 `json:"monto_total"`
	ImpuestoTotal float64 `json:"impuesto_total"`
	Clientes      int     `json:"clientes"`
}

type ReporteVentas struct {
	GeneradoEn string               `json:"generado_en"`
	Filtros    map[string]any       `json:"filtros"`
	Resumen    ReporteVentasResumen `json:"resumen"`
	Data       []VentaResponse      `json:"data"`
}
