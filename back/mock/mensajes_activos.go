package mock

import (
	"sort"
	"sync"

	"back/domain"
)

type ActivosStore struct {
	mu   sync.RWMutex
	data map[int64]domain.MensajePendiente
}

func NewActivosStore() *ActivosStore {
	return &ActivosStore{data: make(map[int64]domain.MensajePendiente)}
}

func (s *ActivosStore) Set(m domain.MensajePendiente) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[m.ID] = m
}

func (s *ActivosStore) Delete(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
}

func (s *ActivosStore) GetAll() []domain.MensajePendiente {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.MensajePendiente, 0, len(s.data))
	for _, m := range s.data {
		result = append(result, m)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].HoraSolicitada.After(result[j].HoraSolicitada)
	})
	return result
}

func (s *ActivosStore) GetByID(id int64) (domain.MensajePendiente, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[id]
	return m, ok
}
