package database

import (
	"context"
	"database/sql"

	"back/domain"
)

type PermisoStore struct {
	db *DB
}

func NewPermisoStore(db *DB) *PermisoStore {
	return &PermisoStore{db: db}
}

func (s *PermisoStore) FindAll(ctx context.Context) []domain.Permission {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nombre, descripcion FROM permissions ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Descripcion); err != nil {
			return out
		}
		out = append(out, p)
	}
	return out
}

func (s *PermisoStore) FindByID(ctx context.Context, id int64) (*domain.Permission, error) {
	q := s.db.rebind(`SELECT id, nombre, descripcion FROM permissions WHERE id = ?`)
	var p domain.Permission
	err := s.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Nombre, &p.Descripcion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &p, nil
}
