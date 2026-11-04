package services

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"back/domain/outputs"
	"back/store"
)

const (
	reportMaxRows     = int(^uint(0) >> 1)
	MaxGeneratedBuses = 1000
)

type ReporteService struct {
	buses  store.BusStore
	ventas store.VentaStore
}

func NewReporteService(buses store.BusStore, ventas store.VentaStore) *ReporteService {
	return &ReporteService{buses: buses, ventas: ventas}
}

func (s *ReporteService) Buses(filters store.BusFilters) outputs.ReporteBuses {
	result := s.buses.List(0, reportMaxRows, filters)
	return buildBusesReport(result.Data, filters, "actual", 0)
}

func (s *ReporteService) GeneratedBuses(count int, filters store.BusFilters) outputs.ReporteBuses {
	if count < 1 {
		count = 1
	}
	if count > MaxGeneratedBuses {
		count = MaxGeneratedBuses
	}
	return buildBusesReport(generateBuses(count, filters), filters, "generado", count)
}

func generateBuses(count int, filters store.BusFilters) []outputs.BusResponse {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()

	marcas := []string{"Mercedes-Benz", "Volvo", "Scania", "Hyundai", "Toyota", "Volkswagen", "MAN", "Iveco"}
	modelos := []string{"OF-1721", "B7R", "K410", "Universe", "Marcopolo", "Torino", "TH-3200", "BC-180"}
	tipos := []string{"BUS", "VAN", "MINIBUS", "MICROBUS"}
	colores := []string{"Blanco", "Rojo", "Azul", "Plateado", "Negro", "Amarillo", "Verde", "Naranja"}

	tipoFiltro := strings.ToUpper(strings.TrimSpace(filters.Tipo))
	qFiltro := strings.TrimSpace(filters.Q)
	desde, desdeOK := parseReportDay(filters.Desde)
	hasta, hastaOK := parseReportDay(filters.Hasta)
	if !desdeOK && !hastaOK {
		desde, hasta = time.Time{}, time.Time{}
	} else if !desdeOK {
		desde = hasta.AddDate(-10, 0, 0)
	} else if !hastaOK {
		hasta = now
	}
	if desdeOK || hastaOK {
		if hasta.Before(desde) {
			desde, hasta = hasta, desde
		}
	}

	data := make([]outputs.BusResponse, 0, count)
	for i := 0; i < count; i++ {
		var ultimo *string
		if rng.Intn(3) > 0 {
			d := now.AddDate(0, 0, -rng.Intn(180)).Format("2006-01-02")
			ultimo = &d
		}

		tipo := tipos[rng.Intn(len(tipos))]
		if tipoFiltro != "" {
			tipo = tipoFiltro
		}
		activo := rng.Intn(5) > 0
		if filters.Activo != nil {
			activo = *filters.Activo
		}
		nombre := fmt.Sprintf("Bus generado %03d", i+1)
		descripcion := "Dato autogenerado para demostracion"
		if qFiltro != "" {
			nombre = fmt.Sprintf("%s %s", nombre, qFiltro)
			descripcion = fmt.Sprintf("%s (filtro: %s)", descripcion, qFiltro)
		}

		fechaCompra := now.AddDate(0, 0, -(365 + rng.Intn(2000)))
		if desdeOK || hastaOK {
			span := int(hasta.Sub(desde).Hours() / 24)
			if span < 0 {
				span = 0
			}
			fechaCompra = desde.AddDate(0, 0, rng.Intn(span+1))
		}

		data = append(data, outputs.BusResponse{
			ID:                  int64(i + 1),
			Placa:               fmt.Sprintf("GEN-%04d", i+1),
			Nombre:              nombre,
			Marca:               marcas[rng.Intn(len(marcas))],
			Modelo:              modelos[rng.Intn(len(modelos))],
			Anio:                2015 + rng.Intn(10),
			Capacidad:           15 + rng.Intn(30),
			Tipo:                tipo,
			Activo:              activo,
			FechaCompra:         fechaCompra.Format("2006-01-02"),
			UltimoMantenimiento: ultimo,
			Precio:              round2(80000 + rng.Float64()*270000),
			Peso:                round2(8 + rng.Float64()*10),
			Color:               colores[rng.Intn(len(colores))],
			Descripcion:         descripcion,
			CreadoEn:            now.Format(time.RFC3339),
			ActualizadoEn:       now.Format(time.RFC3339),
		})
	}
	return data
}

