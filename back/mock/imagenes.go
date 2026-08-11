package mock

import (
	"time"

	"back/domain"
)

type ImagenStore struct {
	store *MockStore[domain.Imagen]
}

func NewImagenStore() *ImagenStore {
	return &ImagenStore{store: NewMockStore[domain.Imagen](nil)}
}

func (s *ImagenStore) Add(tipo string, entidadID int64, fileName string) (domain.Imagen, error) {
	return s.store.Create(domain.Imagen{
		Tipo:      tipo,
		EntidadID: entidadID,
		FileName:  fileName,
		CreadoEn:  time.Now(),
	})
}

func (s *ImagenStore) ListByEntidad(tipo string, entidadID int64) []domain.Imagen {
	var out []domain.Imagen
	for _, i := range s.store.FindAll() {
		if i.Tipo == tipo && i.EntidadID == entidadID {
			out = append(out, i)
		}
	}
	return out
}

func (s *ImagenStore) FindByID(id int64) (*domain.Imagen, error) {
	i, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (s *ImagenStore) Delete(id int64) error {
	return s.store.Delete(id)
}

// DeleteByEntidad borra los registros de una entidad y devuelve los filenames
// eliminados para limpiar el disco después.
func (s *ImagenStore) DeleteByEntidad(tipo string, entidadID int64) []string {
	var removed []string
	for _, i := range s.store.FindAll() {
		if i.Tipo == tipo && i.EntidadID == entidadID {
			removed = append(removed, i.FileName)
			_ = s.store.Delete(i.ID)
		}
	}
	return removed
}
