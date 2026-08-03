package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestHandleImportConnectorProvider_ValidatesRequiredFields(t *testing.T) {
	server := &Server{}
	ctx, _ := newConnectorImportContext(t, "plaid", `{ "client_id": "cid", "access_token": "token" }`, "")
	ctx.Set("user_id", uuid.New())

	err := server.handleImportConnectorProvider(ctx)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", he.Code, http.StatusBadRequest)
	}
	msg, _ := he.Message.(string)
	if !strings.Contains(msg, "missing required field: secret") {
		t.Fatalf("message = %q, want missing secret", msg)
	}
}

func TestHandleImportConnectorProvider_EncryptsAndSyncs(t *testing.T) {
	origEncrypt := encryptConnectorCredentialsFn
	origUpsert := upsertConnectorProviderFn
	origSync := syncConnectorFn
	origCard := createConnectorImportCardFn
	defer func() {
		encryptConnectorCredentialsFn = origEncrypt
		upsertConnectorProviderFn = origUpsert
		syncConnectorFn = origSync
		createConnectorImportCardFn = origCard
	}()

	userID := uuid.New()
	connectorID := uuid.New()
	encryptedOut := json.RawMessage(`{"nonce":"n","cipher":"c"}`)
	sawSync := false
	sawCardStatus := ""

	encryptConnectorCredentialsFn = func(credentials map[string]any) (json.RawMessage, error) {
		if _, ok := credentials["access_token"]; !ok {
			t.Fatalf("expected access_token in credentials")
		}
		return encryptedOut, nil
	}
	upsertConnectorProviderFn = func(_ *Server, gotUserID uuid.UUID, gotProvider string, encrypted []byte) (uuid.UUID, error) {
		if gotUserID != userID {
			t.Fatalf("user_id = %s, want %s", gotUserID, userID)
		}
		if gotProvider != "tesla_fleet" {
			t.Fatalf("provider = %q, want tesla_fleet", gotProvider)
		}
		if string(encrypted) != string(encryptedOut) {
			t.Fatalf("encrypted payload mismatch")
		}
		return connectorID, nil
	}
	syncConnectorFn = func(_ context.Context, _ *storage.Store, gotID uuid.UUID) error {
		sawSync = true
		if gotID != connectorID {
			t.Fatalf("connector id = %s, want %s", gotID, connectorID)
		}
		return nil
	}
	createConnectorImportCardFn = func(_ *Server, _ uuid.UUID, _ string, status string) {
		sawCardStatus = status
	}

	server := &Server{}
	ctx, rec := newConnectorImportContext(t, "tesla_fleet", `{ "access_token": "abc" }`, "")
	ctx.Set("user_id", userID)

	err := server.handleImportConnectorProvider(ctx)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if !sawSync {
		t.Fatalf("expected sync to be called")
	}
	if sawCardStatus != "synced" {
		t.Fatalf("card status = %q, want synced", sawCardStatus)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["status"] != "synced" {
		t.Fatalf("response status = %v, want synced", resp["status"])
	}
}

func TestHandleImportConnectorProvider_SyncFailure(t *testing.T) {
	origEncrypt := encryptConnectorCredentialsFn
	origUpsert := upsertConnectorProviderFn
	origSync := syncConnectorFn
	origCard := createConnectorImportCardFn
	defer func() {
		encryptConnectorCredentialsFn = origEncrypt
		upsertConnectorProviderFn = origUpsert
		syncConnectorFn = origSync
		createConnectorImportCardFn = origCard
	}()

	encryptConnectorCredentialsFn = func(_ map[string]any) (json.RawMessage, error) {
		return json.RawMessage(`{"nonce":"n","cipher":"c"}`), nil
	}
	upsertConnectorProviderFn = func(_ *Server, _ uuid.UUID, _ string, _ []byte) (uuid.UUID, error) {
		return uuid.New(), nil
	}
	syncConnectorFn = func(_ context.Context, _ *storage.Store, _ uuid.UUID) error {
		return errors.New("upstream unavailable")
	}
	cardCalled := false
	createConnectorImportCardFn = func(_ *Server, _ uuid.UUID, _ string, _ string) {
		cardCalled = true
	}

	server := &Server{}
	ctx, _ := newConnectorImportContext(t, "granola", `{ "endpoint": "https://example.com/items" }`, "")
	ctx.Set("user_id", uuid.New())

	err := server.handleImportConnectorProvider(ctx)
	if err == nil {
		t.Fatalf("expected sync failure")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", he.Code, http.StatusBadGateway)
	}
	msg, _ := he.Message.(string)
	if !strings.Contains(msg, "sync failed") {
		t.Fatalf("message = %q, want sync failed hint", msg)
	}
	if cardCalled {
		t.Fatalf("did not expect import card on sync failure")
	}
}

func TestHandleImportConnectorProvider_SyncDisabled(t *testing.T) {
	origEncrypt := encryptConnectorCredentialsFn
	origUpsert := upsertConnectorProviderFn
	origSync := syncConnectorFn
	origCard := createConnectorImportCardFn
	defer func() {
		encryptConnectorCredentialsFn = origEncrypt
		upsertConnectorProviderFn = origUpsert
		syncConnectorFn = origSync
		createConnectorImportCardFn = origCard
	}()

	encryptConnectorCredentialsFn = func(_ map[string]any) (json.RawMessage, error) {
		return json.RawMessage(`{"nonce":"n","cipher":"c"}`), nil
	}
	upsertConnectorProviderFn = func(_ *Server, _ uuid.UUID, _ string, _ []byte) (uuid.UUID, error) {
		return uuid.New(), nil
	}
	syncCalled := false
	syncConnectorFn = func(_ context.Context, _ *storage.Store, _ uuid.UUID) error {
		syncCalled = true
		return nil
	}
	statusCaptured := ""
	createConnectorImportCardFn = func(_ *Server, _ uuid.UUID, _ string, status string) {
		statusCaptured = status
	}

	server := &Server{}
	ctx, rec := newConnectorImportContext(t, "flomo", `{ "endpoint": "https://example.com/items" }`, "sync=false")
	ctx.Set("user_id", uuid.New())

	err := server.handleImportConnectorProvider(ctx)
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if syncCalled {
		t.Fatalf("did not expect sync call when sync=false")
	}
	if statusCaptured != "imported" {
		t.Fatalf("card status = %q, want imported", statusCaptured)
	}
}

func newConnectorImportContext(t *testing.T, provider, body, rawQuery string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	path := "/api/v1/connectors/:provider/import"
	if rawQuery != "" {
		path = path + "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/api/v1/connectors/:provider/import")
	ctx.SetParamNames("provider")
	ctx.SetParamValues(provider)
	return ctx, rec
}