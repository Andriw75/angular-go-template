package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"back/domain"
	"back/domain/inputs"
	"back/store"
)

type VentaStore struct {
	db *DB
}

func NewVentaStore(db *DB) *VentaStore {
	return &VentaStore{db: db}
}

const ventaColumns = `id, serie, correlativo, codigo, cliente_nombre, cliente_documento,
	cliente_direccion, fecha, moneda, subtotal, descuento, impuesto, total, estado,
	observaciones, user_id, creado_en, actualizado_en`

const ventaItemColumns = `id, venta_id, producto_id, descripcion, cantidad, precio_unitario,
	descuento, impuesto_porcentaje, subtotal, impuesto, total`

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func normalizeVenta(input inputs.VentaInput) (domain.Venta, error) {
	if len(input.Items) == 0 {
		return domain.Venta{}, errors.New("la venta debe tener al menos un item")
	}

	fecha := time.Now()
	if input.Fecha != "" {
		if t, err := time.Parse("2006-01-02", input.Fecha); err == nil {
			fecha = t
		}
	}

	serie := strings.ToUpper(strings.TrimSpace(input.Serie))
	if serie == "" {
		serie = "FAC"
	}
	moneda := strings.ToUpper(strings.TrimSpace(input.Moneda))
	if moneda == "" {
		moneda = "PEN"
	}

	items := make([]domain.VentaItem, 0, len(input.Items))
	var subtotal, descuento, impuesto, total float64
	for _, it := range input.Items {
		if it.Cantidad <= 0 {
			return domain.Venta{}, errors.New("la cantidad debe ser mayor a cero")
		}
		if it.PrecioUnitario < 0 {
			return domain.Venta{}, errors.New("el precio no puede ser negativo")
		}
		if strings.TrimSpace(it.Descripcion) == "" {
			return domain.Venta{}, errors.New("la descripcion del item es obligatoria")
		}

		base := it.Cantidad*it.PrecioUnitario - it.Descuento
		if base < 0 {
			base = 0
		}
		lineTax := base * it.ImpuestoPorcentaje / 100
		lineTotal := base + lineTax

		subtotal += base
		descuento += it.Descuento
		impuesto += lineTax
		total += lineTotal

		items = append(items, domain.VentaItem{
			ProductoID:         it.ProductoID,
			Descripcion:        strings.TrimSpace(it.Descripcion),
			Cantidad:           it.Cantidad,
			PrecioUnitario:     it.PrecioUnitario,
			Descuento:          it.Descuento,
			ImpuestoPorcentaje: it.ImpuestoPorcentaje,
			Subtotal:           round2(base),
			Impuesto:           round2(lineTax),
			Total:              round2(lineTotal),
		})
	}

	return domain.Venta{
		Serie:            serie,
		ClienteNombre:    strings.TrimSpace(input.ClienteNombre),
		ClienteDocumento: strings.TrimSpace(input.ClienteDocumento),
		ClienteDireccion: strings.TrimSpace(input.ClienteDireccion),
		Fecha:            fecha,
		Moneda:           moneda,
		Subtotal:         round2(subtotal),
		Descuento:        round2(descuento),
		Impuesto:         round2(impuesto),
		Total:            round2(total),
		Estado:           "borrador",
		Observaciones:    strings.TrimSpace(input.Observaciones),
		Items:            items,
	}, nil
}

