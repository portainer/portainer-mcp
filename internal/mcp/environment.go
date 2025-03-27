package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddEnvironmentFeatures() {
	s.addToolIfExists(ToolListEnvironments, s.handleGetEnvironments())
	s.addToolIfExists(ToolUpdateEnvironment, s.handleUpdateEnvironment())
}

func (s *PortainerMCPServer) handleGetEnvironments() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environments, err := s.cli.GetEnvironments()
		if err != nil {
			return nil, fmt.Errorf("failed to get environments: %w", err)
		}

		data, err := json.Marshal(environments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environments: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironment() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		tagIds, err := parser.GetArrayOfIntegers("tagIds", true)
		if err != nil {
			return nil, err
		}

		userAccesses, err := parser.GetArrayOfObjects("userAccesses", false)
		if err != nil {
			return nil, err
		}
		userAccessesMap, err := parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}

		teamAccesses, err := parser.GetArrayOfObjects("teamAccesses", false)
		if err != nil {
			return nil, err
		}
		teamAccessesMap, err := parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}

		err = s.cli.UpdateEnvironment(id, tagIds, userAccessesMap, teamAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("error updating environment. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment updated successfully"), nil
	}
}
