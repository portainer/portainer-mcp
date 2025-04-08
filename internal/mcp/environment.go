package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddEnvironmentFeatures() {
	s.addToolIfExists(ToolListEnvironments, s.HandleGetEnvironments())

	if !s.readOnly {
		s.addToolIfExists(ToolUpdateEnvironmentTags, s.HandleUpdateEnvironmentTags())
		s.addToolIfExists(ToolUpdateEnvironmentUserAccesses, s.HandleUpdateEnvironmentUserAccesses())
		s.addToolIfExists(ToolUpdateEnvironmentTeamAccesses, s.HandleUpdateEnvironmentTeamAccesses())
	}
}

func (s *PortainerMCPServer) HandleGetEnvironments() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateEnvironmentTags() server.ToolHandlerFunc {
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

		err = s.cli.UpdateEnvironmentTags(id, tagIds)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment tags: %w", err)
		}

		return mcp.NewToolResultText("Environment tags updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateEnvironmentUserAccesses() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		userAccesses, err := parser.GetArrayOfObjects("userAccesses", true)
		if err != nil {
			return nil, err
		}

		userAccessesMap, err := parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}

		err = s.cli.UpdateEnvironmentUserAccesses(id, userAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment user accesses: %w", err)
		}

		return mcp.NewToolResultText("Environment user accesses updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateEnvironmentTeamAccesses() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		teamAccesses, err := parser.GetArrayOfObjects("teamAccesses", true)
		if err != nil {
			return nil, err
		}

		teamAccessesMap, err := parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}

		err = s.cli.UpdateEnvironmentTeamAccesses(id, teamAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment team accesses: %w", err)
		}

		return mcp.NewToolResultText("Environment team accesses updated successfully"), nil
	}
}
