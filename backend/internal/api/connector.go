package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	var connectors []models.Connector
	err := s.store.DB.Select(&connectors, "SELECT id, user_id, provider, is_active, last_sync_at, created_at FROM connectors WHERE user_id = $1", userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, connectors)
}

func (s *Server) handleConnectorAuth(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	provider := c.Param("provider")
	var req ConnectorAuthRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(req.Credentials) == 0 {
		state := uuid.NewString()
		authURL := fmt.Sprintf("/api/v1/connectors/%s/authorize?state=%s", provider, state)
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

	var id uuid.UUID
	encrypted, err := connector.EncryptCredentials(req.Credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	err = s.store.DB.QueryRow(
		"INSERT INTO connectors (user_id, provider, credentials) VALUES ($1, $2, $3) RETURNING id",
		userID, provider, encrypted,
	).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"id":       id,
		"provider": provider,
		"status":   "connected",
	})
}

func (s *Server) handleConnectorAuthorize(c echo.Context) error {
	provider := c.Param("provider")
	state := c.QueryParam("state")
	if state == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing state")
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

	code := uuid.NewString()
	redirectURL := fmt.Sprintf("/api/v1/connectors/%s/callback?state=%s&code=%s", provider, state, code)
	return c.Redirect(http.StatusFound, redirectURL)
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
	encrypted, err := connector.EncryptCredentials(credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var id uuid.UUID
	err = s.store.DB.QueryRow(
		"INSERT INTO connectors (user_id, provider, credentials) VALUES ($1, $2, $3) RETURNING id",
		userID, provider, encrypted,
	).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_ = s.store.Delete(c.Request().Context(), "connector_oauth:"+state, storage.TierSession)
	return c.Redirect(http.StatusFound, "/connectors?connected="+provider)
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

	if err := connector.SyncConnector(c.Request().Context(), s.store.DB, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "syncing"})
}
