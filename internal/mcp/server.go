// Package mcp provides the MCP (Model Context Protocol) server implementation
// for google-contacts, enabling AI assistants to manage contacts remotely.
package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"google-contacts/internal/contacts"
	"google-contacts/pkg/auth"
)

// Config holds the MCP server configuration.
type Config struct {
	Host           string
	Port           int
	BaseURL        string // Base URL for OAuth callbacks (e.g., https://example.com)
	SecretName     string // Secret Manager secret name for OAuth credentials
	SecretProject  string // GCP project for Secret Manager
	CredentialFile string // Local credential file path (fallback)
}

// Server wraps the MCP server and HTTP server.
type Server struct {
	config       *Config
	mcpServer    *mcp.Server
	httpServer   *http.Server
	oauth2Server *OAuth2Server
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *Config) *Server {
	// Create the MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "google-contacts",
		Version: "1.0.0",
	}, nil)

	return &Server{
		config:    cfg,
		mcpServer: mcpServer,
	}
}

// extractBearerToken extracts the token from the Authorization header.
// Expected format: "Bearer <token>"
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(authHeader, bearerPrefix)
}

// authMiddleware wraps an HTTP handler with OAuth2 Bearer token authentication.
// When no token is provided, returns 401 with WWW-Authenticate header pointing to the
// OAuth2 protected resource metadata endpoint (RFC 9728).
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract Bearer token from Authorization header
		accessToken := extractBearerToken(r)

		// If no token provided, return 401 with proper WWW-Authenticate header
		if accessToken == "" {
			// RFC 9728: WWW-Authenticate header with resource_metadata URL
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer resource_metadata="%s/.well-known/oauth-protected-resource"`,
				s.config.BaseURL,
			))
			http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
			return
		}

		// Validate the access token and get OAuth config
		if s.oauth2Server == nil {
			http.Error(w, "OAuth not configured", http.StatusInternalServerError)
			return
		}

		oauthConfig, token, err := s.oauth2Server.ValidateAccessToken(ctx, accessToken)
		if err != nil {
			log.Printf("Token validation error: %v", err)
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer error="invalid_token", resource_metadata="%s/.well-known/oauth-protected-resource"`,
				s.config.BaseURL,
			))
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// Inject OAuth config and token into context for People API
		ctx = auth.WithOAuthConfig(ctx, oauthConfig)
		ctx = auth.WithAccessToken(ctx, token)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// PhoneInput represents a phone number with type for MCP tools.
type PhoneInput struct {
	Value string `json:"value" jsonschema:"Phone number (e.g. +33612345678)"`
	Type  string `json:"type,omitempty" jsonschema:"Phone type: mobile work home main other. Default: mobile"`
}

// EmailInput represents an email address with type for MCP tools.
type EmailInput struct {
	Value string `json:"value" jsonschema:"Email address"`
	Type  string `json:"type,omitempty" jsonschema:"Email type: work home other. Default: work"`
}

// AddressInput represents a postal address with type for MCP tools.
type AddressInput struct {
	Value string `json:"value" jsonschema:"Address (e.g. 10 Rue Example 75001 Paris France)"`
	Type  string `json:"type,omitempty" jsonschema:"Address type: home work other. Default: home"`
}

// CreateInput is the input schema for contacts_create tool.
type CreateInput struct {
	FirstName string         `json:"firstName" jsonschema:"First name of the contact"`
	LastName  string         `json:"lastName" jsonschema:"Last name of the contact"`
	Phones    []PhoneInput   `json:"phones" jsonschema:"Phone numbers with optional types (required)"`
	Emails    []EmailInput   `json:"emails,omitempty" jsonschema:"Email addresses with optional types"`
	Addresses []AddressInput `json:"addresses,omitempty" jsonschema:"Postal addresses with optional types"`
	Company   string         `json:"company,omitempty" jsonschema:"Company name"`
	Position  string         `json:"position,omitempty" jsonschema:"Job title/position"`
	Notes     string         `json:"notes,omitempty" jsonschema:"Notes about the contact"`
	Birthday  string         `json:"birthday,omitempty" jsonschema:"Birthday in YYYY-MM-DD or --MM-DD format"`
}

