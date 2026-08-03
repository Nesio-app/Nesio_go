-- Nesio v2 — run once in Railway Postgres or Supabase SQL editor
-- Safe to re-run: all statements use IF NOT EXISTS

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text UNIQUE NOT NULL,
    password_hash text,
    timezone text NOT NULL DEFAULT 'Asia/Shanghai',
    locale text NOT NULL DEFAULT 'zh',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_password_reset_user_expires ON password_reset_tokens(user_id, expires_at DESC);

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
