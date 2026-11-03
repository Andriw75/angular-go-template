package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"back/store"
)

// LabStore ejecuta consultas largas acotadas contra PostgreSQL. Las tareas
// asincronas viven en memoria y se notifican por SSE; el modo sync se ejecuta
// dentro de la peticion HTTP y se cancela con su contexto.
type LabStore struct {
	db         *DB
	notifier   store.SlowQueryNotifier
	mu         sync.Mutex
	queries    map[string]*labQuery
	ttl        time.Duration
	maxActive  int
	maxPerUser int
	maxSecs    int
	maxRows    int64
}

type labQuery struct {
	info      store.SlowQueryInfo
	ownerID   int64
	cancel    context.CancelFunc
	done      chan struct{}
	expiraEn  time.Time
	cancelado bool
	created   bool
}

func NewLabStore(db *DB, notifier store.SlowQueryNotifier, maxActive, maxPerUser, maxSeconds int, maxRows int64) *LabStore {
	if maxActive < 1 {
		maxActive = 10
	}
	if maxPerUser < 1 {
		maxPerUser = 5
	}
	if maxSeconds < 1 {
		maxSeconds = 600
	}
	if maxRows < 1 {
		maxRows = 2_000_000
	}
	return &LabStore{
		db:         db,
		notifier:   notifier,
		queries:    make(map[string]*labQuery),
		ttl:        5 * time.Minute,
		maxActive:  maxActive,
		maxPerUser: maxPerUser,
		maxSecs:    maxSeconds,
		maxRows:    maxRows,
	}
}

// Start crea una tarea en segundo plano.
func (s *LabStore) Start(ctx context.Context, ownerID int64, input store.SlowQueryInput) (store.SlowQueryInfo, error) {
	input, err := s.validate(input)
	if err != nil {
		return store.SlowQueryInfo{}, err
	}

	s.mu.Lock()
	s.cleanupLocked()
	active, userActive := 0, 0
	for _, q := range s.queries {
		if q.isActive() {
			active++
			if q.ownerID == ownerID {
				userActive++
			}
		}
	}
	if active >= s.maxActive || userActive >= s.maxPerUser {
		s.mu.Unlock()
		return store.SlowQueryInfo{}, store.ErrSlowQueryLimit
	}

	queryCtx, cancel := context.WithCancel(context.Background())
	q := &labQuery{
		ownerID: ownerID,
		cancel:  cancel,
		done:    make(chan struct{}),
		info: store.SlowQueryInfo{
			ID:               newLabID(),
			OwnerID:          ownerID,
			Modo:             input.Modo,
			Estado:           store.SlowQueryEjecutando,
			Segundos:         input.Segundos,
			FilasSolicitadas: input.Filas,
			IniciadoEn:       time.Now(),
		},
	}
	s.queries[q.info.ID] = q
	s.mu.Unlock()

	fail := func(err error) (store.SlowQueryInfo, error) {
		cancel()
		s.finish(q, err, 0)
		return store.SlowQueryInfo{}, err
	}

	conn, pid, err := s.prepareConn(input)
	if err != nil {
		return fail(err)
	}

	s.mu.Lock()
	q.info.PID = pid
	q.created = true
	info := q.info
	s.mu.Unlock()
	s.notify("create", info)

	go func() {
		defer close(q.done)
		defer conn.Close()

		filas, err := s.runQuery(queryCtx, conn, input)
		s.resetConn(conn)
		s.finish(q, err, filas)
	}()

	return info, nil
}

