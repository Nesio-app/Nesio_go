package api

import (
	"net/http"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CreateMemoryRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

func (s *Server) handleListMemories(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var nodes []models.LifeNode
	err := s.store.DB.Select(&nodes, `
		SELECT id, user_id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at
		FROM life_nodes
		WHERE user_id = $1 AND type = 'memory'
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nodes)
}

func (s *Server) handleCreateMemory(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req CreateMemoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	node := models.LifeNode{
		UserID: userID,
		Type:   "memory",
		Title:  req.Title,
		Status: "active",
		Tags:   req.Tags,
		Attributes: map[string]any{"source_text": req.Body},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if req.Body != "" {
		summary := req.Body
		if aiResp, err := callAIService(req.Body, "standard", "memory"); err == nil && aiResp.Content != "" {
			summary = aiResp.Content
		}
		node.Body = &summary
	}

	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, body, status, tags, attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, node.UserID, node.Type, node.Title, node.Body, node.Status, node.Tags, node.Attributes, node.CreatedAt, node.UpdatedAt).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}
