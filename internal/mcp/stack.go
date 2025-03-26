package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddStackFeatures() {
	stacksResource := mcp.NewResource("portainer://stacks",
		"Portainer Stacks",
		mcp.WithResourceDescription("Lists all available stacks"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(stacksResource, s.handleGetStacks())

	createStackTool := s.tools[ToolCreateStack]
	s.srv.AddTool(createStackTool, s.handleCreateStack())

	updateStackTool := s.tools[ToolUpdateStack]
	s.srv.AddTool(updateStackTool, s.handleUpdateStack())

	getStackFileTool := s.tools[ToolGetStackFile]
	s.srv.AddTool(getStackFileTool, s.handleGetStackFile())
}

func (s *PortainerMCPServer) handleGetStacks() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		stacks, err := s.cli.GetStacks()
		if err != nil {
			return nil, fmt.Errorf("failed to get stacks: %w", err)
		}

		data, err := json.Marshal(stacks)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal stacks: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://stacks",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleGetStackFile() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("stack ID is required")
		}

		content, err := s.cli.GetStackFile(int(id))
		if err != nil {
			return nil, fmt.Errorf("failed to get stack file. Error: %w", err)
		}

		return mcp.NewToolResultText(content), nil
	}
}

func (s *PortainerMCPServer) handleCreateStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := request.Params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("stack name is required")
		}

		file, ok := request.Params.Arguments["file"].(string)
		if !ok {
			return nil, fmt.Errorf("stack file is required")
		}

		environmentGroupIds, ok := request.Params.Arguments["environmentGroupIds"].([]any)
		if !ok {
			return nil, fmt.Errorf("environment group IDs are required")
		}

		environmentGroupIdsInt, err := parseNumericArray(environmentGroupIds)
		if err != nil {
			return nil, fmt.Errorf("invalid environment group IDs. Error: %w", err)
		}

		id, err := s.cli.CreateStack(name, file, environmentGroupIdsInt)
		if err != nil {
			return nil, fmt.Errorf("error creating stack. Error: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Stack created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("stack ID is required")
		}

		file, ok := request.Params.Arguments["file"].(string)
		if !ok {
			return nil, fmt.Errorf("stack file is required")
		}

		environmentGroupIds, ok := request.Params.Arguments["environmentGroupIds"].([]any)
		if !ok {
			return nil, fmt.Errorf("environment group IDs are required")
		}

		environmentGroupIdsInt, err := parseNumericArray(environmentGroupIds)
		if err != nil {
			return nil, fmt.Errorf("invalid environment group IDs. Error: %w", err)
		}

		err = s.cli.UpdateStack(int(id), file, environmentGroupIdsInt)
		if err != nil {
			return nil, fmt.Errorf("error updating stack. Error: %w", err)
		}

		return mcp.NewToolResultText("Stack updated successfully"), nil
	}
}