// RunSync ejecuta la consulta dentro de la peticion HTTP. Si el cliente corta
// la peticion, el contexto se cancela y se aborta la consulta en PostgreSQL.
func (s *LabStore) RunSync(ctx context.Context, ownerID int64, input store.SlowQueryInput) (store.SlowQueryInfo, error) {
	input, err := s.validate(input)
	if err != nil {
		return store.SlowQueryInfo{}, err
	}

	start := time.Now()
	conn, pid, err := s.prepareConn(input)
	if err != nil {
		return store.SlowQueryInfo{}, err
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			s.cancelBackend(pid)
		case <-done:
		}
	}()

	filas, runErr := s.runQuery(ctx, conn, input)
	s.resetConn(conn)

	now := time.Now()
	info := store.SlowQueryInfo{
		ID:               newLabID(),
		OwnerID:          ownerID,
		Modo:             input.Modo,
		PID:              pid,
		Segundos:         input.Segundos,
		FilasSolicitadas: input.Filas,
		FilasProcesadas:  filas,
		IniciadoEn:       start,
		FinalizadoEn:     &now,
	}
	switch {
	case ctx.Err() != nil:
		info.Estado = store.SlowQueryCancelada
	case runErr != nil:
		info.Estado = store.SlowQueryError
		info.Error = runErr.Error()
	default:
		info.Estado = store.SlowQueryCompletada
	}
	return info, nil
}

func (s *LabStore) List() []store.SlowQueryInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	res := make([]store.SlowQueryInfo, 0, len(s.queries))
	for _, q := range s.queries {
		res = append(res, q.info)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].IniciadoEn.After(res[j].IniciadoEn)
	})
	return res
}

func (s *LabStore) Cancel(ctx context.Context, id string) (store.SlowQueryInfo, error) {
	s.mu.Lock()
	s.cleanupLocked()
	q, ok := s.queries[id]
	if !ok {
		s.mu.Unlock()
		return store.SlowQueryInfo{}, store.ErrSlowQueryNotFound
	}
	if !q.isActive() {
		info := q.info
		s.mu.Unlock()
		return info, nil
	}

	q.cancelado = true
	q.info.Estado = store.SlowQueryCancelando
	pid := q.info.PID
	done := q.done
	cancel := q.cancel
	info := q.info
	s.mu.Unlock()

	if q.created {
		s.notify("update", info)
	}
	s.cancelInternal(ctx, pid, cancel, done)

	s.mu.Lock()
	info = q.info
	s.mu.Unlock()
	if q.created {
		s.notify("update", info)
	}
	return info, nil
}

func (s *LabStore) CancelAll(ctx context.Context) {
	type pending struct {
		pid    int
		cancel context.CancelFunc
		done   <-chan struct{}
	}

	s.mu.Lock()
	active := make([]pending, 0, len(s.queries))
	for _, q := range s.queries {
		if q.isActive() {
			q.cancelado = true
			q.info.Estado = store.SlowQueryCancelando
			active = append(active, pending{pid: q.info.PID, cancel: q.cancel, done: q.done})
		}
	}
	s.mu.Unlock()

	for _, p := range active {
		s.cancelInternal(ctx, p.pid, p.cancel, p.done)
	}
}

func (s *LabStore) validate(input store.SlowQueryInput) (store.SlowQueryInput, error) {
	if s.db.Type() != "postgres" {
		return input, store.ErrSlowQueryUnsupported
	}
	switch input.Modo {
	case store.SlowQuerySleep, store.SlowQueryCPU:
	default:
		return input, store.ErrSlowQueryInvalid
	}

	input.Segundos = clampInt(input.Segundos, 1, s.maxSecs)
	if input.Filas <= 0 {
		input.Filas = 100_000
	}
	if input.Filas > s.maxRows {
		input.Filas = s.maxRows
	}
	return input, nil
}

func (s *LabStore) prepareConn(input store.SlowQueryInput) (*sql.Conn, int, error) {
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer setupCancel()

	conn, err := s.db.Conn(setupCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("no se pudo reservar conexion: %w", err)
	}

	timeoutMS := 60_000
	if input.Modo == store.SlowQuerySleep {
		timeoutMS = (input.Segundos + 30) * 1000
	}

	setup := []string{
		`SET application_name = 'template-lab'`,
		`SET default_transaction_read_only = on`,
		`SET work_mem = '16MB'`,
		fmt.Sprintf(`SET statement_timeout = %d`, timeoutMS),
	}
	for _, stmt := range setup {
		if _, err := conn.ExecContext(setupCtx, stmt); err != nil {
			conn.Close()
			return nil, 0, fmt.Errorf("no se pudo configurar sesion: %w", err)
		}
	}

	var pid int
	if err := conn.QueryRowContext(setupCtx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		conn.Close()
		return nil, 0, fmt.Errorf("no se pudo obtener pid: %w", err)
	}
	return conn, pid, nil
}

