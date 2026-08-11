package mock

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
)

type ProductoStore struct {
	store *MockStore[domain.Producto]
}

type ProductoFilters struct {
	Q           string
	CategoriaID int64
	Activo      *bool
}

const productoSeedCount = 40

var productoNombres = []string{
	"Smartphone", "Laptop", "Auriculares", "Teclado", "Mouse", "Monitor", "Tablet",
	"Camiseta", "Pantalón", "Chaqueta", "Zapatos", "Gorra",
	"Arroz", "Aceite", "Café", "Azúcar", "Pasta",
	"Sofá", "Lámpara", "Cortina", "Cojín", "Espejo",
	"Balón", "Bicicleta", "Pesa", "Cuerda", "Raqueta",
	"Muñeca", "Bloques", "Rompecabezas", "Carrito",
	"Cuaderno", "Bolígrafo", "Resaltador", "Carpeta",
	"Crema", "Perfume", "Shampoo", "Labial", "Cargador",
}

func NewProductoStore() *ProductoStore {
	s := &ProductoStore{store: NewMockStore[domain.Producto](nil)}
	s.seed()
	return s
}

func (s *ProductoStore) seed() {
	rng := rand.New(rand.NewSource(7))
	now := time.Now()

	for i := 1; i <= productoSeedCount; i++ {
		categoriaID := int64(rng.Intn(8) + 1)
		s.store.Insert(int64(i), domain.Producto{
			ID:            int64(i),
			Nombre:        productoNombres[i-1],
			Descripcion:   fmt.Sprintf("Descripción del producto %d", i),
			Precio:        float64(int((5.0+rng.Float64()*995.0)*100)) / 100,
			Stock:         rng.Intn(200),
			CategoriaID:   categoriaID,
			Activo:        rng.Intn(5) > 0,
			CreadoEn:      now.Add(-time.Duration(rng.Intn(120)) * 24 * time.Hour),
			ActualizadoEn: now.Add(-time.Duration(rng.Intn(30)) * 24 * time.Hour),
		})
	}
}

func (s *ProductoStore) Count(filters ProductoFilters) int64 {
	var count int64
	for _, p := range s.store.FindAll() {
		if s.matchFilters(&p, filters) {
			count++
		}
	}
	return count
}

func (s *ProductoStore) List(offset, limit int, filters ProductoFilters) outputs.PaginatedResponse[outputs.ProductoResponse] {
	all := s.store.FindAll()

	var filtered []domain.Producto
	for _, p := range all {
		if s.matchFilters(&p, filters) {
			filtered = append(filtered, p)
		}
	}

	total := int64(len(filtered))

	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	if limit < 1 {
		limit = 10
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	data := make([]outputs.ProductoResponse, 0, end-offset)
	for i := offset; i < end; i++ {
		data = append(data, outputs.ToProductoResponse(&filtered[i]))
	}

	return outputs.PaginatedResponse[outputs.ProductoResponse]{
		Data:   data,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
}

func (s *ProductoStore) matchFilters(p *domain.Producto, f ProductoFilters) bool {
	if f.Q != "" {
		q := strings.ToLower(f.Q)
		if !strings.Contains(strings.ToLower(p.Nombre), q) &&
			!strings.Contains(strings.ToLower(p.Descripcion), q) {
			return false
		}
	}
	if f.CategoriaID > 0 && p.CategoriaID != f.CategoriaID {
		return false
	}
	if f.Activo != nil && p.Activo != *f.Activo {
		return false
	}
	return true
}

func (s *ProductoStore) FindByID(id int64) (*domain.Producto, error) {
	p, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProductoStore) Create(p domain.Producto) (domain.Producto, error) {
	p.CreadoEn = time.Now()
	p.ActualizadoEn = time.Now()
	return s.store.Create(p)
}

func (s *ProductoStore) Update(id int64, p domain.Producto) error {
	p.ActualizadoEn = time.Now()
	return s.store.Update(id, p)
}

func (s *ProductoStore) Delete(id int64) error {
	return s.store.Delete(id)
}

func (s *ProductoStore) CountByCategoria(categoriaID int64) int64 {
	var count int64
	for _, p := range s.store.FindAll() {
		if p.CategoriaID == categoriaID {
			count++
		}
	}
	return count
}

func (s *ProductoStore) ListByCategoria(categoriaID int64) []domain.Producto {
	var out []domain.Producto
	for _, p := range s.store.FindAll() {
		if p.CategoriaID == categoriaID {
			out = append(out, p)
		}
	}
	return out
}

func (s *ProductoStore) DeleteByCategoria(categoriaID int64) {
	for _, p := range s.store.FindAll() {
		if p.CategoriaID == categoriaID {
			_ = s.store.Delete(p.ID)
		}
	}
}
