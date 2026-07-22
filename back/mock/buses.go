package mock

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
)

type BusStore struct {
	store *MockStore[domain.Bus]
}

type BusFilters struct {
	Q      string
	Tipo   string
	Activo *bool
}

const seedCount = 60

var marcas = []string{"Mercedes-Benz", "Volvo", "Scania", "Hyundai", "Toyota", "Volkswagen", "MAN", "Iveco"}
var modelos = []string{"OF-1721", "B7R", "K410", "Universe", "Marcopolo", "Torino", "TH-3200", "BC-180"}
var tipos = []string{"BUS", "VAN", "MINIBUS", "MICROBUS"}
var colores = []string{"Blanco", "Rojo", "Azul", "Plateado", "Negro", "Amarillo", "Verde", "Naranja"}
var descs = []string{
	"Bus de dos pisos con capacidad para 40 pasajeros, aire acondicionado y baño.",
	"Vehículo tipo Van ideal para rutas cortas, 15 asientos reclinables.",
	"Microbús urbano con piso bajo, capacidad para 25 pasajeros de pie y 15 sentados.",
	"Bus interprovincial de lujo con butacas reclinables, TV y refrigerio.",
	"Minibús ejecutivo para 20 pasajeros, ideal para empresas.",
}

func NewBusStore() *BusStore {
	s := &BusStore{store: NewMockStore[domain.Bus](nil)}
	s.seed()
	return s
}

func (s *BusStore) seed() {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	for i := 1; i <= seedCount; i++ {
		anio := 2015 + rng.Intn(10)
		capacidad := 15 + rng.Intn(30)
		precio := 80000.0 + rng.Float64()*270000.0
		peso := 8.0 + rng.Float64()*10.0
		diasCompra := rng.Intn(2000) + 365
		fechaCompra := now.Add(-time.Duration(diasCompra) * 24 * time.Hour)

		var ultimoMantenimiento *time.Time
		if rng.Intn(3) > 0 {
			t := now.Add(-time.Duration(rng.Intn(180)) * 24 * time.Hour)
			ultimoMantenimiento = &t
		}

		s.store.Insert(int64(i), domain.Bus{
			ID:                  int64(i),
			Placa:               fmt.Sprintf("%c%c%c-%d%d%d", 'A'+rng.Intn(26), 'A'+rng.Intn(26), 'A'+rng.Intn(26), rng.Intn(10), rng.Intn(10), rng.Intn(10)),
			Nombre:              fmt.Sprintf("Bus %03d", i),
			Marca:               marcas[rng.Intn(len(marcas))],
			Modelo:              modelos[rng.Intn(len(modelos))],
			Anio:                anio,
			Capacidad:           capacidad,
			Tipo:                tipos[rng.Intn(len(tipos))],
			Activo:              rng.Intn(5) > 0,
			FechaCompra:         fechaCompra,
			UltimoMantenimiento: ultimoMantenimiento,
			Precio:              float64(int(precio*100)) / 100,
			Peso:                float64(int(peso*100)) / 100,
			Color:               colores[rng.Intn(len(colores))],
			Descripcion:         descs[rng.Intn(len(descs))],
			CreadoEn:            fechaCompra,
			ActualizadoEn:       now.Add(-time.Duration(rng.Intn(90)) * 24 * time.Hour),
		})
	}
}

func (s *BusStore) List(offset, limit int, filters BusFilters) outputs.PaginatedResponse[outputs.BusResponse] {
	all := s.store.FindAll()

	var filtered []domain.Bus
	for _, b := range all {
		if !s.matchFilters(&b, filters) {
			continue
		}
		filtered = append(filtered, b)
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

	data := make([]outputs.BusResponse, 0, end-offset)
	for i := offset; i < end; i++ {
		data = append(data, outputs.ToBusResponse(&filtered[i]))
	}

	return outputs.PaginatedResponse[outputs.BusResponse]{
		Data:   data,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
}

func (s *BusStore) matchFilters(b *domain.Bus, f BusFilters) bool {
	if f.Q != "" {
		q := strings.ToLower(f.Q)
		if !strings.Contains(strings.ToLower(b.Placa), q) &&
			!strings.Contains(strings.ToLower(b.Nombre), q) &&
			!strings.Contains(strings.ToLower(b.Marca), q) &&
			!strings.Contains(strings.ToLower(b.Modelo), q) {
			return false
		}
	}
	if f.Tipo != "" && !strings.EqualFold(b.Tipo, f.Tipo) {
		return false
	}
	if f.Activo != nil && b.Activo != *f.Activo {
		return false
	}
	return true
}

func (s *BusStore) FindByID(id int64) (*domain.Bus, error) {
	b, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *BusStore) Create(b domain.Bus) (domain.Bus, error) {
	b.CreadoEn = time.Now()
	b.ActualizadoEn = time.Now()
	return s.store.Create(b)
}

func (s *BusStore) Update(id int64, b domain.Bus) error {
	b.ActualizadoEn = time.Now()
	return s.store.Update(id, b)
}

func (s *BusStore) Delete(id int64) error {
	return s.store.Delete(id)
}
