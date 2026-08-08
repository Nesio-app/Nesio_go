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

type askServiceRequest struct {
	Question string `json:"question"`
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

	sources, contextText, err := s.buildAskContext(c, userID, question)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	aiURL := strings.TrimSpace(os.Getenv("AI_SERVICE_URL"))
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}

	payload, err := json.Marshal(askServiceRequest{Question: contextText})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, aiURL+"/ask", bytes.NewReader(payload))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fallback, fallbackErr := callAIService(question, "standard", "chat")
		if fallbackErr != nil {
			return c.JSON(http.StatusOK, map[string]any{
				"type":    "direct",
				"answer":  synthesizeAskAnswer(question, sources),
				"sources": sources,
			})
		}
		if strings.Contains(strings.ToLower(fallback.Content), "ai not configured") {
			return c.JSON(http.StatusOK, map[string]any{
				"type":    "direct",
				"answer":  synthesizeAskAnswer(question, sources),
				"sources": sources,
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":    "direct",
			"answer":  fallback.Content,
			"sources": sources,
		})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fallback, fallbackErr := callAIService(question, "standard", "chat")
		if fallbackErr != nil {
			return c.JSON(http.StatusOK, map[string]any{
				"type":    "direct",
				"answer":  synthesizeAskAnswer(question, sources),
				"sources": sources,
			})
		}
		if strings.Contains(strings.ToLower(fallback.Content), "ai not configured") {
			return c.JSON(http.StatusOK, map[string]any{
				"type":    "direct",
				"answer":  synthesizeAskAnswer(question, sources),
				"sources": sources,
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"type":    "direct",
			"answer":  fallback.Content,
			"sources": sources,
		})
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "invalid ask response")
	}
	if _, ok := data["sources"]; !ok {
		data["sources"] = sources
	}
	if answer, ok := data["answer"].(string); ok && strings.Contains(strings.ToLower(answer), "ai not configured") {
		data["answer"] = synthesizeAskAnswer(question, sources)
	}
	return c.JSON(http.StatusOK, data)
}

func synthesizeAskAnswer(question string, sources []map[string]any) string {
	q := strings.TrimSpace(question)
	if len(sources) == 0 {
		return "暂时没有找到足够的本地资料来回答这个问题。"
	}

	lines := []string{fmt.Sprintf("我根据你本地的资料找到了 %d 条相关内容：", len(sources))}
	for idx, source := range sources {
		if idx >= 3 {
			break
		}
		kind, _ := source["kind"].(string)
		title := ""
		if v, ok := source["title"].(string); ok {
			title = v
		}
		if title == "" {
			if v, ok := source["name"].(string); ok {
				title = v
			}
		}
		switch kind {
		case "memory":
			lines = append(lines, fmt.Sprintf("- 记忆：%s", title))
		case "task":
			lines = append(lines, fmt.Sprintf("- 任务：%s", title))
		case "thing":
			lines = append(lines, fmt.Sprintf("- 物品：%s", title))
		case "today_card":
			lines = append(lines, fmt.Sprintf("- 今日卡片：%s", title))
		case "reminder":
			lines = append(lines, fmt.Sprintf("- 提醒：%s", title))
		default:
			lines = append(lines, fmt.Sprintf("- %s", title))
		}
	}
	lines = append(lines, "", "你可以继续问我更具体一点，我会基于这些资料继续帮你找。")
	if q != "" {
		return strings.Join(lines, "\n")
	}
	return "我找到了相关资料，但还需要你把问题说得更具体一点。"
}

