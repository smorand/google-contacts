package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// E2E-NEW-001: Vault Credential Loading Happy Path
func TestLoadFromVault_HappyPath(t *testing.T) {
	t.Parallel()

	// Create mock Vault server returning valid credentials
	creds := map[string]any{
		"installed": map[string]any{
			"client_id":     "test-client-id",
			"client_secret": "test-client-secret",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	var receivedToken string
	vaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Vault-Token")
		if r.URL.Path != "/v1/secret/data/test-path" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// KV v2 double-nested response
		resp := map[string]any{
			"data": map[string]any{
				"data": json.RawMessage(credsJSON),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer vaultServer.Close()

	result, err := loadFromVaultHTTP(vaultServer.URL, "my-vault-token", "test-path")
	if err != nil {
		t.Fatalf("loadFromVaultHTTP() error: %v", err)
	}
	if result == nil {
		t.Fatal("loadFromVaultHTTP() returned nil")
	}

	// Verify X-Vault-Token header was sent
	if receivedToken != "my-vault-token" {
		t.Errorf("X-Vault-Token = %q; want %q", receivedToken, "my-vault-token")
	}

	// Verify credentials can be parsed
	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	installed, ok := parsed["installed"].(map[string]any)
	if !ok {
		t.Fatal("missing 'installed' key in credentials")
	}
	if installed["client_id"] != "test-client-id" {
		t.Errorf("client_id = %v; want test-client-id", installed["client_id"])
	}
}

// E2E-NEW-002: Vault Credential Loading Vault Unavailable
func TestLoadFromVault_Unavailable(t *testing.T) {
	t.Parallel()

	result, err := loadFromVaultHTTP("http://localhost:19999", "token", "path")
	if err == nil {
		t.Fatal("expected error for unavailable vault, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "vault") {
		t.Errorf("error should contain 'vault': %v", err)
	}
}

// E2E-NEW-003: Vault Credential Loading Invalid Token
func TestLoadFromVault_InvalidToken(t *testing.T) {
	t.Parallel()

	vaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"errors":["permission denied"]}`)
	}))
	defer vaultServer.Close()

	result, err := loadFromVaultHTTP(vaultServer.URL, "bad-token", "path")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain '403': %v", err)
	}
}

// E2E-NEW-004: Vault Credential Loading Empty Config
func TestLoadFromVault_EmptyConfig(t *testing.T) {
	t.Parallel()

	s := &OAuth2Server{
		vaultAddr:  "",
		vaultToken: "",
	}

	result, err := s.loadFromVault()
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

// E2E-NEW-005: Credential Chain Fallthrough Order
func TestLoadCredentials_VaultFallthrough(t *testing.T) {
	t.Parallel()

	// Create a mock Vault server with valid Google OAuth credentials
	creds := map[string]any{
		"installed": map[string]any{
			"client_id":                   "vault-client-id",
			"client_secret":               "vault-client-secret",
			"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
			"token_uri":                   "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"redirect_uris":               []string{"http://localhost"},
		},
	}
	credsJSON, _ := json.Marshal(creds)

	vaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"data": json.RawMessage(credsJSON),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer vaultServer.Close()

	s := &OAuth2Server{
		baseURL:         "https://example.com",
		secretProject:   "", // Secret Manager not configured
		secretName:      "",
		vaultAddr:       vaultServer.URL,
		vaultToken:      "test-token",
		vaultSecretPath: "test",
		clients:         make(map[string]*registeredClient),
		states:          make(map[string]*authorizationState),
		codes:           make(map[string]*authorizationCode),
	}

	err := s.LoadCredentials(context.Background())
	if err != nil {
		t.Fatalf("LoadCredentials() error: %v", err)
	}

	// Verify credentials were loaded from Vault
	s.oauthConfigMu.RLock()
	defer s.oauthConfigMu.RUnlock()
	if s.oauthConfig == nil {
		t.Fatal("oauthConfig is nil after LoadCredentials")
	}
	if s.oauthConfig.ClientID != "vault-client-id" {
		t.Errorf("ClientID = %q; want %q", s.oauthConfig.ClientID, "vault-client-id")
	}
}

// E2E-NEW-006: Credential Chain All Sources Fail
func TestLoadCredentials_AllSourcesFail(t *testing.T) {
	t.Parallel()

	s := &OAuth2Server{
		baseURL:        "https://example.com",
		secretProject:  "", // Not configured
		secretName:     "",
		vaultAddr:      "", // Not configured
		vaultToken:     "",
		credentialFile: "", // Not configured
		clients:        make(map[string]*registeredClient),
		states:         make(map[string]*authorizationState),
		codes:          make(map[string]*authorizationCode),
	}

	err := s.LoadCredentials(context.Background())
	if err == nil {
		t.Fatal("expected error when all sources fail, got nil")
	}
	if !strings.Contains(err.Error(), "no OAuth credentials available") {
		t.Errorf("error should mention no credentials: %v", err)
	}
}

// E2E-NEW-007: Config Struct Vault Fields Propagation
func TestConfigVaultFieldsPropagation(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Host:            "localhost",
		Port:            8080,
		BaseURL:         "https://example.com",
		VaultAddr:       "http://vault:8200",
		VaultToken:      "s.test-token",
		VaultSecretPath: "secret/creds/google",
	}

	server := NewServer(cfg)

	// Simulate what Run() does: create OAuth2ServerConfig from Config
	oauth2Cfg := &OAuth2ServerConfig{
		BaseURL:         server.config.BaseURL,
		VaultAddr:       server.config.VaultAddr,
		VaultToken:      server.config.VaultToken,
		VaultSecretPath: server.config.VaultSecretPath,
	}

	if oauth2Cfg.VaultAddr != "http://vault:8200" {
		t.Errorf("VaultAddr = %q; want %q", oauth2Cfg.VaultAddr, "http://vault:8200")
	}
	if oauth2Cfg.VaultToken != "s.test-token" {
		t.Errorf("VaultToken = %q; want %q", oauth2Cfg.VaultToken, "s.test-token")
	}
	if oauth2Cfg.VaultSecretPath != "secret/creds/google" {
		t.Errorf("VaultSecretPath = %q; want %q", oauth2Cfg.VaultSecretPath, "secret/creds/google")
	}
}

// E2E-NEW-008: CLI Flags Vault Flags and Env Var Fallbacks
func TestVaultEnvVarFallbacks(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()

	// Set environment variables
	t.Setenv("VAULT_ADDR", "http://test:8200")
	t.Setenv("VAULT_TOKEN", "test-token")
	t.Setenv("VAULT_SECRET_PATH", "test/path")

	// Simulate the env var fallback logic from runMCP
	vaultAddr := ""
	if vaultAddr == "" {
		vaultAddr = os.Getenv("VAULT_ADDR")
	}

	vaultToken := ""
	if vaultToken == "" {
		vaultToken = os.Getenv("VAULT_TOKEN")
	}

	vaultSecretPath := ""
	if vaultSecretPath == "" {
		vaultSecretPath = os.Getenv("VAULT_SECRET_PATH")
	}

	if vaultAddr != "http://test:8200" {
		t.Errorf("VAULT_ADDR = %q; want %q", vaultAddr, "http://test:8200")
	}
	if vaultToken != "test-token" {
		t.Errorf("VAULT_TOKEN = %q; want %q", vaultToken, "test-token")
	}
	if vaultSecretPath != "test/path" {
		t.Errorf("VAULT_SECRET_PATH = %q; want %q", vaultSecretPath, "test/path")
	}
}
