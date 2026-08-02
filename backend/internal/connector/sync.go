package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/Nesio-app/Nesio_go/internal/signal"
	"github.com/Nesio-app/Nesio_go/internal/storage"
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

type googleCredentials struct {
	AccessToken string `json:"access_token"`
	Account     string `json:"account"`
	CalendarID  string `json:"calendar_id"`
}

type gmailListResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type gmailMessageResponse struct {
	ID       string `json:"id"`
	Snippet  string `json:"snippet"`
	Internal string `json:"internalDate"`
	Payload  struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
	} `json:"payload"`
}

type calendarEventsResponse struct {
	Items []struct {
		ID          string `json:"id"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Location    string `json:"location"`
		Start       struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
	} `json:"items"`
}

func SyncConnector(ctx context.Context, store *storage.Store, connectorID uuid.UUID) error {
	db := store.DB
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

	processor := signal.NewProcessor(store)
	signals, err := fetchConnectorSignals(ctx, conn.Provider, credentials)
	if err != nil {
		return err
	}
	if len(signals) == 0 {
		signals = append(signals, buildFallbackSignal(conn.Provider, connectorID, credentials))
	}
	for _, sig := range signals {
		if _, err := processor.Process(ctx, conn.UserID, sig); err != nil {
			// fingerprint duplicates and muted signals should not abort the whole sync
			continue
		}
	}

	_, err = db.Exec("UPDATE connectors SET last_sync_at = now() WHERE id = $1", connectorID)
	return err
}

func SyncAllConnectors(ctx context.Context, db *sqlx.DB) error {
	return fmt.Errorf("SyncAllConnectors with db only is no longer supported")
}

func SyncAllConnectorsWithStore(ctx context.Context, store *storage.Store) error {
	db := store.DB
	var ids []uuid.UUID
	if err := db.Select(&ids, "SELECT id FROM connectors WHERE is_active = true AND (last_sync_at IS NULL OR last_sync_at < now() - INTERVAL '1 hour')"); err != nil {
		return err
	}
	for _, id := range ids {
		_ = SyncConnector(ctx, store, id)
	}
	return nil
}

func fetchConnectorSignals(ctx context.Context, provider string, credentials map[string]any) ([]models.Signal, error) {
	switch normalizeProvider(provider) {
	case "gmail":
		return fetchGmailSignals(ctx, credentials)
	case "calendar", "googlecalendar":
		return fetchCalendarSignals(ctx, credentials)
	default:
		return nil, nil
	}
}

func normalizeProvider(provider string) string {
	return strings.ReplaceAll(strings.ToLower(provider), "-", "")
}

func buildFallbackSignal(provider string, connectorID uuid.UUID, credentials map[string]any) models.Signal {
	rawData := fmt.Sprintf("Synced %s connector", provider)
	if account, ok := credentials["account"].(string); ok && account != "" {
		rawData = fmt.Sprintf("Synced %s connector for %s", provider, account)
	}
	fields := map[string]any{"provider": provider}
	if account, ok := credentials["account"].(string); ok && account != "" {
		fields["account"] = account
	}
	if calendarID, ok := credentials["calendar_id"].(string); ok && calendarID != "" {
		fields["calendar_id"] = calendarID
	}
	return models.Signal{
		Source:    provider,
		AnchorID:  fmt.Sprintf("sync-%s-%d", connectorID.String(), time.Now().Unix()),
		Fields:    fields,
		RawData:   rawData,
		Timestamp: time.Now().UTC(),
	}
}

func fetchGmailSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	creds := parseGoogleCredentials(credentials)
	if creds.AccessToken == "" {
		return nil, fmt.Errorf("gmail connector missing access_token")
	}

	var list gmailListResponse
	if err := doGoogleJSON(ctx, creds.AccessToken, "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=10&q=newer_than:2d", &list); err != nil {
		return nil, err
	}

	results := make([]models.Signal, 0, len(list.Messages))
	for _, msg := range list.Messages {
		var detail gmailMessageResponse
		detailURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=From&metadataHeaders=Subject", url.PathEscape(msg.ID))
		if err := doGoogleJSON(ctx, creds.AccessToken, detailURL, &detail); err != nil {
			continue
		}
		from := headerValue(detail.Payload.Headers, "From")
		subject := headerValue(detail.Payload.Headers, "Subject")
		rawData := subject
		if detail.Snippet != "" {
			rawData = subject + "\n" + detail.Snippet
		}
		results = append(results, models.Signal{
			Source:   "gmail",
			AnchorID: detail.ID,
			Fields: map[string]any{
				"from":    from,
				"subject": subject,
			},
			RawData:   rawData,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func fetchCalendarSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	creds := parseGoogleCredentials(credentials)
	if creds.AccessToken == "" {
		return nil, fmt.Errorf("calendar connector missing access_token")
	}
	calendarID := creds.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}
	timeMin := url.QueryEscape(time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339))
	endpoint := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?singleEvents=true&orderBy=startTime&timeMin=%s&maxResults=20", url.PathEscape(calendarID), timeMin)

	var events calendarEventsResponse
	if err := doGoogleJSON(ctx, creds.AccessToken, endpoint, &events); err != nil {
		return nil, err
	}

	results := make([]models.Signal, 0, len(events.Items))
	for _, event := range events.Items {
		start := event.Start.DateTime
		if start == "" {
			start = event.Start.Date
		}
		rawData := event.Summary
		if event.Description != "" {
			rawData = rawData + "\n" + event.Description
		}
		results = append(results, models.Signal{
			Source:   "calendar",
			AnchorID: event.ID,
			Fields: map[string]any{
				"title":       event.Summary,
				"location":    event.Location,
				"start_time":  start,
				"description": event.Description,
			},
			RawData:   rawData,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func parseGoogleCredentials(credentials map[string]any) googleCredentials {
	creds := googleCredentials{}
	if token, ok := credentials["access_token"].(string); ok {
		creds.AccessToken = token
	}
	if account, ok := credentials["account"].(string); ok {
		creds.Account = account
	}
	if calendarID, ok := credentials["calendar_id"].(string); ok {
		creds.CalendarID = calendarID
	}
	return creds
}

func doGoogleJSON(ctx context.Context, accessToken, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("google api error: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func headerValue(headers []struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}, name string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}
