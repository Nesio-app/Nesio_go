package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type askRequest struct {
	Question string `json:"question"`
	UserID   string `json:"user_id,omitempty"`
}

type extractionAnalyzeRequest struct {
	Text string `json:"text"`
}

type mentionNode struct {
	ID    uuid.UUID `db:"id" json:"id"`
	Type  string    `db:"type" json:"type"`
	Title string    `db:"title" json:"title"`
}

func (s *Server) handleAsk(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req askRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "question is required")
	}

	aiURL := strings.TrimSpace(os.Getenv("AI_SERVICE_URL"))
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}

	payload, _ := json.Marshal(map[string]any{
		"question": question,
		"user_id":  userID.String(),
	})

	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, aiURL+"/ask", bytes.NewReader(payload))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fallback, fallbackErr := callAIService(question, "standard", "chat")
		if fallbackErr != nil {
			return echo.NewHTTPError(http.StatusBadGateway, "ask unavailable")
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":    "direct",
			"answer":  fallback.Content,
			"sources": []any{},
		})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fallback, fallbackErr := callAIService(question, "standard", "chat")
		if fallbackErr != nil {
			return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("ask failed: %s", strings.TrimSpace(string(body))))
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":    "direct",
			"answer":  fallback.Content,
			"sources": []any{},
		})
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "invalid ask response")
	}
	return c.JSON(http.StatusOK, data)
}

func (s *Server) handleExtractionAnalyze(c echo.Context) error {
	var req extractionAnalyzeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	parsed, _ := s.parseIntakeWithAI(c, text)
	result := map[string]any{
		"type":       parsed.Intent,
		"name":       parsed.Title,
		"attributes": map[string]any{"intent": parsed.Intent, "confidence": parsed.Confidence, "should_remind": parsed.ShouldRemind},
		"tags":       parsed.Tags,
		"raw_input":  text,
	}
	return c.JSON(http.StatusOK, map[string]any{"extracted": []map[string]any{result}})
}

func (s *Server) handleExtractionUpload(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	note := strings.TrimSpace(c.FormValue("note"))

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid multipart form")
	}
	files := form.File["files"]
	if len(files) == 0 {
		if single, singleErr := c.FormFile("file"); singleErr == nil && single != nil {
			files = []*multipart.FileHeader{single}
		}
	}
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "missing files")
	}

	extracted := make([]map[string]any, 0, len(files))
	for _, fileHeader := range files {
		file, openErr := fileHeader.Open()
		if openErr != nil {
			continue
		}
		content, readErr := io.ReadAll(file)
		_ = file.Close()
		if readErr != nil {
			continue
		}

		if isImageUpload(fileHeader.Filename, content) {
			analyzed, analyzeErr := s.forwardAnalyzeToAI(c, userID, fileHeader.Filename, content)
			if analyzeErr != nil {
				continue
			}
			name, _ := analyzed.Extraction["name"].(string)
			if strings.TrimSpace(name) == "" {
				name = strings.TrimSpace(fileHeader.Filename)
			}
			nodeID, createErr := s.createThingNodeFromExtraction(userID, name, analyzed.Extraction, analyzed.VisualHash)
			if createErr != nil {
				continue
			}
			extracted = append(extracted, map[string]any{
				"id":         nodeID,
				"type":       "thing",
				"name":       name,
				"attributes": analyzed.Extraction,
				"tags":       analyzed.Extraction["tags"],
			})
			continue
		}

		title := strings.TrimSpace(fileHeader.Filename)
		if title == "" {
			title = "上传文件"
		}
		body := extractTextFromBytes(content)
		if note != "" {
			if body != "" {
				body = note + "\n\n" + body
			} else {
				body = note
			}
		}
		nodeID, createErr := s.createMindNodeFromUpload(userID, title, body)
		if createErr != nil {
			continue
		}
		extracted = append(extracted, map[string]any{
			"id":         nodeID,
			"type":       "mind",
			"name":       title,
			"attributes": map[string]any{"source": "upload", "kind": "file"},
			"tags":       []string{"上传", "文件"},
		})
	}

	if len(extracted) == 0 {
		return echo.NewHTTPError(http.StatusBadGateway, "no file extracted")
	}
	return c.JSON(http.StatusOK, map[string]any{"extracted": extracted})
}

