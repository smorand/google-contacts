// Package mcp provides the MCP (Model Context Protocol) server implementation
// for google-contacts, enabling AI assistants to manage contacts remotely.
// This file implements OAuth2 authentication endpoints for API key generation.
package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	people "google.golang.org/api/people/v1"

	"google-contacts/pkg/auth"
)

// OAuthScopes contains all OAuth2 scopes for Gmail and People APIs.
// Matches the scopes in pkg/auth/auth.go for consistency.
var OAuthScopes = []string{
	gmail.GmailModifyScope,
	gmail.GmailSendScope,
	gmail.GmailLabelsScope,
	people.ContactsScope,
	people.ContactsOtherReadonlyScope,
}

// stateEntry stores OAuth state parameters with expiration.
type stateEntry struct {
	CreatedAt time.Time
}

// AuthHandler manages OAuth2 authentication flows.
type AuthHandler struct {
	clientID       string
	clientSecret   string
	redirectURI    string
	baseURL        string // Base URL for constructing redirect URIs
	server         *Server
	oauthConfig    *oauth2.Config
	oauthConfigMu  sync.RWMutex
	stateMu        sync.RWMutex
	stateStore     map[string]stateEntry // In-memory state store with TTL
	stateExpiry    time.Duration         // How long states remain valid
	secretProject  string                // GCP project for Secret Manager
	secretName     string                // Secret name for OAuth credentials
	credentialFile string                // Local credential file path (fallback)
}

// AuthHandlerConfig holds configuration for creating an AuthHandler.
type AuthHandlerConfig struct {
	BaseURL        string // Base URL for redirect URIs (e.g., https://example.com)
	Server         *Server
	SecretProject  string // GCP project for Secret Manager
	SecretName     string // Secret Manager secret name for OAuth credentials
	CredentialFile string // Fallback: local credential file path
}

// NewAuthHandler creates a new AuthHandler with the given configuration.
func NewAuthHandler(cfg *AuthHandlerConfig) *AuthHandler {
	h := &AuthHandler{
		baseURL:        cfg.BaseURL,
		server:         cfg.Server,
		stateStore:     make(map[string]stateEntry),
		stateExpiry:    10 * time.Minute, // States expire after 10 minutes
		secretProject:  cfg.SecretProject,
		secretName:     cfg.SecretName,
		credentialFile: cfg.CredentialFile,
	}

	// Start background goroutine to clean up expired states
	go h.cleanupExpiredStates()

	return h
}

// cleanupExpiredStates periodically removes expired state entries.
func (h *AuthHandler) cleanupExpiredStates() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.stateMu.Lock()
		now := time.Now()
		for state, entry := range h.stateStore {
			if now.Sub(entry.CreatedAt) > h.stateExpiry {
				delete(h.stateStore, state)
			}
		}
		h.stateMu.Unlock()
	}
}

// loadOAuthCredentials loads OAuth client credentials from Secret Manager or file.
// Priority: 1) Secret Manager (if configured), 2) Local file (fallback)
func (h *AuthHandler) loadOAuthCredentials(ctx context.Context) error {
	h.oauthConfigMu.Lock()
	defer h.oauthConfigMu.Unlock()

	// Already loaded
	if h.oauthConfig != nil {
		return nil
	}

	var credentialsJSON []byte
	var err error

	// Try Secret Manager first
	if h.secretProject != "" && h.secretName != "" {
		credentialsJSON, err = h.loadFromSecretManager(ctx)
		if err != nil {
			log.Printf("Failed to load credentials from Secret Manager: %v", err)
			// Fall through to try file
		} else {
			log.Printf("OAuth credentials loaded from Secret Manager: %s/%s", h.secretProject, h.secretName)
		}
	}

	// Fall back to local file if Secret Manager didn't work
	if credentialsJSON == nil && h.credentialFile != "" {
		credentialsJSON, err = os.ReadFile(h.credentialFile)
		if err != nil {
			return fmt.Errorf("failed to read credentials file %s: %w", h.credentialFile, err)
		}
		log.Printf("OAuth credentials loaded from file: %s", h.credentialFile)
	}

	if credentialsJSON == nil {
		return fmt.Errorf("no OAuth credentials available: configure Secret Manager or credential file")
	}

	// Parse credentials
	config, err := google.ConfigFromJSON(credentialsJSON, OAuthScopes...)
	if err != nil {
		return fmt.Errorf("failed to parse OAuth credentials: %w", err)
	}

	// Set redirect URI based on base URL
	if h.baseURL != "" {
		config.RedirectURL = h.baseURL + "/auth/callback"
	} else {
		// Default for local development
		config.RedirectURL = "http://localhost:8080/auth/callback"
	}

	h.oauthConfig = config
	h.clientID = config.ClientID
	h.clientSecret = config.ClientSecret
	h.redirectURI = config.RedirectURL

	return nil
}

// loadFromSecretManager loads credentials from Google Secret Manager.
func (h *AuthHandler) loadFromSecretManager(ctx context.Context) ([]byte, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Secret Manager client: %w", err)
	}
	defer client.Close()

	// Access the latest version of the secret
	secretPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", h.secretProject, h.secretName)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access secret %s: %w", secretPath, err)
	}

	return result.Payload.Data, nil
}

// getOAuthConfig returns the OAuth2 configuration, loading it if necessary.
func (h *AuthHandler) getOAuthConfig(ctx context.Context) (*oauth2.Config, error) {
	h.oauthConfigMu.RLock()
	config := h.oauthConfig
	h.oauthConfigMu.RUnlock()

	if config != nil {
		return config, nil
	}

	if err := h.loadOAuthCredentials(ctx); err != nil {
		return nil, err
	}

	h.oauthConfigMu.RLock()
	defer h.oauthConfigMu.RUnlock()
	return h.oauthConfig, nil
}

