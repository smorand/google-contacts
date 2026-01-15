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
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	config     *Config
	mcpServer  *mcp.Server
	httpServer *http.Server
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

// RegisterTools registers all contact management tools with the MCP server.
// Tools will be implemented in US-00029.
func (s *Server) RegisterTools() {
	// Placeholder - tools will be added in US-00029
	// For now, just register a simple ping tool for testing
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
}

// Run starts the HTTP server and blocks until shutdown.
func (s *Server) Run(ctx context.Context) error {
	// Register tools
	s.RegisterTools()

	// Create the streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless: false, // Enable session tracking
	})

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
