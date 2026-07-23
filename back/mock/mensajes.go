package mock

import (
	"fmt"
	"math/rand"
	"time"

	"back/domain"
	"back/domain/inputs"
)

type MensajeStore struct {
	store *MockStore[domain.MensajePendiente]
}

func NewMensajeStore() *MensajeStore {
	s := &MensajeStore{store: NewMockStore[domain.MensajePendiente](nil)}
	s.seed()
	return s
}

func (s *MensajeStore) seed() {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	for i := 1; i <= 15; i++ {
		estado := "pendiente"
		if i > 12 {
			estado = "finalizado"
		}

		horaSol := now.Add(-time.Duration(rng.Intn(120)+10) * time.Minute)
		horaDesac := horaSol.Add(time.Duration(rng.Intn(180)+30) * time.Minute)

		m := domain.MensajePendiente{
			ID:                int64(i),
			Telefono:          fmt.Sprintf("999%03d%03d", rng.Intn(1000), rng.Intn(1000)),
			HoraSolicitada:    horaSol,
			HoraDesactivacion: horaDesac,
			Estado:            estado,
			CreadoEn:          horaSol,
			ActualizadoEn:     horaSol,
		}

		if i > 10 && i <= 12 {
			u := "andriwdv"
			m.UsuarioAcargo = &u
			t := horaSol.Add(5 * time.Minute)
			m.HoraUsuarioAsignado = &t
			m.Estado = "asignado"
		}

		s.store.Insert(int64(i), m)
	}
}

func (s *MensajeStore) Create(input inputs.MensajeInput, now time.Time) (domain.MensajePendiente, error) {
	horaDesac, err := time.Parse(time.RFC3339, input.HoraDesactivacion)
	if err != nil {
		horaDesac, _ = time.Parse("2006-01-02T15:04", input.HoraDesactivacion)
	}
	m := domain.MensajePendiente{
		Telefono:          input.Telefono,
		HoraSolicitada:    now,
		HoraDesactivacion: horaDesac,
		Estado:            "pendiente",
		CreadoEn:          now,
		ActualizadoEn:     now,
	}
	return s.store.Create(m)
}

func (s *MensajeStore) Update(id int64, input inputs.MensajeUpdateInput, now time.Time, username string) (domain.MensajePendiente, error) {
	existing, err := s.store.FindByID(id)
	if err != nil {
		return domain.MensajePendiente{}, err
	}

	if input.Finalizar != nil && *input.Finalizar {
		existing.HoraDesactivacion = now
		existing.Estado = "finalizado"
	}

	if input.UsuarioAcargo != nil && *input.UsuarioAcargo != "" {
		u := *input.UsuarioAcargo
		existing.UsuarioAcargo = &u
		t := now
		existing.HoraUsuarioAsignado = &t
		if existing.Estado == "pendiente" {
			existing.Estado = "asignado"
		}
	}

	existing.ActualizadoEn = now
	if err := s.store.Update(id, existing); err != nil {
		return domain.MensajePendiente{}, err
	}
	return existing, nil
}

func (s *MensajeStore) Delete(id int64) error {
	return s.store.Delete(id)
}

func (s *MensajeStore) FindByID(id int64) (*domain.MensajePendiente, error) {
	m, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *MensajeStore) ListPasadas(offset, limit int) ([]domain.MensajePendiente, int) {
	all := s.store.FindAll()
	var pasadas []domain.MensajePendiente
	for _, m := range all {
		if m.Estado == "finalizado" {
			pasadas = append(pasadas, m)
		}
	}
	total := len(pasadas)
	if offset > len(pasadas) {
		offset = len(pasadas)
	}
	end := offset + limit
	if end > len(pasadas) {
		end = len(pasadas)
	}
	return pasadas[offset:end], total
}

func (s *MensajeStore) CountPasadas() int {
	all := s.store.FindAll()
	count := 0
	for _, m := range all {
		if m.Estado == "finalizado" {
			count++
		}
	}
	return count
}

func (s *MensajeStore) FindAll() []domain.MensajePendiente {
	return s.store.FindAll()
}