func (s *LabStore) runQuery(ctx context.Context, conn *sql.Conn, input store.SlowQueryInput) (int64, error) {
	var filas int64
	var err error
	switch input.Modo {
	case store.SlowQuerySleep:
		err = conn.QueryRowContext(ctx, s.db.rebind(
			`WITH espera AS MATERIALIZED (SELECT pg_sleep(?::double precision))
			 SELECT (SELECT count(*)::bigint FROM generate_series(1::bigint, ?::bigint)) AS filas
			 FROM espera`), float64(input.Segundos), input.Filas).Scan(&filas)
	case store.SlowQueryCPU:
		err = conn.QueryRowContext(ctx, s.db.rebind(
			`SELECT count(*)::bigint
			 FROM generate_series(1::bigint, ?::bigint) g
			 WHERE md5(g::text) LIKE '%0%'`), input.Filas).Scan(&filas)
	}
	return filas, err
}

func (s *LabStore) resetConn(conn *sql.Conn) {
	resetCtx, resetCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer resetCancel()
	for _, stmt := range []string{
		`SET default_transaction_read_only = off`,
		`SET statement_timeout = DEFAULT`,
		`SET work_mem = DEFAULT`,
		`SET application_name = DEFAULT`,
	} {
		if _, err := conn.ExecContext(resetCtx, stmt); err != nil {
			break
		}
	}
}

func (s *LabStore) cancelInternal(ctx context.Context, pid int, cancel context.CancelFunc, done <-chan struct{}) {
	s.cancelBackend(pid)
	cancel()

	select {
	case <-done:
		return
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
	}

	// Respaldo: si la consulta no termino, cerrar el backend en el servidor.
	slog.Warn("la consulta no termino tras cancelar; forzando terminate", "pid", pid)
	s.terminateBackend(pid)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (s *LabStore) cancelBackend(pid int) {
	if pid <= 0 {
		return
	}
	cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ok bool
	if err := s.db.QueryRowContext(cctx, s.db.rebind(`SELECT pg_cancel_backend(?::int4)`), int32(pid)).Scan(&ok); err != nil {
		slog.Warn("no se pudo cancelar la consulta en postgres", "pid", pid, "error", err)
	}
}

func (s *LabStore) terminateBackend(pid int) {
	if pid <= 0 {
		return
	}
	tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ok bool
	if err := s.db.QueryRowContext(tctx, s.db.rebind(`SELECT pg_terminate_backend(?::int4)`), int32(pid)).Scan(&ok); err != nil {
		slog.Warn("no se pudo terminar el backend en postgres", "pid", pid, "error", err)
	}
}

func (s *LabStore) finish(q *labQuery, err error, filas int64) {
	s.mu.Lock()
	now := time.Now()
	q.info.FinalizadoEn = &now
	q.info.FilasProcesadas = filas
	q.expiraEn = now.Add(s.ttl)

	switch {
	case q.cancelado:
		q.info.Estado = store.SlowQueryCancelada
	case err == nil:
		q.info.Estado = store.SlowQueryCompletada
	default:
		q.info.Estado = store.SlowQueryError
		q.info.Error = err.Error()
	}
	info := q.info
	created := q.created
	s.mu.Unlock()

	if created {
		s.notify("update", info)
	}
}

func (s *LabStore) notify(event string, info store.SlowQueryInfo) {
	if s.notifier != nil {
		s.notifier.Notify(event, info)
	}
}

func (s *LabStore) cleanupLocked() {
	now := time.Now()
	for id, q := range s.queries {
		if !q.isActive() && !q.expiraEn.IsZero() && now.After(q.expiraEn) {
			delete(s.queries, id)
		}
	}
}

func (q *labQuery) isActive() bool {
	return q.info.Estado == store.SlowQueryEjecutando || q.info.Estado == store.SlowQueryCancelando
}

func newLabID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
