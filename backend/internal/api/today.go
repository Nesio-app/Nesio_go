package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleGetToday(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	localDay := c.QueryParam("local_day")
	if localDay == "" {
		localDay = s.getUserLocalDay(userID)
	}

	slot := c.QueryParam("slot")
	minSeverityStr := c.QueryParam("min_severity")
	minSeverity := 0
	if minSeverityStr != "" {
		minSeverity, _ = strconv.Atoi(minSeverityStr)
	}

	var cards []models.TodayCard
	query := `
		SELECT id, user_id, local_day, slot, node_id, title, body, severity, action_label, fingerprints, dismissed_at, created_at
		FROM today_cards
		WHERE user_id = $1 AND local_day = $2 AND dismissed_at IS NULL
		AND ($3 = '' OR slot = $3)
		AND ($4::int = 0 OR severity >= $4::int)
		ORDER BY 
			CASE slot 
				WHEN 'pinned' THEN 0 
				WHEN 'guidance' THEN 1 
				WHEN 'task' THEN 2 
			END,
			severity DESC,
			created_at ASC
	`
	err := s.store.DB.Select(&cards, query, userID, localDay, slot, minSeverity)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"cards": cards, "local_day": localDay})
}

func (s *Server) getUserLocalDay(userID uuid.UUID) string {
	var timezone string
	err := s.store.DB.Get(&timezone, "SELECT timezone FROM users WHERE id = $1", userID)
	if err != nil || timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("UTC", 0)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func (s *Server) handleDismissCard(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec(
		"UPDATE today_cards SET dismissed_at = now() WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleMuteCard(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Get fingerprints from card
	var fps []string
	err = s.store.DB.Select(&fps, "SELECT UNNEST(fingerprints) FROM today_cards WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Mark fingerprints as muted
	for _, fp := range fps {
		_, _ = s.store.DB.Exec(`
			INSERT INTO fingerprints (user_id, hash, source, is_muted)
			VALUES ($1, $2, 'card', true)
			ON CONFLICT (user_id, hash) DO UPDATE SET is_muted = true
		`, userID, fp)
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDoneCard(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Update card and associated life_node
	_, err = s.store.DB.Exec(
		"UPDATE today_cards SET dismissed_at = now() WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Also mark node as done if linked
	_, _ = s.store.DB.Exec(`
		UPDATE life_nodes SET status = 'done' 
		WHERE id = (SELECT node_id FROM today_cards WHERE id = $1)
	`, id)

	return c.NoContent(http.StatusNoContent)
}
