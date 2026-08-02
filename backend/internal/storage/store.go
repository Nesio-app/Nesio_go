package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	DB  *sqlx.DB
	RDB *redis.Client
}

func New(ctx context.Context, dbURL, redisURL string) (*Store, error) {
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return &Store{DB: db, RDB: rdb}, nil
}

func (s *Store) Close() error {
	if s.RDB != nil {
		s.RDB.Close()
	}
	return s.DB.Close()
}

func (s *Store) Migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text UNIQUE NOT NULL,
    password_hash text,
    timezone text NOT NULL DEFAULT 'Asia/Shanghai',
    locale text NOT NULL DEFAULT 'zh',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS connectors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    credentials jsonb NOT NULL DEFAULT '{}',
    is_active boolean NOT NULL DEFAULT true,
    last_sync_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS life_nodes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    type text NOT NULL,
    domain text,
    title text NOT NULL,
    body text,
    status text NOT NULL DEFAULT 'active',
    due_date timestamptz,
    tags text[] NOT NULL DEFAULT '{}',
    attributes jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nodes_user_status ON life_nodes(user_id, status, due_date);
CREATE INDEX IF NOT EXISTS idx_nodes_user_domain ON life_nodes(user_id, domain);

CREATE TABLE IF NOT EXISTS today_cards (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    local_day text NOT NULL,
    slot text NOT NULL DEFAULT 'task',
    node_id uuid REFERENCES life_nodes(id) ON DELETE CASCADE,
    title text NOT NULL,
    body text,
    severity int NOT NULL DEFAULT 1,
    action_label text,
    fingerprints text[] NOT NULL DEFAULT '{}',
    dismissed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_today_user_day ON today_cards(user_id, local_day, slot, severity DESC);

CREATE TABLE IF NOT EXISTS fingerprints (
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    hash text NOT NULL,
    source text NOT NULL,
    is_muted boolean NOT NULL DEFAULT false,
    dismissed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, hash)
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL,
    content text NOT NULL,
    actions jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS energy_readings (
    time timestamptz NOT NULL,
    user_id uuid NOT NULL,
    value int NOT NULL CHECK (value BETWEEN 0 AND 100),
    note text,
    is_weekend boolean NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS memory_embeddings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    qdrant_id text NOT NULL UNIQUE,
    source text NOT NULL,
    content_preview text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS kv_store (
    key text PRIMARY KEY,
    value jsonb NOT NULL,
    tier text NOT NULL,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_kv_store_tier ON kv_store(tier);
CREATE INDEX IF NOT EXISTS idx_kv_store_expires_at ON kv_store(expires_at);
`
	_, err := s.DB.Exec(schema)
	return err
}

type StorageTier string

const (
	TierDurable StorageTier = "durable"
	TierCache   StorageTier = "cache"
	TierSession StorageTier = "session"
)

type SetOptions struct {
	TTL  time.Duration
	Tier StorageTier
}

func (s *Store) Set(ctx context.Context, key string, value any, tier StorageTier, ttl time.Duration) error {
	switch tier {
	case TierCache, TierSession:
		return s.RDB.Set(ctx, key, value, ttl).Err()
	case TierDurable:
		payload, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var expiresAt *time.Time
		if ttl > 0 {
			t := time.Now().Add(ttl)
			expiresAt = &t
		}
		_, err = s.DB.Exec(`
			INSERT INTO kv_store (key, value, tier, expires_at, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (key) DO UPDATE SET value = $2, tier = $3, expires_at = $4, updated_at = now()
		`, key, payload, string(tier), expiresAt)
		return err
	default:
		return fmt.Errorf("unsupported storage tier: %s", tier)
	}
}

func (s *Store) Get(ctx context.Context, key string, tier StorageTier) (string, error) {
	switch tier {
	case TierCache, TierSession:
		return s.RDB.Get(ctx, key).Result()
	case TierDurable:
		var raw sql.NullString
		err := s.DB.Get(&raw, `
			SELECT value::text
			FROM kv_store
			WHERE key = $1
			AND (expires_at IS NULL OR expires_at > now())
		`, key)
		if err != nil {
			return "", err
		}
		if !raw.Valid {
			return "", sql.ErrNoRows
		}
		return raw.String, nil
	default:
		return "", fmt.Errorf("unsupported storage tier: %s", tier)
	}
}

func (s *Store) Delete(ctx context.Context, key string, tier StorageTier) error {
	switch tier {
	case TierCache, TierSession:
		return s.RDB.Del(ctx, key).Err()
	case TierDurable:
		_, err := s.DB.Exec(`DELETE FROM kv_store WHERE key = $1`, key)
		return err
	default:
		return fmt.Errorf("unsupported storage tier: %s", tier)
	}
}
