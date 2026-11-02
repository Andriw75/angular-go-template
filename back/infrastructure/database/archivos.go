package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"back/domain"
)

type ArchivoStore struct {
	db *DB
}

func NewArchivoStore(db *DB) *ArchivoStore {
	return &ArchivoStore{db: db}
}

const archivoColumns = `id, file_name, nombre, tipo, tamano, visibilidad, user_id, creado_en`

func (s *ArchivoStore) Create(fileName, nombre, tipo string, tamano int64, visibilidad domain.Visibilidad, userID int64) (domain.Archivo, error) {
	if visibilidad == "" {
		visibilidad = domain.VisibilidadPublico
	}

	a := domain.Archivo{
		FileName:    fileName,
		Nombre:      nombre,
		Tipo:        tipo,
		Tamano:      tamano,
		Visibilidad: visibilidad,
		UserID:      userID,
		CreadoEn:    time.Now(),
	}

	q := s.db.rebind(`INSERT INTO archivos (file_name, nombre, tipo, tamano, visibilidad, user_id, creado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	err := s.db.QueryRowContext(context.Background(), q,
		a.FileName, a.Nombre, a.Tipo, a.Tamano, string(a.Visibilidad), a.UserID, formatTime(a.CreadoEn),
	).Scan(&a.ID)
	if err != nil {
		return domain.Archivo{}, err
	}
	return a, nil
}

func (s *ArchivoStore) FindByID(id int64) (*domain.Archivo, error) {
	q := s.db.rebind(`SELECT ` + archivoColumns + ` FROM archivos WHERE id = ?`)
	return scanArchivo(s.db.QueryRowContext(context.Background(), q, id))
}

func (s *ArchivoStore) FindByFileName(fileName string) (*domain.Archivo, error) {
	q := s.db.rebind(`SELECT ` + archivoColumns + ` FROM archivos WHERE file_name = ?`)
	return scanArchivo(s.db.QueryRowContext(context.Background(), q, fileName))
}

func (s *ArchivoStore) List(offset, limit int, q string) ([]domain.Archivo, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}

	ctx := context.Background()
	var total int
	var args []any
	where := ""
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = ` WHERE (LOWER(nombre) LIKE ? OR LOWER(file_name) LIKE ?)`
		args = append(args, like, like)
	}

	if err := s.db.QueryRowContext(ctx, s.db.rebind(`SELECT COUNT(*) FROM archivos`+where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := s.db.rebind(`SELECT ` + archivoColumns + ` FROM archivos` + where + ` ORDER BY creado_en DESC LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.Archivo
	for rows.Next() {
		a, err := scanArchivo(rows)
		if err != nil {
			return out, total, err
		}
		out = append(out, *a)
	}
	return out, total, nil
}

func (s *ArchivoStore) UpdateVisibilidad(id int64, vis domain.Visibilidad) error {
	q := s.db.rebind(`UPDATE archivos SET visibilidad = ? WHERE id = ?`)
	res, err := s.db.ExecContext(context.Background(), q, string(vis), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ArchivoStore) Delete(id int64) error {
	res, err := s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM archivos WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanArchivo(sc rowScanner) (*domain.Archivo, error) {
	var (
		a      domain.Archivo
		vis    string
		creado string
	)
	if err := sc.Scan(&a.ID, &a.FileName, &a.Nombre, &a.Tipo, &a.Tamano, &vis, &a.UserID, &creado); err != nil {
		return nil, err
	}
	a.Visibilidad = domain.Visibilidad(vis)
	a.CreadoEn = parseTime(creado)
	return &a, nil
}
