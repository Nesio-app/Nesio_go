package api

import (
	"net/http"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type upsertRoomRequest struct {
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	SortOrder *int   `json:"sort_order,omitempty"`
}

type upsertContainerRequest struct {
	Name              string     `json:"name"`
	Icon              string     `json:"icon,omitempty"`
	RoomID            *uuid.UUID `json:"room_id,omitempty"`
	ParentContainerID *uuid.UUID `json:"parent_container_id,omitempty"`
	SortOrder         *int       `json:"sort_order,omitempty"`
}

func (s *Server) handleListRooms(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	rooms := make([]models.Room, 0)
	err := s.store.DB.Select(&rooms, `
		SELECT id, user_id, name, icon, sort_order, created_at
		FROM rooms
		WHERE user_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rooms)
}

func (s *Server) handleCreateRoom(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req upsertRoomRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	icon := req.Icon
	if icon == "" {
		icon = "🏠"
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO rooms (user_id, name, icon, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, req.Name, icon, sortOrder).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdateRoom(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req upsertRoomRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	_, err = s.store.DB.Exec(`
		UPDATE rooms
		SET name = COALESCE(NULLIF($1, ''), name),
		    icon = COALESCE(NULLIF($2, ''), icon),
		    sort_order = COALESCE($3, sort_order)
		WHERE id = $4 AND user_id = $5
	`, req.Name, req.Icon, req.SortOrder, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDeleteRoom(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec(`DELETE FROM rooms WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleListContainers(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	room := c.QueryParam("room")
	containers := make([]models.Container, 0)
	err := s.store.DB.Select(&containers, `
		SELECT id, user_id, room_id, name, icon, parent_container_id, sort_order, created_at
		FROM containers
		WHERE user_id = $1
		AND ($2 = '' OR room_id = $2::uuid)
		ORDER BY sort_order ASC, created_at ASC
	`, userID, room)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, containers)
}

func (s *Server) handleCreateContainer(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req upsertContainerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	icon := req.Icon
	if icon == "" {
		icon = "📦"
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO containers (user_id, room_id, name, icon, parent_container_id, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, req.RoomID, req.Name, icon, req.ParentContainerID, sortOrder).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdateContainer(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req upsertContainerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	_, err = s.store.DB.Exec(`
		UPDATE containers
		SET name = COALESCE(NULLIF($1, ''), name),
		    icon = COALESCE(NULLIF($2, ''), icon),
		    room_id = COALESCE($3, room_id),
		    parent_container_id = COALESCE($4, parent_container_id),
		    sort_order = COALESCE($5, sort_order)
		WHERE id = $6 AND user_id = $7
	`, req.Name, req.Icon, req.RoomID, req.ParentContainerID, req.SortOrder, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDeleteContainer(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec(`DELETE FROM containers WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