// CreateOutput is the output schema for contacts_create tool.
type CreateOutput struct {
	ResourceName string `json:"resourceName" jsonschema:"Google Contact ID (e.g. people/c123456789)"`
	DisplayName  string `json:"displayName" jsonschema:"Full display name of the created contact"`
	Message      string `json:"message" jsonschema:"Success message"`
}

// SearchInput is the input schema for contacts_search tool.
type SearchInput struct {
	Query string `json:"query" jsonschema:"Search query (matches name phone email company)"`
}

// SearchResultItem represents a single search result for MCP output.
type SearchResultItem struct {
	ResourceName string `json:"resourceName" jsonschema:"Google Contact ID"`
	DisplayName  string `json:"displayName" jsonschema:"Full display name"`
	Phone        string `json:"phone,omitempty" jsonschema:"Primary phone number"`
	Email        string `json:"email,omitempty" jsonschema:"Primary email address"`
	Company      string `json:"company,omitempty" jsonschema:"Company name"`
	Position     string `json:"position,omitempty" jsonschema:"Job title"`
}

// SearchOutput is the output schema for contacts_search tool.
type SearchOutput struct {
	Results []SearchResultItem `json:"results" jsonschema:"List of matching contacts"`
	Count   int                `json:"count" jsonschema:"Number of results found"`
}

// ShowInput is the input schema for contacts_show tool.
type ShowInput struct {
	ContactID string `json:"contactId" jsonschema:"Contact ID (e.g. c123456789 or people/c123456789)"`
}

// PhoneOutput represents a phone number in contact details output.
type PhoneOutput struct {
	Value string `json:"value" jsonschema:"Phone number"`
	Type  string `json:"type" jsonschema:"Phone type (mobile work home etc)"`
}

// EmailOutput represents an email address in contact details output.
type EmailOutput struct {
	Value string `json:"value" jsonschema:"Email address"`
	Type  string `json:"type" jsonschema:"Email type (work home etc)"`
}

// AddressOutput represents a postal address in contact details output.
type AddressOutput struct {
	Value string `json:"value" jsonschema:"Full address"`
	Type  string `json:"type" jsonschema:"Address type (home work etc)"`
}

// ShowOutput is the output schema for contacts_show tool.
type ShowOutput struct {
	ResourceName string          `json:"resourceName" jsonschema:"Google Contact ID"`
	FirstName    string          `json:"firstName" jsonschema:"First name"`
	LastName     string          `json:"lastName" jsonschema:"Last name"`
	DisplayName  string          `json:"displayName" jsonschema:"Full display name"`
	Phones       []PhoneOutput   `json:"phones" jsonschema:"All phone numbers with types"`
	Emails       []EmailOutput   `json:"emails" jsonschema:"All email addresses with types"`
	Addresses    []AddressOutput `json:"addresses" jsonschema:"All postal addresses with types"`
	Company      string          `json:"company,omitempty" jsonschema:"Company name"`
	Position     string          `json:"position,omitempty" jsonschema:"Job title"`
	Notes        string          `json:"notes,omitempty" jsonschema:"Notes about contact"`
	Birthday     string          `json:"birthday,omitempty" jsonschema:"Birthday (YYYY-MM-DD or --MM-DD)"`
	UpdatedAt    string          `json:"updatedAt,omitempty" jsonschema:"Last update timestamp"`
}