func (s *Server) handleNodesMention(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	q := strings.TrimSpace(c.QueryParam("q"))
	if q == "" {
		return c.JSON(http.StatusOK, []mentionNode{})
	}
	limit := 20
	if limitStr := strings.TrimSpace(c.QueryParam("limit")); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			if parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}
	}

	rows := make([]mentionNode, 0)
	err := s.store.DB.Select(&rows, `
		SELECT id, type, title
		FROM life_nodes
		WHERE user_id = $1 AND (title ILIKE '%' || $2 || '%' OR COALESCE(body, '') ILIKE '%' || $2 || '%')
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, q, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rows)
}

func (s *Server) handleMedicineOCR(c echo.Context) error {
	// Compatibility alias: currently reuses medication create endpoint payload.
	return s.handleCreateMedication(c)
}

func (s *Server) handleMedicineReminder(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	medID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var medName string
	if err := s.store.DB.Get(&medName, `SELECT name FROM medications WHERE id = $1 AND user_id = $2`, medID, userID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "medicine not found")
	}

	remindAt := time.Now().Add(2 * time.Hour)
	if err := s.createReminderAndCard(s.store.DB, userID, nil, fmt.Sprintf("吃药提醒：%s", medName), "来自 medicine reminder", remindAt, false); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{"status": "created"})
}

func (s *Server) handleDailyBriefAlias(c echo.Context) error {
	if day := strings.TrimSpace(c.QueryParam("day")); day != "" {
		q := c.QueryParams()
		q.Set("local_day", day)
		c.Request().URL.RawQuery = q.Encode()
	}
	return s.handleGetDailyBrief(c)
}

func (s *Server) handleMarkDailyBriefRead(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	res, err := s.store.DB.Exec(`UPDATE daily_briefs SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "brief not found")
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleExportData(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)

	type exportReminder struct {
		ID         uuid.UUID      `db:"id" json:"id"`
		NodeID     *uuid.UUID     `db:"node_id" json:"node_id,omitempty"`
		Title      string         `db:"title" json:"title"`
		RemindAt   time.Time      `db:"remind_at" json:"remind_at"`
		RepeatRule models.JSONMap `db:"repeat_rule" json:"repeat_rule,omitempty"`
		IsDone     bool           `db:"is_done" json:"is_done"`
		Source     *string        `db:"source" json:"source,omitempty"`
		CreatedAt  time.Time      `db:"created_at" json:"created_at"`
	}
	type exportMedication struct {
		ID               uuid.UUID      `db:"id" json:"id"`
		Name             string         `db:"name" json:"name"`
		Dosage           *string        `db:"dosage" json:"dosage,omitempty"`
		Frequency        *string        `db:"frequency" json:"frequency,omitempty"`
		Schedule         models.JSONMap `db:"schedule" json:"schedule,omitempty"`
		StartDate        *time.Time     `db:"start_date" json:"start_date,omitempty"`
		EndDate          *time.Time     `db:"end_date" json:"end_date,omitempty"`
		OCRRaw           *string        `db:"ocr_raw" json:"ocr_raw,omitempty"`
		ImageURL         *string        `db:"image_url" json:"image_url,omitempty"`
		LocationReminder models.JSONMap `db:"location_reminder" json:"location_reminder,omitempty"`
		CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	}
	type exportConnector struct {
		ID         uuid.UUID  `db:"id" json:"id"`
		Provider   string     `db:"provider" json:"provider"`
		IsActive   bool       `db:"is_active" json:"is_active"`
		LastSyncAt *time.Time `db:"last_sync_at" json:"last_sync_at,omitempty"`
		CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	}

	nodes := make([]models.LifeNode, 0)
	reminders := make([]exportReminder, 0)
	cards := make([]models.TodayCard, 0)
	medications := make([]exportMedication, 0)
	connectors := make([]exportConnector, 0)

	if err := s.store.DB.Select(&nodes, `SELECT id, type, domain, title, body, status, due_date, tags, attributes, created_at, updated_at FROM life_nodes WHERE user_id = $1 ORDER BY created_at DESC`, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := s.store.DB.Select(&reminders, `SELECT id, node_id, title, remind_at, repeat_rule, is_done, source, created_at FROM reminders WHERE user_id = $1 ORDER BY remind_at ASC`, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := s.store.DB.Select(&cards, `SELECT id, local_day, slot, node_id, title, body, severity, action_label, fingerprints, dismissed_at, created_at FROM today_cards WHERE user_id = $1 ORDER BY created_at DESC`, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := s.store.DB.Select(&medications, `SELECT id, name, dosage, frequency, schedule, start_date, end_date, ocr_raw, image_url, location_reminder, created_at FROM medications WHERE user_id = $1 ORDER BY created_at DESC`, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := s.store.DB.Select(&connectors, `SELECT id, provider, is_active, last_sync_at, created_at FROM connectors WHERE user_id = $1 ORDER BY created_at DESC`, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user_id":      userID,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"nodes":        nodes,
		"reminders":    reminders,
		"today_cards":  cards,
		"medications":  medications,
		"connectors":   connectors,
	})
}

func (s *Server) createMindNodeFromUpload(userID uuid.UUID, title, body string) (uuid.UUID, error) {
	if strings.TrimSpace(body) == "" {
		body = fmt.Sprintf("已上传文件：%s", title)
	}
	var nodeID uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, body, status, tags, attributes)
		VALUES ($1, 'mind', $2, $3, 'active', $4, $5::jsonb)
		RETURNING id
	`, userID, title, body, pq.Array([]string{"上传", "文件"}), encodeJSONB(map[string]any{"source": "upload", "kind": "file"})).Scan(&nodeID)
	return nodeID, err
}

func (s *Server) createThingNodeFromExtraction(userID uuid.UUID, name string, extraction map[string]any, visualHash string) (uuid.UUID, error) {
	if strings.TrimSpace(name) == "" {
		name = "上传物品"
	}
	var nodeID uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, status, tags, attributes)
		VALUES ($1, 'thing', $2, 'active', $3, $4::jsonb)
		RETURNING id
	`, userID, name, pq.Array([]string{"上传", "智能识别"}), encodeJSONB(extraction)).Scan(&nodeID)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := s.store.DB.Exec(`INSERT INTO item_details (node_id, quantity, updated_at) VALUES ($1, 1, now())`, nodeID); err != nil {
		return uuid.Nil, err
	}
	if strings.TrimSpace(visualHash) != "" {
		_, _ = s.store.DB.Exec(`INSERT INTO item_visual_fingerprints (user_id, node_id, visual_hash) VALUES ($1, $2, $3)`, userID, nodeID, strings.TrimSpace(visualHash))
	}
	return nodeID, nil
}
