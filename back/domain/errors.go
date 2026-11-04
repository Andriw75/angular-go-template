package domain

import "fmt"

// ErrorKind clasifica un error de negocio para que la capa HTTP elija el
// código de estado sin inspeccionar mensajes del driver.
type ErrorKind int

const (
	KindInternal ErrorKind = iota
	KindNotFound
	KindConflict
	KindValidation
	KindUnauthorized
	KindForbidden
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func NewError(kind ErrorKind, code, message string, err error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: err}
}

func NotFound(message string) *Error {
	return NewError(KindNotFound, "not_found", message, nil)
}

func Conflict(code, message string) *Error {
	if code == "" {
		code = "conflict"
	}
	return NewError(KindConflict, code, message, nil)
}

func Validation(message string) *Error {
	return NewError(KindValidation, "validation_error", message, nil)
}

func Unauthorized(code, message string) *Error {
	if code == "" {
		code = "unauthorized"
	}
	return NewError(KindUnauthorized, code, message, nil)
}

func Forbidden(message string) *Error {
	return NewError(KindForbidden, "forbidden", message, nil)
}