func (s *Server) buildAskContext(c echo.Context, userID uuid.UUID, question string) ([]map[string]any, string, error) {
	query := strings.TrimSpace(question)
	if query == "" {
		return []map[string]any{}, "", nil
	}

	sources := make([]map[string]any, 0, 12)
	seen := map[string]struct{}{}
	var snippets []string
	appendSource := func(kind string, id uuid.UUID, title string, extra map[string]any, snippet string) {
		key := kind + ":" + id.String()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		entry := map[string]any{
			"kind":  kind,
			"id":    id,
			"title": title,
		}
		for k, v := range extra {
			entry[k] = v
		}
		sources = append(sources, entry)
		if strings.TrimSpace(snippet) != "" {
			snippets = append(snippets, snippet)
		}
	}

	if memories, err := s.queryAskRecallNodesByTypes(userID, query, []string{lifeNodeTypeMemory, lifeNodeTypeMind}, 5); err == nil {
		for _, row := range memories {
			appendSource("memory", row.ID, row.Title, map[string]any{"type": row.Type}, askSnippetFromTitleBody(row.Title, row.Body))
		}
	}

	var tasks []struct {
		ID       uuid.UUID `db:"id"`
		Title    string    `db:"title"`
		Body     *string   `db:"body"`
		Status   string    `db:"status"`
		DueDate  *time.Time `db:"due_date"`
	}
	if err := s.store.DB.Select(&tasks, `
		SELECT id, title, body, status, due_date
		FROM life_nodes
		WHERE user_id = $1 AND type = 'task'
		  AND (
			 title ILIKE '%' || $2 || '%'
			 OR COALESCE(body, '') ILIKE '%' || $2 || '%'
			 OR EXISTS (
				 SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || $2 || '%'
			 )
		  )
		ORDER BY updated_at DESC
		LIMIT 5
	`, userID, query); err == nil {
		for _, row := range tasks {
			snippet := row.Title
			if row.Body != nil && strings.TrimSpace(*row.Body) != "" {
				snippet += "\n" + strings.TrimSpace(*row.Body)
			}
			if row.DueDate != nil {
				snippet += fmt.Sprintf("\n截止时间：%s", row.DueDate.Format(time.RFC3339))
			}
			appendSource("task", row.ID, row.Title, map[string]any{"status": row.Status}, snippet)
		}
	}

	var things []struct {
		ID    uuid.UUID `db:"id"`
		Title string    `db:"title"`
		Body  *string   `db:"body"`
		Room  *string   `db:"room_name"`
		Container *string `db:"container_name"`
	}
	if err := s.store.DB.Select(&things, `
		SELECT n.id, n.title, n.body, r.name AS room_name, ct.name AS container_name
		FROM life_nodes n
		LEFT JOIN item_details i ON i.node_id = n.id
		LEFT JOIN rooms r ON r.id = i.room_id
		LEFT JOIN containers ct ON ct.id = i.container_id
		WHERE n.user_id = $1 AND n.type = 'thing'
		  AND (
			 n.title ILIKE '%' || $2 || '%'
			 OR COALESCE(n.body, '') ILIKE '%' || $2 || '%'
			 OR EXISTS (
				 SELECT 1 FROM unnest(n.tags) AS tag WHERE tag ILIKE '%' || $2 || '%'
			 )
		  )
		ORDER BY n.updated_at DESC
		LIMIT 5
	`, userID, query); err == nil {
		for _, row := range things {
			snippet := row.Title
			if row.Body != nil && strings.TrimSpace(*row.Body) != "" {
				snippet += "\n" + strings.TrimSpace(*row.Body)
			}
			locationParts := make([]string, 0, 2)
			if row.Room != nil && strings.TrimSpace(*row.Room) != "" {
				locationParts = append(locationParts, *row.Room)
			}
			if row.Container != nil && strings.TrimSpace(*row.Container) != "" {
				locationParts = append(locationParts, *row.Container)
			}
			if len(locationParts) > 0 {
				snippet += "\n位置：" + strings.Join(locationParts, " · ")
			}
			appendSource("thing", row.ID, row.Title, nil, snippet)
		}
	}

	if nodeRefs, err := s.queryAskRecallNodesByTypes(userID, query, []string{lifeNodeTypePerson, lifeNodeTypeEvent, lifeNodeTypeCollection}, 5); err == nil {
		for _, row := range nodeRefs {
			appendSource(row.Type, row.ID, row.Title, nil, askSnippetFromTitleBody(row.Title, row.Body))
		}
	}

	var cards []struct {
		ID     uuid.UUID `db:"id"`
		Title  string    `db:"title"`
		Body   *string   `db:"body"`
		Slot   string    `db:"slot"`
		Severity int     `db:"severity"`
	}
	if err := s.store.DB.Select(&cards, `
		SELECT id, title, body, slot, severity
		FROM today_cards
		WHERE user_id = $1
		  AND dismissed_at IS NULL
		  AND (title ILIKE '%' || $2 || '%' OR COALESCE(body, '') ILIKE '%' || $2 || '%')
		ORDER BY severity DESC, created_at DESC
		LIMIT 3
	`, userID, query); err == nil {
		for _, row := range cards {
			appendSource("today_card", row.ID, row.Title, map[string]any{"slot": row.Slot, "severity": row.Severity}, askSnippetFromTitleBody(row.Title, row.Body))
		}
	}

	var reminders []struct {
		ID       uuid.UUID `db:"id"`
		Title    string    `db:"title"`
		RemindAt time.Time  `db:"remind_at"`
	}
	if err := s.store.DB.Select(&reminders, `
		SELECT id, title, remind_at
		FROM reminders
		WHERE user_id = $1
		  AND is_done = false
		  AND title ILIKE '%' || $2 || '%'
		ORDER BY remind_at ASC
		LIMIT 3
	`, userID, query); err == nil {
		for _, row := range reminders {
			appendSource("reminder", row.ID, row.Title, map[string]any{"remind_at": row.RemindAt}, fmt.Sprintf("%s\n提醒时间：%s", row.Title, row.RemindAt.Format(time.RFC3339)))
		}
	}

	var briefs []struct {
		ID       uuid.UUID `db:"id"`
		LocalDay string    `db:"local_day"`
		Content  string    `db:"content"`
		IsRead   bool      `db:"is_read"`
	}
	if err := s.store.DB.Select(&briefs, `
		SELECT id, local_day, content, is_read
		FROM daily_briefs
		WHERE user_id = $1
		ORDER BY generated_at DESC
		LIMIT 2
	`, userID); err == nil {
		for _, row := range briefs {
			label := "日报"
			if strings.TrimSpace(row.LocalDay) != "" {
				label = fmt.Sprintf("日报 %s", row.LocalDay)
			}
			status := "未读"
			if row.IsRead {
				status = "已读"
			}
			appendSource("daily_brief", row.ID, label, map[string]any{"local_day": row.LocalDay, "is_read": row.IsRead}, fmt.Sprintf("%s（%s）\n%s", label, status, strings.TrimSpace(row.Content)))
		}
	}

	contextText := question
	if len(snippets) > 0 {
		contextText = fmt.Sprintf("你是 Nesio 的问一问，请基于以下本地资料回答用户问题。\n\n问题：%s\n\n相关资料：\n%s\n\n要求：\n- 优先使用相关资料回答\n- 如果资料不足，明确说明不确定\n- 答案简洁、直接、可执行\n", question, strings.Join(snippets, "\n\n---\n\n"))
	}
	return sources, contextText, nil
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
		WHERE user_id = $1 AND (
			title ILIKE '%' || $2 || '%'
			OR COALESCE(body, '') ILIKE '%' || $2 || '%'
			OR EXISTS (
				SELECT 1 FROM unnest(tags) AS tag WHERE tag ILIKE '%' || $2 || '%'
			)
		)
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