// generateState creates a cryptographically secure random state token.
func (h *AuthHandler) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Store state with timestamp
	h.stateMu.Lock()
	h.stateStore[state] = stateEntry{CreatedAt: time.Now()}
	h.stateMu.Unlock()

	return state, nil
}

// validateState checks if the state parameter is valid and not expired.
// Returns true and removes the state from the store if valid.
func (h *AuthHandler) validateState(state string) bool {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	entry, exists := h.stateStore[state]
	if !exists {
		return false
	}

	// Check expiration
	if time.Since(entry.CreatedAt) > h.stateExpiry {
		delete(h.stateStore, state)
		return false
	}

	// Remove state after validation (single-use)
	delete(h.stateStore, state)
	return true
}

// HandleAuth initiates the OAuth2 authorization flow.
// GET /auth - Redirects to Google OAuth consent page.
func (h *AuthHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Load OAuth config
	config, err := h.getOAuthConfig(ctx)
	if err != nil {
		log.Printf("Failed to load OAuth config: %v", err)
		http.Error(w, "OAuth configuration error", http.StatusInternalServerError)
		return
	}

	// Generate state parameter for CSRF protection
	state, err := h.generateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Generate authorization URL
	// AccessTypeOffline ensures we get a refresh token
	// ApprovalForce ensures we always get a refresh token, even if user already authorized
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	log.Printf("Redirecting to OAuth consent page, state=%s...", state[:10])

	// Redirect to Google OAuth consent page
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback handles the OAuth2 callback from Google.
// GET /auth/callback?code=xxx&state=xxx
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check for OAuth error response
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		log.Printf("OAuth error: %s - %s", errParam, errDesc)
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errParam, errDesc), http.StatusBadRequest)
		return
	}

	// Validate state parameter
	state := r.URL.Query().Get("state")
	if state == "" {
		log.Printf("Missing state parameter")
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	if !h.validateState(state) {
		log.Printf("Invalid or expired state parameter")
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("Missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Load OAuth config
	config, err := h.getOAuthConfig(ctx)
	if err != nil {
		log.Printf("Failed to load OAuth config: %v", err)
		http.Error(w, "OAuth configuration error", http.StatusInternalServerError)
		return
	}

	// Exchange authorization code for tokens
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange code for tokens: %v", err)
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusBadRequest)
		return
	}

	// Get user email from the token (for logging/storage)
	userEmail, err := h.getUserEmail(ctx, config, token)
	if err != nil {
		log.Printf("Failed to get user email: %v", err)
		// Non-fatal, continue without email
		userEmail = ""
	}

	log.Printf("OAuth callback successful for user: %s, refresh_token_present=%v",
		userEmail, token.RefreshToken != "")

	// Check if we have a refresh token
	if token.RefreshToken == "" {
		log.Printf("Warning: No refresh token received. User may need to revoke access and re-authorize.")
		http.Error(w, "No refresh token received. Please revoke access at https://myaccount.google.com/permissions and try again.", http.StatusBadRequest)
		return
	}

	// Store token and generate API key using server's Firestore client
	// This will be implemented in US-00037
	// For now, we'll return a success page with the token info
	h.serveCallbackResponse(w, token, userEmail)
}

// getUserEmail fetches the user's email address using the OAuth token.
func (h *AuthHandler) getUserEmail(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (string, error) {
	client := config.Client(ctx, token)

	// Use People API to get user's primary email
	svc, err := people.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create People API client: %w", err)
	}

	// Get the authenticated user's profile
	person, err := svc.People.Get("people/me").PersonFields("emailAddresses").Do()
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}

	// Find primary email
	for _, email := range person.EmailAddresses {
		if email.Metadata != nil && email.Metadata.Primary {
			return email.Value, nil
		}
	}

	// Return first email if no primary found
	if len(person.EmailAddresses) > 0 {
		return person.EmailAddresses[0].Value, nil
	}

	return "", fmt.Errorf("no email address found")
}

// serveCallbackResponse sends the success response after OAuth callback.
// This temporary implementation returns token info for testing.
// Will be replaced with API key generation in US-00037.
func (h *AuthHandler) serveCallbackResponse(w http.ResponseWriter, token *oauth2.Token, userEmail string) {
	// For now, return a JSON response with token info
	// In US-00037, this will generate an API key and store the token in Firestore
	response := map[string]interface{}{
		"success":               true,
		"message":               "OAuth authentication successful",
		"user_email":            userEmail,
		"has_refresh_token":     token.RefreshToken != "",
		"access_token_expiry":   token.Expiry.Format(time.RFC3339),
		"next_step":             "API key generation will be implemented in US-00037",
		"refresh_token_preview": truncateToken(token.RefreshToken),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// truncateToken returns a truncated preview of a token for logging.
func truncateToken(token string) string {
	if len(token) <= 10 {
		return token
	}
	return token[:10] + "..."
}

// SetupAuthRoutes configures the HTTP routes for OAuth authentication.
// This should be called from the Server's Run method.
func (h *AuthHandler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth", h.HandleAuth)
	mux.HandleFunc("/auth/callback", h.HandleCallback)
}

// GetBaseURL returns the configured base URL.
func (h *AuthHandler) GetBaseURL() string {
	return h.baseURL
}

// GetRedirectURI returns the OAuth redirect URI.
func (h *AuthHandler) GetRedirectURI() string {
	h.oauthConfigMu.RLock()
	defer h.oauthConfigMu.RUnlock()
	return h.redirectURI
}

// LocalCredentialsPath returns the default local credentials path.
func LocalCredentialsPath() string {
	return auth.GetCredentialsPath() + "/" + auth.CredentialsFile
}
