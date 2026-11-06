package services

import (
	"sync"
	"testing"
	"time"
)

func TestJobStoreLifecycle(t *testing.T) {
	store := NewJobStore(time.Minute)

	job := store.Create("test", func(j *Job) {
		j.SetProgress(50)
		store.Update(j, "listo", 100, "", nil, true)
	})

	select {
	case <-job.Wait():
	case <-time.After(2 * time.Second):
		t.Fatal("el job no terminó a tiempo")
	}

	snap, ok := store.Get(job.ID)
	if !ok {
		t.Fatal("el job debería existir")
	}
	if snap.Estado != "listo" || snap.Progreso != 100 {
		t.Fatalf("estado = %q, progreso = %d", snap.Estado, snap.Progreso)
	}
	if !snap.Exists {
		t.Fatal("Exists debería ser true")
	}
}

func TestJobStoreExpiredIsNotFound(t *testing.T) {
	store := NewJobStore(10 * time.Millisecond)

	job := store.Create("test", func(j *Job) {
		store.Update(j, "listo", 100, "", nil, false)
	})
	<-job.Wait()

	time.Sleep(30 * time.Millisecond)

	if _, ok := store.Get(job.ID); ok {
		t.Fatal("un job expirado no debe encontrarse")
	}
}

func TestJobStoreConcurrentSnapshots(t *testing.T) {
	store := NewJobStore(time.Minute)

	job := store.Create("test", func(j *Job) {
		for i := 0; i <= 100; i++ {
			j.SetProgress(i)
		}
		store.Update(j, "listo", 100, "", nil, false)
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < 100; k++ {
				_, _ = store.Get(job.ID)
			}
		}()
	}

	wg.Wait()
	<-job.Wait()
}
