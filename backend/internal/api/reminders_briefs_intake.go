package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type reminderItem struct {
	ID       uuid.UUID  `db:"id" json:"id"`
	NodeID   *uuid.UUID `db:"node_id" json:"node_id,omitempty"`
	Title    string     `db:"title" json:"title"`
	RemindAt time.Time  `db:"remind_at" json:"remind_at"`
	IsDone   bool       `db:"is_done" json:"is_done"`
	Source   *string    `db:"source" json:"source,omitempty"`
}

type createReminderRequest struct {
	NodeID    *uuid.UUID `json:"node_id,omitempty"`
	Title     string     `json:"title"`
	RemindAt  *time.Time `json:"remind_at,omitempty"`
	Source    string     `json:"source,omitempty"`
	Body      *string    `json:"body,omitempty"`
	Important bool       `json:"important,omitempty"`
}

type medicationItem struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	Name             string     `db:"name" json:"name"`
	Dosage           *string    `db:"dosage" json:"dosage,omitempty"`
	Frequency        *string    `db:"frequency" json:"frequency,omitempty"`
	StartDate        *time.Time `db:"start_date" json:"start_date,omitempty"`
	EndDate          *time.Time `db:"end_date" json:"end_date,omitempty"`
	ImageURL         *string    `db:"image_url" json:"image_url,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
}

type createMedicationRequest struct {
	Name      string  `json:"name"`
	Dosage    *string `json:"dosage,omitempty"`
	Frequency *string `json:"frequency,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	ImageURL  *string `json:"image_url,omitempty"`
}

type dailyBrief struct {
	ID         uuid.UUID `db:"id" json:"id"`
	LocalDay   string    `db:"local_day" json:"local_day"`
	Content    string    `db:"content" json:"content"`
	AudioURL   *string   `db:"audio_url" json:"audio_url,omitempty"`
	IsRead     bool      `db:"is_read" json:"is_read"`
	GeneratedAt time.Time `db:"generated_at" json:"generated_at"`
}

type intakeRequest struct {
	Text string `json:"text"`
}

