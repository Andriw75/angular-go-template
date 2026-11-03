package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"back/domain"
	"back/domain/outputs"
	"back/store"
)

type BusStore struct {
	db *DB
}

func NewBusStore(db *DB) *BusStore {
	return &BusStore{db: db}
}

const busColumns = `id, placa, nombre, marca, modelo, anio, capacidad, tipo, activo,
	fecha_compra, ultimo_mantenimiento, precio, peso, color, descripcion, creado_en, actualizado_en`

func busWhere(f store.BusFilters) (string, []any) {
	var conds []string
	var args []any

	if f.Q != "" {
		like := "%" + strings.ToLower(f.Q) + "%"
		conds = append(conds, `(LOWER(placa) LIKE ? OR LOWER(nombre) LIKE ? OR LOWER(marca) LIKE ? OR LOWER(modelo) LIKE ?)`)
		args = append(args, like, like, like, like)
	}
	if f.Tipo != "" {
		conds = append(conds, `LOWER(tipo) = LOWER(?)`)
		args = append(args, f.Tipo)
	}
	if f.Activo != nil {
		conds = append(conds, `activo = ?`)
		args = append(args, boolInt(*f.Activo))
	}
	if f.Desde != "" {
		conds = append(conds, `fecha_compra >= ?`)
		args = append(args, f.Desde)
	}
	if f.Hasta != "" {
		conds = append(conds, `fecha_compra <= ?`)
		args = append(args, f.Hasta)
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (s *BusStore) Count(filters store.BusFilters) int64 {
	where, args := busWhere(filters)
	var count int64
	if err := s.db.QueryRowContext(context.Background(), s.db.rebind(`SELECT COUNT(*) FROM buses`+where), args...).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *BusStore) List(offset, limit int, filters store.BusFilters) outputs.PaginatedResponse[outputs.BusResponse] {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 10
	}

	where, args := busWhere(filters)
	q := s.db.rebind(`SELECT ` + busColumns + ` FROM buses` + where + ` ORDER BY id LIMIT ? OFFSET ?`)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(context.Background(), q, args...)
	if err != nil {
		return outputs.PaginatedResponse[outputs.BusResponse]{Data: []outputs.BusResponse{}, Limit: limit, Offset: offset}
	}
	defer rows.Close()

	data := []outputs.BusResponse{}
	for rows.Next() {
		b, err := scanBus(rows)
		if err != nil {
			break
		}
		data = append(data, outputs.ToBusResponse(b))
	}

	return outputs.PaginatedResponse[outputs.BusResponse]{
		Data:   data,
		Total:  s.Count(filters),
		Offset: offset,
		Limit:  limit,
	}
}

func (s *BusStore) FindByID(id int64) (*domain.Bus, error) {
	q := s.db.rebind(`SELECT ` + busColumns + ` FROM buses WHERE id = ?`)
	return scanBus(s.db.QueryRowContext(context.Background(), q, id))
}

func (s *BusStore) Create(b domain.Bus) (domain.Bus, error) {
	now := time.Now()
	b.CreadoEn = now
	b.ActualizadoEn = now

	q := s.db.rebind(`INSERT INTO buses
		(placa, nombre, marca, modelo, anio, capacidad, tipo, activo, fecha_compra,
		 ultimo_mantenimiento, precio, peso, color, descripcion, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)

	err := s.db.QueryRowContext(context.Background(), q,
		b.Placa, b.Nombre, b.Marca, b.Modelo, b.Anio, b.Capacidad, b.Tipo, boolInt(b.Activo),
		b.FechaCompra.Format("2006-01-02"), formatTimePtr(b.UltimoMantenimiento), b.Precio, b.Peso,
		b.Color, b.Descripcion, formatTime(now), formatTime(now),
	).Scan(&b.ID)
	if err != nil {
		return domain.Bus{}, err
	}
	return b, nil
}

func (s *BusStore) Update(id int64, b domain.Bus) error {
	b.ActualizadoEn = time.Now()
	q := s.db.rebind(`UPDATE buses SET
		placa = ?, nombre = ?, marca = ?, modelo = ?, anio = ?, capacidad = ?, tipo = ?, activo = ?,
		fecha_compra = ?, ultimo_mantenimiento = ?, precio = ?, peso = ?, color = ?, descripcion = ?, actualizado_en = ?
		WHERE id = ?`)

	res, err := s.db.ExecContext(context.Background(), q,
		b.Placa, b.Nombre, b.Marca, b.Modelo, b.Anio, b.Capacidad, b.Tipo, boolInt(b.Activo),
		b.FechaCompra.Format("2006-01-02"), formatTimePtr(b.UltimoMantenimiento), b.Precio, b.Peso,
		b.Color, b.Descripcion, formatTime(b.ActualizadoEn), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *BusStore) Delete(id int64) error {
	q := s.db.rebind(`DELETE FROM buses WHERE id = ?`)
	res, err := s.db.ExecContext(context.Background(), q, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanBus(sc rowScanner) (*domain.Bus, error) {
	var (
		b           domain.Bus
		activo      int64
		fecha       string
		ultimo      sql.NullString
		creado      string
		actualizado string
	)
	if err := sc.Scan(
		&b.ID, &b.Placa, &b.Nombre, &b.Marca, &b.Modelo, &b.Anio, &b.Capacidad,
		&b.Tipo, &activo, &fecha, &ultimo, &b.Precio, &b.Peso, &b.Color,
		&b.Descripcion, &creado, &actualizado,
	); err != nil {
		return nil, err
	}
	b.Activo = intBool(activo)
	b.FechaCompra = parseTime(fecha)
	if ultimo.Valid {
		t := parseTime(ultimo.String)
		b.UltimoMantenimiento = &t
	}
	b.CreadoEn = parseTime(creado)
	b.ActualizadoEn = parseTime(actualizado)
	return &b, nil
}
