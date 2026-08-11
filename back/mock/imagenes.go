package mock

import (
	"fmt"
	"sort"
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
	next := 1
	for _, i := range s.ListByEntidad(tipo, entidadID) {
		if i.Orden >= next {
			next = i.Orden + 1
		}
	}
	return s.store.Create(domain.Imagen{
		Tipo:      tipo,
		EntidadID: entidadID,
		FileName:  fileName,
		Orden:     next,
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
	sort.Slice(out, func(a, b int) bool {
		if out[a].Orden != out[b].Orden {
			return out[a].Orden < out[b].Orden
		}
		return out[a].ID < out[b].ID
	})
	return out
}

// Reorder actualiza el orden de las imágenes existentes. ids debe contener
// exactamente los IDs de las imágenes de la entidad, en el nuevo orden.
func (s *ImagenStore) Reorder(tipo string, entidadID int64, ids []int64) error {
	existing := s.ListByEntidad(tipo, entidadID)
	if len(existing) != len(ids) {
		return fmt.Errorf("la cantidad de imágenes no coincide")
	}
	idSet := make(map[int64]domain.Imagen, len(existing))
	for _, i := range existing {
		idSet[i.ID] = i
	}
	for ord, id := range ids {
		img, ok := idSet[id]
		if !ok {
			return fmt.Errorf("imagen no encontrada")
		}
		img.Orden = ord + 1
		if err := s.store.Update(id, img); err != nil {
			return err
		}
	}
	return nil
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
