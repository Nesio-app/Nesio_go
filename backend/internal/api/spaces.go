package api

import (
	"net/http"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type defaultRoomSeed struct {
	Name      string
	Icon      string
	SortOrder int
}

type defaultContainerSeed struct {
	Name     string
	Icon     string
	RoomName *string
	SortOrder int
}

var defaultRooms = []defaultRoomSeed{
	{Name: "客厅", Icon: "🛋️", SortOrder: 1},
	{Name: "卧室", Icon: "🛏️", SortOrder: 2},
	{Name: "厨房", Icon: "🍳", SortOrder: 3},
	{Name: "书房", Icon: "📚", SortOrder: 4},
	{Name: "卫生间", Icon: "🚿", SortOrder: 5},
	{Name: "玄关", Icon: "🚪", SortOrder: 6},
	{Name: "储藏室", Icon: "📦", SortOrder: 7},
	{Name: "阳台", Icon: "🌿", SortOrder: 8},
}

func strPtr(s string) *string { return &s }

var defaultContainers = []defaultContainerSeed{
	{Name: "冰箱", Icon: "❄️", RoomName: strPtr("厨房"), SortOrder: 1},
	{Name: "橱柜", Icon: "🗄️", RoomName: strPtr("厨房"), SortOrder: 2},
	{Name: "衣柜", Icon: "👔", RoomName: strPtr("卧室"), SortOrder: 1},
	{Name: "床头柜", Icon: "🛏️", RoomName: strPtr("卧室"), SortOrder: 2},
	{Name: "书架", Icon: "📚", RoomName: strPtr("书房"), SortOrder: 1},
	{Name: "鞋柜", Icon: "👟", RoomName: strPtr("玄关"), SortOrder: 1},
	{Name: "药箱", Icon: "💊", RoomName: nil, SortOrder: 1},
}

func (s *Server) ensureDefaultSpaces(userID uuid.UUID) error {
	tx, err := s.store.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var roomCount int
	if err := tx.Get(&roomCount, `SELECT COUNT(1) FROM rooms WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if roomCount == 0 {
		for _, room := range defaultRooms {
			if _, err := tx.Exec(`
				INSERT INTO rooms (user_id, name, icon, sort_order)
				VALUES ($1, $2, $3, $4)
			`, userID, room.Name, room.Icon, room.SortOrder); err != nil {
				return err
			}
		}
	}

	var containerCount int
	if err := tx.Get(&containerCount, `SELECT COUNT(1) FROM containers WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if containerCount == 0 {
		for _, container := range defaultContainers {
			if container.RoomName == nil {
				if _, err := tx.Exec(`
					INSERT INTO containers (user_id, room_id, name, icon, parent_container_id, sort_order)
					VALUES ($1, NULL, $2, $3, NULL, $4)
				`, userID, container.Name, container.Icon, container.SortOrder); err != nil {
					return err
				}
				continue
			}

			if _, err := tx.Exec(`
				INSERT INTO containers (user_id, room_id, name, icon, parent_container_id, sort_order)
				SELECT $1, r.id, $2, $3, NULL, $4
				FROM rooms r
				WHERE r.user_id = $1 AND r.name = $5
				LIMIT 1
			`, userID, container.Name, container.Icon, container.SortOrder, *container.RoomName); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

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
	if err := s.ensureDefaultSpaces(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
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
	if err := s.ensureDefaultSpaces(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
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
