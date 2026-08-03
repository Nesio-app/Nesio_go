package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type relationRow struct {
	ID        uuid.UUID `db:"id" json:"id"`
	FromNode  uuid.UUID `db:"from_node" json:"from_node"`
	ToNode    uuid.UUID `db:"to_node" json:"to_node"`
	Relation  string    `db:"relation" json:"relation"`
}

type createRelationRequest struct {
	FromNode uuid.UUID `json:"from_node"`
	ToNode   uuid.UUID `json:"to_node"`
	Relation string    `json:"relation"`
}

type searchResult struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Type      string    `db:"type" json:"type"`
	Title     string    `db:"title" json:"title"`
	Body      *string   `db:"body" json:"body,omitempty"`
	ImageURL  *string   `db:"image_url" json:"image_url,omitempty"`
	CreatedAt string    `db:"created_at" json:"created_at"`
}

func (s *Server) handleCreateRelation(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req createRelationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Relation == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "relation is required")
	}

	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO relations (from_node, to_node, relation)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT 1 FROM life_nodes WHERE id = $1 AND user_id = $4)
		  AND EXISTS (SELECT 1 FROM life_nodes WHERE id = $2 AND user_id = $4)
		RETURNING id
	`, req.FromNode, req.ToNode, req.Relation, userID).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid relation nodes")
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleListRelations(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	nodeID := c.QueryParam("node_id")

	relations := make([]relationRow, 0)
	err := s.store.DB.Select(&relations, `
		SELECT r.id, r.from_node, r.to_node, r.relation
		FROM relations r
		JOIN life_nodes f ON f.id = r.from_node
		JOIN life_nodes t ON t.id = r.to_node
		WHERE f.user_id = $1 AND t.user_id = $1
		  AND ($2 = '' OR r.from_node = $2::uuid OR r.to_node = $2::uuid)
		ORDER BY r.created_at DESC
		LIMIT 200
	`, userID, nodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, relations)
}

func (s *Server) handleDeleteRelation(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec(`
		DELETE FROM relations r
		USING life_nodes f, life_nodes t
		WHERE r.id = $1
		  AND f.id = r.from_node
		  AND t.id = r.to_node
		  AND f.user_id = $2
		  AND t.user_id = $2
	`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleSearch(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusOK, []searchResult{})
	}
	limit := 20
	if limitStr := strings.TrimSpace(c.QueryParam("limit")); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			if parsed < 1 {
				parsed = 1
			}
			if parsed > 100 {
				parsed = 100
			}
			limit = parsed
		}
	}

	merged, err := s.parallelSearchRecall(userID, query, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, merged)
}
