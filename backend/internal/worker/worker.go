package worker

import (
	"context"
	"log"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/storage"
)

type Worker struct {
	store  *storage.Store
	ctx    context.Context
	cancel context.CancelFunc
}

func New(store *storage.Store) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{store: store, ctx: ctx, cancel: cancel}
}

func (w *Worker) Start() {
	log.Println("Worker started")

	// Daily cleanup ticker
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.runDailyCleanup()
		}
	}
}

func (w *Worker) Stop() {
	w.cancel()
}

func (w *Worker) runDailyCleanup() {
	// Move 7-day old active tasks without due_date to 'later'
	_, err := w.store.DB.Exec(`
		UPDATE life_nodes 
		SET status = 'later', updated_at = now()
		WHERE status = 'active' 
		  AND due_date IS NULL 
		  AND created_at < NOW() - INTERVAL '7 days'
	`)
	if err != nil {
		log.Printf("cleanup later error: %v", err)
	}

	// Archive 30-day old 'later' tasks
	_, err = w.store.DB.Exec(`
		UPDATE life_nodes 
		SET status = 'archived', updated_at = now()
		WHERE status = 'later' 
		  AND updated_at < NOW() - INTERVAL '30 days'
	`)
	if err != nil {
		log.Printf("cleanup archive error: %v", err)
	}

	log.Println("Daily cleanup completed")
}
