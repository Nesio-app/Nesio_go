package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/Nesio-app/Nesio_go/internal/models"
)

func (s *Server) handleGetMe(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var user models.User
	err := s.store.DB.Get(&user, "SELECT id, email, timezone, locale, created_at FROM users WHERE id = $1", userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, user)
}

type UpdateMeRequest struct {
	Timezone *string `json:"timezone,omitempty"`
	Locale   *string `json:"locale,omitempty"`
}

func (s *Server) handleUpdateMe(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req UpdateMeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// Simplified: just return current user for now
	return s.handleGetMe(c)
}
