package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type CreateTaskRequest struct {
	Title   string     `json:"title" validate:"required"`
	Type    string     `json:"type"`
	Domain  *string    `json:"domain,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
	Tags    []string   `json:"tags,omitempty"`
}

type UpdateTaskRequest struct {
	Status  *string    `json:"status,omitempty"`
	Title   *string    `json:"title,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

func (s *Server) handleCreateTask(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return s.createTaskForUser(c, userID, req)
}

func (s *Server) createTaskForUser(c echo.Context, userID uuid.UUID, req CreateTaskRequest) error {
	tags := pq.StringArray(req.Tags)
	if tags == nil {
		tags = pq.StringArray{}
	}
	node := models.LifeNode{
		UserID: userID,
		Type:   "task",
		Title:  req.Title,
		Status: "active",
		Tags:   tags,
	}
	if req.Type != "" {
		node.Type = req.Type
	}
	if req.Domain != nil {
		node.Domain = req.Domain
	}
	if req.DueDate != nil {
		node.DueDate = req.DueDate
	}

	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, domain, title, status, due_date, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, node.UserID, node.Type, node.Domain, node.Title, node.Status, node.DueDate, pq.Array(node.Tags)).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if node.Domain != nil {
		_ = s.insertTodayReflection(userID, *node.Domain, node.Title, node.Body, id)
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdateTask(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Title != nil {
		trimmedTitle := strings.TrimSpace(*req.Title)
		if trimmedTitle == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "title must not be empty")
		}
		req.Title = &trimmedTitle
	}

	if req.Status == nil && req.Title == nil && req.DueDate == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one field is required")
	}

	result, err := s.store.DB.Exec(`
		UPDATE life_nodes
		SET title = COALESCE($1, title),
		    status = COALESCE($2, status),
		    due_date = COALESCE($3, due_date),
		    updated_at = now()
		WHERE id = $4 AND user_id = $5 AND type = 'task'
	`, req.Title, req.Status, req.DueDate, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if updated == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "task not found")
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleListTasks(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	status := c.QueryParam("status")
	if status == "" {
		status = "active"
	}
	domain := c.QueryParam("domain")

	var nodes []models.LifeNode
	query := `
		SELECT id, user_id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at
		FROM life_nodes
		WHERE user_id = $1 AND type = 'task' AND status = $2
		AND ($3 = '' OR domain = $3)
		ORDER BY due_date ASC NULLS LAST, created_at DESC
	`
	err := s.store.DB.Select(&nodes, query, userID, status, domain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nodes)
}
