package api

import (
	"fmt"
	"net/http"
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
	{Provider: "tesla_fleet", Label: "Tesla Fleet", Status: "ready_for_token"},
	{Provider: "granola", Label: "Granola", Status: "ready_for_token"},
	{Provider: "plaid", Label: "Plaid", Status: "ready_for_token"},
	{Provider: "google_timeline", Label: "Google Timeline", Status: "ready_for_token"},
	{Provider: "flomo", Label: "Flomo", Status: "ready_for_token"},
	{Provider: "apple_health", Label: "Apple Health", Status: "ready_for_token"},
}

var encryptConnectorCredentialsFn = connector.EncryptCredentials
var syncConnectorFn = connector.SyncConnector

var upsertConnectorProviderFn = func(s *Server, userID uuid.UUID, provider string, encrypted []byte) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO connectors (user_id, provider, credentials, is_active, last_sync_at)
		VALUES ($1, $2, $3, true, NULL)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET credentials = EXCLUDED.credentials, is_active = true, last_sync_at = NULL
		RETURNING id
	`, userID, provider, encrypted).Scan(&id)
	return id, err
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
