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

	updateEnvironmentTool := mcp.NewTool("updateEnvironment",
		mcp.WithDescription("Update an existing environment"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the environment to update"),
		),
		mcp.WithString("tagIds",
			mcp.Required(),
			mcp.Description("The IDs of the tags that are associated with the environment, separated by commas."+
				"Must include all the tag IDs that are associated with the environment - this includes new tags and the existing tags that are already associated with the environment."),
		),
	)

	s.srv.AddResource(environmentsResource, s.handleGetEnvironments())
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

		tagIds, ok := request.Params.Arguments["tagIds"].(string)
		if !ok {
			return nil, fmt.Errorf("tag IDs are required")
		}

		tagIdsInt, err := ParseCommaSeparatedInts(tagIds)
		if err != nil {
			return nil, fmt.Errorf("invalid tag IDs. Error: %w", err)
		}

		err = s.cli.UpdateEnvironment(int(id), tagIdsInt)
		if err != nil {
			return nil, fmt.Errorf("error updating environment. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment updated successfully"), nil
	}
}
