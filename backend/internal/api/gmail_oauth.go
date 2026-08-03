package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/connector"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const googleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
const googleTokenURL = "https://oauth2.googleapis.com/token"

var gmailScopes = "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.send https://www.googleapis.com/auth/userinfo.email"

// handleGmailOAuthAuthorize initiates the Google OAuth2 authorization code flow.
func (s *Server) handleGmailOAuthAuthorize(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if !isValidGoogleClientID(clientID) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "GOOGLE_CLIENT_ID not configured")
	}

	state := uuid.NewString()
	redirectURI := googleRedirectURI()

	stateData, _ := json.Marshal(map[string]string{"user_id": userID.String()})
	if err := s.store.Set(c.Request().Context(), "google_oauth:"+state, string(stateData), storage.TierSession, 10*time.Minute); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	params := url.Values{
		"client_id":       {clientID},
		"redirect_uri":    {redirectURI},
		"response_type":   {"code"},
		"scope":           {gmailScopes},
		"access_type":     {"offline"},
		"prompt":          {"consent"},
		"state":           {state},
	}
	authURL := googleAuthURL + "?" + params.Encode()
	return c.JSON(http.StatusOK, map[string]string{"auth_url": authURL})
}

// handleGmailOAuthCallback exchanges the authorization code for access + refresh tokens.
func (s *Server) handleGmailOAuthCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	if code == "" || state == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing code or state")
	}

	raw, err := s.store.Get(c.Request().Context(), "google_oauth:"+state, storage.TierSession)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired state – please try connecting again")
	}
	_ = s.store.Delete(c.Request().Context(), "google_oauth:"+state, storage.TierSession)

	var stateData struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(raw), &stateData); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	userID, err := uuid.Parse(stateData.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	tokens, err := exchangeGoogleCode(c.Request().Context(), code, googleRedirectURI())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "token exchange failed: "+err.Error())
	}

	account, _ := fetchGoogleEmail(c.Request().Context(), tokens.AccessToken)

	creds := map[string]any{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"token_expiry":  time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).UTC().Format(time.RFC3339),
		"account":       account,
	}
	encrypted, err := connector.EncryptCredentials(creds)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Upsert connector row.
	_, _ = s.store.DB.Exec(`DELETE FROM connectors WHERE user_id = $1 AND provider = 'gmail'`, userID)
	var id uuid.UUID
	if err := s.store.DB.QueryRow(
		`INSERT INTO connectors (user_id, provider, credentials) VALUES ($1, 'gmail', $2) RETURNING id`,
		userID, encrypted,
	).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.Redirect(http.StatusFound, "http://localhost:5173/?connected=gmail&account="+url.QueryEscape(account))
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func exchangeGoogleCode(ctx context.Context, code, redirectURI string) (*googleTokenResponse, error) {
	return doTokenRequest(ctx, url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	})
}

func refreshGoogleToken(ctx context.Context, refreshToken string) (*googleTokenResponse, error) {
	return doTokenRequest(ctx, url.Values{
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
}

func doTokenRequest(ctx context.Context, params url.Values) (*googleTokenResponse, error) {
	params.Set("client_id", os.Getenv("GOOGLE_CLIENT_ID"))
	params.Set("client_secret", os.Getenv("GOOGLE_CLIENT_SECRET"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google token error %d: %s", resp.StatusCode, string(body))
	}

	var t googleTokenResponse
	return &t, json.Unmarshal(body, &t)
}

func fetchGoogleEmail(ctx context.Context, accessToken string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	return info.Email, nil
}

func googleRedirectURI() string {
	if v := os.Getenv("GOOGLE_REDIRECT_URI"); v != "" {
		return v
	}
	return "http://localhost:8080/api/v1/connectors/gmail/oauth/callback"
}

func isValidGoogleClientID(clientID string) bool {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return false
	}
	if strings.HasPrefix(clientID, "your_client_id") {
		return false
	}
	if strings.Contains(strings.ToLower(clientID), "placeholder") {
		return false
	}
	return strings.HasSuffix(clientID, ".apps.googleusercontent.com")
}