// UpdateInput is the input schema for contacts_update tool.
type UpdateInput struct {
	ContactID       string         `json:"contactId" jsonschema:"Contact ID to update"`
	FirstName       string         `json:"firstName,omitempty" jsonschema:"New first name"`
	LastName        string         `json:"lastName,omitempty" jsonschema:"New last name"`
	Phones          []PhoneInput   `json:"phones,omitempty" jsonschema:"Replace ALL phones with these"`
	AddPhones       []PhoneInput   `json:"addPhones,omitempty" jsonschema:"Add phones without removing existing"`
	RemovePhones    []string       `json:"removePhones,omitempty" jsonschema:"Remove phones by value"`
	Emails          []EmailInput   `json:"emails,omitempty" jsonschema:"Replace ALL emails with these"`
	AddEmails       []EmailInput   `json:"addEmails,omitempty" jsonschema:"Add emails without removing existing"`
	RemoveEmails    []string       `json:"removeEmails,omitempty" jsonschema:"Remove emails by value"`
	Addresses       []AddressInput `json:"addresses,omitempty" jsonschema:"Replace ALL addresses with these"`
	AddAddresses    []AddressInput `json:"addAddresses,omitempty" jsonschema:"Add addresses without removing existing"`
	RemoveAddresses []string       `json:"removeAddresses,omitempty" jsonschema:"Remove addresses by street content"`
	Company         string         `json:"company,omitempty" jsonschema:"New company name"`
	Position        string         `json:"position,omitempty" jsonschema:"New job title"`
	Notes           string         `json:"notes,omitempty" jsonschema:"New notes"`
	Birthday        string         `json:"birthday,omitempty" jsonschema:"New birthday (YYYY-MM-DD or --MM-DD)"`
	ClearBirthday   bool           `json:"clearBirthday,omitempty" jsonschema:"Set true to remove birthday"`
}

// UpdateOutput is the output schema for contacts_update tool.
type UpdateOutput struct {
	ShowOutput
	Message string `json:"message" jsonschema:"Success message"`
}

// DeleteInput is the input schema for contacts_delete tool.
type DeleteInput struct {
	ContactID string `json:"contactId" jsonschema:"Contact ID to delete"`
}

// DeleteOutput is the output schema for contacts_delete tool.
type DeleteOutput struct {
	Message     string `json:"message" jsonschema:"Success message"`
	DeletedID   string `json:"deletedId" jsonschema:"ID of deleted contact"`
	DisplayName string `json:"displayName,omitempty" jsonschema:"Name of deleted contact"`
}

// RegisterTools registers all contact management tools with the MCP server.
func (s *Server) RegisterTools() {
	// Register ping tool for connectivity testing
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ping",
		Description: "Test connectivity with the MCP server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (
		*mcp.CallToolResult,
		struct {
			Message string `json:"message"`
			Time    string `json:"time"`
		},
		error,
	) {
		return nil, struct {
			Message string `json:"message"`
			Time    string `json:"time"`
		}{
			Message: "pong",
			Time:    time.Now().Format(time.RFC3339),
		}, nil
	})

	// Register contacts_create tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "contacts_create",
		Description: "Create a new contact in Google Contacts",
	}, s.handleCreateContact)

	// Register contacts_search tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "contacts_search",
		Description: "Search contacts by name, phone, email, or company",
	}, s.handleSearchContacts)

	// Register contacts_show tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "contacts_show",
		Description: "Get full details of a contact by ID",
	}, s.handleShowContact)

	// Register contacts_update tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "contacts_update",
		Description: "Update an existing contact (only specified fields are modified)",
	}, s.handleUpdateContact)

	// Register contacts_delete tool
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "contacts_delete",
		Description: "Delete a contact by ID",
	}, s.handleDeleteContact)
}

