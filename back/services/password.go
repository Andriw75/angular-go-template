package services

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// commonPasswords es una lista mínima de contraseñas prohibidas. No pretende
// ser exhaustiva; evita lo obvio y la contraseña histórica del template.
var commonPasswords = map[string]struct{}{
	"password":      {},
	"password123":   {},
	"123456":        {},
	"12345678":      {},
	"123456789":     {},
	"1234567890":    {},
	"qwerty":        {},
	"qwerty123":     {},
	"abc123":        {},
	"admin":         {},
	"admin123":      {},
	"administrador": {},
	"iloveyou":      {},
	"welcome":       {},
	"letmein":       {},
	"monkey":        {},
	"dragon":        {},
	"football":      {},
	"master":        {},
	"sunshine":      {},
	"princess":      {},
	"login":         {},
	"passw0rd":      {},
}

// ValidatePassword aplica la política mínima: 12 caracteres, máximo 72 bytes
// (límite de bcrypt) y rechazo de contraseñas comunes.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < 12 {
		return fmt.Errorf("la contraseña debe tener al menos 12 caracteres")
	}
	if len([]byte(password)) > 72 {
		return fmt.Errorf("la contraseña no puede superar los 72 bytes")
	}
	if _, ok := commonPasswords[strings.ToLower(strings.TrimSpace(password))]; ok {
		return fmt.Errorf("la contraseña es demasiado común")
	}
	return nil
}

// ValidatePasswordForUser añade la comprobación de que la contraseña no sea
// igual al nombre de usuario.
func ValidatePasswordForUser(username, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(username), strings.TrimSpace(password)) {
		return fmt.Errorf("la contraseña no puede ser igual al nombre de usuario")
	}
	return nil
}
