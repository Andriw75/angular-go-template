package mock

import (
	"errors"

	"back/domain"
	"back/infrastructure/auth"
)

type UserStore struct {
	store *MockStore[domain.User]
}

func NewUserStore(crypt *auth.CryptManager) *UserStore {
	s := &UserStore{store: NewMockStore[domain.User](nil)}

	hash, err := crypt.Hash("abc123zyx")
	if err != nil {
		panic(err)
	}

	s.store.Insert(1, domain.User{
		ID:       1,
		Username: "andriwdv",
		Email:    "admin@example.com",
		Password: hash,
		Activo:   true,
		Permisos: []string{"admin"},
	})

	return s
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
