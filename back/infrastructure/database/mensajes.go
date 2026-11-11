package database

import (
	"context"
	"database/sql"
	"time"

	"back/domain"
	"back/domain/inputs"
)

type MensajeStore struct {
	db *DB
}

func NewMensajeStore(db *DB) *MensajeStore {
	return &MensajeStore{db: db}
}

const mensajeColumns = `id, telefono, hora_solicitada, hora_desactivacion, usuario_acargo,
	hora_usuario_asignado, estado, creado_en, actualizado_en`

func (s *MensajeStore) Create(ctx context.Context, input inputs.MensajeInput, now time.Time) (domain.MensajePendiente, error) {
	horaDesac, err := time.Parse(time.RFC3339, input.HoraDesactivacion)
	if err != nil {
		horaDesac, _ = time.Parse("2006-01-02T15:04", input.HoraDesactivacion)
	}

	m := domain.MensajePendiente{
		Telefono:          input.Telefono,
		HoraSolicitada:    now,
		HoraDesactivacion: horaDesac,
		Estado:            "pendiente",
		CreadoEn:          now,
		ActualizadoEn:     now,
	}

	q := s.db.rebind(`INSERT INTO mensajes_pendientes
		(telefono, hora_solicitada, hora_desactivacion, usuario_acargo, hora_usuario_asignado, estado, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	err = s.db.QueryRowContext(ctx, q,
		m.Telefono, formatTime(m.HoraSolicitada), formatTime(m.HoraDesactivacion),
		nil, nil, m.Estado, formatTime(now), formatTime(now),
	).Scan(&m.ID)
	if err != nil {
		return domain.MensajePendiente{}, err
	}
	return m, nil
}

func (s *MensajeStore) Update(ctx context.Context, id int64, input inputs.MensajeUpdateInput, now time.Time, username string) (domain.MensajePendiente, error) {
	existing, err := s.FindByID(ctx, id)
	if err != nil {
		return domain.MensajePendiente{}, err
	}

	if input.Finalizar != nil && *input.Finalizar {
		existing.HoraDesactivacion = now
		existing.Estado = "finalizado"
	}

	if input.UsuarioAcargo != nil && *input.UsuarioAcargo != "" {
		u := *input.UsuarioAcargo
		existing.UsuarioAcargo = &u
		t := now
		existing.HoraUsuarioAsignado = &t
		if existing.Estado == "pendiente" {
			existing.Estado = "asignado"
		}
	}

	oldUpdated := existing.ActualizadoEn
	existing.ActualizadoEn = now

	q := s.db.rebind(`UPDATE mensajes_pendientes SET
		hora_desactivacion = ?, usuario_acargo = ?, hora_usuario_asignado = ?, estado = ?, actualizado_en = ?
		WHERE id = ? AND actualizado_en = ?`)
	res, err := s.db.ExecContext(ctx, q,
		formatTime(existing.HoraDesactivacion), strPtr(existing.UsuarioAcargo),
		formatTimePtr(existing.HoraUsuarioAsignado), existing.Estado, formatTime(now), id, formatTime(oldUpdated),
	)
	if err != nil {
		return domain.MensajePendiente{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.MensajePendiente{}, domain.Conflict("conflict", "el mensaje fue modificado por otra sesión; recarga e intenta de nuevo")
	}
	return *existing, nil
}

func (s *MensajeStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, s.db.rebind(`DELETE FROM mensajes_pendientes WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("mensaje no encontrado")
	}
	return nil
}

func (s *MensajeStore) FindByID(ctx context.Context, id int64) (*domain.MensajePendiente, error) {
	q := s.db.rebind(`SELECT ` + mensajeColumns + ` FROM mensajes_pendientes WHERE id = ?`)
	return scanMensaje(s.db.QueryRowContext(ctx, q, id))
}

func (s *MensajeStore) FindAll(ctx context.Context) []domain.MensajePendiente {
	rows, err := s.db.QueryContext(ctx, `SELECT `+mensajeColumns+` FROM mensajes_pendientes ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []domain.MensajePendiente
	for rows.Next() {
		m, err := scanMensaje(rows)
		if err != nil {
			return out
		}
		out = append(out, *m)
	}
	return out
}

func (s *MensajeStore) ListPasadas(ctx context.Context, offset, limit int) ([]domain.MensajePendiente, bool) {
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 10
	}

	q := s.db.rebind(`SELECT ` + mensajeColumns + ` FROM mensajes_pendientes
		WHERE estado = 'finalizado' ORDER BY hora_solicitada DESC LIMIT ? OFFSET ?`)
	rows, err := s.db.QueryContext(ctx, q, limit+1, offset)
	if err != nil {
		return []domain.MensajePendiente{}, false
	}
	defer rows.Close()

	var out []domain.MensajePendiente
	hasMore := false
	for rows.Next() {
		if len(out) == limit {
			hasMore = true
			break
		}
		m, err := scanMensaje(rows)
		if err != nil {
			break
		}
		out = append(out, *m)
	}
	return out, hasMore
}

func (s *MensajeStore) CountPasadas(ctx context.Context) int {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mensajes_pendientes WHERE estado = 'finalizado'`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func scanMensaje(sc rowScanner) (*domain.MensajePendiente, error) {
	var (
		m            domain.MensajePendiente
		horaSol      string
		horaDesac    string
		usuario      sql.NullString
		horaAsignado sql.NullString
		creado       string
		actualizado  string
	)
	if err := sc.Scan(&m.ID, &m.Telefono, &horaSol, &horaDesac, &usuario,
		&horaAsignado, &m.Estado, &creado, &actualizado); err != nil {
		return nil, err
	}
	m.HoraSolicitada = parseTime(horaSol)
	m.HoraDesactivacion = parseTime(horaDesac)
	if usuario.Valid {
		u := usuario.String
		m.UsuarioAcargo = &u
	}
	if horaAsignado.Valid {
		t := parseTime(horaAsignado.String)
		m.HoraUsuarioAsignado = &t
	}
	m.CreadoEn = parseTime(creado)
	m.ActualizadoEn = parseTime(actualizado)
	return &m, nil
}
