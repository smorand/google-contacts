// Package mcp provides the MCP (Model Context Protocol) server implementation.
package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateState(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	// Generate multiple states and ensure they're unique
	states := make(map[string]bool)
	for i := 0; i < 10; i++ {
		state, err := h.generateState()
		if err != nil {
			t.Fatalf("generateState() error: %v", err)
		}
		if state == "" {
			t.Error("generateState() returned empty string")
		}
		if states[state] {
			t.Errorf("generateState() returned duplicate state: %s", state)
		}
		states[state] = true

		// State should be base64 URL encoded (44 chars for 32 bytes)
		if len(state) < 40 {
			t.Errorf("state too short: %d chars, expected >= 40", len(state))
		}
	}
}

func TestValidateState(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	// Generate a state
	state, err := h.generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}

	// Validate should succeed first time
	if !h.validateState(state) {
		t.Error("validateState() returned false for valid state")
	}

	// Validate should fail second time (single-use)
	if h.validateState(state) {
		t.Error("validateState() returned true for already-used state")
	}
}

func TestValidateState_Invalid(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	tests := []struct {
		name  string
		state string
	}{
		{"empty state", ""},
		{"random state", "random-invalid-state"},
		{"partial match", "abc123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if h.validateState(tc.state) {
				t.Errorf("validateState(%q) returned true, expected false", tc.state)
			}
		})
	}
}

func TestTruncateToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{"empty token", "", ""},
		{"short token", "abc", "abc"},
		{"exactly 10 chars", "1234567890", "1234567890"},
		{"longer than 10", "123456789012345", "1234567890..."},
		{"much longer", "this-is-a-very-long-refresh-token-abc123", "this-is-a-..."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateToken(tc.token)
			if result != tc.expected {
				t.Errorf("truncateToken(%q) = %q, want %q", tc.token, result, tc.expected)
			}
		})
	}
}

func TestLocalCredentialsPath(t *testing.T) {
	path := LocalCredentialsPath()
	if path == "" {
		t.Error("LocalCredentialsPath() returned empty string")
	}
	if !strings.Contains(path, ".credentials") {
		t.Errorf("LocalCredentialsPath() = %q, expected to contain '.credentials'", path)
	}
	if !strings.Contains(path, "google_credentials.json") {
		t.Errorf("LocalCredentialsPath() = %q, expected to contain 'google_credentials.json'", path)
	}
}

func TestHandleAuth_MissingCredentials(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{
		// No credentials configured
	})

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()

	h.HandleAuth(rec, req)

	// Should return 500 because no credentials are available
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("HandleAuth() status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleCallback_MissingState(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleCallback() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "Missing state") {
		t.Errorf("HandleCallback() body = %q, expected to contain 'Missing state'", rec.Body.String())
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state=invalid-state", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleCallback() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "Invalid or expired state") {
		t.Errorf("HandleCallback() body = %q, expected to contain 'Invalid or expired state'", rec.Body.String())
	}
}

func TestHandleCallback_MissingCode(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	// First generate a valid state
	state, err := h.generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state="+state, nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleCallback() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "Missing authorization code") {
		t.Errorf("HandleCallback() body = %q, expected to contain 'Missing authorization code'", rec.Body.String())
	}
}

func TestHandleCallback_OAuthError(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?error=access_denied&error_description=User%20denied%20access", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleCallback() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "access_denied") {
		t.Errorf("HandleCallback() body = %q, expected to contain 'access_denied'", rec.Body.String())
	}
}

func TestAuthHandler_SetupRoutes(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})
	mux := http.NewServeMux()

	// This should not panic
	h.SetupRoutes(mux)

	// Verify routes are registered by making requests
	// Note: Actual handler behavior is tested separately

	// Test /auth route exists
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Should get an error (500) because no credentials, not 404
	if rec.Code == http.StatusNotFound {
		t.Error("/auth route not registered")
	}

	// Test /auth/callback route exists
	req = httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Should get an error (400 for missing params), not 404
	if rec.Code == http.StatusNotFound {
		t.Error("/auth/callback route not registered")
	}
}

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"empty", "", ""},
		{"localhost", "http://localhost:8080", "http://localhost:8080"},
		{"cloud run", "https://my-app.run.app", "https://my-app.run.app"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewAuthHandler(&AuthHandlerConfig{BaseURL: tc.baseURL})
			if h.GetBaseURL() != tc.expected {
				t.Errorf("GetBaseURL() = %q, want %q", h.GetBaseURL(), tc.expected)
			}
		})
	}
}
