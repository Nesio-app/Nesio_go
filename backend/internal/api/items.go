package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type createItemRequest struct {
	Name             string     `json:"name"`
	Body             *string    `json:"body,omitempty"`
	RoomID           *uuid.UUID `json:"room_id,omitempty"`
	ContainerID      *uuid.UUID `json:"container_id,omitempty"`
	LocationNote     *string    `json:"location_note,omitempty"`
	ExpiryDate       *string    `json:"expiry_date,omitempty"`
	ExpiryRemindDays *int       `json:"expiry_remind_days,omitempty"`
	IsDocument       bool       `json:"is_document,omitempty"`
	DocumentType     *string    `json:"document_type,omitempty"`
	DocumentNumber   *string    `json:"document_number,omitempty"`
	Quantity         *int       `json:"quantity,omitempty"`
	Unit             *string    `json:"unit,omitempty"`
	PrimaryImageURL  *string    `json:"primary_image_url,omitempty"`
	VisualHash       *string    `json:"visual_hash,omitempty"`
	ReminderLabel    *string    `json:"reminder_label,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
}

type analyzeItemResponse struct {
	Extraction         map[string]any `json:"extraction"`
	Duplicates         []map[string]any `json:"duplicates"`
	VisualHash         string         `json:"visual_hash"`
	SuggestedRoomID    *uuid.UUID     `json:"suggested_room_id,omitempty"`
	SuggestedContainerID *uuid.UUID   `json:"suggested_container_id,omitempty"`
	ImageURL           string         `json:"image_url,omitempty"`
}

type listItemResponse struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	Name             string     `db:"name" json:"name"`
	Type             string     `db:"type" json:"type"`
	Body             *string    `db:"body" json:"body,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	RoomID           *uuid.UUID `db:"room_id" json:"room_id,omitempty"`
	ContainerID      *uuid.UUID `db:"container_id" json:"container_id,omitempty"`
	LocationNote     *string    `db:"location_note" json:"location_note,omitempty"`
	RoomName         *string    `db:"room_name" json:"room_name,omitempty"`
	RoomIcon         *string    `db:"room_icon" json:"room_icon,omitempty"`
	ContainerName    *string    `db:"container_name" json:"container_name,omitempty"`
	ContainerIcon    *string    `db:"container_icon" json:"container_icon,omitempty"`
	ExpiryDate       *time.Time `db:"expiry_date" json:"expiry_date,omitempty"`
	IsDocument       bool       `db:"is_document" json:"is_document"`
	DocumentType     *string    `db:"document_type" json:"document_type,omitempty"`
	DocumentNumber   *string    `db:"document_number" json:"document_number,omitempty"`
	Quantity         int        `db:"quantity" json:"quantity"`
	Unit             *string    `db:"unit" json:"unit,omitempty"`
	PrimaryImageURL  *string    `db:"primary_image_url" json:"primary_image_url,omitempty"`
	DaysUntilExpiry  *int       `db:"days_until_expiry" json:"days_until_expiry,omitempty"`
}