// handleCreateContact implements the contacts_create MCP tool.
func (s *Server) handleCreateContact(ctx context.Context, req *mcp.CallToolRequest, input CreateInput) (
	*mcp.CallToolResult,
	CreateOutput,
	error,
) {
	// Validate required fields
	if input.FirstName == "" {
		return nil, CreateOutput{}, fmt.Errorf("firstName is required")
	}
	if input.LastName == "" {
		return nil, CreateOutput{}, fmt.Errorf("lastName is required")
	}
	if len(input.Phones) == 0 {
		return nil, CreateOutput{}, fmt.Errorf("at least one phone is required")
	}

	// Get the contacts service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return nil, CreateOutput{}, fmt.Errorf("failed to get contacts service: %w", err)
	}

	// Convert input to ContactInput
	contactInput := contacts.ContactInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Company:   input.Company,
		Position:  input.Position,
		Notes:     input.Notes,
		Birthday:  input.Birthday,
	}

	// Convert phones
	for _, phone := range input.Phones {
		phoneType := phone.Type
		if phoneType == "" {
			phoneType = "mobile"
		}
		contactInput.Phones = append(contactInput.Phones, contacts.PhoneEntry{
			Value: phone.Value,
			Type:  phoneType,
		})
	}

	// Convert emails
	for _, email := range input.Emails {
		emailType := email.Type
		if emailType == "" {
			emailType = "work"
		}
		contactInput.Emails = append(contactInput.Emails, contacts.EmailEntry{
			Value: email.Value,
			Type:  emailType,
		})
	}

	// Convert addresses
	for _, addr := range input.Addresses {
		addrType := addr.Type
		if addrType == "" {
			addrType = "home"
		}
		contactInput.Addresses = append(contactInput.Addresses, contacts.AddressEntry{
			Value: addr.Value,
			Type:  addrType,
		})
	}

	// Create the contact
	created, err := srv.CreateContact(ctx, contactInput)
	if err != nil {
		return nil, CreateOutput{}, fmt.Errorf("failed to create contact: %w", err)
	}

	return nil, CreateOutput{
		ResourceName: created.ResourceName,
		DisplayName:  created.DisplayName,
		Message:      fmt.Sprintf("Contact '%s' created successfully", created.DisplayName),
	}, nil
}

// handleSearchContacts implements the contacts_search MCP tool.
func (s *Server) handleSearchContacts(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (
	*mcp.CallToolResult,
	SearchOutput,
	error,
) {
	// Validate required fields
	if input.Query == "" {
		return nil, SearchOutput{}, fmt.Errorf("query is required")
	}

	// Get the contacts service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return nil, SearchOutput{}, fmt.Errorf("failed to get contacts service: %w", err)
	}

	// Search contacts
	results, err := srv.SearchContacts(ctx, input.Query)
	if err != nil {
		return nil, SearchOutput{}, fmt.Errorf("failed to search contacts: %w", err)
	}

	// Convert results - always initialize Results to empty slice to avoid null in JSON
	output := SearchOutput{
		Count:   len(results),
		Results: []SearchResultItem{},
	}
	for _, r := range results {
		output.Results = append(output.Results, SearchResultItem{
			ResourceName: r.ResourceName,
			DisplayName:  r.DisplayName,
			Phone:        r.Phone,
			Email:        r.Email,
			Company:      r.Company,
			Position:     r.Position,
		})
	}

	return nil, output, nil
}

// handleShowContact implements the contacts_show MCP tool.
func (s *Server) handleShowContact(ctx context.Context, req *mcp.CallToolRequest, input ShowInput) (
	*mcp.CallToolResult,
	ShowOutput,
	error,
) {
	// Validate required fields
	if input.ContactID == "" {
		return nil, ShowOutput{}, fmt.Errorf("contactId is required")
	}

	// Get the contacts service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return nil, ShowOutput{}, fmt.Errorf("failed to get contacts service: %w", err)
	}

	// Get contact details
	details, err := srv.GetContactDetails(ctx, input.ContactID)
	if err != nil {
		return nil, ShowOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}

	// Convert to output
	output := ShowOutput{
		ResourceName: details.ResourceName,
		FirstName:    details.FirstName,
		LastName:     details.LastName,
		DisplayName:  details.DisplayName,
		Company:      details.Company,
		Position:     details.Position,
		Notes:        details.Notes,
		Birthday:     details.Birthday,
		UpdatedAt:    details.UpdatedAt,
	}

	// Convert phones
	for _, phone := range details.Phones {
		output.Phones = append(output.Phones, PhoneOutput{
			Value: phone.Value,
			Type:  phone.Type,
		})
	}

	// Convert emails
	for _, email := range details.Emails {
		output.Emails = append(output.Emails, EmailOutput{
			Value: email.Value,
			Type:  email.Type,
		})
	}

	// Convert addresses
	for _, addr := range details.Addresses {
		output.Addresses = append(output.Addresses, AddressOutput{
			Value: addr.Value,
			Type:  addr.Type,
		})
	}

	return nil, output, nil
}