func (s *VentaStore) Create(input inputs.VentaInput, userID int64) (domain.Venta, error) {
	venta, err := normalizeVenta(input)
	if err != nil {
		return domain.Venta{}, err
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Venta{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	q := s.db.rebind(`INSERT INTO ventas
		(serie, correlativo, codigo, cliente_nombre, cliente_documento, cliente_direccion, fecha,
		 moneda, subtotal, descuento, impuesto, total, estado, observaciones, user_id, creado_en, actualizado_en)
		VALUES (?, 0, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)

	err = tx.QueryRowContext(ctx, q,
		venta.Serie, venta.ClienteNombre, venta.ClienteDocumento, venta.ClienteDireccion,
		venta.Fecha.Format("2006-01-02"), venta.Moneda, venta.Subtotal, venta.Descuento,
		venta.Impuesto, venta.Total, venta.Estado, venta.Observaciones, userID,
		formatTime(now), formatTime(now),
	).Scan(&venta.ID)
	if err != nil {
		return domain.Venta{}, err
	}

	if err := s.insertItems(ctx, tx, venta.ID, venta.Items); err != nil {
		return domain.Venta{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Venta{}, err
	}
	return s.getVenta(venta.ID)
}

func (s *VentaStore) insertItems(ctx context.Context, tx *sql.Tx, ventaID int64, items []domain.VentaItem) error {
	q := s.db.rebind(`INSERT INTO venta_items
		(venta_id, producto_id, descripcion, cantidad, precio_unitario, descuento,
		 impuesto_porcentaje, subtotal, impuesto, total)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	for i := range items {
		it := &items[i]
		it.VentaID = ventaID
		if err := tx.QueryRowContext(ctx, q,
			ventaID, it.ProductoID, it.Descripcion, it.Cantidad, it.PrecioUnitario,
			it.Descuento, it.ImpuestoPorcentaje, it.Subtotal, it.Impuesto, it.Total,
		).Scan(&it.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *VentaStore) Update(id int64, input inputs.VentaInput) (domain.Venta, error) {
	venta, err := normalizeVenta(input)
	if err != nil {
		return domain.Venta{}, err
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Venta{}, err
	}
	defer tx.Rollback()

	var estado string
	if err := tx.QueryRowContext(ctx, s.db.rebind(`SELECT estado FROM ventas WHERE id = ?`), id).Scan(&estado); err != nil {
		return domain.Venta{}, err
	}
	if estado != "borrador" {
		return domain.Venta{}, errors.New("solo se puede editar una venta en borrador")
	}

	now := time.Now()
	q := s.db.rebind(`UPDATE ventas SET serie = ?, cliente_nombre = ?, cliente_documento = ?,
		cliente_direccion = ?, fecha = ?, moneda = ?, subtotal = ?, descuento = ?, impuesto = ?,
		total = ?, observaciones = ?, actualizado_en = ? WHERE id = ?`)
	if _, err := tx.ExecContext(ctx, q,
		venta.Serie, venta.ClienteNombre, venta.ClienteDocumento, venta.ClienteDireccion,
		venta.Fecha.Format("2006-01-02"), venta.Moneda, venta.Subtotal, venta.Descuento,
		venta.Impuesto, venta.Total, venta.Observaciones, formatTime(now), id,
	); err != nil {
		return domain.Venta{}, err
	}
	if _, err := tx.ExecContext(ctx, s.db.rebind(`DELETE FROM venta_items WHERE venta_id = ?`), id); err != nil {
		return domain.Venta{}, err
	}
	if err := s.insertItems(ctx, tx, id, venta.Items); err != nil {
		return domain.Venta{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Venta{}, err
	}
	return s.getVenta(id)
}

func (s *VentaStore) Delete(id int64) error {
	ctx := context.Background()
	var estado string
	if err := s.db.QueryRowContext(ctx, s.db.rebind(`SELECT estado FROM ventas WHERE id = ?`), id).Scan(&estado); err != nil {
		return err
	}
	if estado != "borrador" {
		return errors.New("solo se puede eliminar una venta en borrador")
	}
	res, err := s.db.ExecContext(ctx, s.db.rebind(`DELETE FROM ventas WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *VentaStore) FindByID(id int64) (*domain.Venta, error) {
	ctx := context.Background()
	q := s.db.rebind(`SELECT ` + ventaColumns + ` FROM ventas WHERE id = ?`)
	venta, err := scanVenta(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, err
	}
	items, err := s.loadItems(ctx, id)
	if err != nil {
		return nil, err
	}
	venta.Items = items
	return venta, nil
}

func (s *VentaStore) loadItems(ctx context.Context, ventaID int64) ([]domain.VentaItem, error) {
	q := s.db.rebind(`SELECT ` + ventaItemColumns + ` FROM venta_items WHERE venta_id = ? ORDER BY id`)
	rows, err := s.db.QueryContext(ctx, q, ventaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.VentaItem, 0)
	for rows.Next() {
		var (
			it         domain.VentaItem
			productoID sql.NullInt64
		)
		if err := rows.Scan(&it.ID, &it.VentaID, &productoID, &it.Descripcion, &it.Cantidad,
			&it.PrecioUnitario, &it.Descuento, &it.ImpuestoPorcentaje, &it.Subtotal,
			&it.Impuesto, &it.Total); err != nil {
			return nil, err
		}
		if productoID.Valid {
			v := productoID.Int64
			it.ProductoID = &v
		}
		items = append(items, it)
	}
	return items, nil
}

func ventaWhere(f store.VentaFilters) (string, []any) {
	var conds []string
	var args []any
	if f.Q != "" {
		like := "%" + strings.ToLower(f.Q) + "%"
		conds = append(conds, `(LOWER(cliente_nombre) LIKE ? OR LOWER(COALESCE(codigo,'')) LIKE ? OR LOWER(cliente_documento) LIKE ?)`)
		args = append(args, like, like, like)
	}
	if f.Serie != "" {
		conds = append(conds, `serie = ?`)
		args = append(args, strings.ToUpper(f.Serie))
	}
	if f.Estado != "" {
		conds = append(conds, `estado = ?`)
		args = append(args, f.Estado)
	}
	if f.Desde != "" {
		conds = append(conds, `fecha >= ?`)
		args = append(args, f.Desde)
	}
	if f.Hasta != "" {
		conds = append(conds, `fecha <= ?`)
		args = append(args, f.Hasta)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (s *VentaStore) Count(filters store.VentaFilters) int {
	where, args := ventaWhere(filters)
	var total int
	if err := s.db.QueryRowContext(context.Background(), s.db.rebind(`SELECT COUNT(*) FROM ventas`+where), args...).Scan(&total); err != nil {
		return 0
	}
	return total
}

func (s *VentaStore) List(offset, limit int, filters store.VentaFilters) ([]domain.Venta, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}

	ctx := context.Background()
	where, args := ventaWhere(filters)

	var total int
	if err := s.db.QueryRowContext(ctx, s.db.rebind(`SELECT COUNT(*) FROM ventas`+where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := s.db.rebind(`SELECT ` + ventaColumns + ` FROM ventas` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
	rows, err := s.db.QueryContext(ctx, q, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	ventas := make([]domain.Venta, 0)
	for rows.Next() {
		v, err := scanVenta(rows)
		if err != nil {
			return nil, 0, err
		}
		items, err := s.loadItems(ctx, v.ID)
		if err != nil {
			return nil, 0, err
		}
		v.Items = items
		ventas = append(ventas, *v)
	}
	return ventas, total, nil
}

func (s *VentaStore) Emit(id int64) (domain.Venta, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Venta{}, err
	}
	defer tx.Rollback()

	var (
		estado string
		serie  string
		fecha  string
	)
	if err := tx.QueryRowContext(ctx, s.db.rebind(`SELECT estado, serie, fecha FROM ventas WHERE id = ?`), id).Scan(&estado, &serie, &fecha); err != nil {
		return domain.Venta{}, err
	}
	if estado != "borrador" {
		return domain.Venta{}, errors.New("la venta ya fue emitida o anulada")
	}
	if serie == "" {
		serie = "FAC"
	}
	anio := time.Now().Year()
	if t, err := time.Parse("2006-01-02", fecha); err == nil {
		anio = t.Year()
	}

	var correlativo int64
	q := s.db.rebind(`INSERT INTO venta_secuencias (serie, anio, ultimo) VALUES (?, ?, 1)
		ON CONFLICT (serie, anio) DO UPDATE SET ultimo = venta_secuencias.ultimo + 1
		RETURNING ultimo`)
	if err := tx.QueryRowContext(ctx, q, serie, anio).Scan(&correlativo); err != nil {
		return domain.Venta{}, err
	}
	codigo := fmt.Sprintf("%s-%d-%06d", serie, anio, correlativo)

	now := time.Now()
	if _, err := tx.ExecContext(ctx, s.db.rebind(`UPDATE ventas SET serie = ?, correlativo = ?, codigo = ?,
		estado = 'emitido', actualizado_en = ? WHERE id = ?`), serie, correlativo, codigo, formatTime(now), id); err != nil {
		return domain.Venta{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Venta{}, err
	}
	return s.getVenta(id)
}

func (s *VentaStore) Annul(id int64, motivo string, userID int64) (domain.Venta, error) {
	ctx := context.Background()
	now := time.Now()
	var res sql.Result
	var err error
	if motivo != "" {
		res, err = s.db.ExecContext(ctx, s.db.rebind(`UPDATE ventas SET estado = 'anulado',
			observaciones = ?, actualizado_en = ? WHERE id = ? AND estado = 'emitido'`),
			motivo, formatTime(now), id)
	} else {
		res, err = s.db.ExecContext(ctx, s.db.rebind(`UPDATE ventas SET estado = 'anulado',
			actualizado_en = ? WHERE id = ? AND estado = 'emitido'`), formatTime(now), id)
	}
	if err != nil {
		return domain.Venta{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.Venta{}, errors.New("solo se puede anular una venta emitida")
	}
	return s.getVenta(id)
}

func scanVenta(sc rowScanner) (*domain.Venta, error) {
	var (
		v           domain.Venta
		codigo      sql.NullString
		fecha       string
		creado      string
		actualizado string
	)
	if err := sc.Scan(&v.ID, &v.Serie, &v.Correlativo, &codigo, &v.ClienteNombre,
		&v.ClienteDocumento, &v.ClienteDireccion, &fecha, &v.Moneda, &v.Subtotal,
		&v.Descuento, &v.Impuesto, &v.Total, &v.Estado, &v.Observaciones,
		&v.UserID, &creado, &actualizado); err != nil {
		return nil, err
	}
	if codigo.Valid {
		v.Codigo = codigo.String
	}
	v.Fecha = parseTime(fecha)
	v.CreadoEn = parseTime(creado)
	v.ActualizadoEn = parseTime(actualizado)
	v.Items = []domain.VentaItem{}
	return &v, nil
}

func (s *VentaStore) getVenta(id int64) (domain.Venta, error) {
	v, err := s.FindByID(id)
	if err != nil {
		return domain.Venta{}, err
	}
	return *v, nil
}
