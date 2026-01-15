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

	// Test /auth/success route exists
	req = httptest.NewRequest(http.MethodGet, "/auth/success?key=test-key", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	// Should get 200 OK since we provided the key
	if rec.Code == http.StatusNotFound {
		t.Error("/auth/success route not registered")
	}
}

func TestHandleSuccess_ValidKey(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{
		BaseURL: "https://example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/success?key=550e8400-e29b-41d4-a716-446655440000&email=user@example.com", nil)
	rec := httptest.NewRecorder()

	h.HandleSuccess(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()

	// Check Content-Type
	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("HandleSuccess() Content-Type = %q, want text/html", contentType)
	}

	// Check that API key is displayed
	if !strings.Contains(body, "550e8400-e29b-41d4-a716-446655440000") {
		t.Error("HandleSuccess() response does not contain API key")
	}

	// Check that email is displayed
	if !strings.Contains(body, "user@example.com") {
		t.Error("HandleSuccess() response does not contain user email")
	}

	// Check that server URL is displayed
	if !strings.Contains(body, "https://example.com") {
		t.Error("HandleSuccess() response does not contain server URL")
	}

	// Check that copy button exists
	if !strings.Contains(body, "copyToClipboard") {
		t.Error("HandleSuccess() response does not contain copy button JavaScript")
	}

	// Check for security warning
	if !strings.Contains(body, "Security Notice") || !strings.Contains(body, "securely") {
		t.Error("HandleSuccess() response does not contain security warning")
	}
}

func TestHandleSuccess_MissingKey(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	req := httptest.NewRequest(http.MethodGet, "/auth/success", nil)
	rec := httptest.NewRecorder()

	h.HandleSuccess(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSuccess() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "Missing API key") {
		t.Errorf("HandleSuccess() body = %q, expected to contain 'Missing API key'", rec.Body.String())
	}
}

func TestHandleSuccess_EmptyKey(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{})

	req := httptest.NewRequest(http.MethodGet, "/auth/success?key=", nil)
	rec := httptest.NewRecorder()

	h.HandleSuccess(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleSuccess() status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSuccess_NoEmail(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{
		BaseURL: "https://example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/success?key=test-api-key-123", nil)
	rec := httptest.NewRecorder()

	h.HandleSuccess(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()

	// Check that API key is displayed
	if !strings.Contains(body, "test-api-key-123") {
		t.Error("HandleSuccess() response does not contain API key")
	}

	// Email should not be displayed (optional field)
	// But the page should still render correctly
}

func TestHandleSuccess_FallbackServerURL(t *testing.T) {
	h := NewAuthHandler(&AuthHandlerConfig{
		// No BaseURL configured - should use request host
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/success?key=test-key", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()

	h.HandleSuccess(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HandleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()

	// Should show http://localhost:8080 as the server URL (http since no TLS)
	if !strings.Contains(body, "http://localhost:8080") {
		t.Error("HandleSuccess() response does not contain fallback server URL")
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

func TestGenerateAPIKey(t *testing.T) {
	// Generate multiple keys and verify they are valid UUIDs
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key := GenerateAPIKey()
		if key == "" {
			t.Error("GenerateAPIKey() returned empty string")
		}

		// UUID v4 format: 8-4-4-4-12 (total 36 chars including hyphens)
		if len(key) != 36 {
			t.Errorf("GenerateAPIKey() returned %q, expected 36 char UUID", key)
		}

		// Check for duplicates
		if keys[key] {
			t.Errorf("GenerateAPIKey() returned duplicate key: %s", key)
		}
		keys[key] = true

		// Check UUID format (8-4-4-4-12 pattern)
		parts := strings.Split(key, "-")
		if len(parts) != 5 {
			t.Errorf("GenerateAPIKey() = %q, expected 5 parts separated by hyphens", key)
		}
		expectedLengths := []int{8, 4, 4, 4, 12}
		for j, part := range parts {
			if len(part) != expectedLengths[j] {
				t.Errorf("GenerateAPIKey() part %d = %q (len %d), expected len %d", j, part, len(part), expectedLengths[j])
			}
		}
	}
}

func TestGenerateAPIKey_UniquePerCall(t *testing.T) {
	// Verify that multiple calls generate unique keys
	key1 := GenerateAPIKey()
	key2 := GenerateAPIKey()
	key3 := GenerateAPIKey()

	if key1 == key2 || key2 == key3 || key1 == key3 {
		t.Errorf("GenerateAPIKey() returned duplicate keys: %s, %s, %s", key1, key2, key3)
	}
}
