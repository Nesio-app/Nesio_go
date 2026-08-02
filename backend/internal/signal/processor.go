package signal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Processor struct {
	DB *sqlx.DB
}

type aiCardResponse struct {
	Content string `json:"content"`
	Card    *struct {
		Title    string  `json:"title"`
		Body     *string `json:"body,omitempty"`
		Severity int     `json:"severity"`
	} `json:"card,omitempty"`
}

type aiRequest struct {
	Message string `json:"message"`
	Tier    string `json:"tier"`
	Mode    string `json:"mode"`
}

type aiResponse struct {
	Content string   `json:"content"`
	Sources []string `json:"sources,omitempty"`
	Card    *struct {
		Title    string  `json:"title"`
		Body     *string `json:"body,omitempty"`
		Severity int     `json:"severity"`
	} `json:"card,omitempty"`
}

func NewProcessor(db *sqlx.DB) *Processor {
	return &Processor{DB: db}
}

func (p *Processor) Process(ctx context.Context, userID uuid.UUID, signal models.Signal) (*models.TodayCard, error) {
	fp := fingerprint(signal)

	// Check muted
	var muted bool
	err := p.DB.Get(&muted, "SELECT EXISTS(SELECT 1 FROM fingerprints WHERE user_id = $1 AND hash = $2 AND is_muted = true)", userID, fp)
	if err == nil && muted {
		return nil, fmt.Errorf("signal muted")
	}

	// Check dismissed today
	var dismissed bool
	err = p.DB.Get(&dismissed, "SELECT EXISTS(SELECT 1 FROM fingerprints WHERE user_id = $1 AND hash = $2 AND dismissed_at > $3)", userID, fp, time.Now().Add(-24*time.Hour))
	if err == nil && dismissed {
		return nil, fmt.Errorf("signal dismissed today")
	}

	localDay := p.getUserLocalDay(userID)

	// Check daily quota (max 50 cards/day)
	var count int
	err = p.DB.Get(&count, "SELECT COUNT(*) FROM today_cards WHERE user_id = $1 AND local_day = $2", userID, localDay)
	if err == nil && count >= 50 {
		return nil, fmt.Errorf("daily quota exceeded")
	}

	// Classify severity
	severity := classifySignal(signal)

	// Generate card
	card := &models.TodayCard{
		UserID:       userID,
		LocalDay:     localDay,
		Slot:         "task",
		Title:        generateTitle(signal),
		Body:         generateBody(signal),
		Severity:     severity,
		Fingerprints: []string{fp},
		CreatedAt:    time.Now().UTC(),
	}

	if signal.Source == "gmail" || signal.Source == "email" || signal.Source == "note" || signal.Source == "calendar" {
		if aiCard, err := p.callAIForCard(signal); err == nil && aiCard != nil {
			card.Title = aiCard.Title
			card.Body = aiCard.Body
			card.Severity = aiCard.Severity
		}
	}

	if card.Severity == 3 {
		card.Slot = "pinned"
	}

	// Insert card
	_, err = p.DB.NamedExec(`
		INSERT INTO today_cards (user_id, local_day, slot, title, body, severity, fingerprints)
		VALUES (:user_id, :local_day, :slot, :title, :body, :severity, :fingerprints)
	`, card)
	if err != nil {
		return nil, fmt.Errorf("insert card: %w", err)
	}

	// Record fingerprint
	_, _ = p.DB.Exec(`
		INSERT INTO fingerprints (user_id, hash, source, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, hash) DO NOTHING
	`, userID, fp, signal.Source, time.Now().UTC())

	return card, nil
}

func (p *Processor) getUserLocalDay(userID uuid.UUID) string {
	var timezone string
	err := p.DB.Get(&timezone, "SELECT timezone FROM users WHERE id = $1", userID)
	if err != nil || timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("UTC", 0)
	}
	return time.Now().In(loc).Format("2006-01-02")
}

func fingerprint(signal models.Signal) string {
	keys := make([]string, 0, len(signal.Fields))
	for k := range signal.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonical string
	for _, k := range keys {
		canonical += fmt.Sprintf("%s=%v;", k, signal.Fields[k])
	}

	content := fmt.Sprintf("v2:%s:%s:%s", signal.Source, signal.AnchorID, canonical)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8])
}

func (p *Processor) callAIForCard(signal models.Signal) (*struct {
	Title    string  `json:"title"`
	Body     *string `json:"body,omitempty"`
	Severity int     `json:"severity"`
}, error) {
	aiURL := os.Getenv("AI_SERVICE_URL")
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}

	payload := aiRequest{
		Message: signal.RawData,
		Tier:    "standard",
		Mode:    "card",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(aiURL+"/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result aiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if result.Card == nil {
		return nil, fmt.Errorf("no card returned from AI")
	}
	if result.Card.Title == "" {
		return nil, fmt.Errorf("empty card title from AI")
	}
	if result.Card.Severity < 1 || result.Card.Severity > 3 {
		result.Card.Severity = 2
	}
	return result.Card, nil
}

func classifySignal(signal models.Signal) int {
	switch signal.Source {
	case "calendar":
		if start, ok := signal.Fields["start_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				if time.Until(t) <= 2*time.Hour && time.Until(t) > 0 {
					return 3 // critical
				}
			}
		}
		return 1
	case "gmail", "email":
		return 2 // important - needs AI review
	default:
		return 1
	}
}

func generateTitle(signal models.Signal) string {
	switch signal.Source {
	case "calendar":
		if title, ok := signal.Fields["title"].(string); ok {
			return title
		}
	case "plaid":
		if desc, ok := signal.Fields["description"].(string); ok {
			return "账单提醒: " + desc
		}
	}
	return "新信号"
}

func generateBody(signal models.Signal) *string {
	var body string
	switch signal.Source {
	case "calendar":
		if loc, ok := signal.Fields["location"].(string); ok && loc != "" {
			body = "地点: " + loc
		}
	case "gmail":
		if from, ok := signal.Fields["from"].(string); ok {
			body = "来自: " + from
		}
	}
	if body == "" {
		return nil
	}
	return &body
}
