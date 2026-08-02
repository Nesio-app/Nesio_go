package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/Nesio-app/Nesio_go/internal/signal"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type connectorRecord struct {
	ID          uuid.UUID       `db:"id"`
	UserID      uuid.UUID       `db:"user_id"`
	Provider    string          `db:"provider"`
	Credentials json.RawMessage `db:"credentials"`
	IsActive    bool            `db:"is_active"`
}

func SyncConnector(ctx context.Context, db *sqlx.DB, connectorID uuid.UUID) error {
	var conn connectorRecord
	if err := db.Get(&conn, "SELECT id, user_id, provider, credentials, is_active FROM connectors WHERE id = $1", connectorID); err != nil {
		return err
	}
	if !conn.IsActive {
		return fmt.Errorf("connector is inactive")
	}

	credentials, err := DecryptCredentials(conn.Credentials)
	if err != nil {
		return err
	}

	rawData := fmt.Sprintf("Synced %s connector", conn.Provider)
	if account, ok := credentials["account"].(string); ok && account != "" {
		rawData = fmt.Sprintf("Synced %s connector for %s", conn.Provider, account)
	}

	fields := map[string]any{
		"provider": conn.Provider,
	}
	for key, value := range credentials {
		fields[key] = value
	}

	sig := models.Signal{
		Source:    conn.Provider,
		AnchorID:  fmt.Sprintf("sync-%s-%d", connectorID.String(), time.Now().Unix()),
		Fields:    fields,
		RawData:   rawData,
		Timestamp: time.Now().UTC(),
	}

	processor := signal.NewProcessor(db)
	_, err = processor.Process(ctx, conn.UserID, sig)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE connectors SET last_sync_at = now() WHERE id = $1", connectorID)
	return err
}

func SyncAllConnectors(ctx context.Context, db *sqlx.DB) error {
	var ids []uuid.UUID
	if err := db.Select(&ids, "SELECT id FROM connectors WHERE is_active = true AND (last_sync_at IS NULL OR last_sync_at < now() - INTERVAL '1 hour')"); err != nil {
		return err
	}
	for _, id := range ids {
		_ = SyncConnector(ctx, db, id)
	}
	return nil
}
