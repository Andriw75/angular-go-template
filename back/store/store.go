package store

import (
	"context"
	"errors"
	"time"

	"back/domain"
	"back/domain/inputs"
	"back/domain/outputs"
	"back/infrastructure/auth"
)

// Errores del laboratorio de consultas largas.
var (
	ErrSlowQueryNotFound    = errors.New("consulta no encontrada")
	ErrSlowQueryUnsupported = errors.New("el laboratorio solo esta disponible en postgres")
	ErrSlowQueryLimit       = errors.New("limite de consultas concurrentes alcanzado")
	ErrSlowQueryInvalid     = errors.New("parametros de consulta invalidos")
)

type SlowQueryModo string

const (
	SlowQuerySleep SlowQueryModo = "sleep"
	SlowQueryCPU   SlowQueryModo = "cpu"
)

type SlowQueryEstado string

const (
	SlowQueryEjecutando SlowQueryEstado = "ejecutando"
	SlowQueryCancelando SlowQueryEstado = "cancelando"
	SlowQueryCancelada  SlowQueryEstado = "cancelada"
	SlowQueryCompletada SlowQueryEstado = "completada"
	SlowQueryError      SlowQueryEstado = "error"
)

type SlowQueryInput struct {
	Modo     SlowQueryModo
	Segundos int
	Filas    int64
}

type SlowQueryInfo struct {
	ID               string          `json:"id"`
	OwnerID          int64           `json:"owner_id"`
	Modo             SlowQueryModo   `json:"modo"`
	PID              int             `json:"pid"`
	Estado           SlowQueryEstado `json:"estado"`
	Segundos         int             `json:"segundos"`
	FilasSolicitadas int64           `json:"filas_solicitadas"`
	FilasProcesadas  int64           `json:"filas_procesadas"`
	IniciadoEn       time.Time       `json:"iniciado_en"`
	FinalizadoEn     *time.Time      `json:"finalizado_en,omitempty"`
	Error            string          `json:"error,omitempty"`
}

// SlowQueryNotifier recibe los cambios de estado de las tareas del laboratorio
// para publicarlos por SSE.
type SlowQueryNotifier interface {
	Notify(event string, info SlowQueryInfo)
}

// SlowQueryStore ejecuta consultas largas acotadas y permite cancelarlas
// terminando realmente la consulta en PostgreSQL.
type SlowQueryStore interface {
	Start(ctx context.Context, ownerID int64, input SlowQueryInput) (SlowQueryInfo, error)
	RunSync(ctx context.Context, ownerID int64, input SlowQueryInput) (SlowQueryInfo, error)
	List() []SlowQueryInfo
	Cancel(ctx context.Context, id string) (SlowQueryInfo, error)
	CancelAll(ctx context.Context)
}

type BusFilters struct {
	Q      string
	Tipo   string
	Activo *bool
	Desde  string
	Hasta  string
}

type CategoriaFilters struct {
	Q      string
	Activo *bool
}

type ProductoFilters struct {
	Q           string
	CategoriaID int64
	Activo      *bool
}

type VentaFilters struct {
	Q      string
	Serie  string
	Estado string
	Desde  string
	Hasta  string
}

type UserStore interface {
	Create(input inputs.UserInput) (domain.User, error)
	Update(id int64, input inputs.UserUpdateInput) error
	UpdatePassword(id int64, hash string, mustChange bool) error
	Delete(id int64) error
	FindByUsername(username string) (*domain.User, error)
	FindByID(id int64) (*domain.User, error)
	FindAll() []domain.User
	IsValidPassword(crypt *auth.CryptManager, username, password string) (*domain.User, bool)
}

type PermisoStore interface {
	FindAll() []domain.Permission
	FindByID(id int64) (*domain.Permission, error)
}

type BusStore interface {
	Count(filters BusFilters) int64
	List(offset, limit int, filters BusFilters) outputs.PaginatedResponse[outputs.BusResponse]
	FindByID(id int64) (*domain.Bus, error)
	Create(b domain.Bus) (domain.Bus, error)
	Update(id int64, b domain.Bus) error
	Delete(id int64) error
}

