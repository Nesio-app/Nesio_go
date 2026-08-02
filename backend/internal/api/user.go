package api

import (
	"fmt"
	"net/http"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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

	query := "UPDATE users SET"
	args := []any{}
	idx := 1
	if req.Timezone != nil {
		query += fmt.Sprintf(" timezone = $%d", idx)
		args = append(args, *req.Timezone)
		idx++
	}
	if req.Locale != nil {
		if idx > 1 {
			query += ","
		}
		query += fmt.Sprintf(" locale = $%d", idx)
		args = append(args, *req.Locale)
		idx++
	}
	if len(args) == 0 {
		return s.handleGetMe(c)
	}
	query += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, userID)

	_, err := s.store.DB.Exec(query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return s.handleGetMe(c)
}
