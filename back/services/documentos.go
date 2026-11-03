package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
	pdfreport "back/infrastructure/report"
	"back/store"
)

const DocumentTemplateVersion = "v1"

type BusesDocumentParams struct {
	Serie     string `json:"serie"`
	Plantilla string `json:"plantilla"`
	Politica  string `json:"politica"`
	Q         string `json:"q"`
	Tipo      string `json:"tipo"`
	Activo    *bool  `json:"activo,omitempty"`
	Desde     string `json:"desde"`
	Hasta     string `json:"hasta"`
	Modo      string `json:"modo"`
	Cantidad  int    `json:"cantidad"`
}

type VentaDocumentParams struct {
	VentaID   int64  `json:"venta_id"`
	Serie     string `json:"serie"`
	Plantilla string `json:"plantilla"`
	Politica  string `json:"politica"`
}

type DocumentoService struct {
	store   store.DocumentoStore
	ventas  store.VentaStore
	reports *ReporteService
	cfg     *Config
	jobs    *JobStore
}

func NewDocumentoService(store store.DocumentoStore, ventas store.VentaStore, reports *ReporteService, cfg *Config) *DocumentoService {
	_ = os.MkdirAll(cfg.DocumentosDir(), 0o755)
	return &DocumentoService{
		store:   store,
		ventas:  ventas,
		reports: reports,
		cfg:     cfg,
		jobs:    NewJobStore(time.Duration(cfg.JobTTLMinutes) * time.Minute),
	}
}

func (s *DocumentoService) Jobs() *JobStore { return s.jobs }

func (s *DocumentoService) SyncTimeout() time.Duration {
	return time.Duration(s.cfg.ReportSyncTimeout) * time.Second
}

func documentHash(tipo string, params any) (hash string, paramsJSON string, err error) {
	b, err := json.Marshal(params)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(tipo + "|" + DocumentTemplateVersion + "|" + string(b)))
	return hex.EncodeToString(sum[:]), string(b), nil
}

func normalizeBusesParams(p BusesDocumentParams) BusesDocumentParams {
	if p.Serie == "" {
		p.Serie = "BUS"
	}
	if p.Plantilla == "" {
		p.Plantilla = "buses-a4-p"
	}
	if p.Politica == "" {
		p.Politica = "inmutable"
	}
	if p.Modo != "generado" {
		p.Modo = "actual"
	}
	if p.Modo == "generado" {
		if p.Cantidad < 1 {
			p.Cantidad = 50
		}
		if p.Cantidad > MaxGeneratedBuses {
			p.Cantidad = MaxGeneratedBuses
		}
	}
	p.Serie = strings.ToUpper(p.Serie)
	return p
}

func (s *DocumentoService) busesReport(p BusesDocumentParams) outputs.ReporteBuses {
	filters := store.BusFilters{Q: p.Q, Tipo: p.Tipo, Activo: p.Activo, Desde: p.Desde, Hasta: p.Hasta}
	if p.Modo == "generado" {
		return s.reports.GeneratedBuses(p.Cantidad, filters)
	}
	return s.reports.Buses(filters)
}

func normalizeVentaParams(p VentaDocumentParams) VentaDocumentParams {
	if p.Plantilla == "" {
		p.Plantilla = "venta-factura"
	}
	if p.Politica == "" {
		p.Politica = "inmutable"
	}
	return p
}

func (s *DocumentoService) GenerateBuses(ctx context.Context, p BusesDocumentParams, userID int64, progress func(int)) (domain.Documento, bool, error) {
	p = normalizeBusesParams(p)
	tipo := "reporte_buses"
	hash, paramsJSON, err := documentHash(tipo, p)
	if err != nil {
		return domain.Documento{}, false, err
	}
	if progress != nil {
		progress(10)
	}

	existing, err := s.store.FindByHash(tipo, hash)
	if err == nil && existing != nil {
		if p.Politica == "inmutable" || existing.Politica == "inmutable" {
			return *existing, true, nil
		}
		rep := s.busesReport(p)
		snapshot, _ := json.Marshal(rep)
		var written []string
		doc, gerr := s.store.AddVersion(existing.ID, string(snapshot), func(codigo string, version int) ([]domain.DocumentoArchivo, error) {
			files, werr := s.writeBusesFiles(codigo, version, rep, p.Plantilla)
			for _, f := range files {
				written = append(written, f.FileName)
			}
			return files, werr
		})
		if gerr != nil {
			s.removeFiles(written)
			return domain.Documento{}, true, gerr
		}
		return doc, true, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Documento{}, false, err
	}

	if progress != nil {
		progress(30)
	}
	rep := s.busesReport(p)
	snapshot, _ := json.Marshal(rep)
	if progress != nil {
		progress(60)
	}

	input := store.DocumentoCreateInput{
		Tipo: tipo, Serie: p.Serie, Politica: p.Politica, Estado: "emitido",
		Plantilla: p.Plantilla, PlantillaVersion: DocumentTemplateVersion,
		ParametrosHash: hash, ParametrosJSON: paramsJSON, SnapshotJSON: string(snapshot), UserID: userID,
	}

	var written []string
	doc, err := s.store.CreateWithCorrelativo(input, func(codigo string) ([]domain.DocumentoArchivo, error) {
		files, werr := s.writeBusesFiles(codigo, 1, rep, p.Plantilla)
		for _, f := range files {
			written = append(written, f.FileName)
		}
		return files, werr
	})
	if err != nil {
		s.removeFiles(written)
		if dup, derr := s.store.FindByHash(tipo, hash); derr == nil && dup != nil {
			return *dup, true, nil
		}
		return domain.Documento{}, false, err
	}
	if progress != nil {
		progress(100)
	}
	return doc, false, nil
}

