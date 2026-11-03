package mock

import (
	"errors"
	"time"

	"back/domain"
	"back/domain/inputs"
	"back/infrastructure/auth"
)

type UserStore struct {
	store *MockStore[domain.User]
	crypt *auth.CryptManager
}

func NewUserStore(crypt *auth.CryptManager) *UserStore {
	// Sin usuarios sembrados: las credenciales por defecto no existen.
	return &UserStore{store: NewMockStore[domain.User](nil), crypt: crypt}
}

func (s *UserStore) Create(input inputs.UserInput) (domain.User, error) {
	hash, err := s.crypt.Hash(input.Password)
	if err != nil {
		return domain.User{}, err
	}
	now := time.Now()
	user := domain.User{
		Username:           input.Username,
		Email:              input.Email,
		Password:           hash,
		Activo:             input.Activo,
		Permisos:           input.Permisos,
		MustChangePassword: input.MustChangePassword,
		CreadoEn:           now,
		ActualizadoEn:      now,
	}
	return s.store.Create(user)
}

func (s *UserStore) Update(id int64, input inputs.UserUpdateInput) error {
	existing, err := s.store.FindByID(id)
	if err != nil {
		return err
	}

	if input.Username != nil {
		existing.Username = *input.Username
	}
	if input.Email != nil {
		existing.Email = *input.Email
	}
	if input.Activo != nil {
		existing.Activo = *input.Activo
	}
	if input.Permisos != nil {
		existing.Permisos = *input.Permisos
	}
	if input.Password != nil && *input.Password != "" {
		hash, err := s.crypt.Hash(*input.Password)
		if err != nil {
			return err
		}
		existing.Password = hash
		existing.MustChangePassword = true
	}
	if input.MustChangePassword != nil {
		existing.MustChangePassword = *input.MustChangePassword
	}

	existing.ActualizadoEn = time.Now()
	return s.store.Update(id, existing)
}

func (s *UserStore) UpdatePassword(id int64, hash string, mustChange bool) error {
	existing, err := s.store.FindByID(id)
	if err != nil {
		return err
	}
	existing.Password = hash
	existing.MustChangePassword = mustChange
	existing.ActualizadoEn = time.Now()
	return s.store.Update(id, existing)
}

func (s *UserStore) Delete(id int64) error {
	return s.store.Delete(id)
}

func (s *UserStore) FindByUsername(username string) (*domain.User, error) {
	for _, u := range s.store.FindAll() {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (s *UserStore) FindByID(id int64) (*domain.User, error) {
	u, err := s.store.FindByID(id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserStore) FindAll() []domain.User {
	return s.store.FindAll()
}

func (s *UserStore) IsValidPassword(crypt *auth.CryptManager, username, password string) (*domain.User, bool) {
	user, err := s.FindByUsername(username)
	if err != nil {
		return nil, false
	}
	if !crypt.Verify(user.Password, password) {
		return nil, false
	}
	return user, true
}
