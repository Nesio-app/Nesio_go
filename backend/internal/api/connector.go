package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/Nesio-app/Nesio_go/internal/models"
)

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

	// Simplified: just create a connector record
	var id uuid.UUID
	err := s.store.DB.QueryRow(
		"INSERT INTO connectors (user_id, provider) VALUES ($1, $2) RETURNING id",
		userID, provider,
	).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"id":       id,
		"provider": provider,
		"status":   "pending_oauth",
		"auth_url": "/oauth/" + provider + "?connector_id=" + id.String(),
	})
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

	// Trigger async sync job
	// In production, enqueue to Asynq
	_, err = s.store.DB.Exec(
		"UPDATE connectors SET last_sync_at = now() WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "syncing"})
}
