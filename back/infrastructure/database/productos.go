package database

import (
	"context"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
	"back/store"
)

type ProductoStore struct {
	db *DB
}

func NewProductoStore(db *DB) *ProductoStore {
	return &ProductoStore{db: db}
}

const productoColumns = `id, nombre, descripcion, precio, stock, categoria_id, activo, creado_en, actualizado_en`

func productoWhere(f store.ProductoFilters) (string, []any) {
	var conds []string
	var args []any
	if f.Q != "" {
		like := "%" + strings.ToLower(f.Q) + "%"
		conds = append(conds, `(LOWER(nombre) LIKE ? OR LOWER(descripcion) LIKE ?)`)
		args = append(args, like, like)
	}
	if f.CategoriaID > 0 {
		conds = append(conds, `categoria_id = ?`)
		args = append(args, f.CategoriaID)
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

func (s *ProductoStore) Count(filters store.ProductoFilters) int64 {
	where, args := productoWhere(filters)
	var count int64
	if err := s.db.QueryRowContext(context.Background(), s.db.rebind(`SELECT COUNT(*) FROM productos`+where), args...).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *ProductoStore) List(offset, limit int, filters store.ProductoFilters) outputs.PaginatedResponse[outputs.ProductoResponse] {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 10
	}

	where, args := productoWhere(filters)
	q := s.db.rebind(`SELECT ` + productoColumns + ` FROM productos` + where + ` ORDER BY id LIMIT ? OFFSET ?`)
	args = append(args, limit+1, offset)

	rows, err := s.db.QueryContext(context.Background(), q, args...)
	if err != nil {
		return outputs.PaginatedResponse[outputs.ProductoResponse]{Data: []outputs.ProductoResponse{}, Limit: limit, Offset: offset}
	}
	defer rows.Close()

	data := []outputs.ProductoResponse{}
	hasMore := false
	for rows.Next() {
		if len(data) == limit {
			hasMore = true
			break
		}
		p, err := scanProducto(rows)
		if err != nil {
			break
		}
		data = append(data, outputs.ToProductoResponse(p))
	}

	return outputs.PaginatedResponse[outputs.ProductoResponse]{
		Data:    data,
		Offset:  offset,
		Limit:   limit,
		HasMore: hasMore,
	}
}

func (s *ProductoStore) FindByID(id int64) (*domain.Producto, error) {
	q := s.db.rebind(`SELECT ` + productoColumns + ` FROM productos WHERE id = ?`)
	producto, err := scanProducto(s.db.QueryRowContext(context.Background(), q, id))
	if err != nil {
		return nil, translateDBError(err, "producto no encontrado")
	}
	return producto, nil
}

func (s *ProductoStore) Create(p domain.Producto) (domain.Producto, error) {
	now := time.Now()
	p.CreadoEn = now
	p.ActualizadoEn = now

	q := s.db.rebind(`INSERT INTO productos (nombre, descripcion, precio, stock, categoria_id, activo, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	err := s.db.QueryRowContext(context.Background(), q,
		p.Nombre, p.Descripcion, p.Precio, p.Stock, p.CategoriaID, boolInt(p.Activo), formatTime(now), formatTime(now),
	).Scan(&p.ID)
	if err != nil {
		return domain.Producto{}, err
	}
	return p, nil
}

func (s *ProductoStore) Update(id int64, p domain.Producto) error {
	p.ActualizadoEn = time.Now()
	q := s.db.rebind(`UPDATE productos SET nombre = ?, descripcion = ?, precio = ?, stock = ?,
		categoria_id = ?, activo = ?, actualizado_en = ? WHERE id = ?`)
	res, err := s.db.ExecContext(context.Background(), q,
		p.Nombre, p.Descripcion, p.Precio, p.Stock, p.CategoriaID, boolInt(p.Activo), formatTime(p.ActualizadoEn), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("producto no encontrado")
	}
	return nil
}

func (s *ProductoStore) Delete(id int64) error {
	res, err := s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM productos WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("producto no encontrado")
	}
	return nil
}

func (s *ProductoStore) CountByCategoria(categoriaID int64) int64 {
	var count int64
	q := s.db.rebind(`SELECT COUNT(*) FROM productos WHERE categoria_id = ?`)
	if err := s.db.QueryRowContext(context.Background(), q, categoriaID).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *ProductoStore) ListByCategoria(categoriaID int64) []domain.Producto {
	q := s.db.rebind(`SELECT ` + productoColumns + ` FROM productos WHERE categoria_id = ? ORDER BY id`)
	rows, err := s.db.QueryContext(context.Background(), q, categoriaID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.Producto
	for rows.Next() {
		p, err := scanProducto(rows)
		if err != nil {
			return out
		}
		out = append(out, *p)
	}
	return out
}

func (s *ProductoStore) DeleteByCategoria(categoriaID int64) {
	_, _ = s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM productos WHERE categoria_id = ?`), categoriaID)
}

func scanProducto(sc rowScanner) (*domain.Producto, error) {
	var (
		p           domain.Producto
		activo      int64
		creado      string
		actualizado string
	)
	if err := sc.Scan(&p.ID, &p.Nombre, &p.Descripcion, &p.Precio, &p.Stock,
		&p.CategoriaID, &activo, &creado, &actualizado); err != nil {
		return nil, err
	}
	p.Activo = intBool(activo)
	p.CreadoEn = parseTime(creado)
	p.ActualizadoEn = parseTime(actualizado)
	return &p, nil
}
