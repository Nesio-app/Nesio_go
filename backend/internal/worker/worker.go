package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/google/uuid"
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

	if err := w.runExpiryReminderJob(ctx); err != nil {
		log.Printf("expiry reminder job error: %v", err)
	}

	if err := w.runDocumentReminderJob(ctx); err != nil {
		log.Printf("document reminder job error: %v", err)
	}

	if err := w.runDailyBriefJob(ctx); err != nil {
		log.Printf("daily brief generation error: %v", err)
	}

	log.Printf("Daily cleanup and connector sync completed: %s", task.Type())
	return nil
}

func (w *Worker) ensureDailyBriefs(ctx context.Context) error {
	type userRow struct {
		ID       string `db:"id"`
		Timezone string `db:"timezone"`
	}
	users := make([]userRow, 0)
	if err := w.store.DB.Select(&users, `SELECT id::text as id, timezone FROM users`); err != nil {
		return err
	}

	for _, user := range users {
		tz := strings.TrimSpace(user.Timezone)
		if tz == "" {
			tz = "Asia/Shanghai"
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.FixedZone("UTC", 0)
		}
		nowLocal := time.Now().In(loc)
		if nowLocal.Hour() != 8 {
			continue
		}
		localDay := nowLocal.Format("2006-01-02")

		uid, err := uuid.Parse(user.ID)
		if err != nil {
			continue
		}

		var taskCount int
		var reminderCount int
		var cardCount int
		_ = w.store.DB.Get(&taskCount, `SELECT COUNT(*) FROM life_nodes WHERE user_id = $1 AND type = 'task' AND status IN ('active', 'later')`, uid)
		_ = w.store.DB.Get(&reminderCount, `SELECT COUNT(*) FROM reminders WHERE user_id = $1 AND is_done = false`, uid)
		_ = w.store.DB.Get(&cardCount, `SELECT COUNT(*) FROM today_cards WHERE user_id = $1 AND local_day = $2 AND dismissed_at IS NULL`, uid, localDay)

		content := fmt.Sprintf("早上好。今天待办 %d 项，提醒 %d 条，今日卡片 %d 条。先处理最重要的一条。", taskCount, reminderCount, cardCount)
		_, _ = w.store.DB.Exec(`
			INSERT INTO daily_briefs (user_id, local_day, content)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, local_day)
			DO UPDATE SET content = EXCLUDED.content, generated_at = now()
		`, uid, localDay, content)
	}

	return nil
}
