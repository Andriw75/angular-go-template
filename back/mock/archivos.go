package mock

import (
	"fmt"
	"strings"
	"time"

	"back/domain"
)

type ArchivoStore struct {
	store *MockStore[domain.Archivo]
}

func NewArchivoStore() *ArchivoStore {
	return &ArchivoStore{store: NewMockStore[domain.Archivo](nil)}
}

func (s *ArchivoStore) Create(fileName, nombre, tipo string, tamano int64, visibilidad domain.Visibilidad, userID int64) (domain.Archivo, error) {
	if visibilidad == "" {
		visibilidad = domain.VisibilidadPublico
	}
	return s.store.Create(domain.Archivo{
		FileName:    fileName,
		Nombre:      nombre,
		Tipo:        tipo,
		Tamano:      tamano,
		Visibilidad: visibilidad,
		UserID:      userID,
		CreadoEn:    time.Now(),
	})
}

func (s *ArchivoStore) FindByID(id int64) (*domain.Archivo, error) {
	a, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *ArchivoStore) FindByFileName(fileName string) (*domain.Archivo, error) {
	for _, a := range s.store.FindAll() {
		if a.FileName == fileName {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("archivo not found")
}

func (s *ArchivoStore) List(offset, limit int, q string) ([]domain.Archivo, int, error) {
	all := s.store.FindAll()

	if q != "" {
		q = strings.ToLower(q)
		filtered := make([]domain.Archivo, 0)
		for _, a := range all {
			if strings.Contains(strings.ToLower(a.Nombre), q) || strings.Contains(strings.ToLower(a.FileName), q) {
				filtered = append(filtered, a)
			}
		}
		all = filtered
	}

	total := len(all)

	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].CreadoEn.After(all[i].CreadoEn) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if offset >= len(all) {
		return []domain.Archivo{}, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (s *ArchivoStore) UpdateVisibilidad(id int64, vis domain.Visibilidad) error {
	a, err := s.store.FindByID(id)
	if err != nil {
		return err
	}
	a.Visibilidad = vis
	return s.store.Update(id, a)
}

func (s *ArchivoStore) Delete(id int64) error {
	return s.store.Delete(id)
}
