package mock

import (
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
)

type CategoriaStore struct {
	store *MockStore[domain.Categoria]
}

type CategoriaFilters struct {
	Q      string
	Activo *bool
}

func NewCategoriaStore() *CategoriaStore {
	s := &CategoriaStore{store: NewMockStore[domain.Categoria](nil)}
	s.seed()
	return s
}

func (s *CategoriaStore) seed() {
	now := time.Now()
	nombres := []struct {
		nombre      string
		descripcion string
	}{
		{"Electrónica", "Dispositivos y accesorios electrónicos"},
		{"Ropa", "Prendas de vestir para todas las temporadas"},
		{"Alimentos", "Productos alimenticios y bebidas"},
		{"Hogar", "Artículos para el hogar y decoración"},
		{"Deportes", "Equipamiento e indumentaria deportiva"},
		{"Juguetes", "Juguetes y juegos para niños"},
		{"Librería", "Libros, papelería y artículos de oficina"},
		{"Belleza", "Cosméticos y productos de cuidado personal"},
	}

	for i, c := range nombres {
		id := int64(i + 1)
		s.store.Insert(id, domain.Categoria{
			ID:            id,
			Nombre:        c.nombre,
			Descripcion:   c.descripcion,
			Activo:        true,
			CreadoEn:      now,
			ActualizadoEn: now,
		})
	}
}

func (s *CategoriaStore) Count(filters CategoriaFilters) int64 {
	var count int64
	for _, c := range s.store.FindAll() {
		if s.matchFilters(&c, filters) {
			count++
		}
	}
	return count
}

func (s *CategoriaStore) List(offset, limit int, filters CategoriaFilters) outputs.PaginatedResponse[outputs.CategoriaResponse] {
	all := s.store.FindAll()

	var filtered []domain.Categoria
	for _, c := range all {
		if s.matchFilters(&c, filters) {
			filtered = append(filtered, c)
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

	data := make([]outputs.CategoriaResponse, 0, end-offset)
	for i := offset; i < end; i++ {
		data = append(data, outputs.ToCategoriaResponse(&filtered[i]))
	}

	return outputs.PaginatedResponse[outputs.CategoriaResponse]{
		Data:   data,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
}

func (s *CategoriaStore) matchFilters(c *domain.Categoria, f CategoriaFilters) bool {
	if f.Q != "" {
		q := strings.ToLower(f.Q)
		if !strings.Contains(strings.ToLower(c.Nombre), q) &&
			!strings.Contains(strings.ToLower(c.Descripcion), q) {
			return false
		}
	}
	if f.Activo != nil && c.Activo != *f.Activo {
		return false
	}
	return true
}

func (s *CategoriaStore) FindByID(id int64) (*domain.Categoria, error) {
	c, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CategoriaStore) Create(c domain.Categoria) (domain.Categoria, error) {
	c.CreadoEn = time.Now()
	c.ActualizadoEn = time.Now()
	return s.store.Create(c)
}

func (s *CategoriaStore) Update(id int64, c domain.Categoria) error {
	c.ActualizadoEn = time.Now()
	return s.store.Update(id, c)
}

func (s *CategoriaStore) Delete(id int64) error {
	return s.store.Delete(id)
}
