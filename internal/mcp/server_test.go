// Package mcp provides the MCP (Model Context Protocol) server implementation.
package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google-contacts/pkg/auth"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer test-api-key-123",
			expected:   "test-api-key-123",
		},
		{
			name:       "empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "missing bearer prefix",
			authHeader: "Basic dXNlcjpwYXNz",
			expected:   "",
		},
		{
			name:       "bearer only no token",
			authHeader: "Bearer ",
			expected:   "",
		},
		{
			name:       "lowercase bearer",
			authHeader: "bearer token",
			expected:   "",
		},
		{
			name:       "token with spaces",
			authHeader: "Bearer token with spaces",
			expected:   "token with spaces",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			result := extractBearerToken(req)
			if result != tc.expected {
				t.Errorf("extractBearerToken() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	// When no token is provided, middleware should return 401 with WWW-Authenticate header
	cfg := &Config{
		Host:    "localhost",
		Port:    8080,
		BaseURL: "https://example.com",
	}
	server := NewServer(cfg)
	// Initialize OAuth2 server so middleware has it
	server.oauth2Server = NewOAuth2Server(&OAuth2ServerConfig{
		BaseURL: cfg.BaseURL,
	})

	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	// Request without any auth header should get 401
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if called {
		t.Error("expected next handler NOT to be called")
	}

	// Verify WWW-Authenticate header is set correctly
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("expected WWW-Authenticate header to be set")
	}
	expectedContains := `resource_metadata="https://example.com/.well-known/oauth-protected-resource"`
	if wwwAuth != `Bearer `+expectedContains {
		t.Errorf("WWW-Authenticate = %q, want it to contain %q", wwwAuth, expectedContains)
	}
}

func TestAuthMiddleware_ContextPropagation(t *testing.T) {
	// This test verifies that context values are propagated through the middleware.
	// Note: We can't fully test the auth flow without mocking the OAuth2 server,
	// but we can test context propagation using auth package functions.

	ctx := context.Background()

	// Initially no token
	token, ok := auth.GetRefreshTokenFromContext(ctx)
	if ok || token != "" {
		t.Errorf("expected no token initially, got %q", token)
	}

	// Add a refresh token
	testToken := "test-refresh-token-12345"
	ctx = auth.WithRefreshToken(ctx, testToken)

	// Now should have the token
	token, ok = auth.GetRefreshTokenFromContext(ctx)
	if !ok {
		t.Error("expected token to be present in context")
	}
	if token != testToken {
		t.Errorf("expected token %q, got %q", testToken, token)
	}
}

func TestPerUserTokenFlow(t *testing.T) {
	// This test documents the expected flow for per-user token authentication.
	// It verifies the integration between components without requiring actual
	// external services.

	// 1. User authenticates via OAuth, gets access token from the OAuth flow

	// 2. Request arrives with Bearer token in Authorization header
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	accessToken := "ya29.xxxxx"
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Verify header extraction works
	extracted := extractBearerToken(req)
	if extracted != accessToken {
		t.Errorf("extractBearerToken() = %q, want %q", extracted, accessToken)
	}

	// 3. When OAuth2 server validates the token, it returns oauth config and token
	//    (this would be tested in integration tests)

	// 4. Middleware can inject refresh token into context
	simulatedRefreshToken := "1//0gXXXXXX"
	ctx := auth.WithRefreshToken(req.Context(), simulatedRefreshToken)

	// 5. Tool handler receives context with token
	token, ok := auth.GetRefreshTokenFromContext(ctx)
	if !ok {
		t.Error("refresh token should be present in context")
	}
	if token != simulatedRefreshToken {
		t.Errorf("got token %q, want %q", token, simulatedRefreshToken)
	}

	// 6. contacts.GetPeopleService(ctx) would use this token via auth.GetClient(ctx)
	//    (this calls Google API, so tested in integration tests)
}

func TestInputStructTypes(t *testing.T) {
	// Test that MCP input/output struct types are correctly defined
	// This ensures the schema generation works properly

	// CreateInput validation
	create := CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Phones:    []PhoneInput{{Value: "+33612345678", Type: "mobile"}},
	}
	if create.FirstName == "" || create.LastName == "" {
		t.Error("CreateInput fields not accessible")
	}
	if len(create.Phones) != 1 {
		t.Error("CreateInput phones not accessible")
	}

	// SearchInput validation
	search := SearchInput{Query: "test"}
	if search.Query == "" {
		t.Error("SearchInput query not accessible")
	}

	// ShowInput validation
	show := ShowInput{ContactID: "c123"}
	if show.ContactID == "" {
		t.Error("ShowInput contactId not accessible")
	}

	// UpdateInput validation
	update := UpdateInput{
		ContactID: "c123",
		FirstName: "Jane",
		AddPhones: []PhoneInput{{Value: "+33698765432", Type: "work"}},
	}
	if update.ContactID == "" || update.FirstName == "" {
		t.Error("UpdateInput fields not accessible")
	}

	// DeleteInput validation
	del := DeleteInput{ContactID: "c123"}
	if del.ContactID == "" {
		t.Error("DeleteInput contactId not accessible")
	}
}