type MensajeStore interface {
	Create(input inputs.MensajeInput, now time.Time) (domain.MensajePendiente, error)
	Update(id int64, input inputs.MensajeUpdateInput, now time.Time, username string) (domain.MensajePendiente, error)
	Delete(id int64) error
	FindByID(id int64) (*domain.MensajePendiente, error)
	ListPasadas(offset, limit int) ([]domain.MensajePendiente, bool)
	CountPasadas() int
	FindAll() []domain.MensajePendiente
}

type CategoriaStore interface {
	Count(filters CategoriaFilters) int64
	List(offset, limit int, filters CategoriaFilters) outputs.PaginatedResponse[outputs.CategoriaResponse]
	FindByID(id int64) (*domain.Categoria, error)
	Create(c domain.Categoria) (domain.Categoria, error)
	Update(id int64, c domain.Categoria) error
	Delete(id int64) error
}

type ProductoStore interface {
	Count(filters ProductoFilters) int64
	List(offset, limit int, filters ProductoFilters) outputs.PaginatedResponse[outputs.ProductoResponse]
	FindByID(id int64) (*domain.Producto, error)
	Create(p domain.Producto) (domain.Producto, error)
	Update(id int64, p domain.Producto) error
	Delete(id int64) error
	CountByCategoria(categoriaID int64) int64
	ListByCategoria(categoriaID int64) []domain.Producto
	DeleteByCategoria(categoriaID int64)
}

type ImagenStore interface {
	Add(tipo string, entidadID int64, fileName string) (domain.Imagen, error)
	ListAll() []domain.Imagen
	ListByEntidad(tipo string, entidadID int64) []domain.Imagen
	Reorder(tipo string, entidadID int64, ids []int64) error
	FindByID(id int64) (*domain.Imagen, error)
	FindByFileName(fileName string) (*domain.Imagen, error)
	Delete(id int64) error
	DeleteByEntidad(tipo string, entidadID int64) []string
}

type ArchivoStore interface {
	Create(fileName, nombre, tipo string, tamano int64, visibilidad domain.Visibilidad, userID int64) (domain.Archivo, error)
	FindByID(id int64) (*domain.Archivo, error)
	FindByFileName(fileName string) (*domain.Archivo, error)
	List(offset, limit int, q string) ([]domain.Archivo, bool, error)
	Count(q string) (int, error)
	UpdateVisibilidad(id int64, vis domain.Visibilidad) error
	Delete(id int64) error
}

type InstalacionStore interface {
	List() []domain.Instalacion
	FindByID(id int64) (*domain.Instalacion, error)
	Create(i domain.Instalacion) (domain.Instalacion, error)
	Update(id int64, i domain.Instalacion) error
	Delete(id int64) error
}

type DocumentoFilters struct {
	Q      string
	Tipo   string
	Serie  string
	Estado string
}

type DocumentoCreateInput struct {
	Anio             int
	Tipo             string
	Serie            string
	Politica         string
	Estado           string
	Plantilla        string
	PlantillaVersion string
	ParametrosHash   string
	ParametrosJSON   string
	SnapshotJSON     string
	UserID           int64
}

type DocumentoStore interface {
	FindByHash(tipo, hash string) (*domain.Documento, error)
	FindByID(id int64) (*domain.Documento, error)
	List(offset, limit int, filters DocumentoFilters) ([]domain.Documento, bool, error)
	Count(filters DocumentoFilters) (int, error)
	CreateWithCorrelativo(input DocumentoCreateInput, build func(codigo string) ([]domain.DocumentoArchivo, error)) (domain.Documento, error)
	AddVersion(id int64, snapshotJSON string, build func(codigo string, version int) ([]domain.DocumentoArchivo, error)) (domain.Documento, error)
	Annul(id int64, motivo string, userID int64) (domain.Documento, error)
}

type VentaStore interface {
	List(offset, limit int, filters VentaFilters) ([]domain.Venta, bool, error)
	Count(filters VentaFilters) int
	FindByID(id int64) (*domain.Venta, error)
	Create(input inputs.VentaInput, userID int64) (domain.Venta, error)
	Update(id int64, input inputs.VentaInput) (domain.Venta, error)
	Delete(id int64) error
	Emit(id int64) (domain.Venta, error)
	Annul(id int64, motivo string, userID int64) (domain.Venta, error)
}
