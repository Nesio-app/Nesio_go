package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type JSONMap map[string]any

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}

	var raw []byte
	switch value := src.(type) {
	case []byte:
		raw = value
	case string:
		raw = []byte(value)
	default:
		return fmt.Errorf("unsupported JSONMap scan type %T", src)
	}

	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}

	decoded := JSONMap{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*m = decoded
	return nil
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Timezone     string    `db:"timezone" json:"timezone"`
	Locale       string    `db:"locale" json:"locale"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Connector struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Provider   string    `db:"provider" json:"provider"`
	Credentials JSONMap `db:"credentials" json:"-"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	LastSyncAt *time.Time `db:"last_sync_at" json:"last_sync_at,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type LifeNode struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Type       string    `db:"type" json:"type"`
	Domain     *string   `db:"domain" json:"domain,omitempty"`
	Title      string    `db:"title" json:"title"`
	Body       *string   `db:"body" json:"body,omitempty"`
	Status     string    `db:"status" json:"status"`
	DueDate    *time.Time `db:"due_date" json:"due_date,omitempty"`
	Tags       pq.StringArray  `db:"tags" json:"tags"`
	Attributes JSONMap `db:"attributes" json:"attributes"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type TodayCard struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	UserID       uuid.UUID  `db:"user_id" json:"user_id"`
	LocalDay     string     `db:"local_day" json:"local_day"`
	Slot         string     `db:"slot" json:"slot"`
	NodeID       *uuid.UUID `db:"node_id" json:"node_id,omitempty"`
	Title        string     `db:"title" json:"title"`
	Body         *string    `db:"body" json:"body,omitempty"`
	Severity     int        `db:"severity" json:"severity"`
	ActionLabel  *string    `db:"action_label" json:"action_label,omitempty"`
	Fingerprints pq.StringArray   `db:"fingerprints" json:"fingerprints"`
	DismissedAt  *time.Time `db:"dismissed_at" json:"dismissed_at,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

type ChatMessage struct {
	ID        uuid.UUID      `db:"id" json:"id"`
	UserID    uuid.UUID      `db:"user_id" json:"user_id"`
	Role      string         `db:"role" json:"role"`
	Content   string         `db:"content" json:"content"`
	Actions   JSONMap `db:"actions" json:"actions,omitempty"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}

type Signal struct {
	Source    string         `json:"source"`
	AnchorID  string         `json:"anchor_id"`
	Fields    map[string]any `json:"fields"`
	RawData   string         `json:"raw_data"`
	Timestamp time.Time      `json:"timestamp"`
}