func parseReportDay(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", value)
	return d, err == nil
}

func buildBusesReport(data []outputs.BusResponse, filters store.BusFilters, modo string, cantidad int) outputs.ReporteBuses {
	resumen := outputs.ReporteBusesResumen{Total: len(data)}
	porTipo := map[string]*outputs.ReporteBusesTipo{}
	var capacidadTotal float64

	for i := range data {
		b := &data[i]
		if b.Activo {
			resumen.Activos++
		} else {
			resumen.Inactivos++
		}
		resumen.PrecioTotal += b.Precio
		resumen.PesoTotal += b.Peso
		capacidadTotal += float64(b.Capacidad)

		t := porTipo[b.Tipo]
		if t == nil {
			t = &outputs.ReporteBusesTipo{Tipo: b.Tipo}
			porTipo[b.Tipo] = t
		}
		t.Cantidad++
		t.PrecioTotal += b.Precio
		t.CapacidadPromedio += float64(b.Capacidad)
	}

	if resumen.Total > 0 {
		resumen.CapacidadPromedio = round2(capacidadTotal / float64(resumen.Total))
	}
	resumen.PrecioTotal = round2(resumen.PrecioTotal)
	resumen.PesoTotal = round2(resumen.PesoTotal)

	porTipoList := make([]outputs.ReporteBusesTipo, 0, len(porTipo))
	for _, t := range porTipo {
		if t.Cantidad > 0 {
			t.CapacidadPromedio = round2(t.CapacidadPromedio / float64(t.Cantidad))
		}
		t.PrecioTotal = round2(t.PrecioTotal)
		porTipoList = append(porTipoList, *t)
	}

	filtros := map[string]any{
		"q": filters.Q, "tipo": filters.Tipo, "activo": filters.Activo,
		"desde": filters.Desde, "hasta": filters.Hasta, "modo": modo,
	}
	if modo == "generado" {
		filtros["cantidad"] = cantidad
	}

	return outputs.ReporteBuses{
		GeneradoEn: time.Now().Format(time.RFC3339),
		Filtros:    filtros,
		Resumen:    resumen,
		PorTipo:    porTipoList,
		Data:       data,
	}
}

func (s *ReporteService) Ventas(filters store.VentaFilters) outputs.ReporteVentas {
	ventas, _, _ := s.ventas.List(0, reportMaxRows, filters)
	total := s.ventas.Count(filters)

	resumen := outputs.ReporteVentasResumen{Total: total}
	clientes := map[string]struct{}{}
	for i := range ventas {
		v := &ventas[i]
		switch v.Estado {
		case "emitido":
			resumen.Emitidas++
		case "anulado":
			resumen.Anuladas++
		}
		resumen.MontoTotal += v.Total
		resumen.ImpuestoTotal += v.Impuesto
		if v.ClienteNombre != "" {
			clientes[v.ClienteNombre] = struct{}{}
		}
	}
	resumen.MontoTotal = round2(resumen.MontoTotal)
	resumen.ImpuestoTotal = round2(resumen.ImpuestoTotal)
	resumen.Clientes = len(clientes)

	return outputs.ReporteVentas{
		GeneradoEn: time.Now().Format(time.RFC3339),
		Filtros: map[string]any{
			"q": filters.Q, "serie": filters.Serie, "estado": filters.Estado,
			"desde": filters.Desde, "hasta": filters.Hasta,
		},
		Resumen: resumen,
		Data:    outputs.ToVentaResponses(ventas),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
