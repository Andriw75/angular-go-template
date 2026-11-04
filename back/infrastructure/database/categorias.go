package database

import (
	"context"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
	"back/store"
)

type CategoriaStore struct {
	db *DB
}

func NewCategoriaStore(db *DB) *CategoriaStore {
	return &CategoriaStore{db: db}
}

const categoriaColumns = `id, nombre, descripcion, activo, creado_en, actualizado_en`

func categoriaWhere(f store.CategoriaFilters) (string, []any) {
	var conds []string
	var args []any
	if f.Q != "" {
		like := "%" + strings.ToLower(f.Q) + "%"
		conds = append(conds, `(LOWER(nombre) LIKE ? OR LOWER(descripcion) LIKE ?)`)
		args = append(args, like, like)
	}
	if f.Activo != nil {
		conds = append(conds, `activo = ?`)
		args = append(args, boolInt(*f.Activo))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (s *CategoriaStore) Count(filters store.CategoriaFilters) int64 {
	where, args := categoriaWhere(filters)
	var count int64
	if err := s.db.QueryRowContext(context.Background(), s.db.rebind(`SELECT COUNT(*) FROM categorias`+where), args...).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *CategoriaStore) List(offset, limit int, filters store.CategoriaFilters) outputs.PaginatedResponse[outputs.CategoriaResponse] {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 10
	}

	where, args := categoriaWhere(filters)
	q := s.db.rebind(`SELECT ` + categoriaColumns + ` FROM categorias` + where + ` ORDER BY id LIMIT ? OFFSET ?`)
	args = append(args, limit+1, offset)

	rows, err := s.db.QueryContext(context.Background(), q, args...)
	if err != nil {
		return outputs.PaginatedResponse[outputs.CategoriaResponse]{Data: []outputs.CategoriaResponse{}, Limit: limit, Offset: offset}
	}
	defer rows.Close()

	data := []outputs.CategoriaResponse{}
	hasMore := false
	for rows.Next() {
		if len(data) == limit {
			hasMore = true
			break
		}
		c, err := scanCategoria(rows)
		if err != nil {
			break
		}
		data = append(data, outputs.ToCategoriaResponse(c))
	}

	return outputs.PaginatedResponse[outputs.CategoriaResponse]{
		Data:    data,
		Offset:  offset,
		Limit:   limit,
		HasMore: hasMore,
	}
}

func (s *CategoriaStore) FindByID(id int64) (*domain.Categoria, error) {
	q := s.db.rebind(`SELECT ` + categoriaColumns + ` FROM categorias WHERE id = ?`)
	categoria, err := scanCategoria(s.db.QueryRowContext(context.Background(), q, id))
	if err != nil {
		return nil, translateDBError(err, "categoria no encontrada")
	}
	return categoria, nil
}

func (s *CategoriaStore) Create(c domain.Categoria) (domain.Categoria, error) {
	now := time.Now()
	c.CreadoEn = now
	c.ActualizadoEn = now

	q := s.db.rebind(`INSERT INTO categorias (nombre, descripcion, activo, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?) RETURNING id`)
	err := s.db.QueryRowContext(context.Background(), q,
		c.Nombre, c.Descripcion, boolInt(c.Activo), formatTime(now), formatTime(now),
	).Scan(&c.ID)
	if err != nil {
		return domain.Categoria{}, err
	}
	return c, nil
}

func (s *CategoriaStore) Update(id int64, c domain.Categoria) error {
	c.ActualizadoEn = time.Now()
	q := s.db.rebind(`UPDATE categorias SET nombre = ?, descripcion = ?, activo = ?, actualizado_en = ? WHERE id = ?`)
	res, err := s.db.ExecContext(context.Background(), q, c.Nombre, c.Descripcion, boolInt(c.Activo), formatTime(c.ActualizadoEn), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("categoria no encontrada")
	}
	return nil
}

func (s *CategoriaStore) Delete(id int64) error {
	res, err := s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM categorias WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("categoria no encontrada")
	}
	return nil
}

func scanCategoria(sc rowScanner) (*domain.Categoria, error) {
	var (
		c           domain.Categoria
		activo      int64
		creado      string
		actualizado string
	)
	if err := sc.Scan(&c.ID, &c.Nombre, &c.Descripcion, &activo, &creado, &actualizado); err != nil {
		return nil, err
	}
	c.Activo = intBool(activo)
	c.CreadoEn = parseTime(creado)
	c.ActualizadoEn = parseTime(actualizado)
	return &c, nil
}