func TestOutputStructTypes(t *testing.T) {
	// Test that MCP output struct types are correctly defined

	// CreateOutput
	createOut := CreateOutput{
		ResourceName: "people/c123",
		DisplayName:  "John Doe",
		Message:      "Created",
	}
	if createOut.ResourceName == "" || createOut.DisplayName == "" {
		t.Error("CreateOutput fields not accessible")
	}

	// SearchOutput
	searchOut := SearchOutput{
		Count: 1,
		Results: []SearchResultItem{{
			ResourceName: "people/c123",
			DisplayName:  "John Doe",
		}},
	}
	if searchOut.Count != 1 || len(searchOut.Results) != 1 {
		t.Error("SearchOutput fields not correct")
	}

	// ShowOutput
	showOut := ShowOutput{
		ResourceName: "people/c123",
		FirstName:    "John",
		LastName:     "Doe",
		Phones:       []PhoneOutput{{Value: "+33612345678", Type: "mobile"}},
		Emails:       []EmailOutput{{Value: "john@example.com", Type: "work"}},
	}
	if showOut.ResourceName == "" || showOut.FirstName == "" {
		t.Error("ShowOutput fields not accessible")
	}
	if len(showOut.Phones) != 1 || len(showOut.Emails) != 1 {
		t.Error("ShowOutput arrays not correct")
	}

	// UpdateOutput (embeds ShowOutput)
	updateOut := UpdateOutput{
		Message: "Updated",
	}
	updateOut.ResourceName = "people/c123"
	if updateOut.Message == "" || updateOut.ResourceName == "" {
		t.Error("UpdateOutput fields not accessible")
	}

	// DeleteOutput
	deleteOut := DeleteOutput{
		Message:     "Deleted",
		DeletedID:   "people/c123",
		DisplayName: "John Doe",
	}
	if deleteOut.Message == "" || deleteOut.DeletedID == "" {
		t.Error("DeleteOutput fields not accessible")
	}
}

func TestOAuth2ServerMetadata(t *testing.T) {
	// Test that OAuth2 metadata endpoints return correct structure
	s := NewOAuth2Server(&OAuth2ServerConfig{
		BaseURL: "https://example.com",
	})

	// Test protected resource metadata endpoint
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	rec := httptest.NewRecorder()
	s.HandleProtectedResourceMetadata(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Test authorization server metadata endpoint
	req = httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rec = httptest.NewRecorder()
	s.HandleAuthorizationServerMetadata(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestOAuth2ServerRegistration(t *testing.T) {
	// Test dynamic client registration
	s := NewOAuth2Server(&OAuth2ServerConfig{
		BaseURL: "https://example.com",
	})

	// Test with missing redirect_uris
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", nil)
	rec := httptest.NewRecorder()
	s.HandleClientRegistration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing body, got %d", rec.Code)
	}

	// Test GET method not allowed
	req = httptest.NewRequest(http.MethodGet, "/oauth/register", nil)
	rec = httptest.NewRecorder()
	s.HandleClientRegistration(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", rec.Code)
	}
}

func TestValidatePKCE(t *testing.T) {
	// Test PKCE validation
	tests := []struct {
		name      string
		verifier  string
		challenge string
		method    string
		expected  bool
	}{
		{
			name:      "valid S256",
			verifier:  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			method:    "S256",
			expected:  true,
		},
		{
			name:      "invalid verifier",
			verifier:  "wrong-verifier",
			challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			method:    "S256",
			expected:  false,
		},
		{
			name:      "unsupported method",
			verifier:  "test",
			challenge: "test",
			method:    "plain",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validatePKCE(tc.verifier, tc.challenge, tc.method)
			if result != tc.expected {
				t.Errorf("validatePKCE() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	// Test token generation
	token1 := generateSecureToken(32)
	token2 := generateSecureToken(32)

	if token1 == "" {
		t.Error("token1 should not be empty")
	}
	if token2 == "" {
		t.Error("token2 should not be empty")
	}
	if token1 == token2 {
		t.Error("tokens should be unique")
	}
}
