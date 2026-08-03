package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"net/http"
	"os"
	"bytes"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type providerInfo struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
	Status   string `json:"status"`
}

var supportedProviders = []providerInfo{
	{Provider: "tesla_fleet", Label: "Tesla Fleet", Status: "oauth_link"},
	{Provider: "granola", Label: "Granola", Status: "oauth_link"},
	{Provider: "plaid", Label: "Plaid", Status: "oauth_link"},
	{Provider: "google_timeline", Label: "Google Timeline", Status: "oauth_link"},
	{Provider: "flomo", Label: "Flomo", Status: "oauth_link"},
	{Provider: "apple_health", Label: "Apple Health", Status: "oauth_link"},
}

var providerOAuthAuthURLEnv = map[string]string{
	"tesla_fleet":     "CONNECTOR_TESLA_FLEET_AUTH_URL",
	"granola":         "CONNECTOR_GRANOLA_AUTH_URL",
	"plaid":           "CONNECTOR_PLAID_AUTH_URL",
	"google_timeline": "CONNECTOR_GOOGLE_TIMELINE_AUTH_URL",
	"flomo":           "CONNECTOR_FLOMO_AUTH_URL",
	"apple_health":    "CONNECTOR_APPLE_HEALTH_AUTH_URL",
}

var providerOAuthTokenURLEnv = map[string]string{
	"tesla_fleet":     "CONNECTOR_TESLA_FLEET_TOKEN_URL",
	"granola":         "CONNECTOR_GRANOLA_TOKEN_URL",
	"plaid":           "CONNECTOR_PLAID_TOKEN_URL",
	"google_timeline": "CONNECTOR_GOOGLE_TIMELINE_TOKEN_URL",
	"flomo":           "CONNECTOR_FLOMO_TOKEN_URL",
	"apple_health":    "CONNECTOR_APPLE_HEALTH_TOKEN_URL",
}

var providerOAuthClientIDEnv = map[string]string{
	"tesla_fleet":     "CONNECTOR_TESLA_FLEET_CLIENT_ID",
	"granola":         "CONNECTOR_GRANOLA_CLIENT_ID",
	"plaid":           "CONNECTOR_PLAID_CLIENT_ID",
	"google_timeline": "CONNECTOR_GOOGLE_TIMELINE_CLIENT_ID",
	"flomo":           "CONNECTOR_FLOMO_CLIENT_ID",
	"apple_health":    "CONNECTOR_APPLE_HEALTH_CLIENT_ID",
}

var providerOAuthClientSecretEnv = map[string]string{
	"tesla_fleet":     "CONNECTOR_TESLA_FLEET_CLIENT_SECRET",
	"granola":         "CONNECTOR_GRANOLA_CLIENT_SECRET",
	"plaid":           "CONNECTOR_PLAID_CLIENT_SECRET",
	"google_timeline": "CONNECTOR_GOOGLE_TIMELINE_CLIENT_SECRET",
	"flomo":           "CONNECTOR_FLOMO_CLIENT_SECRET",
	"apple_health":    "CONNECTOR_APPLE_HEALTH_CLIENT_SECRET",
}

var encryptConnectorCredentialsFn = connector.EncryptCredentials
var syncConnectorFn = connector.SyncConnector

var upsertConnectorProviderFn = func(s *Server, userID uuid.UUID, provider string, encrypted []byte) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.store.DB.Get(&id, "SELECT id FROM connectors WHERE user_id = $1 AND provider = $2 ORDER BY created_at DESC LIMIT 1", userID, provider)
	if err == nil {
		_, execErr := s.store.DB.Exec(
			"UPDATE connectors SET credentials = $3, is_active = true, last_sync_at = NULL WHERE id = $1 AND user_id = $2",
			id, userID, encrypted,
		)
		if execErr != nil {
			return uuid.Nil, execErr
		}
		return id, nil
	}

	var createdID uuid.UUID
	err = s.store.DB.QueryRow(
		"INSERT INTO connectors (user_id, provider, credentials, is_active, last_sync_at) VALUES ($1, $2, $3, true, NULL) RETURNING id",
		userID, provider, encrypted,
	).Scan(&createdID)
	if err != nil {
		return uuid.Nil, err
	}
	return createdID, nil
}

var createConnectorImportCardFn = func(s *Server, userID uuid.UUID, provider, status string) {
	title := fmt.Sprintf("%s 数据已导入", provider)
	body := fmt.Sprintf("%s 连接器已完成导入并同步", provider)
	if status == "imported" {
		body = fmt.Sprintf("%s 连接器已导入，等待同步", provider)
	}
	localDay := s.getUserLocalDay(userID)
	_, _ = s.store.DB.Exec(`
		INSERT INTO today_cards (user_id, local_day, slot, title, body, severity, fingerprints)
		VALUES ($1, $2, 'guidance', $3, $4, 2, $5)
	`, userID, localDay, title, body, pq.Array([]string{fmt.Sprintf("connector-import:%s:%d", provider, time.Now().Unix())}))
}

