package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deviantony/portainer-mcp/pkg/portainer/client"
	"github.com/mark3labs/mcp-go/server"
)

// ServerOption represents a functional option for configuring the server
type ServerOption func(*PortainerMCPServer)

// WithLogger configures the server with a custom logger
func WithLogger(logger *log.Logger) ServerOption {
	return func(s *PortainerMCPServer) {
		s.logger = logger
	}
}

// WithSkipTLSVerify configures the client to skip TLS verification
func WithSkipTLSVerify(skip bool) ServerOption {
	return func(s *PortainerMCPServer) {
		s.skipTLSVerify = skip
	}
}

// PortainerMCPServer is the main server structure
type PortainerMCPServer struct {
	srv           *server.MCPServer
	cli           *client.PortainerClient
	logger        *log.Logger
	serverURL     string
	token         string
	skipTLSVerify bool
	resourceMap   map[string]*ResourceHandler
	toolMap       map[string]*ToolHandler
}

// NewPortainerMCPServer creates a new Portainer MCP server
func NewPortainerMCPServer(serverURL, token string, options ...ServerOption) *PortainerMCPServer {
	s := &PortainerMCPServer{
		srv: server.NewMCPServer(
			"Portainer MCP Server",
			"0.1.0",
			server.WithResourceCapabilities(true, true),
			server.WithLogging(),
		),
		serverURL:     serverURL,
		token:         token,
		logger:        log.New(os.Stderr, "[PORTAINER-MCP] ", log.LstdFlags),
		skipTLSVerify: true,
		resourceMap:   make(map[string]*ResourceHandler),
		toolMap:       make(map[string]*ToolHandler),
	}

	// Apply options
	for _, option := range options {
		option(s)
	}

	// Initialize client
	s.cli = client.NewPortainerClient(
		serverURL,
		token,
		client.WithSkipTLSVerify(s.skipTLSVerify),
	)

	return s
}

// RegisterResourceHandler registers a resource handler
func (s *PortainerMCPServer) RegisterResourceHandler(handler *ResourceHandler) {
	s.resourceMap[handler.URI] = handler
	s.srv.AddResource(handler.GetResource(), handler.GetHandlerFunc())
	s.logger.Printf("Registered resource handler: %s (%s)", handler.Name, handler.URI)
}

// RegisterToolHandler registers a tool handler
func (s *PortainerMCPServer) RegisterToolHandler(handler *ToolHandler) {
	s.toolMap[handler.Name] = handler
	s.srv.AddTool(handler.GetTool(), handler.GetHandlerFunc())
	s.logger.Printf("Registered tool handler: %s", handler.Name)
}

// AddFeatures adds all feature handlers to the server
func (s *PortainerMCPServer) AddFeatures() {
	// Register resource handlers
	envHandler := CreateEnvironmentsResourceHandler(s)
	s.RegisterResourceHandler(envHandler)

	userHandler := CreateUsersResourceHandler(s)
	s.RegisterResourceHandler(userHandler)

	teamHandler := CreateTeamsResourceHandler(s)
	s.RegisterResourceHandler(teamHandler)

	accessGroupHandler := CreateAccessGroupsResourceHandler(s)
	s.RegisterResourceHandler(accessGroupHandler)

	tagHandler := CreateTagsResourceHandler(s)
	s.RegisterResourceHandler(tagHandler)

	stackHandler := CreateStacksResourceHandler(s)
	s.RegisterResourceHandler(stackHandler)
	
	settingsHandler := CreateSettingsResourceHandler(s)
	s.RegisterResourceHandler(settingsHandler)

	// Register tool handlers
	updateEnvHandler := CreateUpdateEnvironmentToolHandler(s)
	s.RegisterToolHandler(updateEnvHandler)

	updateUserHandler := CreateUpdateUserToolHandler(s)
	s.RegisterToolHandler(updateUserHandler)

	updateTeamHandler := CreateUpdateTeamToolHandler(s)
	s.RegisterToolHandler(updateTeamHandler)

	createAccessGroupHandler := CreateCreateAccessGroupToolHandler(s)
	s.RegisterToolHandler(createAccessGroupHandler)

	updateAccessGroupHandler := CreateUpdateAccessGroupToolHandler(s)
	s.RegisterToolHandler(updateAccessGroupHandler)

	addEnvToAccessGroupHandler := CreateAddEnvironmentToAccessGroupToolHandler(s)
	s.RegisterToolHandler(addEnvToAccessGroupHandler)

	removeEnvFromAccessGroupHandler := CreateRemoveEnvironmentFromAccessGroupToolHandler(s)
	s.RegisterToolHandler(removeEnvFromAccessGroupHandler)
}

// Start starts the server with graceful shutdown
func (s *PortainerMCPServer) Start() error {
	// Set up graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s.logger.Printf("Starting Portainer MCP server with server URL: %s", s.serverURL)

	// Start the server
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ServeStdio(s.srv)
	}()

	// Wait for signals
	select {
	case <-ctx.Done():
		s.logger.Println("Shutdown signal received, waiting for in-flight requests to complete...")
		// Give time for graceful shutdown
		time.Sleep(5 * time.Second)
		s.logger.Println("Server stopped gracefully")
		return nil
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}

// Debug logs a debug message if debugging is enabled
func (s *PortainerMCPServer) Debug(format string, v ...interface{}) {
	s.logger.Printf("[DEBUG] "+format, v...)
}

// Info logs an info message
func (s *PortainerMCPServer) Info(format string, v ...interface{}) {
	s.logger.Printf("[INFO] "+format, v...)
}

// Error logs an error message
func (s *PortainerMCPServer) Error(format string, v ...interface{}) {
	s.logger.Printf("[ERROR] "+format, v...)
}

// Client returns the Portainer client
func (s *PortainerMCPServer) Client() *client.PortainerClient {
	return s.cli
}