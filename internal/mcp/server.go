package mcp

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/client"
	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type PortainerMCPServer struct {
	srv   *server.MCPServer
	cli   *client.PortainerClient
	tools map[string]mcp.Tool
}

func NewPortainerMCPServer(serverURL, token, toolsPath string) (*PortainerMCPServer, error) {
	tools, err := toolgen.LoadToolsFromYAML(toolsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tools: %w", err)
	}

	return &PortainerMCPServer{
		srv: server.NewMCPServer(
			"Portainer MCP Server",
			"0.1.0",
			server.WithResourceCapabilities(true, true),
			server.WithLogging(),
		),
		cli:   client.NewPortainerClient(serverURL, token, client.WithSkipTLSVerify(true)),
		tools: tools,
	}, nil
}

func (s *PortainerMCPServer) Start() error {
	return server.ServeStdio(s.srv)
}

// addToolIfExists adds a tool to the server if it exists in the tools map
func (s *PortainerMCPServer) addToolIfExists(toolName string, handler server.ToolHandlerFunc) {
	if tool, exists := s.tools[toolName]; exists {
		s.srv.AddTool(tool, handler)
	}
}
