package domain

import "time"

type DocumentoArchivo struct {
	ID          int64
	DocumentoID int64
	Formato     string
	FileName    string
	Mime        string
	Tamano      int64
	Version     int
	CreadoEn    time.Time
}

type DocumentoEvento struct {
	ID          int64
	DocumentoID int64
	Accion      string
	Detalle     string
	UserID      int64
	CreadoEn    time.Time
}

type Documento struct {
	ID               int64
	Tipo             string
	Serie            string
	Anio             int
	Correlativo      int64
	Codigo           string
	Politica         string
	Estado           string
	Plantilla        string
	PlantillaVersion string
	ParametrosHash   string
	ParametrosJSON   string
	SnapshotJSON     string
	Version          int
	UserID           int64
	CreadoEn         time.Time
	ActualizadoEn    time.Time
	EmitidoEn        time.Time
	AnuladoEn        *time.Time
	MotivoAnulacion  string
	OrigenID         *int64
	Archivos         []DocumentoArchivo
	Eventos          []DocumentoEvento
}
