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

	"cloud.google.com/go/firestore"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"google-contacts/internal/contacts"
	"google-contacts/pkg/auth"
)

// Config holds the MCP server configuration.
type Config struct {
	Host             string
	Port             int
	APIKey           string // Static API key for authentication (optional)
	FirestoreProject string // GCP project for Firestore API key validation (optional)
}

// Server wraps the MCP server and HTTP server.
type Server struct {
	config          *Config
	mcpServer       *mcp.Server
	httpServer      *http.Server
	firestoreClient *firestore.Client
}

// APIKeyDocument represents the structure stored in Firestore for API keys.
// Collection: api_keys, Document ID: the API key itself
type APIKeyDocument struct {
	RefreshToken string `firestore:"refresh_token"`
	UserEmail    string `firestore:"user_email,omitempty"`
	CreatedAt    string `firestore:"created_at,omitempty"`
	Description  string `firestore:"description,omitempty"`
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

// initFirestore initializes the Firestore client if a project is configured.
func (s *Server) initFirestore(ctx context.Context) error {
	if s.config.FirestoreProject == "" {
		return nil
	}

	client, err := firestore.NewClient(ctx, s.config.FirestoreProject)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %w", err)
	}
	s.firestoreClient = client
	log.Printf("Firestore client initialized for project: %s", s.config.FirestoreProject)
	return nil
}

// validateAPIKey validates an API key and returns the associated refresh token.
// Returns:
// - refreshToken: the OAuth refresh token if valid
// - valid: true if the API key is valid
// - err: error if validation failed unexpectedly
func (s *Server) validateAPIKey(ctx context.Context, apiKey string) (refreshToken string, valid bool, err error) {
	// No authentication configured - allow unauthenticated access
	if s.config.APIKey == "" && s.config.FirestoreProject == "" {
		return "", true, nil
	}

	// API key is required when auth is configured
	if apiKey == "" {
		return "", false, nil
	}

	// Check static API key first (for local development)
	if s.config.APIKey != "" {
		if apiKey == s.config.APIKey {
			// Static API key is valid, no refresh token (will use local token file)
			return "", true, nil
		}
		// Static key configured but doesn't match
		return "", false, nil
	}

	// Check Firestore for API key validation
	if s.firestoreClient != nil {
		doc, err := s.firestoreClient.Collection("api_keys").Doc(apiKey).Get(ctx)
		if err != nil {
			// Document not found or other error - treat as invalid key
			log.Printf("API key validation failed: %v", err)
			return "", false, nil
		}

		var keyDoc APIKeyDocument
		if err := doc.DataTo(&keyDoc); err != nil {
			log.Printf("Failed to parse API key document: %v", err)
			return "", false, nil
		}

		if keyDoc.RefreshToken == "" {
			log.Printf("API key document has no refresh token")
			return "", false, nil
		}

		return keyDoc.RefreshToken, true, nil
	}

	return "", false, nil
}

// extractBearerToken extracts the API key from the Authorization header.
// Expected format: "Bearer <api_key>"
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

