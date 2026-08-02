package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/hibiken/asynq"
)

const taskDailyMaintenance = "maintenance:daily"

type maintenancePayload struct{}

type Worker struct {
	store  *storage.Store
	ctx    context.Context
	cancel context.CancelFunc
	client *asynq.Client
	server *asynq.Server
	sched  *asynq.Scheduler
}

func New(store *storage.Store) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	redisOptions := store.RDB.Options()
	asynqOptions := asynq.RedisClientOpt{
		Addr:      redisOptions.Addr,
		Username:  redisOptions.Username,
		Password:  redisOptions.Password,
		DB:        redisOptions.DB,
		TLSConfig: redisOptions.TLSConfig,
	}
	server := asynq.NewServer(
		asynqOptions,
		asynq.Config{
			Concurrency: 5,
		},
	)
	sched := asynq.NewScheduler(
		asynqOptions,
		&asynq.SchedulerOpts{},
	)
	client := asynq.NewClient(asynqOptions)
	return &Worker{store: store, ctx: ctx, cancel: cancel, client: client, server: server, sched: sched}
}

func (w *Worker) Start() {
	log.Println("Worker started with Asynq")

	mux := asynq.NewServeMux()
	mux.HandleFunc(taskDailyMaintenance, func(ctx context.Context, task *asynq.Task) error {
		return w.runDailyMaintenance(ctx, task)
	})

	payload, _ := json.Marshal(maintenancePayload{})
	if _, err := w.sched.Register("@every 1h", asynq.NewTask(taskDailyMaintenance, payload)); err != nil {
		log.Printf("register scheduler error: %v", err)
	}

	go func() {
		if err := w.sched.Run(); err != nil {
			log.Printf("scheduler stopped: %v", err)
		}
	}()

	if _, err := w.client.EnqueueContext(w.ctx, asynq.NewTask(taskDailyMaintenance, payload)); err != nil {
		log.Printf("enqueue initial maintenance error: %v", err)
	}

	if err := w.server.Run(mux); err != nil {
		log.Printf("worker server stopped: %v", err)
	}
}

func (w *Worker) Stop() {
	w.cancel()
	if w.sched != nil {
		w.sched.Shutdown()
	}
	if w.server != nil {
		w.server.Shutdown()
	}
	if w.client != nil {
		if err := w.client.Close(); err != nil {
			log.Printf("worker client close error: %v", err)
		}
	}
}

func (w *Worker) runDailyMaintenance(ctx context.Context, task *asynq.Task) error {
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
		return err
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
		return err
	}

	if err := connector.SyncAllConnectorsWithStore(ctx, w.store); err != nil {
		log.Printf("connector sync error: %v", err)
		return err
	}

	log.Printf("Daily cleanup and connector sync completed: %s", task.Type())
	return nil
}
