package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (w *Worker) runExpiryReminderJob(ctx context.Context) error {
	type row struct {
		UserID        uuid.UUID `db:"user_id"`
		NodeID        uuid.UUID `db:"node_id"`
		Name          string    `db:"name"`
		DaysRemaining int       `db:"days_remaining"`
	}

	rows := make([]row, 0)
	err := w.store.DB.SelectContext(ctx, &rows, `
		SELECT n.user_id, n.id AS node_id, n.title AS name, (i.expiry_date - CURRENT_DATE) AS days_remaining
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		WHERE n.type = 'thing'
		  AND i.expiry_date IS NOT NULL
		  AND i.is_document = false
		  AND i.expiry_date BETWEEN CURRENT_DATE AND CURRENT_DATE + 7
	`)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, item := range rows {
		title := fmt.Sprintf("%s 即将到期", item.Name)
		body := fmt.Sprintf("%s 将在 %d 天内到期", item.Name, item.DaysRemaining)
		_, _ = w.store.DB.ExecContext(ctx, `
			INSERT INTO reminders (user_id, node_id, title, remind_at, source)
			SELECT $1, $2, $3, $4, 'item-expiry'
			WHERE NOT EXISTS (
				SELECT 1 FROM reminders
				WHERE user_id = $1 AND node_id = $2 AND source = 'item-expiry' AND is_done = false
			)
		`, item.UserID, item.NodeID, title, now)
		_, _ = w.store.DB.ExecContext(ctx, `
			INSERT INTO today_cards (user_id, local_day, slot, node_id, title, body, severity, fingerprints)
			VALUES ($1, $2, 'guidance', $3, $4, $5, 2, $6)
		`, item.UserID, now.Format("2006-01-02"), item.NodeID, title, body, []string{"item-expiry:" + item.NodeID.String()})
	}

	return nil
}
