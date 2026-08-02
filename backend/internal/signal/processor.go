package signal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/Nesio-app/Nesio_go/pkg/fingerprint"
)

type Processor struct {
	DB *sqlx.DB
}

func NewProcessor(db *sqlx.DB) *Processor {
	return &Processor{DB: db}
}

func (p *Processor) Process(ctx context.Context, userID uuid.UUID, signal models.Signal) (*models.TodayCard, error) {
	fp := fingerprint.Fingerprint(signal)

	// Check muted
	var muted bool
	err := p.DB.Get(&muted, "SELECT EXISTS(SELECT 1 FROM fingerprints WHERE user_id = $1 AND hash = $2 AND is_muted = true)", userID, fp)
	if err == nil && muted {
		return nil, fmt.Errorf("signal muted")
	}

	// Check dismissed today
	var dismissed bool
	err = p.DB.Get(&dismissed, "SELECT EXISTS(SELECT 1 FROM fingerprints WHERE user_id = $1 AND hash = $2 AND dismissed_at > $3)", userID, fp, time.Now().Add(-24*time.Hour))
	if err == nil && dismissed {
		return nil, fmt.Errorf("signal dismissed today")
	}

	// Check daily quota (max 50 cards/day)
	var count int
	err = p.DB.Get(&count, "SELECT COUNT(*) FROM today_cards WHERE user_id = $1 AND local_day = $2", userID, time.Now().Format("2006-01-02"))
	if err == nil && count >= 50 {
		return nil, fmt.Errorf("daily quota exceeded")
	}

	// Classify severity
	severity := classifySignal(signal)

	// Generate card
	card := &models.TodayCard{
		UserID:       userID,
		LocalDay:     time.Now().Format("2006-01-02"),
		Slot:         "task",
		Title:        generateTitle(signal),
		Body:         generateBody(signal),
		Severity:     severity,
		Fingerprints: []string{fp},
		CreatedAt:    time.Now(),
	}

	if severity == 3 {
		card.Slot = "pinned"
	}

	// Insert card
	_, err = p.DB.NamedExec(`
		INSERT INTO today_cards (user_id, local_day, slot, title, body, severity, fingerprints)
		VALUES (:user_id, :local_day, :slot, :title, :body, :severity, :fingerprints)
	`, card)
	if err != nil {
		return nil, fmt.Errorf("insert card: %w", err)
	}

	// Record fingerprint
	_, _ = p.DB.Exec(`
		INSERT INTO fingerprints (user_id, hash, source, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, hash) DO NOTHING
	`, userID, fp, signal.Source, time.Now())

	return card, nil
}

func classifySignal(signal models.Signal) int {
	switch signal.Source {
	case "calendar":
		// Check if event starts within 2 hours
		if start, ok := signal.Fields["start_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				if time.Until(t) <= 2*time.Hour && time.Until(t) > 0 {
					return 3 // critical
				}
			}
		}
		return 1
	case "plaid":
		if overdue, ok := signal.Fields["overdue"].(bool); ok && overdue {
			return 3
		}
		return 1
	case "gmail", "email":
		return 2 // important - needs AI review
	default:
		return 1
	}
}

func generateTitle(signal models.Signal) string {
	switch signal.Source {
	case "calendar":
		if title, ok := signal.Fields["title"].(string); ok {
			return title
		}
	case "plaid":
		if desc, ok := signal.Fields["description"].(string); ok {
			return "账单提醒: " + desc
		}
	}
	return "新信号"
}

func generateBody(signal models.Signal) *string {
	var body string
	switch signal.Source {
	case "calendar":
		if loc, ok := signal.Fields["location"].(string); ok && loc != "" {
			body = "地点: " + loc
		}
	case "gmail":
		if from, ok := signal.Fields["from"].(string); ok {
			body = "来自: " + from
		}
	}
	if body == "" {
		return nil
	}
	return &body
}
