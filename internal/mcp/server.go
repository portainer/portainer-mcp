package mcp

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/portainer/client"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

const (
	// MinimumToolsVersion is the minimum supported version of the tools.yaml file
	MinimumToolsVersion = "1.0"
)

// PortainerClient defines the interface for the wrapper client used by the MCP server
type PortainerClient interface {
	// Tag methods
	GetEnvironmentTags() ([]models.EnvironmentTag, error)
	CreateEnvironmentTag(name string) (int, error)

	// Environment methods
	GetEnvironments() ([]models.Environment, error)
	UpdateEnvironmentTags(id int, tagIds []int) error
	UpdateEnvironmentUserAccesses(id int, userAccesses map[int]string) error
	UpdateEnvironmentTeamAccesses(id int, teamAccesses map[int]string) error

	// Environment Group methods
	GetEnvironmentGroups() ([]models.Group, error)
	CreateEnvironmentGroup(name string, environmentIds []int) (int, error)
	UpdateEnvironmentGroupName(id int, name string) error
	UpdateEnvironmentGroupEnvironments(id int, name string, environmentIds []int) error
	UpdateEnvironmentGroupTags(id int, name string, tagIds []int) error

	// Access Group methods
	GetAccessGroups() ([]models.AccessGroup, error)
	CreateAccessGroup(name string, environmentIds []int) (int, error)
	UpdateAccessGroupName(id int, name string) error
	UpdateAccessGroupUserAccesses(id int, userAccesses map[int]string) error
	UpdateAccessGroupTeamAccesses(id int, teamAccesses map[int]string) error
	AddEnvironmentToAccessGroup(id int, environmentId int) error
	RemoveEnvironmentFromAccessGroup(id int, environmentId int) error

	// Stack methods
	GetStacks() ([]models.Stack, error)
	GetStackFile(id int) (string, error)
	CreateStack(name string, file string, environmentGroupIds []int) (int, error)
	UpdateStack(id int, file string, environmentGroupIds []int) error

	// Team methods
	CreateTeam(name string) (int, error)
	GetTeams() ([]models.Team, error)
	UpdateTeamName(id int, name string) error
	UpdateTeamMembers(id int, userIds []int) error

	// User methods
	GetUsers() ([]models.User, error)
	UpdateUserRole(id int, role string) error

	// Settings methods
	GetSettings() (models.PortainerSettings, error)
}

type PortainerMCPServer struct {
	srv   *server.MCPServer
	cli   PortainerClient
	tools map[string]mcp.Tool
}

func NewPortainerMCPServer(serverURL, token, toolsPath string) (*PortainerMCPServer, error) {
	tools, err := toolgen.LoadToolsFromYAML(toolsPath, MinimumToolsVersion)
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
	} else {
		log.Printf("Tool %s not found, will not be registered for MCP usage", toolName)
	}
}
