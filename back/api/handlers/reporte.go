package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"back/domain/outputs"
	pdfreport "back/infrastructure/report"
	"back/services"
	"back/store"
)

type ReporteHandler struct {
	deps *Dependencies
	svc  *services.ReporteService
}

func NewReporteHandler(deps *Dependencies) *ReporteHandler {
	return &ReporteHandler{
		deps: deps,
		svc:  services.NewReporteService(deps.BusStore, deps.VentaStore),
	}
}

func parseBusReportFilters(r *http.Request) store.BusFilters {
	q := r.URL.Query()
	f := store.BusFilters{
		Q:     strings.TrimSpace(q.Get("q")),
		Tipo:  strings.TrimSpace(q.Get("tipo")),
		Desde: strings.TrimSpace(q.Get("desde")),
		Hasta: strings.TrimSpace(q.Get("hasta")),
	}
	if s := q.Get("activo"); s != "" {
		if v, err := strconv.ParseBool(s); err == nil {
			f.Activo = &v
		}
	}
	return f
}

func parseVentaReportFilters(r *http.Request) store.VentaFilters {
	q := r.URL.Query()
	return store.VentaFilters{
		Q:      strings.TrimSpace(q.Get("q")),
		Serie:  strings.TrimSpace(q.Get("serie")),
		Estado: strings.TrimSpace(q.Get("estado")),
		Desde:  strings.TrimSpace(q.Get("desde")),
		Hasta:  strings.TrimSpace(q.Get("hasta")),
	}
}

func reportMode(r *http.Request) (string, int) {
	mode := r.URL.Query().Get("modo")
	if mode != "generado" {
		mode = "actual"
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("cantidad"))
	if count < 1 {
		count = 50
	}
	if count > services.MaxGeneratedBuses {
		count = services.MaxGeneratedBuses
	}
	return mode, count
}

func (h *ReporteHandler) busesReport(r *http.Request) outputs.ReporteBuses {
	mode, count := reportMode(r)
	filters := parseBusReportFilters(r)
	if mode == "generado" {
		return h.svc.GeneratedBuses(count, filters)
	}
	return h.svc.Buses(filters)
}

func (h *ReporteHandler) Buses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.busesReport(r))
}

func (h *ReporteHandler) BusesCount(w http.ResponseWriter, r *http.Request) {
	mode, count := reportMode(r)
	if mode == "generado" {
		writeJSON(w, http.StatusOK, map[string]int{"total": count})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"total": h.deps.BusStore.Count(parseBusReportFilters(r))})
}

func (h *ReporteHandler) Ventas(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Ventas(parseVentaReportFilters(r)))
}

func (h *ReporteHandler) VentasCount(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"total": h.deps.VentaStore.Count(parseVentaReportFilters(r))})
}

func writeDownload(w http.ResponseWriter, data []byte, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *ReporteHandler) BusesPDF(w http.ResponseWriter, r *http.Request) {
	template := r.URL.Query().Get("template")
	rep := h.busesReport(r)
	data, err := pdfreport.GenerateBusesPDF(rep, template)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al generar PDF")
		return
	}
	writeDownload(w, data, "application/pdf", "reporte-buses-"+time.Now().Format("20060102-1504")+".pdf")
}

func (h *ReporteHandler) BusesXLSX(w http.ResponseWriter, r *http.Request) {
	rep := h.busesReport(r)
	data, err := pdfreport.GenerateBusesExcel(rep)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al generar Excel")
		return
	}
	writeDownload(w, data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "reporte-buses-"+time.Now().Format("20060102-1504")+".xlsx")
}

func (h *ReporteHandler) VentaPDF(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	venta, err := h.deps.VentaStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "venta not found")
		return
	}
	data, err := pdfreport.GenerateVentaPDF(outputs.ToVentaResponse(venta), r.URL.Query().Get("template"))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al generar PDF")
		return
	}
	writeDownload(w, data, "application/pdf", "venta-"+venta.Codigo+".pdf")
}

func (h *ReporteHandler) VentaXLSX(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	venta, err := h.deps.VentaStore.FindByID(id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "venta not found")
		return
	}
	data, err := pdfreport.GenerateVentaExcel(outputs.ToVentaResponse(venta))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error al generar Excel")
		return
	}
	writeDownload(w, data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "venta-"+venta.Codigo+".xlsx")
}