// authMiddleware wraps an HTTP handler with API key authentication.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract API key from Authorization header
		apiKey := extractBearerToken(r)

		// Validate the API key
		refreshToken, valid, err := s.validateAPIKey(ctx, apiKey)
		if err != nil {
			log.Printf("API key validation error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !valid {
			http.Error(w, "Unauthorized: invalid or missing API key", http.StatusUnauthorized)
			return
		}

		// If we have a refresh token from Firestore, inject it into context
		if refreshToken != "" {
			ctx = auth.WithRefreshToken(ctx, refreshToken)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// PhoneInput represents a phone number with type for MCP tools.
type PhoneInput struct {
	Value string `json:"value" jsonschema:"description=Phone number (e.g. +33612345678)"`
	Type  string `json:"type,omitempty" jsonschema:"description=Phone type: mobile work home main other. Default: mobile"`
}

// EmailInput represents an email address with type for MCP tools.
type EmailInput struct {
	Value string `json:"value" jsonschema:"description=Email address"`
	Type  string `json:"type,omitempty" jsonschema:"description=Email type: work home other. Default: work"`
}

// AddressInput represents a postal address with type for MCP tools.
type AddressInput struct {
	Value string `json:"value" jsonschema:"description=Address (e.g. 10 Rue Example 75001 Paris France)"`
	Type  string `json:"type,omitempty" jsonschema:"description=Address type: home work other. Default: home"`
}

// CreateInput is the input schema for contacts_create tool.
type CreateInput struct {
	FirstName string         `json:"firstName" jsonschema:"required,description=First name of the contact"`
	LastName  string         `json:"lastName" jsonschema:"required,description=Last name of the contact"`
	Phones    []PhoneInput   `json:"phones" jsonschema:"required,description=Phone numbers with optional types"`
	Emails    []EmailInput   `json:"emails,omitempty" jsonschema:"description=Email addresses with optional types"`
	Addresses []AddressInput `json:"addresses,omitempty" jsonschema:"description=Postal addresses with optional types"`
	Company   string         `json:"company,omitempty" jsonschema:"description=Company name"`
	Position  string         `json:"position,omitempty" jsonschema:"description=Job title/position"`
	Notes     string         `json:"notes,omitempty" jsonschema:"description=Notes about the contact"`
	Birthday  string         `json:"birthday,omitempty" jsonschema:"description=Birthday in YYYY-MM-DD or --MM-DD format"`
}

// CreateOutput is the output schema for contacts_create tool.
type CreateOutput struct {
	ResourceName string `json:"resourceName" jsonschema:"description=Google Contact ID (e.g. people/c123456789)"`
	DisplayName  string `json:"displayName" jsonschema:"description=Full display name of the created contact"`
	Message      string `json:"message" jsonschema:"description=Success message"`
}

// SearchInput is the input schema for contacts_search tool.
type SearchInput struct {
	Query string `json:"query" jsonschema:"required,description=Search query (matches name phone email company)"`
}

// SearchResultItem represents a single search result for MCP output.
type SearchResultItem struct {
	ResourceName string `json:"resourceName" jsonschema:"description=Google Contact ID"`
	DisplayName  string `json:"displayName" jsonschema:"description=Full display name"`
	Phone        string `json:"phone,omitempty" jsonschema:"description=Primary phone number"`
	Email        string `json:"email,omitempty" jsonschema:"description=Primary email address"`
	Company      string `json:"company,omitempty" jsonschema:"description=Company name"`
	Position     string `json:"position,omitempty" jsonschema:"description=Job title"`
}

// SearchOutput is the output schema for contacts_search tool.
type SearchOutput struct {
	Results []SearchResultItem `json:"results" jsonschema:"description=List of matching contacts"`
	Count   int                `json:"count" jsonschema:"description=Number of results found"`
}

// ShowInput is the input schema for contacts_show tool.
type ShowInput struct {
	ContactID string `json:"contactId" jsonschema:"required,description=Contact ID (e.g. c123456789 or people/c123456789)"`
}

// PhoneOutput represents a phone number in contact details output.
type PhoneOutput struct {
	Value string `json:"value" jsonschema:"description=Phone number"`
	Type  string `json:"type" jsonschema:"description=Phone type (mobile work home etc)"`
}

// EmailOutput represents an email address in contact details output.
type EmailOutput struct {
	Value string `json:"value" jsonschema:"description=Email address"`
	Type  string `json:"type" jsonschema:"description=Email type (work home etc)"`
}

// AddressOutput represents a postal address in contact details output.
type AddressOutput struct {
	Value string `json:"value" jsonschema:"description=Full address"`
	Type  string `json:"type" jsonschema:"description=Address type (home work etc)"`
}

// ShowOutput is the output schema for contacts_show tool.
type ShowOutput struct {
	ResourceName string          `json:"resourceName" jsonschema:"description=Google Contact ID"`
	FirstName    string          `json:"firstName" jsonschema:"description=First name"`
	LastName     string          `json:"lastName" jsonschema:"description=Last name"`
	DisplayName  string          `json:"displayName" jsonschema:"description=Full display name"`
	Phones       []PhoneOutput   `json:"phones" jsonschema:"description=All phone numbers with types"`
	Emails       []EmailOutput   `json:"emails" jsonschema:"description=All email addresses with types"`
	Addresses    []AddressOutput `json:"addresses" jsonschema:"description=All postal addresses with types"`
	Company      string          `json:"company,omitempty" jsonschema:"description=Company name"`
	Position     string          `json:"position,omitempty" jsonschema:"description=Job title"`
	Notes        string          `json:"notes,omitempty" jsonschema:"description=Notes about contact"`
	Birthday     string          `json:"birthday,omitempty" jsonschema:"description=Birthday (YYYY-MM-DD or --MM-DD)"`
	UpdatedAt    string          `json:"updatedAt,omitempty" jsonschema:"description=Last update timestamp"`
}

// UpdateInput is the input schema for contacts_update tool.
type UpdateInput struct {
	ContactID       string         `json:"contactId" jsonschema:"required,description=Contact ID to update"`
	FirstName       string         `json:"firstName,omitempty" jsonschema:"description=New first name"`
	LastName        string         `json:"lastName,omitempty" jsonschema:"description=New last name"`
	Phones          []PhoneInput   `json:"phones,omitempty" jsonschema:"description=Replace ALL phones with these"`
	AddPhones       []PhoneInput   `json:"addPhones,omitempty" jsonschema:"description=Add phones without removing existing"`
	RemovePhones    []string       `json:"removePhones,omitempty" jsonschema:"description=Remove phones by value"`
	Emails          []EmailInput   `json:"emails,omitempty" jsonschema:"description=Replace ALL emails with these"`
	AddEmails       []EmailInput   `json:"addEmails,omitempty" jsonschema:"description=Add emails without removing existing"`
	RemoveEmails    []string       `json:"removeEmails,omitempty" jsonschema:"description=Remove emails by value"`
	Addresses       []AddressInput `json:"addresses,omitempty" jsonschema:"description=Replace ALL addresses with these"`
	AddAddresses    []AddressInput `json:"addAddresses,omitempty" jsonschema:"description=Add addresses without removing existing"`
	RemoveAddresses []string       `json:"removeAddresses,omitempty" jsonschema:"description=Remove addresses by street content"`
	Company         string         `json:"company,omitempty" jsonschema:"description=New company name"`
	Position        string         `json:"position,omitempty" jsonschema:"description=New job title"`
	Notes           string         `json:"notes,omitempty" jsonschema:"description=New notes"`
	Birthday        string         `json:"birthday,omitempty" jsonschema:"description=New birthday (YYYY-MM-DD or --MM-DD)"`
	ClearBirthday   bool           `json:"clearBirthday,omitempty" jsonschema:"description=Set true to remove birthday"`
}

// UpdateOutput is the output schema for contacts_update tool.
type UpdateOutput struct {
	ShowOutput
	Message string `json:"message" jsonschema:"description=Success message"`
}

// DeleteInput is the input schema for contacts_delete tool.
type DeleteInput struct {
	ContactID string `json:"contactId" jsonschema:"required,description=Contact ID to delete"`
}

// DeleteOutput is the output schema for contacts_delete tool.
type DeleteOutput struct {
	Message     string `json:"message" jsonschema:"description=Success message"`
	DeletedID   string `json:"deletedId" jsonschema:"description=ID of deleted contact"`
	DisplayName string `json:"displayName,omitempty" jsonschema:"description=Name of deleted contact"`
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

	// Convert results
	output := SearchOutput{
		Count: len(results),
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
	// Initialize Firestore client if configured
	if err := s.initFirestore(ctx); err != nil {
		return err
	}
	defer func() {
		if s.firestoreClient != nil {
			s.firestoreClient.Close()
		}
	}()

	// Register tools
	s.RegisterTools()

	// Create the streamable HTTP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: false, // Enable session tracking
	})

	// Wrap with authentication middleware
	handler := s.authMiddleware(mcpHandler)

	// Log authentication mode
	if s.config.APIKey != "" {
		log.Println("Authentication mode: static API key")
	} else if s.config.FirestoreProject != "" {
		log.Println("Authentication mode: Firestore API keys")
	} else {
		log.Println("Authentication mode: disabled (no API key or Firestore project configured)")
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting MCP server on %s", addr)
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
