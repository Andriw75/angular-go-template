package database

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	sqlite "modernc.org/sqlite"

	"back/domain"
)

// isUniqueViolation reconoce el conflicto de unicidad tanto en PostgreSQL
// (SQLSTATE 23505) como en SQLite (constraint UNIQUE / PRIMARY KEY).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case 2067, 1555: // SQLITE_CONSTRAINT_UNIQUE, SQLITE_CONSTRAINT_PRIMARYKEY
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "unique violation")
}

// translateDBError convierte errores de persistencia en errores de negocio.
// Un error desconocido se devuelve tal cual para que la capa HTTP lo registre
// y responda 500 sin filtrar detalles internos.
func translateDBError(err error, notFoundMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NotFound(notFoundMessage)
	}
	if isUniqueViolation(err) {
		return domain.Conflict("conflict", "el registro ya existe")
	}
	return err
}
