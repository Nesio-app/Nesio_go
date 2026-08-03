package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	case "teslafleet":
		return fetchTeslaFleetSignals(ctx, credentials)
	case "plaid":
		return fetchPlaidSignals(ctx, credentials)
	case "granola":
		return fetchGranolaSignals(ctx, credentials)
	case "flomo":
		return fetchFlomoSignals(ctx, credentials)
	case "googletimeline":
		return fetchGoogleTimelineSignals(credentials)
	case "applehealth":
		return fetchAppleHealthSignals(credentials)
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

func fetchTeslaFleetSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	accessToken, _ := credentials["access_token"].(string)
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("tesla_fleet connector missing access_token")
	}
	endpoint, _ := credentials["endpoint"].(string)
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://fleet-api.prd.na.vn.cloud.tesla.com/api/1/vehicles"
	}

	var payload struct {
		Response []struct {
			ID           int64  `json:"id"`
			DisplayName  string `json:"display_name"`
			State        string `json:"state"`
			BatteryLevel *int   `json:"battery_level"`
			Vin          string `json:"vin"`
		} `json:"response"`
	}
	if err := doBearerJSON(ctx, accessToken, endpoint, &payload); err != nil {
		return nil, err
	}
	results := make([]models.Signal, 0, len(payload.Response))
	for _, vehicle := range payload.Response {
		raw := fmt.Sprintf("Tesla %s 状态 %s", vehicle.DisplayName, vehicle.State)
		fields := map[string]any{"name": vehicle.DisplayName, "state": vehicle.State, "vin": vehicle.Vin}
		if vehicle.BatteryLevel != nil {
			fields["battery_level"] = *vehicle.BatteryLevel
			raw = fmt.Sprintf("%s，电量 %d%%", raw, *vehicle.BatteryLevel)
		}
		results = append(results, models.Signal{
			Source:    "tesla_fleet",
			AnchorID:  fmt.Sprintf("tesla-%d", vehicle.ID),
			Fields:    fields,
			RawData:   raw,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func fetchPlaidSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	clientID, _ := credentials["client_id"].(string)
	secret, _ := credentials["secret"].(string)
	accessToken, _ := credentials["access_token"].(string)
	if strings.TrimSpace(clientID) == "" {
		clientID = strings.TrimSpace(os.Getenv("CONNECTOR_PLAID_CLIENT_ID"))
	}
	if strings.TrimSpace(secret) == "" {
		secret = strings.TrimSpace(os.Getenv("CONNECTOR_PLAID_CLIENT_SECRET"))
	}
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(secret) == "" || strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("plaid connector missing client_id/secret/access_token")
	}
	endpoint, _ := credentials["endpoint"].(string)
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://production.plaid.com/accounts/balance/get"
	}

	reqBody, _ := json.Marshal(map[string]any{
		"client_id":    clientID,
		"secret":       secret,
		"access_token": accessToken,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("plaid api error: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			Name      string `json:"name"`
			Subtype   string `json:"subtype"`
			Balances  struct {
				Available *float64 `json:"available"`
				Current   float64  `json:"current"`
			} `json:"balances"`
		} `json:"accounts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]models.Signal, 0, len(payload.Accounts))
	for _, account := range payload.Accounts {
		raw := fmt.Sprintf("%s 当前余额 %.2f", account.Name, account.Balances.Current)
		fields := map[string]any{
			"account_name": account.Name,
			"subtype":      account.Subtype,
			"current":      account.Balances.Current,
		}
		if account.Balances.Available != nil {
			fields["available"] = *account.Balances.Available
		}
		results = append(results, models.Signal{
			Source:    "plaid",
			AnchorID:  account.AccountID,
			Fields:    fields,
			RawData:   raw,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func fetchGranolaSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	return fetchGenericJSONSignals(ctx, credentials, "granola")
}

func fetchFlomoSignals(ctx context.Context, credentials map[string]any) ([]models.Signal, error) {
	return fetchGenericJSONSignals(ctx, credentials, "flomo")
}

func fetchGoogleTimelineSignals(credentials map[string]any) ([]models.Signal, error) {
	events, ok := credentials["timeline_events"].([]any)
	if !ok || len(events) == 0 {
		return nil, fmt.Errorf("google_timeline connector missing timeline_events")
	}
	results := make([]models.Signal, 0, len(events))
	for idx, item := range events {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := row["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = "timeline_event"
		}
		results = append(results, models.Signal{
			Source:    "google_timeline",
			AnchorID:  fmt.Sprintf("timeline-%d", idx),
			Fields:    row,
			RawData:   name,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func fetchAppleHealthSignals(credentials map[string]any) ([]models.Signal, error) {
	entries, ok := credentials["entries"].([]any)
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("apple_health connector missing entries")
	}
	results := make([]models.Signal, 0, len(entries))
	for idx, item := range entries {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		title, _ := row["metric"].(string)
		if strings.TrimSpace(title) == "" {
			title = "health_metric"
		}
		results = append(results, models.Signal{
			Source:    "apple_health",
			AnchorID:  fmt.Sprintf("health-%d", idx),
			Fields:    row,
			RawData:   title,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func fetchGenericJSONSignals(ctx context.Context, credentials map[string]any, source string) ([]models.Signal, error) {
	endpoint, _ := credentials["endpoint"].(string)
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("%s connector missing endpoint", source)
	}
	token, _ := credentials["access_token"].(string)

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := doOptionalBearerJSON(ctx, token, endpoint, &payload); err != nil {
		return nil, err
	}
	results := make([]models.Signal, 0, len(payload.Items))
	for idx, item := range payload.Items {
		raw := source
		if title, ok := item["title"].(string); ok && strings.TrimSpace(title) != "" {
			raw = title
		}
		results = append(results, models.Signal{
			Source:    source,
			AnchorID:  fmt.Sprintf("%s-%d", source, idx),
			Fields:    item,
			RawData:   raw,
			Timestamp: time.Now().UTC(),
		})
	}
	return results, nil
}

func doBearerJSON(ctx context.Context, accessToken, endpoint string, target any) error {
	if strings.TrimSpace(accessToken) == "" {
		return fmt.Errorf("missing access_token")
	}
	return doOptionalBearerJSON(ctx, accessToken, endpoint, target)
}

func doOptionalBearerJSON(ctx context.Context, accessToken, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(accessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("provider api error: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
