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

func TestValidateAPIKey_NoAuthConfigured(t *testing.T) {
	// When no auth is configured, all requests should be allowed
	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "",
		FirestoreProject: "",
	}
	server := NewServer(cfg)
	ctx := context.Background()

	refreshToken, valid, err := server.validateAPIKey(ctx, "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true when no auth configured")
	}
	if refreshToken != "" {
		t.Errorf("expected empty refresh token, got %q", refreshToken)
	}

	// Even with a random API key, should be valid
	refreshToken, valid, err = server.validateAPIKey(ctx, "some-key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid=true when no auth configured")
	}
	if refreshToken != "" {
		t.Errorf("expected empty refresh token, got %q", refreshToken)
	}
}

func TestValidateAPIKey_StaticKey(t *testing.T) {
	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "secret-key-123",
		FirestoreProject: "",
	}
	server := NewServer(cfg)
	ctx := context.Background()

	tests := []struct {
		name               string
		apiKey             string
		expectedValid      bool
		expectedHasRefresh bool
	}{
		{
			name:               "correct static key",
			apiKey:             "secret-key-123",
			expectedValid:      true,
			expectedHasRefresh: false,
		},
		{
			name:               "wrong static key",
			apiKey:             "wrong-key",
			expectedValid:      false,
			expectedHasRefresh: false,
		},
		{
			name:               "empty key when auth required",
			apiKey:             "",
			expectedValid:      false,
			expectedHasRefresh: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refreshToken, valid, err := server.validateAPIKey(ctx, tc.apiKey)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if valid != tc.expectedValid {
				t.Errorf("valid = %v, want %v", valid, tc.expectedValid)
			}
			hasRefresh := refreshToken != ""
			if hasRefresh != tc.expectedHasRefresh {
				t.Errorf("hasRefresh = %v, want %v", hasRefresh, tc.expectedHasRefresh)
			}
		})
	}
}

func TestAuthMiddleware_NoAuthConfigured(t *testing.T) {
	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "",
		FirestoreProject: "",
	}
	server := NewServer(cfg)

	// Create a test handler that records if it was called
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	// Request without any auth header should succeed
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("expected next handler to be called")
	}
}

func TestAuthMiddleware_StaticKey_Valid(t *testing.T) {
	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "secret-key",
		FirestoreProject: "",
	}
	server := NewServer(cfg)

	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("expected next handler to be called")
	}
}

func TestAuthMiddleware_StaticKey_Invalid(t *testing.T) {
	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "secret-key",
		FirestoreProject: "",
	}
	server := NewServer(cfg)

	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	tests := []struct {
		name       string
		authHeader string
	}{
		{"wrong key", "Bearer wrong-key"},
		{"no header", ""},
		{"invalid format", "Basic dXNlcjpwYXNz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rec.Code)
			}
			if called {
				t.Error("expected next handler NOT to be called")
			}
		})
	}
}

func TestAuthMiddleware_ContextPropagation(t *testing.T) {
	// This test verifies that when auth middleware processes a request,
	// it properly propagates context values to the next handler.
	// This is crucial for per-user token integration where the refresh token
	// is injected into context for tool handlers to use.

	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "", // No static key - allows any request
		FirestoreProject: "",
	}
	server := NewServer(cfg)

	var receivedCtx context.Context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	// Create request with custom context value
	type testKey string
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), testKey("test"), "value"))
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify context was propagated
	if receivedCtx == nil {
		t.Fatal("context was not propagated to handler")
	}
	if v := receivedCtx.Value(testKey("test")); v != "value" {
		t.Errorf("context value not propagated, got %v", v)
	}
}

func TestAuthMiddleware_RefreshTokenInjection(t *testing.T) {
	// This test verifies that auth middleware does NOT inject refresh token
	// when using static API key (refresh token only comes from Firestore).
	// Static API key mode uses local file-based auth instead.

	cfg := &Config{
		Host:             "localhost",
		Port:             8080,
		APIKey:           "static-key",
		FirestoreProject: "",
	}
	server := NewServer(cfg)

	var receivedCtx context.Context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authMiddleware(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer static-key")
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify no refresh token in context (static key mode uses file-based auth)
	if receivedCtx == nil {
		t.Fatal("context was not propagated to handler")
	}

	// The refresh token should NOT be present for static key auth
	// (refresh token injection only happens with Firestore-based keys)
	token, ok := auth.GetRefreshTokenFromContext(receivedCtx)
	if ok || token != "" {
		t.Errorf("expected no refresh token in context for static key auth, got %q", token)
	}
}

func TestAuthMiddleware_WithRefreshToken(t *testing.T) {
	// This test simulates what happens when a refresh token is injected into context.
	// While we can't test Firestore integration in unit tests, we can verify that
	// the auth package's context functions work correctly.

	// Test the auth package's context functions directly
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
	// external services (Firestore, Google API).

	// 1. User authenticates via OAuth, gets API key stored in Firestore
	//    (tested in auth_test.go)

	// 2. Request arrives with API key in Authorization header
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	apiKey := "test-api-key-uuid"
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Verify header extraction works
	extracted := extractBearerToken(req)
	if extracted != apiKey {
		t.Errorf("extractBearerToken() = %q, want %q", extracted, apiKey)
	}

	// 3. When Firestore validates the key, it returns a refresh token
	//    (this would be mocked in integration tests)
	simulatedRefreshToken := "1//0gXXXXXX"

	// 4. Middleware injects refresh token into context
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