func (s *DocumentoService) GenerateVenta(ctx context.Context, p VentaDocumentParams, userID int64, progress func(int)) (domain.Documento, bool, error) {
	p = normalizeVentaParams(p)
	venta, err := s.ventas.FindByID(p.VentaID)
	if err != nil {
		return domain.Documento{}, false, err
	}
	if venta.Codigo == "" {
		return domain.Documento{}, false, errors.New("la venta debe estar emitida")
	}
	if p.Serie == "" {
		p.Serie = venta.Serie
	}
	tipo := "venta"
	hash, paramsJSON, err := documentHash(tipo, p)
	if err != nil {
		return domain.Documento{}, false, err
	}
	if progress != nil {
		progress(10)
	}

	existing, err := s.store.FindByHash(tipo, hash)
	if err == nil && existing != nil {
		if p.Politica == "inmutable" || existing.Politica == "inmutable" {
			return *existing, true, nil
		}
		ventaResp := outputs.ToVentaResponse(venta)
		snapshot, _ := json.Marshal(ventaResp)
		var written []string
		doc, gerr := s.store.AddVersion(existing.ID, string(snapshot), func(codigo string, version int) ([]domain.DocumentoArchivo, error) {
			files, werr := s.writeVentaFiles(codigo, version, ventaResp, p.Plantilla)
			for _, f := range files {
				written = append(written, f.FileName)
			}
			return files, werr
		})
		if gerr != nil {
			s.removeFiles(written)
			return domain.Documento{}, true, gerr
		}
		return doc, true, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Documento{}, false, err
	}

	ventaResp := outputs.ToVentaResponse(venta)
	snapshot, _ := json.Marshal(ventaResp)
	if progress != nil {
		progress(50)
	}

	input := store.DocumentoCreateInput{
		Tipo: tipo, Serie: p.Serie, Politica: p.Politica, Estado: "emitido", Anio: ventaAnio(venta),
		Plantilla: p.Plantilla, PlantillaVersion: DocumentTemplateVersion,
		ParametrosHash: hash, ParametrosJSON: paramsJSON, SnapshotJSON: string(snapshot), UserID: userID,
	}
	var written []string
	doc, err := s.store.CreateWithCorrelativo(input, func(codigo string) ([]domain.DocumentoArchivo, error) {
		files, werr := s.writeVentaFiles(codigo, 1, ventaResp, p.Plantilla)
		for _, f := range files {
			written = append(written, f.FileName)
		}
		return files, werr
	})
	if err != nil {
		s.removeFiles(written)
		if dup, derr := s.store.FindByHash(tipo, hash); derr == nil && dup != nil {
			return *dup, true, nil
		}
		return domain.Documento{}, false, err
	}
	if progress != nil {
		progress(100)
	}
	return doc, false, nil
}

func (s *DocumentoService) writeBusesFiles(codigo string, version int, rep outputs.ReporteBuses, template string) ([]domain.DocumentoArchivo, error) {
	pdfBytes, err := pdfreport.GenerateBusesPDF(rep, template)
	if err != nil {
		return nil, err
	}
	xlsxBytes, err := pdfreport.GenerateBusesExcel(rep)
	if err != nil {
		return nil, err
	}
	return s.writeFiles(codigo, version, pdfBytes, xlsxBytes)
}

func (s *DocumentoService) writeVentaFiles(codigo string, version int, venta outputs.VentaResponse, template string) ([]domain.DocumentoArchivo, error) {
	pdfBytes, err := pdfreport.GenerateVentaPDF(venta, template)
	if err != nil {
		return nil, err
	}
	xlsxBytes, err := pdfreport.GenerateVentaExcel(venta)
	if err != nil {
		return nil, err
	}
	return s.writeFiles(codigo, version, pdfBytes, xlsxBytes)
}

func (s *DocumentoService) writeFiles(codigo string, version int, pdfBytes, xlsxBytes []byte) ([]domain.DocumentoArchivo, error) {
	dir := s.cfg.DocumentosDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	pdfName := fmt.Sprintf("%s-v%d.pdf", codigo, version)
	xlsxName := fmt.Sprintf("%s-v%d.xlsx", codigo, version)
	if err := os.WriteFile(filepath.Join(dir, pdfName), pdfBytes, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, xlsxName), xlsxBytes, 0o644); err != nil {
		_ = os.Remove(filepath.Join(dir, pdfName))
		return nil, err
	}
	return []domain.DocumentoArchivo{
		{Formato: "pdf", FileName: pdfName, Mime: "application/pdf", Tamano: int64(len(pdfBytes)), Version: version},
		{Formato: "xlsx", FileName: xlsxName, Mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Tamano: int64(len(xlsxBytes)), Version: version},
	}, nil
}

func (s *DocumentoService) removeFiles(names []string) {
	for _, name := range names {
		_ = os.Remove(filepath.Join(s.cfg.DocumentosDir(), name))
	}
}

func ventaAnio(v *domain.Venta) int {
	if t, err := time.Parse("2006-01-02", v.Fecha.Format("2006-01-02")); err == nil {
		return t.Year()
	}
	return time.Now().Year()
}
