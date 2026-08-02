package api
import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ChatRequest struct {
	Message           string `json:"message" validate:"required"`
	Tier              string `json:"tier,omitempty"`
	RequiresReasoning bool   `json:"requires_reasoning,omitempty"`
}

type ChatResponse struct {
	Content string   `json:"content"`
	Sources []string `json:"sources,omitempty"`
}

func (s *Server) handleChat(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req ChatRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Save user message
	_, _ = s.store.DB.Exec(
		"INSERT INTO chat_messages (user_id, role, content) VALUES ($1, 'user', $2)",
		userID, req.Message,
	)

	// Route to AI service
	tier := selectTier(req)
	aiResp, err := callAIService(req.Message, tier, "chat")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "AI service error: "+err.Error())
	}

	// Save assistant message
	_, _ = s.store.DB.Exec(
		"INSERT INTO chat_messages (user_id, role, content, actions) VALUES ($1, 'assistant', $2, $3)",
		userID, aiResp.Content, nil,
	)

	return c.JSON(http.StatusOK, aiResp)
}

func (s *Server) handleChatHistory(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var messages []models.ChatMessage
	err := s.store.DB.Select(&messages, `
		SELECT id, user_id, role, content, actions, created_at
		FROM chat_messages
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, messages)
}

func selectTier(req ChatRequest) string {
	if req.Tier != "" {
		return req.Tier
	}
	if req.RequiresReasoning || len(req.Message) > 200 {
		return "deep"
	}
	if len(req.Message) < 50 {
		return "quick"
	}
	return "standard"
}

func callAIService(message, tier, mode string) (*ChatResponse, error) {
	aiURL := os.Getenv("AI_SERVICE_URL")
	if aiURL == "" {
		aiURL = "http://ai-service:8000"
	}

	payload := map[string]any{
		"message": message,
		"tier":    tier,
		"mode":    mode,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(aiURL+"/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result ChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
