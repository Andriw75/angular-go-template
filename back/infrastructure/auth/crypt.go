package auth

import "golang.org/x/crypto/bcrypt"

type CryptManager struct{}

func NewCryptManager() *CryptManager {
	return &CryptManager{}
}

const bcryptCost = 12

func (m *CryptManager) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

func (m *CryptManager) Verify(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