// handleUpdateContact implements the contacts_update MCP tool.
func (s *Server) handleUpdateContact(ctx context.Context, req *mcp.CallToolRequest, input UpdateInput) (
	*mcp.CallToolResult,
	UpdateOutput,
	error,
) {
	// Validate required fields
	if input.ContactID == "" {
		return nil, UpdateOutput{}, fmt.Errorf("contactId is required")
	}

	// Get the contacts service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return nil, UpdateOutput{}, fmt.Errorf("failed to get contacts service: %w", err)
	}

	// Build UpdateInput with pointers for optional fields
	updateInput := contacts.UpdateInput{
		ClearBirthday: input.ClearBirthday,
	}

	// Set string pointers only if non-empty
	if input.FirstName != "" {
		updateInput.FirstName = &input.FirstName
	}
	if input.LastName != "" {
		updateInput.LastName = &input.LastName
	}
	if input.Company != "" {
		updateInput.Company = &input.Company
	}
	if input.Position != "" {
		updateInput.Position = &input.Position
	}
	if input.Notes != "" {
		updateInput.Notes = &input.Notes
	}
	if input.Birthday != "" {
		updateInput.Birthday = &input.Birthday
	}

	// Convert phones
	for _, phone := range input.Phones {
		phoneType := phone.Type
		if phoneType == "" {
			phoneType = "mobile"
		}
		updateInput.Phones = append(updateInput.Phones, contacts.PhoneEntry{
			Value: phone.Value,
			Type:  phoneType,
		})
	}

	for _, phone := range input.AddPhones {
		phoneType := phone.Type
		if phoneType == "" {
			phoneType = "mobile"
		}
		updateInput.AddPhones = append(updateInput.AddPhones, contacts.PhoneEntry{
			Value: phone.Value,
			Type:  phoneType,
		})
	}

	updateInput.RemovePhones = input.RemovePhones

	// Convert emails
	for _, email := range input.Emails {
		emailType := email.Type
		if emailType == "" {
			emailType = "work"
		}
		updateInput.Emails = append(updateInput.Emails, contacts.EmailEntry{
			Value: email.Value,
			Type:  emailType,
		})
	}

	for _, email := range input.AddEmails {
		emailType := email.Type
		if emailType == "" {
			emailType = "work"
		}
		updateInput.AddEmails = append(updateInput.AddEmails, contacts.EmailEntry{
			Value: email.Value,
			Type:  emailType,
		})
	}

	updateInput.RemoveEmails = input.RemoveEmails

	// Convert addresses
	for _, addr := range input.Addresses {
		addrType := addr.Type
		if addrType == "" {
			addrType = "home"
		}
		updateInput.Addresses = append(updateInput.Addresses, contacts.AddressEntry{
			Value: addr.Value,
			Type:  addrType,
		})
	}

	for _, addr := range input.AddAddresses {
		addrType := addr.Type
		if addrType == "" {
			addrType = "home"
		}
		updateInput.AddAddresses = append(updateInput.AddAddresses, contacts.AddressEntry{
			Value: addr.Value,
			Type:  addrType,
		})
	}

	updateInput.RemoveAddresses = input.RemoveAddresses

	// Update the contact
	details, err := srv.UpdateContact(ctx, input.ContactID, updateInput)
	if err != nil {
		return nil, UpdateOutput{}, fmt.Errorf("failed to update contact: %w", err)
	}

	// Convert to output
	output := UpdateOutput{
		Message: fmt.Sprintf("Contact '%s' updated successfully", details.DisplayName),
	}
	output.ResourceName = details.ResourceName
	output.FirstName = details.FirstName
	output.LastName = details.LastName
	output.DisplayName = details.DisplayName
	output.Company = details.Company
	output.Position = details.Position
	output.Notes = details.Notes
	output.Birthday = details.Birthday
	output.UpdatedAt = details.UpdatedAt

	// Convert phones
	for _, phone := range details.Phones {
		output.Phones = append(output.Phones, PhoneOutput{
			Value: phone.Value,
			Type:  phone.Type,
		})
	}

	// Convert emails
	for _, email := range details.Emails {
		output.Emails = append(output.Emails, EmailOutput{
			Value: email.Value,
			Type:  email.Type,
		})
	}

	// Convert addresses
	for _, addr := range details.Addresses {
		output.Addresses = append(output.Addresses, AddressOutput{
			Value: addr.Value,
			Type:  addr.Type,
		})
	}

	return nil, output, nil
}

