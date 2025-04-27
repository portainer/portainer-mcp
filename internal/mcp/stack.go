package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddStackFeatures() {
	s.addToolIfExists(ToolListStacks, s.HandleGetStacks())
	s.addToolIfExists(ToolGetStackFile, s.HandleGetStackFile())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateStack, s.HandleCreateStack())
		s.addToolIfExists(ToolUpdateStack, s.HandleUpdateStack())
	}
}

func (s *PortainerMCPServer) HandleGetStacks() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stacks, err := s.cli.GetStacks()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get stacks", err), nil
		}

		data, err := json.Marshal(stacks)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal stacks", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleGetStackFile() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		stackFile, err := s.cli.GetStackFile(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get stack file", err), nil
		}

		return mcp.NewToolResultText(stackFile), nil
	}
}

func (s *PortainerMCPServer) HandleCreateStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		file, err := parser.GetString("file", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid file parameter", err), nil
		}

		environmentGroupIds, err := parser.GetArrayOfIntegers("environmentGroupIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentGroupIds parameter", err), nil
		}

		id, err := s.cli.CreateStack(name, file, environmentGroupIds)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("error creating stack", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Stack created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		file, err := parser.GetString("file", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid file parameter", err), nil
		}

		environmentGroupIds, err := parser.GetArrayOfIntegers("environmentGroupIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentGroupIds parameter", err), nil
		}

		err = s.cli.UpdateStack(id, file, environmentGroupIds)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update stack", err), nil
		}

		return mcp.NewToolResultText("Stack updated successfully"), nil
	}
}