func (s *Server) handleListItems(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	roomID := c.QueryParam("room")
	containerID := c.QueryParam("container")
	search := c.QueryParam("q")

	items := make([]listItemResponse, 0)
	err := s.store.DB.Select(&items, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			CASE
				WHEN i.expiry_date IS NOT NULL THEN (i.expiry_date - CURRENT_DATE)
				ELSE NULL
			END AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1
		  AND n.type = 'thing'
		  AND ($2 = '' OR i.room_id = $2::uuid)
		  AND ($3 = '' OR i.container_id = $3::uuid)
		  AND ($4 = '' OR n.title ILIKE '%' || $4 || '%')
		ORDER BY
		  CASE WHEN i.expiry_date IS NOT NULL AND i.expiry_date <= CURRENT_DATE + 7 THEN 0 ELSE 1 END,
		  i.expiry_date ASC NULLS LAST,
		  n.created_at DESC
	`, userID, roomID, containerID, search)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, items)
}

func (s *Server) handleCreateItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req createItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	tags := pq.StringArray(req.Tags)
	if tags == nil {
		tags = pq.StringArray{}
	}

	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ExpiryDate)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "expiry_date must be YYYY-MM-DD")
		}
		expiryDate = &parsed
	}

	expiryRemindDays := 30
	if req.ExpiryRemindDays != nil {
		expiryRemindDays = *req.ExpiryRemindDays
	}

	quantity := 1
	if req.Quantity != nil && *req.Quantity > 0 {
		quantity = *req.Quantity
	}

	unit := req.Unit
	if unit == nil || *unit == "" {
		defaultUnit := "个"
		unit = &defaultUnit
	}

	tx, err := s.store.DB.Beginx()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback()

	var nodeID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, body, status, tags, attributes)
		VALUES ($1, 'thing', $2, $3, 'active', $4, $5::jsonb)
		RETURNING id
	`, userID, req.Name, req.Body, pq.Array(tags), encodeJSONB(map[string]any{})).Scan(&nodeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_, err = tx.Exec(`
		INSERT INTO item_details (
			node_id, room_id, container_id, location_note,
			expiry_date, expiry_remind_days, is_document, document_type, document_number,
			quantity, unit, primary_image_url, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
	`, nodeID, req.RoomID, req.ContainerID, req.LocationNote, expiryDate, expiryRemindDays, req.IsDocument, req.DocumentType, req.DocumentNumber, quantity, unit, req.PrimaryImageURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, tag := range req.Tags {
		if tag == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO item_tags (item_id, tag, source) VALUES ($1, $2, 'manual') ON CONFLICT DO NOTHING`, nodeID, tag); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.VisualHash != nil && strings.TrimSpace(*req.VisualHash) != "" {
		if _, err := tx.Exec(`
			INSERT INTO item_visual_fingerprints (user_id, node_id, visual_hash, image_url)
			VALUES ($1, $2, $3, $4)
		`, userID, nodeID, strings.TrimSpace(*req.VisualHash), req.PrimaryImageURL); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if expiryDate != nil {
		remindAt := expiryDate.AddDate(0, 0, -expiryRemindDays)
		title := fmt.Sprintf("%s 即将到期", req.Name)
		body := fmt.Sprintf("%s 将在 %s 到期", req.Name, expiryDate.Format("2006-01-02"))
		if req.IsDocument {
			title = fmt.Sprintf("证件到期提醒：%s", req.Name)
			body = fmt.Sprintf("证件 %s 将在 %s 到期，请提前办理", req.Name, expiryDate.Format("2006-01-02"))
		}
		if err := s.createReminderAndCard(tx, userID, &nodeID, title, body, remindAt, req.IsDocument); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.ReminderLabel != nil && strings.TrimSpace(*req.ReminderLabel) != "" && expiryDate == nil {
		remindAt := time.Now().Add(2 * time.Hour)
		if err := s.createReminderAndCard(tx, userID, &nodeID, strings.TrimSpace(*req.ReminderLabel), fmt.Sprintf("与物品 %s 相关", req.Name), remindAt, false); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": nodeID})
}

func (s *Server) handleGetItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var item listItemResponse
	err = s.store.DB.Get(&item, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			CASE
				WHEN i.expiry_date IS NOT NULL THEN (i.expiry_date - CURRENT_DATE)
				ELSE NULL
			END AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing' AND n.id = $2
	`, userID, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "item not found")
	}

	return c.JSON(http.StatusOK, item)
}

func (s *Server) handleUpdateItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req createItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ExpiryDate)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "expiry_date must be YYYY-MM-DD")
		}
		expiryDate = &parsed
	}

	if req.Name != "" {
		if _, err := s.store.DB.Exec(`UPDATE life_nodes SET title = $1, body = $2, updated_at = now() WHERE id = $3 AND user_id = $4 AND type = 'thing'`, req.Name, req.Body, id, userID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	_, err = s.store.DB.Exec(`
		UPDATE item_details
		SET room_id = COALESCE($1, room_id),
		    container_id = COALESCE($2, container_id),
		    location_note = COALESCE($3, location_note),
		    expiry_date = COALESCE($4, expiry_date),
		    is_document = $5,
		    document_type = COALESCE($6, document_type),
		    document_number = COALESCE($7, document_number),
		    quantity = COALESCE($8, quantity),
		    unit = COALESCE($9, unit),
		    primary_image_url = COALESCE($10, primary_image_url),
		    updated_at = now()
		WHERE node_id = $11 AND EXISTS (
			SELECT 1 FROM life_nodes n WHERE n.id = $11 AND n.user_id = $12 AND n.type = 'thing'
		)
	`, req.RoomID, req.ContainerID, req.LocationNote, expiryDate, req.IsDocument, req.DocumentType, req.DocumentNumber, req.Quantity, req.Unit, req.PrimaryImageURL, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleDeleteItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	_, err = s.store.DB.Exec(`DELETE FROM life_nodes WHERE id = $1 AND user_id = $2 AND type = 'thing'`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleWhereIsItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found", "answer": "请输入要查找的物品"})
	}

	items := make([]listItemResponse, 0)
	err := s.store.DB.Select(&items, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			NULL::int AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing'
		  AND (n.title ILIKE '%' || $2 || '%' OR n.body ILIKE '%' || $2 || '%')
		ORDER BY n.updated_at DESC
		LIMIT 5
	`, userID, query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if len(items) == 0 {
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found", "answer": "没有找到这个物品，要添加吗？"})
	}

	if len(items) == 1 {
		item := items[0]
		roomName := "未知房间"
		containerName := "未知容器"
		if item.RoomName != nil && *item.RoomName != "" {
			roomName = *item.RoomName
		}
		if item.ContainerName != nil && *item.ContainerName != "" {
			containerName = *item.ContainerName
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":   "found",
			"answer": fmt.Sprintf("%s 放在 %s %s", item.Name, roomName, containerName),
			"item":   item,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"type":   "multiple",
		"answer": fmt.Sprintf("找到了 %d 个相关物品", len(items)),
		"items":  items,
	})
}

func (s *Server) handleAnalyzeItem(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "missing file")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if aiResp, err := s.forwardAnalyzeToAI(c, userID, fileHeader.Filename, content); err == nil {
		duplicates, _ := s.findDuplicates(c, userID, aiResp.VisualHash, aiResp.Extraction)
		aiResp.Duplicates = duplicates
		return c.JSON(http.StatusOK, aiResp)
	}

	name := strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename))
	if name == "" {
		name = "新物品"
	}

	fallback := analyzeItemResponse{
		Extraction: map[string]any{
			"name":                name,
			"category":            "other",
			"quantity":            1,
			"unit":                "piece",
			"is_document":         false,
			"suggested_room":      "storage",
			"suggested_container": "drawer",
			"tags":                []string{"拍照", "待确认"},
		},
		Duplicates: []map[string]any{},
		VisualHash: fmt.Sprintf("fallback-%d", len(content)),
	}
	duplicates, _ := s.findDuplicates(c, userID, fallback.VisualHash, fallback.Extraction)
	fallback.Duplicates = duplicates

	return c.JSON(http.StatusOK, fallback)
}

func (s *Server) handleWhereIsItemPhoto(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "missing file")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	analyzed, err := s.forwardAnalyzeToAI(c, userID, fileHeader.Filename, content)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "photo analyze failed")
	}
	name, _ := analyzed.Extraction["name"].(string)
	if strings.TrimSpace(name) == "" {
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found", "answer": "无法识别图片中的物品"})
	}

	items := make([]listItemResponse, 0)
	err = s.store.DB.Select(&items, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			NULL::int AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing' AND n.title ILIKE '%' || $2 || '%'
		ORDER BY n.updated_at DESC
		LIMIT 3
	`, userID, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if len(items) == 0 {
		return c.JSON(http.StatusOK, map[string]any{"type": "not_found", "answer": "家里还没有这个物品记录"})
	}
	if len(items) == 1 {
		item := items[0]
		roomName := "未知房间"
		containerName := "未知容器"
		if item.RoomName != nil && *item.RoomName != "" {
			roomName = *item.RoomName
		}
		if item.ContainerName != nil && *item.ContainerName != "" {
			containerName = *item.ContainerName
		}
		return c.JSON(http.StatusOK, map[string]any{"type": "found", "answer": fmt.Sprintf("%s 放在 %s %s", item.Name, roomName, containerName), "item": item})
	}
	return c.JSON(http.StatusOK, map[string]any{"type": "multiple", "answer": fmt.Sprintf("找到了 %d 个相关物品", len(items)), "items": items})
}

func (s *Server) forwardAnalyzeToAI(c echo.Context, userID uuid.UUID, filename string, content []byte) (*analyzeItemResponse, error) {
	aiURL := strings.TrimSpace(os.Getenv("AI_SERVICE_URL"))
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := writer.WriteField("user_id", userID.String()); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, aiURL+"/items/analyze", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai analyze failed: %s", resp.Status)
	}

	var result analyzeItemResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Extraction == nil {
		result.Extraction = map[string]any{}
	}
	if result.Duplicates == nil {
		result.Duplicates = []map[string]any{}
	}
	return &result, nil
}

func (s *Server) findDuplicates(c echo.Context, userID uuid.UUID, visualHash string, extraction map[string]any) ([]map[string]any, error) {
	duplicates := make([]map[string]any, 0)
	if strings.TrimSpace(visualHash) != "" {
		rows := make([]struct {
			ID            uuid.UUID `db:"id"`
			Name          string    `db:"name"`
			RoomName      *string   `db:"room_name"`
			ContainerName *string   `db:"container_name"`
		}, 0)
		err := s.store.DB.Select(&rows, `
			SELECT n.id, n.title as name, r.name AS room_name, ct.name AS container_name
			FROM item_visual_fingerprints v
			JOIN life_nodes n ON n.id = v.node_id
			LEFT JOIN item_details i ON i.node_id = n.id
			LEFT JOIN rooms r ON r.id = i.room_id
			LEFT JOIN containers ct ON ct.id = i.container_id
			WHERE v.user_id = $1 AND v.visual_hash = $2
			ORDER BY v.created_at DESC
			LIMIT 3
		`, userID, strings.TrimSpace(visualHash))
		if err == nil {
			for _, row := range rows {
				duplicates = append(duplicates, map[string]any{
					"id": row.ID,
					"name": row.Name,
					"room_name": row.RoomName,
					"container_name": row.ContainerName,
					"match_type": "visual",
				})
			}
		}
	}

	if len(duplicates) == 0 {
		if name, ok := extraction["name"].(string); ok && strings.TrimSpace(name) != "" {
			rows := make([]struct {
				ID            uuid.UUID `db:"id"`
				Name          string    `db:"name"`
				RoomName      *string   `db:"room_name"`
				ContainerName *string   `db:"container_name"`
			}, 0)
			err := s.store.DB.Select(&rows, `
				SELECT n.id, n.title as name, r.name AS room_name, ct.name AS container_name
				FROM life_nodes n
				LEFT JOIN item_details i ON i.node_id = n.id
				LEFT JOIN rooms r ON r.id = i.room_id
				LEFT JOIN containers ct ON ct.id = i.container_id
				WHERE n.user_id = $1 AND n.type = 'thing' AND n.title ILIKE '%' || $2 || '%'
				ORDER BY n.updated_at DESC
				LIMIT 3
			`, userID, strings.TrimSpace(name))
			if err == nil {
				for _, row := range rows {
					duplicates = append(duplicates, map[string]any{
						"id": row.ID,
						"name": row.Name,
						"room_name": row.RoomName,
						"container_name": row.ContainerName,
						"match_type": "name",
					})
				}
			}
		}
	}

	return duplicates, nil
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func (s *Server) createReminderAndCard(tx sqlExecer, userID uuid.UUID, nodeID *uuid.UUID, title, body string, remindAt time.Time, critical bool) error {
	slot := "guidance"
	severity := 2
	if critical {
		slot = "pinned"
		severity = 3
	}
	localDay := s.getUserLocalDay(userID)
	var bodyPtr *string
	if strings.TrimSpace(body) != "" {
		bodyPtr = &body
	}

	if _, err := tx.Exec(`
		INSERT INTO reminders (user_id, node_id, title, remind_at, source)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, nodeID, title, remindAt.UTC(), "item"); err != nil {
		return err
	}

	_, err := tx.Exec(`
		INSERT INTO today_cards (user_id, local_day, slot, node_id, title, body, severity, fingerprints)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, localDay, slot, nodeID, title, bodyPtr, severity, pq.Array([]string{"reminder:" + title}))
	return err
}

func (s *Server) handleListExpiringItems(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	days := 30

	rows := make([]listItemResponse, 0)
	err := s.store.DB.Select(&rows, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			CASE WHEN i.expiry_date IS NOT NULL THEN (i.expiry_date - CURRENT_DATE) ELSE NULL END AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing'
		  AND i.expiry_date IS NOT NULL
		  AND i.expiry_date <= CURRENT_DATE + $2
		ORDER BY i.expiry_date ASC
	`, userID, days)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rows)
}

func (s *Server) handleListDocuments(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)

	rows := make([]listItemResponse, 0)
	err := s.store.DB.Select(&rows, `
		SELECT
			n.id,
			n.title AS name,
			n.type,
			n.body,
			n.created_at,
			i.room_id,
			i.container_id,
			i.location_note,
			r.name AS room_name,
			r.icon AS room_icon,
			ct.name AS container_name,
			ct.icon AS container_icon,
			i.expiry_date,
			i.is_document,
			i.document_type,
			i.document_number,
			i.quantity,
			i.unit,
			i.primary_image_url,
			CASE WHEN i.expiry_date IS NOT NULL THEN (i.expiry_date - CURRENT_DATE) ELSE NULL END AS days_until_expiry
		FROM life_nodes n
		JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing' AND i.is_document = true
		ORDER BY i.expiry_date ASC NULLS LAST, n.updated_at DESC
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rows)
}

func (s *Server) handleMarkItemDuplicate(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	sourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var req struct {
		TargetItemID string `json:"target_item_id"`
		Increment    int    `json:"increment"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	targetID, err := uuid.Parse(strings.TrimSpace(req.TargetItemID))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid target_item_id")
	}
	if sourceID == targetID {
		return echo.NewHTTPError(http.StatusBadRequest, "source and target cannot be same")
	}

	increment := req.Increment
	if increment <= 0 {
		increment = 1
	}

	tx, err := s.store.DB.Beginx()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback()

	var sourceOwned bool
	if err := tx.Get(&sourceOwned, `
		SELECT EXISTS(
			SELECT 1 FROM life_nodes
			WHERE id = $1 AND user_id = $2 AND type = 'thing'
		)
	`, sourceID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !sourceOwned {
		return echo.NewHTTPError(http.StatusNotFound, "source item not found")
	}

	var targetOwned bool
	if err := tx.Get(&targetOwned, `
		SELECT EXISTS(
			SELECT 1 FROM life_nodes
			WHERE id = $1 AND user_id = $2 AND type = 'thing'
		)
	`, targetID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !targetOwned {
		return echo.NewHTTPError(http.StatusNotFound, "target item not found")
	}

	if _, err := tx.Exec(`
		UPDATE item_details
		SET quantity = COALESCE(quantity, 0) + $1,
		    updated_at = now()
		WHERE node_id = $2
	`, increment, targetID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if _, err := tx.Exec(`DELETE FROM life_nodes WHERE id = $1 AND user_id = $2 AND type = 'thing'`, sourceID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "merged", "target_item_id": targetID, "increment": increment})
}

func (s *Server) handleSnoozeExpiry(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	_, err = s.store.DB.Exec(`
		UPDATE item_details
		SET expiry_remind_days = expiry_remind_days + 7,
		    updated_at = now()
		WHERE node_id = $1
		AND EXISTS (
			SELECT 1 FROM life_nodes WHERE id = $1 AND user_id = $2 AND type = 'thing'
		)
	`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "snoozed", "days": 7})
}
