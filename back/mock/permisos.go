package mock

import "back/domain"

type PermisoStore struct {
	store *MockStore[domain.Permission]
}

func NewPermisoStore() *PermisoStore {
	s := &PermisoStore{store: NewMockStore[domain.Permission](nil)}

	s.store.Insert(1, domain.Permission{ID: 1, Nombre: "buses", Descripcion: "Acceso a buses"})
	s.store.Insert(2, domain.Permission{ID: 2, Nombre: "usuarios", Descripcion: "Gestion de usuarios"})

	return s
}

func (s *PermisoStore) FindAll() []domain.Permission {
	return s.store.FindAll()
}

func (s *PermisoStore) FindByID(id int64) (*domain.Permission, error) {
	p, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
