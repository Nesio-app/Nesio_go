package api

import (
	"net/http"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type DomainDetailResponse struct {
	Domain string             `json:"domain"`
	Tasks  []models.LifeNode  `json:"tasks"`
	Memory []models.LifeNode  `json:"memory"`
	Today  []models.TodayCard `json:"today"`
}

type DomainNodeUpdateRequest struct {
	Title   *string    `json:"title,omitempty"`
	Body    *string    `json:"body,omitempty"`
	Status  *string    `json:"status,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

func (s *Server) handleDomainDetail(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	domain := c.Param("domain")

	tasks := []models.LifeNode{}
	memory := []models.LifeNode{}
	todayCards := []models.TodayCard{}

	if err := s.store.DB.Select(&tasks, `
		SELECT id, user_id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at
		FROM life_nodes
		WHERE user_id = $1 AND domain = $2 AND type = 'task'
		ORDER BY updated_at DESC
	`, userID, domain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := s.store.DB.Select(&memory, `
		SELECT id, user_id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at
		FROM life_nodes
		WHERE user_id = $1 AND domain = $2 AND (type = 'memory' OR type = 'mind')
		ORDER BY updated_at DESC
	`, userID, domain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := s.store.DB.Select(&todayCards, `
		SELECT id, user_id, local_day, slot, node_id, title, body, severity, action_label, fingerprints, dismissed_at, created_at
		FROM today_cards
		WHERE user_id = $1 AND dismissed_at IS NULL AND title ILIKE '%' || $2 || '%'
		ORDER BY created_at DESC
		LIMIT 10
	`, userID, domain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, DomainDetailResponse{Domain: domain, Tasks: tasks, Memory: memory, Today: todayCards})
}

func (s *Server) handleCreateDomainTask(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	domain := c.Param("domain")
	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Domain = &domain
	return s.createTaskForUser(c, userID, req)
}

func (s *Server) handleCreateDomainMemory(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	domain := c.Param("domain")
	var req CreateMemoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return s.createMemoryForUser(c, userID, req, &domain)
}

func (s *Server) handleUpdateDomainNode(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	domain := c.Param("domain")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req DomainNodeUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	_, err = s.store.DB.Exec(`
		UPDATE life_nodes
		SET title = COALESCE($1, title),
		    body = COALESCE($2, body),
		    status = COALESCE($3, status),
		    due_date = COALESCE($4, due_date),
		    updated_at = now()
		WHERE id = $5 AND user_id = $6 AND domain = $7
	`, req.Title, req.Body, req.Status, req.DueDate, id, userID, domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDeleteDomainNode(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	domain := c.Param("domain")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	_, err = s.store.DB.Exec(`DELETE FROM life_nodes WHERE id = $1 AND user_id = $2 AND domain = $3`, id, userID, domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	_, _ = s.store.DB.Exec(`DELETE FROM today_cards WHERE node_id = $1 AND user_id = $2`, id, userID)
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) insertTodayReflection(userID uuid.UUID, domain, title string, body *string, nodeID uuid.UUID) error {
	localDay := s.getUserLocalDay(userID)
	_, err := s.store.DB.Exec(`
		INSERT INTO today_cards (user_id, local_day, slot, node_id, title, body, severity, fingerprints)
		VALUES ($1, $2, 'task', $3, $4, $5, 1, $6)
	`, userID, localDay, nodeID, title, body, pq.Array([]string{"domain-reflection:" + nodeID.String()}))
	return err
}
