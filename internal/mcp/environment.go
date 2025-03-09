package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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
			mcp.Description("The IDs of the tags to add to the environment, separated by commas"),
		),
	)

	s.srv.AddResource(environmentsResource, s.handleGetEnvironments())
	s.srv.AddTool(updateEnvironmentTool, s.handleUpdateEnvironment())
}

func (s *PortainerMCPServer) handleGetEnvironments() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		environments, err := s.cli.GetEnvironments()
		if err != nil {
			return nil, fmt.Errorf("failed to get environments: %w", err)
		}

		data, err := json.Marshal(environments)
		if err != nil {
			return nil, err
		}

		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      "portainer://environments",
					MIMEType: "application/json",
				},
				Text: string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironment() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := request.Params.Arguments["id"].(float64)
		tagIds := request.Params.Arguments["tagIds"].(string)

		tagIdsInt := []int{}
		for _, tagId := range strings.Split(tagIds, ",") {
			tagIdInt, err := strconv.Atoi(tagId)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid tag ID: %v\n", err)
				return mcp.NewToolResultError(fmt.Sprintf("invalid tag ID: %v", err)), nil
			}
			tagIdsInt = append(tagIdsInt, tagIdInt)
		}

		err := s.cli.UpdateEnvironment(int(id), tagIdsInt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error updating environment: %v\n", err)
			return mcp.NewToolResultError(fmt.Sprintf("error updating environment: %v", err)), nil
		}

		return mcp.NewToolResultText("Environment updated successfully"), nil
	}
}
