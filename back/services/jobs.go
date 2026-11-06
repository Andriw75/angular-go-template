package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"back/domain"
)

// Job representa una tarea asíncrona de generación de documentos. Sus campos
// mutables se leen y escriben bajo mu para evitar carreras entre la goroutine
// que genera y los handlers que consultan el progreso.
type Job struct {
	mu           sync.RWMutex
	ID           string
	Tipo         string
	Estado       string
	Progreso     int
	Error        string
	Documento    *domain.Documento
	Exists       bool
	CreadoEn     time.Time
	Actualizado  time.Time
	ExpiraEn     time.Time
	Done         chan struct{}
	finishedOnce sync.Once
}

// JobSnapshot es una copia inmutable del estado de un Job.
type JobSnapshot struct {
	ID          string
	Tipo        string
	Estado      string
	Progreso    int
	Error       string
	Documento   *domain.Documento
	Exists      bool
	CreadoEn    time.Time
	Actualizado time.Time
	ExpiraEn    time.Time
}

func (j *Job) Snapshot() JobSnapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return JobSnapshot{
		ID:          j.ID,
		Tipo:        j.Tipo,
		Estado:      j.Estado,
		Progreso:    j.Progreso,
		Error:       j.Error,
		Documento:   j.Documento,
		Exists:      j.Exists,
		CreadoEn:    j.CreadoEn,
		Actualizado: j.Actualizado,
		ExpiraEn:    j.ExpiraEn,
	}
}

func (j *Job) Wait() <-chan struct{} { return j.Done }

// SetProgress actualiza el porcentaje sin tocar el estado final.
func (j *Job) SetProgress(progreso int) {
	j.mu.Lock()
	j.Progreso = progreso
	j.Actualizado = time.Now()
	j.mu.Unlock()
}

func (j *Job) fail(msg string) {
	j.mu.Lock()
	j.Estado = "error"
	j.Error = msg
	j.Actualizado = time.Now()
	j.mu.Unlock()
}

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	ttl  time.Duration
}

func NewJobStore(ttl time.Duration) *JobStore {
	s := &JobStore{jobs: make(map[string]*Job), ttl: ttl}
	go s.cleanupLoop()
	return s
}

func newJobID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

func (s *JobStore) Create(tipo string, run func(job *Job)) *Job {
	s.cleanup()
	now := time.Now()
	job := &Job{
		ID:          newJobID(),
		Tipo:        tipo,
		Estado:      "pendiente",
		CreadoEn:    now,
		Actualizado: now,
		ExpiraEn:    now.Add(s.ttl),
		Done:        make(chan struct{}),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				job.fail("panic en generacion")
			}
			job.finishedOnce.Do(func() { close(job.Done) })
		}()

		job.mu.Lock()
		job.Estado = "generando"
		job.Progreso = 5
		job.Actualizado = time.Now()
		job.mu.Unlock()

		run(job)
	}()

	return job
}

func (s *JobStore) Update(job *Job, estado string, progreso int, errMsg string, doc *domain.Documento, exists bool) {
	job.mu.Lock()
	job.Estado = estado
	job.Progreso = progreso
	job.Error = errMsg
	job.Documento = doc
	job.Exists = exists
	job.Actualizado = time.Now()
	job.mu.Unlock()

	if estado == "listo" || estado == "error" {
		job.finishedOnce.Do(func() { close(job.Done) })
	}
}

func (s *JobStore) Get(id string) (JobSnapshot, bool) {
	s.mu.RLock()
	job, ok := s.jobs[id]
	s.mu.RUnlock()
	if !ok {
		return JobSnapshot{}, false
	}

	snap := job.Snapshot()
	if time.Now().After(snap.ExpiraEn) {
		return JobSnapshot{}, false
	}
	return snap, true
}

func (s *JobStore) cleanupLoop() {
	interval := s.ttl / 2
	if interval < time.Minute {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

func (s *JobStore) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, job := range s.jobs {
		if now.After(job.ExpiraEn) {
			delete(s.jobs, id)
		}
	}
}