func (s *Server) handleListReminders(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	rows := make([]reminderItem, 0)
	err := s.store.DB.Select(&rows, `
		SELECT id, node_id, title, remind_at, is_done, source
		FROM reminders
		WHERE user_id = $1
		ORDER BY remind_at ASC
		LIMIT 200
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rows)
}

func (s *Server) handleCreateReminder(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req createReminderRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Title) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}

	remindAt := time.Now().Add(2 * time.Hour)
	if req.RemindAt != nil {
		remindAt = *req.RemindAt
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}

	tx, err := s.store.DB.Beginx()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback()

	if err := s.createReminderAndCard(tx, userID, req.NodeID, req.Title, valueOrDefault(req.Body, ""), remindAt, req.Important); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if _, err := tx.Exec(`UPDATE reminders SET source = $1 WHERE user_id = $2 AND title = $3 AND remind_at = $4`, source, userID, req.Title, remindAt.UTC()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{"status": "created"})
}

func (s *Server) handleDoneReminder(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	if _, err := s.store.DB.Exec(`UPDATE reminders SET is_done = true WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleListMedications(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	rows := make([]medicationItem, 0)
	err := s.store.DB.Select(&rows, `
		SELECT id, name, dosage, frequency, start_date, end_date, image_url, created_at
		FROM medications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rows)
}

func (s *Server) handleCreateMedication(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req createMedicationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Name) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "start_date must be YYYY-MM-DD")
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "end_date must be YYYY-MM-DD")
	}

	var id uuid.UUID
	err = s.store.DB.QueryRow(`
		INSERT INTO medications (user_id, name, dosage, frequency, start_date, end_date, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userID, req.Name, req.Dosage, req.Frequency, startDate, endDate, req.ImageURL).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.Frequency != nil && strings.TrimSpace(*req.Frequency) != "" {
		_ = s.createReminderAndCard(s.store.DB, userID, nil, fmt.Sprintf("%s 用量提醒", req.Name), fmt.Sprintf("剂量：%s，频率：%s", valueOrDefault(req.Dosage, "按需"), *req.Frequency), time.Now().Add(4*time.Hour), false)
	}

	return c.JSON(http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleGetDailyBrief(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	localDay := c.QueryParam("local_day")
	if localDay == "" {
		localDay = s.getUserLocalDay(userID)
	}

	var brief dailyBrief
	err := s.store.DB.Get(&brief, `
		SELECT id, local_day, content, audio_url, is_read, generated_at
		FROM daily_briefs
		WHERE user_id = $1 AND local_day = $2
	`, userID, localDay)
	if err == nil {
		return c.JSON(http.StatusOK, brief)
	}
	if err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	generated, genErr := s.generateDailyBrief(userID, localDay)
	if genErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, genErr.Error())
	}
	return c.JSON(http.StatusOK, generated)
}

func (s *Server) handleGenerateDailyBrief(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	localDay := s.getUserLocalDay(userID)
	brief, err := s.generateDailyBrief(userID, localDay)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, brief)
}

func (s *Server) generateDailyBrief(userID uuid.UUID, localDay string) (*dailyBrief, error) {
	var taskCount int
	var reminderCount int
	var cardCount int
	_ = s.store.DB.Get(&taskCount, `SELECT COUNT(*) FROM life_nodes WHERE user_id = $1 AND type = 'task' AND status IN ('active', 'later')`, userID)
	_ = s.store.DB.Get(&reminderCount, `SELECT COUNT(*) FROM reminders WHERE user_id = $1 AND is_done = false`, userID)
	_ = s.store.DB.Get(&cardCount, `SELECT COUNT(*) FROM today_cards WHERE user_id = $1 AND local_day = $2 AND dismissed_at IS NULL`, userID, localDay)

	content := fmt.Sprintf("早上好。今天待办 %d 项，提醒 %d 条，今日卡片 %d 条。先处理最重要的一条。", taskCount, reminderCount, cardCount)
	var id uuid.UUID
	err := s.store.DB.QueryRow(`
		INSERT INTO daily_briefs (user_id, local_day, content)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, local_day)
		DO UPDATE SET content = EXCLUDED.content, generated_at = now()
		RETURNING id
	`, userID, localDay, content).Scan(&id)
	if err != nil {
		return nil, err
	}

	brief := &dailyBrief{ID: id, LocalDay: localDay, Content: content, IsRead: false, GeneratedAt: time.Now().UTC()}
	return brief, nil
}

func (s *Server) handleIntakeIngest(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req intakeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "text is required")
	}

	parsed, _ := s.parseIntakeWithAI(c, text)
	title := truncateTitle(text)
	tags := []string{"手记", "输入框"}
	if parsed.Title != "" {
		title = truncateTitle(parsed.Title)
	}
	if len(parsed.Tags) > 0 {
		tags = parsed.Tags
	}
	nodeType := lifeNodeTypeForIntent(parsed.Intent)

	var nodeID uuid.UUID
	if err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, body, status, tags, attributes)
		VALUES ($1, $2, $3, $4, 'active', $5, $6::jsonb)
		RETURNING id
	`, userID, nodeType, title, &text, pq.Array(tags), encodeJSONB(map[string]any{"source": "smart_input", "intent": parsed.Intent, "confidence": parsed.Confidence})).Scan(&nodeID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	createdReminder := false
	var finalRemindAt *time.Time
	if parsed.ShouldRemind {
		createdReminder = true
		remindAt := time.Now().Add(2 * time.Hour)
		if parsed.RemindAt != nil {
			remindAt = *parsed.RemindAt
		} else if fallbackRemindAt, ok := inferReminderTimeFromText(text); ok {
			remindAt = fallbackRemindAt
		}
		finalRemindAt = &remindAt
		_ = s.createReminderAndCard(s.store.DB, userID, &nodeID, title, text, remindAt, strings.Contains(text, "证件"))
	}

	intentLabel := lifeNodeIntentLabel(parsed.Intent)
	var remindAtStr *string
	if finalRemindAt != nil {
		v := finalRemindAt.UTC().Format(time.RFC3339)
		remindAtStr = &v
	}

	return c.JSON(http.StatusOK, map[string]any{
		"node_id":          nodeID,
		"reminder_created": createdReminder,
		"intent":           parsed.Intent,
		"intent_label":     intentLabel,
		"confidence":       parsed.Confidence,
		"remind_at":        remindAtStr,
	})
}

func (s *Server) handleIntakeUpload(c echo.Context) error {
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

	if !isImageUpload(fileHeader.Filename, content) {
		title := strings.TrimSpace(fileHeader.Filename)
		if title == "" {
			title = "上传文件"
		}
		body := extractTextFromBytes(content)
		if body == "" {
			body = fmt.Sprintf("已上传文件：%s（%d bytes）", title, len(content))
		}

		var nodeID uuid.UUID
		if err := s.store.DB.QueryRow(`
			INSERT INTO life_nodes (user_id, type, title, body, status, tags, attributes)
			VALUES ($1, 'mind', $2, $3, 'active', $4, $5::jsonb)
			RETURNING id
		`, userID, title, body, pq.Array([]string{"上传", "文件"}), encodeJSONB(map[string]any{"source": "upload", "kind": "file"})).Scan(&nodeID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]any{"node_id": nodeID, "kind": "file"})
	}

	analyzed, err := s.forwardAnalyzeToAI(c, userID, fileHeader.Filename, content)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	name, _ := analyzed.Extraction["name"].(string)
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(fileHeader.Filename, ".jpg")
	}
	if strings.TrimSpace(name) == "" {
		name = "上传物品"
	}

	var nodeID uuid.UUID
	if err := s.store.DB.QueryRow(`
		INSERT INTO life_nodes (user_id, type, title, status, tags, attributes)
		VALUES ($1, 'thing', $2, 'active', $3, $4::jsonb)
		RETURNING id
	`, userID, name, pq.Array([]string{"上传", "智能识别"}), encodeJSONB(analyzed.Extraction)).Scan(&nodeID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if _, err := s.store.DB.Exec(`INSERT INTO item_details (node_id, quantity, updated_at) VALUES ($1, 1, now())`, nodeID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if strings.TrimSpace(analyzed.VisualHash) != "" {
		_, _ = s.store.DB.Exec(`INSERT INTO item_visual_fingerprints (user_id, node_id, visual_hash) VALUES ($1, $2, $3)`, userID, nodeID, strings.TrimSpace(analyzed.VisualHash))
	}

	return c.JSON(http.StatusOK, map[string]any{"item_id": nodeID, "extraction": analyzed.Extraction, "duplicates": analyzed.Duplicates})
}

func isImageUpload(filename string, content []byte) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif" || ext == ".heic" {
		return true
	}
	contentType := http.DetectContentType(content)
	return strings.HasPrefix(contentType, "image/")
}

func extractTextFromBytes(content []byte) string {
	if len(content) == 0 {
		return ""
	}
	if !utf8.Valid(content) {
		return ""
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > 800 {
		return string(runes[:800])
	}
	return text
}

func inferReminderTimeFromText(text string) (time.Time, bool) {
	now := time.Now()
	lower := strings.ToLower(text)
	if strings.Contains(lower, "明天") {
		return time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location()), true
	}
	if strings.Contains(lower, "后天") {
		return time.Date(now.Year(), now.Month(), now.Day()+2, 9, 0, 0, 0, now.Location()), true
	}
	if strings.Contains(lower, "今天") || strings.Contains(lower, "提醒") || strings.Contains(lower, "到期") || strings.Contains(lower, "点") {
		return now.Add(2 * time.Hour), true
	}
	return time.Time{}, false
}

type intakeParseResult struct {
	Intent       string
	Title        string
	ShouldRemind bool
	RemindAt     *time.Time
	Tags         []string
	Confidence   float64
}

func (s *Server) parseIntakeWithAI(c echo.Context, text string) (*intakeParseResult, error) {
	aiURL := strings.TrimSpace(os.Getenv("AI_SERVICE_URL"))
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}
	payload, _ := json.Marshal(map[string]any{"text": text, "locale": "zh"})
	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, aiURL+"/intake/parse", bytes.NewReader(payload))
	if err != nil {
		return &intakeParseResult{Intent: "memory", Title: truncateTitle(text), ShouldRemind: false, Confidence: 0.3}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &intakeParseResult{Intent: "memory", Title: truncateTitle(text), ShouldRemind: false, Confidence: 0.3}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return &intakeParseResult{Intent: "memory", Title: truncateTitle(text), ShouldRemind: false, Confidence: 0.3}, fmt.Errorf("ai intake parse failed")
	}

	var parsed struct {
		Intent       string   `json:"intent"`
		Title        string   `json:"title"`
		ShouldRemind bool     `json:"should_remind"`
		RemindAt     *string  `json:"remind_at"`
		Tags         []string `json:"tags"`
		Confidence   float64  `json:"confidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return &intakeParseResult{Intent: "memory", Title: truncateTitle(text), ShouldRemind: false, Confidence: 0.3}, err
	}

	result := &intakeParseResult{
		Intent:       parsed.Intent,
		Title:        parsed.Title,
		ShouldRemind: parsed.ShouldRemind,
		Tags:         parsed.Tags,
		Confidence:   parsed.Confidence,
	}
	if result.Intent == "" {
		result.Intent = "memory"
	}
	if result.Title == "" {
		result.Title = truncateTitle(text)
	}
	if parsed.RemindAt != nil && strings.TrimSpace(*parsed.RemindAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*parsed.RemindAt)); err == nil {
			result.RemindAt = &t
		}
	}
	return result, nil
}

func truncateTitle(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= 24 {
		return text
	}
	return string([]rune(text)[:24])
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func encodeJSONB(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil || len(payload) == 0 || string(payload) == "null" {
		return []byte("{}")
	}
	return payload
}
