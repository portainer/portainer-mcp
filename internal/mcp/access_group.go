package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddAccessGroupFeatures() {
	accessGroupResource := mcp.NewResource("portainer://access-groups",
		"Portainer Access Groups",
		mcp.WithResourceDescription("Lists all available access groups"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(accessGroupResource, s.handleGetAccessGroups())

	createAccessGroupTool := s.tools[ToolCreateAccessGroup]
	s.srv.AddTool(createAccessGroupTool, s.handleCreateAccessGroup())

	updateAccessGroupTool := s.tools[ToolUpdateAccessGroup]
	s.srv.AddTool(updateAccessGroupTool, s.handleUpdateAccessGroup())

	addEnvironmentToAccessGroupTool := s.tools[ToolAddEnvironmentToAccessGroup]
	s.srv.AddTool(addEnvironmentToAccessGroupTool, s.handleAddEnvironmentToAccessGroup())

	removeEnvironmentFromAccessGroupTool := s.tools[ToolRemoveEnvironmentFromAccessGroup]
	s.srv.AddTool(removeEnvironmentFromAccessGroupTool, s.handleRemoveEnvironmentFromAccessGroup())
}

func (s *PortainerMCPServer) handleGetAccessGroups() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		accessGroups, err := s.cli.GetAccessGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get access groups: %w", err)
		}

		data, err := json.Marshal(accessGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal access groups: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://access-groups",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

// accessGroupParams represents the parameters needed to create an access group
type accessGroupParams struct {
	Name           string
	EnvironmentIds []int
	UserAccesses   map[int]string
	TeamAccesses   map[int]string
}

// parseAccessMap parses access entries from the request parameters and returns a map of ID to access level
func parseAccessMap(entries []any) (map[int]string, error) {
	accessMap := map[int]string{}

	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid access entry: %v", entry)
		}

		id, ok := entryMap["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid ID: %v", entryMap["id"])
		}

		access, ok := entryMap["access"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid access: %v", entryMap["access"])
		}

		accessMap[int(id)] = access
	}

	return accessMap, nil
}

// parseCreateAccessGroupParams parses and validates the request parameters
func parseCreateAccessGroupParams(request mcp.CallToolRequest) (*accessGroupParams, error) {
	// Parse required name parameter
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("access group name is required")
	}

	// Parse optional environment IDs
	var environmentIds []int
	if envIds, ok := request.Params.Arguments["environmentIds"].([]any); ok {
		var err error
		environmentIds, err = parseNumericArray(envIds)
		if err != nil {
			return nil, fmt.Errorf("invalid environment IDs: %w", err)
		}
	}

	// Parse optional user accesses
	userAccessesMap := map[int]string{}
	if userAccesses, ok := request.Params.Arguments["userAccesses"].([]any); ok {
		var err error
		userAccessesMap, err = parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}
	}

	// Parse optional team accesses
	teamAccessesMap := map[int]string{}
	if teamAccesses, ok := request.Params.Arguments["teamAccesses"].([]any); ok {
		var err error
		teamAccessesMap, err = parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}
	}

	return &accessGroupParams{
		Name:           name,
		EnvironmentIds: environmentIds,
		UserAccesses:   userAccessesMap,
		TeamAccesses:   teamAccessesMap,
	}, nil
}

func (s *PortainerMCPServer) handleCreateAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse and validate parameters
		params, err := parseCreateAccessGroupParams(request)
		if err != nil {
			return nil, err
		}

		// Create access group
		accessGroup := models.AccessGroup{
			Name:           params.Name,
			EnvironmentIds: params.EnvironmentIds,
			UserAccesses:   params.UserAccesses,
			TeamAccesses:   params.TeamAccesses,
		}

		groupID, err := s.cli.CreateAccessGroup(accessGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to create access group: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Access group created successfully with ID: %d", groupID)), nil
	}
}

type updateAccessGroupParams struct {
	ID           int
	Name         string
	UserAccesses map[int]string
	TeamAccesses map[int]string
}

func parseUpdateAccessGroupParams(request mcp.CallToolRequest) (*updateAccessGroupParams, error) {
	// Parse required ID parameter
	id, ok := request.Params.Arguments["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("access group ID is required")
	}

	// Parse optional name parameter
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		name = ""
	}

	// Parse optional user accesses
	userAccessesMap := map[int]string{}
	if userAccesses, ok := request.Params.Arguments["userAccesses"].([]any); ok {
		var err error
		userAccessesMap, err = parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}
	}

	// Parse optional team accesses
	teamAccessesMap := map[int]string{}
	if teamAccesses, ok := request.Params.Arguments["teamAccesses"].([]any); ok {
		var err error
		teamAccessesMap, err = parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}
	}

	return &updateAccessGroupParams{
		ID:           int(id),
		Name:         name,
		UserAccesses: userAccessesMap,
		TeamAccesses: teamAccessesMap,
	}, nil
}

func (s *PortainerMCPServer) handleUpdateAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("access group ID is required")
		}

		params, err := parseUpdateAccessGroupParams(request)
		if err != nil {
			return nil, err
		}

		accessGroup := models.AccessGroup{
			ID:           int(id),
			Name:         params.Name,
			UserAccesses: params.UserAccesses,
			TeamAccesses: params.TeamAccesses,
		}

		err = s.cli.UpdateAccessGroup(accessGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to update access group: %w", err)
		}

		return mcp.NewToolResultText("Access group updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleAddEnvironmentToAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("access group ID is required")
		}

		environmentId, ok := request.Params.Arguments["environmentId"].(float64)
		if !ok {
			return nil, fmt.Errorf("environment ID is required")
		}

		err := s.cli.AddEnvironmentToAccessGroup(int(id), int(environmentId))
		if err != nil {
			return nil, fmt.Errorf("failed to add environment to access group: %w", err)
		}

		return mcp.NewToolResultText("Environment added to access group successfully"), nil
	}
}

func (s *PortainerMCPServer) handleRemoveEnvironmentFromAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("access group ID is required")
		}

		environmentId, ok := request.Params.Arguments["environmentId"].(float64)
		if !ok {
			return nil, fmt.Errorf("environment ID is required")
		}

		err := s.cli.RemoveEnvironmentFromAccessGroup(int(id), int(environmentId))
		if err != nil {
			return nil, fmt.Errorf("failed to remove environment from access group: %w", err)
		}

		return mcp.NewToolResultText("Environment removed from access group successfully"), nil
	}
}