// handleDeleteContact implements the contacts_delete MCP tool.
func (s *Server) handleDeleteContact(ctx context.Context, req *mcp.CallToolRequest, input DeleteInput) (
	*mcp.CallToolResult,
	DeleteOutput,
	error,
) {
	// Validate required fields
	if input.ContactID == "" {
		return nil, DeleteOutput{}, fmt.Errorf("contactId is required")
	}

	// Get the contacts service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return nil, DeleteOutput{}, fmt.Errorf("failed to get contacts service: %w", err)
	}

	// Get contact details before deletion for confirmation
	details, err := srv.GetContactDetails(ctx, input.ContactID)
	if err != nil {
		return nil, DeleteOutput{}, fmt.Errorf("failed to get contact: %w", err)
	}

	displayName := details.DisplayName

	// Delete the contact
	err = srv.DeleteContact(ctx, input.ContactID)
	if err != nil {
		return nil, DeleteOutput{}, fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil, DeleteOutput{
		Message:     fmt.Sprintf("Contact '%s' deleted successfully", displayName),
		DeletedID:   details.ResourceName,
		DisplayName: displayName,
	}, nil
}

// Run starts the HTTP server and blocks until shutdown.
func (s *Server) Run(ctx context.Context) error {
	// Register tools
	s.RegisterTools()

	// Create the streamable HTTP handler for MCP
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: false, // Enable session tracking
	})

	// Create HTTP mux for routing
	mux := http.NewServeMux()

	// Determine credential file path (default to local credentials)
	credFile := s.config.CredentialFile
	if credFile == "" {
		credFile = LocalCredentialsPath()
	}

	// Initialize OAuth2 server
	s.oauth2Server = NewOAuth2Server(&OAuth2ServerConfig{
		BaseURL:        s.config.BaseURL,
		SecretProject:  s.config.SecretProject,
		SecretName:     s.config.SecretName,
		CredentialFile: credFile,
	})

	// Register OAuth2 routes (not protected by auth)
	s.oauth2Server.SetupRoutes(mux)
	log.Println("OAuth2 endpoints enabled:")
	log.Println("  - /.well-known/oauth-protected-resource")
	log.Println("  - /.well-known/oauth-authorization-server")
	log.Println("  - /oauth/register")
	log.Println("  - /oauth/authorize")
	log.Println("  - /oauth/callback")
	log.Println("  - /oauth/token")

	// Health check endpoint (not protected by auth)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap MCP handler with authentication middleware
	authedMCPHandler := s.authMiddleware(mcpHandler)

	// MCP endpoint (protected by OAuth2 Bearer token auth)
	mux.Handle("/", authedMCPHandler)

	log.Println("Authentication mode: OAuth2 Bearer tokens")

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting MCP server on %s", addr)
		log.Printf("Base URL: %s", s.config.BaseURL)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Printf("Received signal %v, shutting down...", sig)
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	log.Println("MCP server stopped")
	return nil
}

// LocalCredentialsPath returns the default local credentials path.
func LocalCredentialsPath() string {
	return auth.GetCredentialsPath() + "/" + auth.CredentialsFile
}
