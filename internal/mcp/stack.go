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

	// It seems that ResourceTemplates are not loaded by Claude.ai

	// stackFileResource := mcp.NewResourceTemplate(
	// 	"portainer://stacks/{id}/file",
	// 	"Compose file associated with a Portainer stack",
	// 	mcp.WithTemplateDescription("Content of the stack file for a specific stack ID."),
	// 	mcp.WithTemplateMIMEType("application/json"),
	// )

	getStackFileTool := mcp.NewTool("getStackFile",
		mcp.WithDescription("Get the compose file for a specific stack ID"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the stack to get the compose file for"),
		),
	)

	// Docker opiniated for POC purposes
	createStackTool := mcp.NewTool("createStack",
		mcp.WithDescription("Create a new stack"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the stack."+
				"Stack name must only consist of lowercase alpha characters, numbers, hyphens, or underscores as well as start with a lowercase character or number"),
		),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Content of the stack file."+
				"The file must be a valid docker-compose.yml file."+
				"example: services:\n web:\n image:nginx"),
		),
		mcp.WithString("environmentGroupIds",
			mcp.Required(),
			mcp.Description("The IDs of the environment groups that the stack belongs to, separated by commas."+
				"Must include at least one environment group ID"),
		),
	)

	updateStackTool := mcp.NewTool("updateStack",
		mcp.WithDescription("Update an existing stack"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the stack to update"),
		),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Content of the stack file."+
				"The file must be a valid docker-compose.yml file."+
				"example: version: 3\n services:\n web:\n image:nginx"),
		),
		mcp.WithString("environmentGroupIds",
			mcp.Required(),
			mcp.Description("The IDs of the environment groups that the stack belongs to, separated by commas."+
				"Must include at least one environment group ID"),
		),
	)

	s.srv.AddResource(stacksResource, s.handleGetStacks())
	s.srv.AddTool(createStackTool, s.handleCreateStack())
	s.srv.AddTool(updateStackTool, s.handleUpdateStack())
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

		environmentGroupIdsStr, ok := request.Params.Arguments["environmentGroupIds"].(string)
		if !ok {
			return nil, fmt.Errorf("environment group IDs are required")
		}

		environmentGroupIds, err := ParseCommaSeparatedInts(environmentGroupIdsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid environment group IDs. Error: %w", err)
		}

		id, err := s.cli.CreateStack(name, file, environmentGroupIds)
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

		environmentGroupIdsStr, ok := request.Params.Arguments["environmentGroupIds"].(string)
		if !ok {
			return nil, fmt.Errorf("environment group IDs are required")
		}

		environmentGroupIds, err := ParseCommaSeparatedInts(environmentGroupIdsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid environment group IDs. Error: %w", err)
		}

		err = s.cli.UpdateStack(int(id), file, environmentGroupIds)
		if err != nil {
			return nil, fmt.Errorf("error updating stack. Error: %w", err)
		}

		return mcp.NewToolResultText("Stack updated successfully"), nil
	}
}
