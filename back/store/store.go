package store

import (
	"time"

	"back/domain"
	"back/domain/inputs"
	"back/domain/outputs"
	"back/infrastructure/auth"
)

type BusFilters struct {
	Q      string
	Tipo   string
	Activo *bool
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

type UserStore interface {
	Create(input inputs.UserInput) (domain.User, error)
	Update(id int64, input inputs.UserUpdateInput) error
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
	ListPasadas(offset, limit int) ([]domain.MensajePendiente, int)
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
	ListByEntidad(tipo string, entidadID int64) []domain.Imagen
	Reorder(tipo string, entidadID int64, ids []int64) error
	FindByID(id int64) (*domain.Imagen, error)
	Delete(id int64) error
	DeleteByEntidad(tipo string, entidadID int64) []string
}

type ArchivoStore interface {
	Create(fileName, nombre, tipo string, tamano int64, visibilidad domain.Visibilidad, userID int64) (domain.Archivo, error)
	FindByID(id int64) (*domain.Archivo, error)
	FindByFileName(fileName string) (*domain.Archivo, error)
	List(offset, limit int, q string) ([]domain.Archivo, int, error)
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
