package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"back/domain"
	"back/store"
)

type DocumentoStore struct {
	db *DB
}

func NewDocumentoStore(db *DB) *DocumentoStore {
	return &DocumentoStore{db: db}
}

const documentoColumns = `id, tipo, serie, anio, correlativo, codigo, politica, estado, plantilla,
	plantilla_version, parametros_hash, parametros_json, snapshot_json, version, user_id,
	creado_en, actualizado_en, emitido_en, anulado_en, motivo_anulacion, origen_id`

const documentoArchivoColumns = `id, documento_id, formato, file_name, mime, tamano, version, creado_en`

const documentoEventoColumns = `id, documento_id, accion, detalle, user_id, creado_en`

func (s *DocumentoStore) FindByHash(tipo, hash string) (*domain.Documento, error) {
	ctx := context.Background()
	q := s.db.rebind(`SELECT ` + documentoColumns + ` FROM documentos WHERE tipo = ? AND parametros_hash = ?`)
	doc, err := scanDocumento(s.db.QueryRowContext(ctx, q, tipo, hash))
	if err != nil {
		return nil, err
	}
	if err := s.loadDetails(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentoStore) FindByID(id int64) (*domain.Documento, error) {
	ctx := context.Background()
	q := s.db.rebind(`SELECT ` + documentoColumns + ` FROM documentos WHERE id = ?`)
	doc, err := scanDocumento(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, err
	}
	if err := s.loadDetails(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *DocumentoStore) loadDetails(ctx context.Context, doc *domain.Documento) error {
	archivos, err := s.loadArchivos(ctx, doc.ID)
	if err != nil {
		return err
	}
	eventos, err := s.loadEventos(ctx, doc.ID)
	if err != nil {
		return err
	}
	doc.Archivos = archivos
	doc.Eventos = eventos
	return nil
}

func (s *DocumentoStore) loadArchivos(ctx context.Context, docID int64) ([]domain.DocumentoArchivo, error) {
	q := s.db.rebind(`SELECT ` + documentoArchivoColumns + ` FROM documento_archivos WHERE documento_id = ? ORDER BY id`)
	rows, err := s.db.QueryContext(ctx, q, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.DocumentoArchivo, 0)
	for rows.Next() {
		var a domain.DocumentoArchivo
		var creado string
		if err := rows.Scan(&a.ID, &a.DocumentoID, &a.Formato, &a.FileName, &a.Mime,
			&a.Tamano, &a.Version, &creado); err != nil {
			return nil, err
		}
		a.CreadoEn = parseTime(creado)
		out = append(out, a)
	}
	return out, nil
}

func (s *DocumentoStore) loadEventos(ctx context.Context, docID int64) ([]domain.DocumentoEvento, error) {
	q := s.db.rebind(`SELECT ` + documentoEventoColumns + ` FROM documento_eventos WHERE documento_id = ? ORDER BY id`)
	rows, err := s.db.QueryContext(ctx, q, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.DocumentoEvento, 0)
	for rows.Next() {
		var e domain.DocumentoEvento
		var creado string
		if err := rows.Scan(&e.ID, &e.DocumentoID, &e.Accion, &e.Detalle, &e.UserID, &creado); err != nil {
			return nil, err
		}
		e.CreadoEn = parseTime(creado)
		out = append(out, e)
	}
	return out, nil
}

func (s *DocumentoStore) CreateWithCorrelativo(input store.DocumentoCreateInput, build func(codigo string) ([]domain.DocumentoArchivo, error)) (domain.Documento, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Documento{}, err
	}
	defer tx.Rollback()

	anio := input.Anio
	if anio == 0 {
		anio = time.Now().Year()
	}
	var correlativo int64
	qSeq := s.db.rebind(`INSERT INTO documento_secuencias (tipo, serie, anio, ultimo) VALUES (?, ?, ?, 1)
		ON CONFLICT (tipo, serie, anio) DO UPDATE SET ultimo = documento_secuencias.ultimo + 1
		RETURNING ultimo`)
	if err := tx.QueryRowContext(ctx, qSeq, input.Tipo, input.Serie, anio).Scan(&correlativo); err != nil {
		return domain.Documento{}, err
	}
	codigo := fmt.Sprintf("%s-%d-%06d", input.Serie, anio, correlativo)

	archivos, err := build(codigo)
	if err != nil {
		return domain.Documento{}, err
	}

	now := time.Now()
	estado := input.Estado
	if estado == "" {
		estado = "emitido"
	}
	politica := input.Politica
	if politica == "" {
		politica = "inmutable"
	}

	q := s.db.rebind(`INSERT INTO documentos
		(tipo, serie, anio, correlativo, codigo, politica, estado, plantilla, plantilla_version,
		 parametros_hash, parametros_json, snapshot_json, version, user_id, creado_en, actualizado_en, emitido_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?) RETURNING id`)
	var id int64
	if err := tx.QueryRowContext(ctx, q, input.Tipo, input.Serie, anio, correlativo, codigo,
		politica, estado, input.Plantilla, input.PlantillaVersion, input.ParametrosHash,
		input.ParametrosJSON, input.SnapshotJSON, input.UserID, formatTime(now), formatTime(now), formatTime(now),
	).Scan(&id); err != nil {
		return domain.Documento{}, err
	}

	if err := insertArchivos(ctx, tx, s.db, id, 1, archivos, now); err != nil {
		return domain.Documento{}, err
	}
	if err := insertEvento(ctx, tx, s.db, id, "creado", "Documento generado", input.UserID, now); err != nil {
		return domain.Documento{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Documento{}, err
	}
	doc, err := s.FindByID(id)
	if err != nil {
		return domain.Documento{}, err
	}
	return *doc, nil
}

func (s *DocumentoStore) AddVersion(id int64, snapshotJSON string, build func(codigo string, version int) ([]domain.DocumentoArchivo, error)) (domain.Documento, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Documento{}, err
	}
	defer tx.Rollback()

	var codigo string
	var version int
	if err := tx.QueryRowContext(ctx, s.db.rebind(`SELECT codigo, version FROM documentos WHERE id = ?`), id).Scan(&codigo, &version); err != nil {
		return domain.Documento{}, err
	}
	newVersion := version + 1
	archivos, err := build(codigo, newVersion)
	if err != nil {
		return domain.Documento{}, err
	}

	now := time.Now()
	if _, err := tx.ExecContext(ctx, s.db.rebind(`UPDATE documentos SET version = ?, snapshot_json = ?, actualizado_en = ? WHERE id = ?`),
		newVersion, snapshotJSON, formatTime(now), id); err != nil {
		return domain.Documento{}, err
	}
	if err := insertArchivos(ctx, tx, s.db, id, newVersion, archivos, now); err != nil {
		return domain.Documento{}, err
	}
	if err := insertEvento(ctx, tx, s.db, id, "regenerado", fmt.Sprintf("Version %d", newVersion), 0, now); err != nil {
		return domain.Documento{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Documento{}, err
	}
	doc, err := s.FindByID(id)
	if err != nil {
		return domain.Documento{}, err
	}
	return *doc, nil
}

func (s *DocumentoStore) Annul(id int64, motivo string, userID int64) (domain.Documento, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Documento{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	res, err := tx.ExecContext(ctx, s.db.rebind(`UPDATE documentos SET estado = 'anulado', anulado_en = ?,
		motivo_anulacion = ?, actualizado_en = ? WHERE id = ? AND estado <> 'anulado'`),
		formatTime(now), motivo, formatTime(now), id)
	if err != nil {
		return domain.Documento{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.Documento{}, sql.ErrNoRows
	}
	if err := insertEvento(ctx, tx, s.db, id, "anulado", motivo, userID, now); err != nil {
		return domain.Documento{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Documento{}, err
	}
	doc, err := s.FindByID(id)
	if err != nil {
		return domain.Documento{}, err
	}
	return *doc, nil
}

func (s *DocumentoStore) List(offset, limit int, filters store.DocumentoFilters) ([]domain.Documento, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}
	ctx := context.Background()

	var conds []string
	var args []any
	if filters.Q != "" {
		like := "%" + strings.ToLower(filters.Q) + "%"
		conds = append(conds, `(LOWER(codigo) LIKE ? OR LOWER(parametros_json) LIKE ?)`)
		args = append(args, like, like)
	}
	if filters.Tipo != "" {
		conds = append(conds, `tipo = ?`)
		args = append(args, filters.Tipo)
	}
	if filters.Serie != "" {
		conds = append(conds, `serie = ?`)
		args = append(args, filters.Serie)
	}
	if filters.Estado != "" {
		conds = append(conds, `estado = ?`)
		args = append(args, filters.Estado)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, s.db.rebind(`SELECT COUNT(*) FROM documentos`+where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := s.db.rebind(`SELECT ` + documentoColumns + ` FROM documentos` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
	rows, err := s.db.QueryContext(ctx, q, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Documento, 0)
	for rows.Next() {
		doc, err := scanDocumento(rows)
		if err != nil {
			return nil, 0, err
		}
		if err := s.loadDetails(ctx, doc); err != nil {
			return nil, 0, err
		}
		out = append(out, *doc)
	}
	return out, total, nil
}

func insertArchivos(ctx context.Context, tx *sql.Tx, db *DB, docID int64, version int, archivos []domain.DocumentoArchivo, now time.Time) error {
	q := db.rebind(`INSERT INTO documento_archivos (documento_id, formato, file_name, mime, tamano, version, creado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	for _, a := range archivos {
		if _, err := tx.ExecContext(ctx, q, docID, a.Formato, a.FileName, a.Mime, a.Tamano, version, formatTime(now)); err != nil {
			return err
		}
	}
	return nil
}

func insertEvento(ctx context.Context, tx *sql.Tx, db *DB, docID int64, accion, detalle string, userID int64, now time.Time) error {
	q := db.rebind(`INSERT INTO documento_eventos (documento_id, accion, detalle, user_id, creado_en)
		VALUES (?, ?, ?, ?, ?)`)
	_, err := tx.ExecContext(ctx, q, docID, accion, detalle, userID, formatTime(now))
	return err
}

func scanDocumento(sc rowScanner) (*domain.Documento, error) {
	var (
		d           domain.Documento
		creado      string
		actualizado string
		emitido     string
		anulado     sql.NullString
		motivo      sql.NullString
		origen      sql.NullInt64
	)
	if err := sc.Scan(&d.ID, &d.Tipo, &d.Serie, &d.Anio, &d.Correlativo, &d.Codigo,
		&d.Politica, &d.Estado, &d.Plantilla, &d.PlantillaVersion, &d.ParametrosHash,
		&d.ParametrosJSON, &d.SnapshotJSON, &d.Version, &d.UserID, &creado,
		&actualizado, &emitido, &anulado, &motivo, &origen); err != nil {
		return nil, err
	}
	d.CreadoEn = parseTime(creado)
	d.ActualizadoEn = parseTime(actualizado)
	d.EmitidoEn = parseTime(emitido)
	if anulado.Valid {
		t := parseTime(anulado.String)
		d.AnuladoEn = &t
	}
	if motivo.Valid {
		d.MotivoAnulacion = motivo.String
	}
	if origen.Valid {
		v := origen.Int64
		d.OrigenID = &v
	}
	return &d, nil
}
