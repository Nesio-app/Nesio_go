package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ConnectorAuthRequest struct {
	Credentials map[string]any `json:"credentials,omitempty"`
}

func (s *Server) handleListConnectors(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	type connectorWithCredentials struct {
		ID          uuid.UUID       `db:"id"`
		UserID      uuid.UUID       `db:"user_id"`
		Provider    string          `db:"provider"`
		Credentials json.RawMessage `db:"credentials"`
		IsActive    bool            `db:"is_active"`
		LastSyncAt  *time.Time      `db:"last_sync_at"`
		CreatedAt   time.Time       `db:"created_at"`
	}

	rows := make([]connectorWithCredentials, 0)
	err := s.store.DB.Select(&rows, "SELECT id, user_id, provider, credentials, is_active, last_sync_at, created_at FROM connectors WHERE user_id = $1", userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	connectors := make([]models.Connector, 0, len(rows))
	for _, row := range rows {
		effectiveActive := row.IsActive
		if row.IsActive {
			credentials, decryptErr := connector.DecryptCredentials(row.Credentials)
			if decryptErr != nil || !connectorCredentialsReady(row.Provider, credentials) {
				effectiveActive = false
				_, _ = s.store.DB.Exec("UPDATE connectors SET is_active = false WHERE id = $1", row.ID)
			}
		}

		connectors = append(connectors, models.Connector{
			ID:         row.ID,
			UserID:     row.UserID,
			Provider:   row.Provider,
			IsActive:   effectiveActive,
			LastSyncAt: row.LastSyncAt,
			CreatedAt:  row.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, connectors)
}

func (s *Server) handleConnectorAuth(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	provider := strings.TrimSpace(c.Param("provider"))
	if !isSupportedProvider(provider) {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported provider")
	}
	var req ConnectorAuthRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(req.Credentials) == 0 {
		if !supportsProviderOAuth(provider) {
			return echo.NewHTTPError(http.StatusBadRequest, "provider does not support oauth link")
		}

		state := uuid.NewString()
		authURL, err := buildProviderOAuthURL(provider, state)
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
		}

		stateData := map[string]any{"user_id": userID.String(), "provider": provider}
		statePayload, err := json.Marshal(stateData)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if err := s.store.Set(c.Request().Context(), "connector_oauth:"+state, string(statePayload), storage.TierSession, 5*time.Minute); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, map[string]any{"auth_url": authURL})
	}

	isActive := connectorCredentialsReady(provider, req.Credentials)
	status := "connected"
	if !isActive {
		status = "oauth_pending_credentials"
	}

	encrypted, err := connector.EncryptCredentials(req.Credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	id, err := upsertConnectorCredentialsNoConflict(s, userID, provider, encrypted, isActive)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"id":       id,
		"provider": provider,
		"status":   status,
	})
}

func (s *Server) handleConnectorAuthorize(c echo.Context) error {
	return echo.NewHTTPError(http.StatusGone, "deprecated route: use POST /api/v1/connectors/:provider/auth to fetch oauth url")
}

func (s *Server) handleConnectorCallback(c echo.Context) error {
	provider := c.Param("provider")
	state := c.QueryParam("state")
	code := c.QueryParam("code")
	if state == "" || code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing state or code")
	}

	raw, err := s.store.Get(c.Request().Context(), "connector_oauth:"+state, storage.TierSession)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired state")
	}

	var sessionData struct {
		UserID   string `json:"user_id"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal([]byte(raw), &sessionData); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if sessionData.Provider != provider {
		return echo.NewHTTPError(http.StatusBadRequest, "provider mismatch")
	}

	userID, err := uuid.Parse(sessionData.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	credentials := map[string]any{"code": code, "provider": provider}
	if hasProviderOAuthTokenConfig(provider) {
		tokens, exchangeErr := exchangeProviderOAuthCode(c.Request().Context(), provider, code)
		if exchangeErr == nil {
			for k, v := range tokens {
				credentials[k] = v
			}
		} else {
			credentials["oauth_exchange_error"] = exchangeErr.Error()
		}
	}

	active := connectorCredentialsReady(provider, credentials)
	encrypted, err := connector.EncryptCredentials(credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_, err = upsertConnectorCredentialsNoConflict(s, userID, provider, encrypted, active)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_ = s.store.Delete(c.Request().Context(), "connector_oauth:"+state, storage.TierSession)
	status := "connected"
	if !active {
		status = "oauth_code_received"
	}
	return c.Redirect(http.StatusFound, connectorOAuthClientRedirect(provider, status))
}

func upsertConnectorCredentialsNoConflict(s *Server, userID uuid.UUID, provider string, encrypted json.RawMessage, isActive bool) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.store.DB.Get(&id, "SELECT id FROM connectors WHERE user_id = $1 AND provider = $2 ORDER BY created_at DESC LIMIT 1", userID, provider)
	if err == nil {
		_, execErr := s.store.DB.Exec(
			"UPDATE connectors SET credentials = $3, is_active = $4, last_sync_at = NULL WHERE id = $1 AND user_id = $2",
			id, userID, encrypted, isActive,
		)
		if execErr != nil {
			return uuid.Nil, execErr
		}
		return id, nil
	}

	var createdID uuid.UUID
	err = s.store.DB.QueryRow(
		"INSERT INTO connectors (user_id, provider, credentials, is_active, last_sync_at) VALUES ($1, $2, $3, $4, NULL) RETURNING id",
		userID, provider, encrypted, isActive,
	).Scan(&createdID)
	if err != nil {
		return uuid.Nil, err
	}
	return createdID, nil
}

func connectorCredentialsReady(provider string, credentials map[string]any) bool {
	if credentials == nil {
		return false
	}
	accessToken, _ := credentials["access_token"].(string)
	accessToken = strings.TrimSpace(accessToken)

	switch connectorProviderKey(provider) {
	case "gmail", "googlemail", "calendar", "googlecalendar", "teslafleet", "plaid", "granola", "flomo", "googletimeline", "applehealth":
		return accessToken != ""
	default:
		return accessToken != ""
	}
}

func connectorProviderKey(provider string) string {
	v := strings.TrimSpace(strings.ToLower(provider))
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "_", "")
	return v
}

func connectorOAuthClientRedirect(provider, status string) string {
	base := strings.TrimSpace(os.Getenv("CONNECTOR_OAUTH_CLIENT_REDIRECT_URL"))
	if base == "" {
		base = "http://localhost:5173/Nesio_go/"
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return fmt.Sprintf("http://localhost:5173/Nesio_go/?connector=%s&status=%s", url.QueryEscape(provider), url.QueryEscape(status))
	}
	query := parsed.Query()
	query.Set("connector", provider)
	query.Set("status", status)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (s *Server) handleDeleteConnector(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec("DELETE FROM connectors WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleSyncConnector(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var owner uuid.UUID
	err = s.store.DB.Get(&owner, "SELECT user_id FROM connectors WHERE id = $1", id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "connector not found")
	}
	if owner != userID {
		return echo.NewHTTPError(http.StatusForbidden, "not authorized")
	}

	if err := connector.SyncConnector(c.Request().Context(), s.store, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "syncing"})
}
