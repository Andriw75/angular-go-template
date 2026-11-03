package outputs

import (
	"time"

	"back/domain"
)

type DocumentoArchivoResponse struct {
	ID       int64  `json:"id"`
	Formato  string `json:"formato"`
	FileName string `json:"file_name"`
	Mime     string `json:"mime"`
	Tamano   int64  `json:"tamano"`
	Version  int    `json:"version"`
	URL      string `json:"url"`
	CreadoEn string `json:"creado_en"`
}

type DocumentoEventoResponse struct {
	ID       int64  `json:"id"`
	Accion   string `json:"accion"`
	Detalle  string `json:"detalle"`
	UserID   int64  `json:"user_id"`
	CreadoEn string `json:"creado_en"`
}

type DocumentoResponse struct {
	ID               int64                       `json:"id"`
	Tipo             string                      `json:"tipo"`
	Serie            string                      `json:"serie"`
	Anio             int                         `json:"anio"`
	Correlativo      int64                       `json:"correlativo"`
	Codigo           string                      `json:"codigo"`
	Politica         string                      `json:"politica"`
	Estado           string                      `json:"estado"`
	Plantilla        string                      `json:"plantilla"`
	PlantillaVersion string                      `json:"plantilla_version"`
	Version          int                         `json:"version"`
	CreadoEn         string                      `json:"creado_en"`
	ActualizadoEn    string                      `json:"actualizado_en"`
	EmitidoEn        string                      `json:"emitido_en"`
	AnuladoEn        *string                     `json:"anulado_en,omitempty"`
	MotivoAnulacion  string                      `json:"motivo_anulacion,omitempty"`
	OrigenID         *int64                      `json:"origen_id,omitempty"`
	Archivos         []DocumentoArchivoResponse  `json:"archivos"`
	Eventos          []DocumentoEventoResponse   `json:"eventos"`
}

func ToDocumentoResponse(d *domain.Documento) DocumentoResponse {
	archivos := make([]DocumentoArchivoResponse, 0, len(d.Archivos))
	for _, a := range d.Archivos {
		archivos = append(archivos, DocumentoArchivoResponse{
			ID: a.ID, Formato: a.Formato, FileName: a.FileName, Mime: a.Mime,
			Tamano: a.Tamano, Version: a.Version, URL: "/documentos/" + itoa(d.ID) + "/file?formato=" + a.Formato,
			CreadoEn: a.CreadoEn.Format(time.RFC3339),
		})
	}
	eventos := make([]DocumentoEventoResponse, 0, len(d.Eventos))
	for _, e := range d.Eventos {
		eventos = append(eventos, DocumentoEventoResponse{
			ID: e.ID, Accion: e.Accion, Detalle: e.Detalle, UserID: e.UserID,
			CreadoEn: e.CreadoEn.Format(time.RFC3339),
		})
	}
	var anulado *string
	if d.AnuladoEn != nil {
		s := d.AnuladoEn.Format(time.RFC3339)
		anulado = &s
	}
	return DocumentoResponse{
		ID: d.ID, Tipo: d.Tipo, Serie: d.Serie, Anio: d.Anio, Correlativo: d.Correlativo,
		Codigo: d.Codigo, Politica: d.Politica, Estado: d.Estado, Plantilla: d.Plantilla,
		PlantillaVersion: d.PlantillaVersion, Version: d.Version,
		CreadoEn: d.CreadoEn.Format(time.RFC3339), ActualizadoEn: d.ActualizadoEn.Format(time.RFC3339),
		EmitidoEn: d.EmitidoEn.Format(time.RFC3339), AnuladoEn: anulado,
		MotivoAnulacion: d.MotivoAnulacion, OrigenID: d.OrigenID,
		Archivos: archivos, Eventos: eventos,
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
