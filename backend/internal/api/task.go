package api

import (
	"net/http"
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

	node := models.LifeNode{
		UserID: userID,
		Type:   "task",
		Title:  req.Title,
		Status: "active",
		Tags:   pq.StringArray(req.Tags),
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

	if req.Status != nil {
		_, err = s.store.DB.Exec("UPDATE life_nodes SET status = $1, updated_at = now() WHERE id = $2 AND user_id = $3", *req.Status, id, userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleListTasks(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	status := c.QueryParam("status")
	if status == "" {
		status = "active"
	}

	var nodes []models.LifeNode
	err := s.store.DB.Select(&nodes, `
		SELECT id, user_id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at
		FROM life_nodes
		WHERE user_id = $1 AND status = $2
		ORDER BY due_date ASC NULLS LAST, created_at DESC
	`, userID, status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nodes)
}
