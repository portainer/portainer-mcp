package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddEnvironmentFeatures() {
	environmentsResource := mcp.NewResource("portainer://environments",
		"Portainer Environments",
		mcp.WithResourceDescription("Lists all available environments"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(environmentsResource, s.handleGetEnvironments())

	updateEnvironmentTool := s.tools[ToolUpdateEnvironment]
	s.srv.AddTool(updateEnvironmentTool, s.handleUpdateEnvironment())
}

func (s *PortainerMCPServer) handleGetEnvironments() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		environments, err := s.cli.GetEnvironments()
		if err != nil {
			return nil, fmt.Errorf("failed to get environments: %w", err)
		}

		data, err := json.Marshal(environments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environments: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://environments",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironment() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("environment ID is required")
		}

		tagIds, ok := request.Params.Arguments["tagIds"].([]any)
		if !ok {
			return nil, fmt.Errorf("tag IDs are required")
		}

		tagIdsInt, err := parseNumericArray(tagIds)
		if err != nil {
			return nil, fmt.Errorf("invalid tag IDs. Error: %w", err)
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

		err = s.cli.UpdateEnvironment(int(id), tagIdsInt, userAccessesMap, teamAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("error updating environment. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment updated successfully"), nil
	}
}
