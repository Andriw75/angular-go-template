package database

import (
	"context"
	"database/sql"
	"time"

	"back/domain"
)

type ImagenStore struct {
	db *DB
}

func NewImagenStore(db *DB) *ImagenStore {
	return &ImagenStore{db: db}
}

const imagenColumns = `id, tipo, entidad_id, file_name, orden, creado_en`

func (s *ImagenStore) Add(tipo string, entidadID int64, fileName string) (domain.Imagen, error) {
	ctx := context.Background()

	var next int
	qNext := s.db.rebind(`SELECT COALESCE(MAX(orden), 0) + 1 FROM imagenes WHERE tipo = ? AND entidad_id = ?`)
	if err := s.db.QueryRowContext(ctx, qNext, tipo, entidadID).Scan(&next); err != nil {
		next = 1
	}

	img := domain.Imagen{
		Tipo:      tipo,
		EntidadID: entidadID,
		FileName:  fileName,
		Orden:     next,
		CreadoEn:  time.Now(),
	}

	q := s.db.rebind(`INSERT INTO imagenes (tipo, entidad_id, file_name, orden, creado_en)
		VALUES (?, ?, ?, ?, ?) RETURNING id`)
	if err := s.db.QueryRowContext(ctx, q, tipo, entidadID, fileName, next, formatTime(img.CreadoEn)).Scan(&img.ID); err != nil {
		return domain.Imagen{}, err
	}
	return img, nil
}

func (s *ImagenStore) ListAll() []domain.Imagen {
	rows, err := s.db.QueryContext(context.Background(), `SELECT `+imagenColumns+` FROM imagenes ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.Imagen
	for rows.Next() {
		img, err := scanImagen(rows)
		if err != nil {
			return out
		}
		out = append(out, *img)
	}
	return out
}

func (s *ImagenStore) ListByEntidad(tipo string, entidadID int64) []domain.Imagen {
	q := s.db.rebind(`SELECT ` + imagenColumns + ` FROM imagenes
		WHERE tipo = ? AND entidad_id = ? ORDER BY orden, id`)
	rows, err := s.db.QueryContext(context.Background(), q, tipo, entidadID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.Imagen
	for rows.Next() {
		img, err := scanImagen(rows)
		if err != nil {
			return out
		}
		out = append(out, *img)
	}
	return out
}

func (s *ImagenStore) Reorder(tipo string, entidadID int64, ids []int64) error {
	existing := s.ListByEntidad(tipo, entidadID)
	if len(existing) != len(ids) {
		return sql.ErrNoRows
	}

	valid := make(map[int64]bool, len(existing))
	for _, img := range existing {
		valid[img.ID] = true
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := s.db.rebind(`UPDATE imagenes SET orden = ? WHERE id = ? AND tipo = ? AND entidad_id = ?`)
	for i, id := range ids {
		if !valid[id] {
			return sql.ErrNoRows
		}
		if _, err := tx.ExecContext(ctx, q, i+1, id, tipo, entidadID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *ImagenStore) FindByID(id int64) (*domain.Imagen, error) {
	q := s.db.rebind(`SELECT ` + imagenColumns + ` FROM imagenes WHERE id = ?`)
	return scanImagen(s.db.QueryRowContext(context.Background(), q, id))
}

func (s *ImagenStore) FindByFileName(fileName string) (*domain.Imagen, error) {
	q := s.db.rebind(`SELECT ` + imagenColumns + ` FROM imagenes WHERE file_name = ?`)
	return scanImagen(s.db.QueryRowContext(context.Background(), q, fileName))
}

func (s *ImagenStore) Delete(id int64) error {
	res, err := s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM imagenes WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ImagenStore) DeleteByEntidad(tipo string, entidadID int64) []string {
	existing := s.ListByEntidad(tipo, entidadID)
	files := make([]string, 0, len(existing))
	for _, img := range existing {
		files = append(files, img.FileName)
	}
	_, _ = s.db.ExecContext(context.Background(),
		s.db.rebind(`DELETE FROM imagenes WHERE tipo = ? AND entidad_id = ?`), tipo, entidadID)
	return files
}

func scanImagen(sc rowScanner) (*domain.Imagen, error) {
	var (
		img    domain.Imagen
		creado string
	)
	if err := sc.Scan(&img.ID, &img.Tipo, &img.EntidadID, &img.FileName, &img.Orden, &creado); err != nil {
		return nil, err
	}
	img.CreadoEn = parseTime(creado)
	return &img, nil
}
