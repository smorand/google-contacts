// Package mcp provides the MCP (Model Context Protocol) server implementation.
package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
