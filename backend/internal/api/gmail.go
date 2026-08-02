package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type gmailConnectorRecord struct {
	ID          uuid.UUID       `db:"id"`
	Credentials json.RawMessage `db:"credentials"`
}

type GmailMessage struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet"`
}

type GmailSendRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type gmailListResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type gmailMessageResponse struct {
	ID      string `json:"id"`
	Snippet string `json:"snippet"`
	Payload struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
	} `json:"payload"`
}

func (s *Server) handleGmailInbox(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	accessToken, err := s.getGmailAccessToken(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var list gmailListResponse
	endpoint := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=20&q=in:inbox newer_than:7d"
	if err := doGoogleJSON(c.Request().Context(), accessToken, endpoint, &list); err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	messages := make([]GmailMessage, 0, len(list.Messages))
	for _, msg := range list.Messages {
		var detail gmailMessageResponse
		detailURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=From&metadataHeaders=Subject", url.PathEscape(msg.ID))
		if err := doGoogleJSON(c.Request().Context(), accessToken, detailURL, &detail); err != nil {
			continue
		}
		messages = append(messages, GmailMessage{
			ID:      detail.ID,
			From:    gmailHeaderValue(detail.Payload.Headers, "From"),
			Subject: gmailHeaderValue(detail.Payload.Headers, "Subject"),
			Snippet: detail.Snippet,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"messages": messages})
}

func (s *Server) handleGmailSend(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	accessToken, err := s.getGmailAccessToken(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var req GmailSendRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.To == "" || req.Subject == "" || req.Body == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "to, subject, and body are required")
	}

	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", req.To, req.Subject, req.Body)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(raw))
	payload, _ := json.Marshal(map[string]string{"raw": encoded})

	httpReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, "https://gmail.googleapis.com/gmail/v1/users/me/messages/send", bytes.NewReader(payload))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return echo.NewHTTPError(http.StatusBadGateway, string(body))
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "sent"})
}

func (s *Server) getGmailAccessToken(userID uuid.UUID) (string, error) {
	var row gmailConnectorRecord
	err := s.store.DB.Get(&row, `
		SELECT id, credentials
		FROM connectors
		WHERE user_id = $1 AND is_active = true AND provider IN ('gmail', 'googlemail')
		ORDER BY created_at DESC
		LIMIT 1
	`, userID)
	if err != nil {
		return "", fmt.Errorf("gmail connector not found")
	}
	credentials, err := connector.DecryptCredentials(row.Credentials)
	if err != nil {
		return "", err
	}
	accessToken, _ := credentials["access_token"].(string)
	if accessToken == "" {
		return "", fmt.Errorf("gmail connector missing access_token")
	}
	return accessToken, nil
}

func doGoogleJSON(ctx context.Context, accessToken, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("google api error: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func gmailHeaderValue(headers []struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}, name string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}
