package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddEnvironmentGroupFeatures() {
	s.addToolIfExists(ToolListEnvironmentGroups, s.HandleGetEnvironmentGroups())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateEnvironmentGroup, s.HandleCreateEnvironmentGroup())
		s.addToolIfExists(ToolUpdateEnvironmentGroupName, s.HandleUpdateEnvironmentGroupName())
		s.addToolIfExists(ToolUpdateEnvironmentGroupEnvironments, s.HandleUpdateEnvironmentGroupEnvironments())
		s.addToolIfExists(ToolUpdateEnvironmentGroupTags, s.HandleUpdateEnvironmentGroupTags())
	}
}

func (s *PortainerMCPServer) HandleGetEnvironmentGroups() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		edgeGroups, err := s.cli.GetEnvironmentGroups()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get environment groups", err), nil
		}

		data, err := json.Marshal(edgeGroups)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal environment groups", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleCreateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentIds parameter", err), nil
		}

		id, err := s.cli.CreateEnvironmentGroup(name, environmentIds)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create environment group", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment group created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupName() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		name, err := parser.GetString("name", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		err = s.cli.UpdateEnvironmentGroupName(id, name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update environment group name", err), nil
		}

		return mcp.NewToolResultText("Environment group name updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupEnvironments() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentIds parameter", err), nil
		}

		err = s.cli.UpdateEnvironmentGroupEnvironments(id, environmentIds)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update environment group environments", err), nil
		}

		return mcp.NewToolResultText("Environment group environments updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupTags() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		tagIds, err := parser.GetArrayOfIntegers("tagIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid tagIds parameter", err), nil
		}

		err = s.cli.UpdateEnvironmentGroupTags(id, tagIds)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update environment group tags", err), nil
		}

		return mcp.NewToolResultText("Environment group tags updated successfully"), nil
	}
}
