package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"back/domain"
)

type Job struct {
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

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	ttl  time.Duration
}

func NewJobStore(ttl time.Duration) *JobStore {
	return &JobStore{jobs: make(map[string]*Job), ttl: ttl}
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
				job.Error = "panic en generacion"
				job.Estado = "error"
			}
			job.finishedOnce.Do(func() { close(job.Done) })
		}()
		job.Estado = "generando"
		job.Progreso = 5
		run(job)
	}()

	return job
}

func (s *JobStore) Update(job *Job, estado string, progreso int, errMsg string, doc *domain.Documento, exists bool) {
	job.Estado = estado
	job.Progreso = progreso
	job.Error = errMsg
	job.Documento = doc
	job.Exists = exists
	job.Actualizado = time.Now()
	if estado == "listo" || estado == "error" {
		job.finishedOnce.Do(func() { close(job.Done) })
	}
}

func (s *JobStore) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok || time.Now().After(job.ExpiraEn) {
		return nil, false
	}
	return job, true
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