func (s *Server) handleListConnectorProviders(c echo.Context) error {
	return c.JSON(http.StatusOK, supportedProviders)
}

func (s *Server) handleImportConnectorProvider(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	provider := strings.TrimSpace(c.Param("provider"))
	if !isSupportedProvider(provider) {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported provider")
	}

	var payload map[string]any
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if err := validateProviderPayload(provider, payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	encrypted, err := encryptConnectorCredentialsFn(payload)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	id, err := upsertConnectorProviderFn(s, userID, provider, encrypted)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	syncNow := c.QueryParam("sync")
	if syncNow == "" || strings.EqualFold(syncNow, "true") || syncNow == "1" {
		if err := syncConnectorFn(c.Request().Context(), s.store, id); err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("connector imported but sync failed: %v", err))
		}
	}

	status := "imported"
	if syncNow == "" || strings.EqualFold(syncNow, "true") || syncNow == "1" {
		status = "synced"
	}

	createConnectorImportCardFn(s, userID, provider, status)

	return c.JSON(http.StatusCreated, map[string]any{"id": id, "provider": provider, "status": status})
}

func isSupportedProvider(provider string) bool {
	for _, p := range supportedProviders {
		if p.Provider == provider {
			return true
		}
	}
	return false
}

func supportsProviderOAuth(provider string) bool {
	_, ok := providerOAuthAuthURLEnv[provider]
	return ok
}

func buildProviderOAuthURL(provider, state string) (string, error) {
	envKey, ok := providerOAuthAuthURLEnv[provider]
	if !ok {
		return "", fmt.Errorf("oauth is not supported for provider: %s", provider)
	}

	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return "", fmt.Errorf("oauth is not configured for %s: set %s", provider, envKey)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid oauth auth url for %s", provider)
	}
	query := parsed.Query()
	query.Set("state", state)
	if query.Get("redirect_uri") == "" {
		query.Set("redirect_uri", connectorProviderRedirectURI(provider))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func connectorProviderRedirectURI(provider string) string {
	base := strings.TrimSpace(os.Getenv("CONNECTOR_OAUTH_REDIRECT_BASE_URL"))
	if base == "" {
		base = "http://localhost:8080"
	}
	base = strings.TrimRight(base, "/")
	return fmt.Sprintf("%s/api/v1/connectors/%s/callback", base, provider)
}

func hasProviderOAuthTokenConfig(provider string) bool {
	tokenKey, ok1 := providerOAuthTokenURLEnv[provider]
	clientIDKey, ok2 := providerOAuthClientIDEnv[provider]
	clientSecretKey, ok3 := providerOAuthClientSecretEnv[provider]
	if !ok1 || !ok2 || !ok3 {
		return false
	}
	return strings.TrimSpace(os.Getenv(tokenKey)) != "" && strings.TrimSpace(os.Getenv(clientIDKey)) != "" && strings.TrimSpace(os.Getenv(clientSecretKey)) != ""
}

func exchangeProviderOAuthCode(ctx context.Context, provider, code string) (map[string]any, error) {
	tokenURLKey, ok1 := providerOAuthTokenURLEnv[provider]
	clientIDKey, ok2 := providerOAuthClientIDEnv[provider]
	clientSecretKey, ok3 := providerOAuthClientSecretEnv[provider]
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("oauth token exchange not supported for provider: %s", provider)
	}

	tokenURL := strings.TrimSpace(os.Getenv(tokenURLKey))
	clientID := strings.TrimSpace(os.Getenv(clientIDKey))
	clientSecret := strings.TrimSpace(os.Getenv(clientSecretKey))
	if tokenURL == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("oauth token exchange not configured for %s", provider)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", connectorProviderRedirectURI(provider))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth token exchange failed: %s", strings.TrimSpace(string(body)))
	}

	tokens := map[string]any{}
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, err
	}
	if accessToken, _ := tokens["access_token"].(string); strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("oauth token exchange succeeded but access_token missing")
	}
	return tokens, nil
}

func validateProviderPayload(provider string, payload map[string]any) error {
	requireString := func(key string) error {
		v, _ := payload[key].(string)
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("missing required field: %s", key)
		}
		return nil
	}

	switch provider {
	case "tesla_fleet":
		return requireString("access_token")
	case "plaid":
		if err := requireString("client_id"); err != nil {
			return err
		}
		if err := requireString("secret"); err != nil {
			return err
		}
		return requireString("access_token")
	case "granola", "flomo":
		return requireString("endpoint")
	case "google_timeline":
		if events, ok := payload["timeline_events"].([]any); !ok || len(events) == 0 {
			return fmt.Errorf("missing required field: timeline_events")
		}
		return nil
	case "apple_health":
		if entries, ok := payload["entries"].([]any); !ok || len(entries) == 0 {
			return fmt.Errorf("missing required field: entries")
		}
		return nil
	default:
		return fmt.Errorf("unsupported provider")
	}
}
