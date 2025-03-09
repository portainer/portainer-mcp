package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddEnvironmentGroupFeatures() {
	environmentGroupsResource := mcp.NewResource("portainer://environment-groups",
		"Portainer Environment Groups",
		mcp.WithResourceDescription("Lists all available environment groups"),
		mcp.WithMIMEType("application/json"),
	)

	createEnvironmentGroupTool := mcp.NewTool("createEnvironmentGroup",
		mcp.WithDescription("Create a new environment group"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the environment group"),
		),
		mcp.WithString("environmentIds",
			mcp.Required(),
			mcp.Description("The IDs of the environments to add to the group, separated by commas"),
		),
	)

	updateEnvironmentGroupTool := mcp.NewTool("updateEnvironmentGroup",
		mcp.WithDescription("Update an existing environment group"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the environment group to update"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the environment group, re-use the existing name to keep the same group name"),
		),
		mcp.WithString("environmentIds",
			mcp.Required(),
			mcp.Description("The IDs of the environments that are part of the group, separated by commas."+
				"Must include all the environment IDs that are part of the group"),
		),
	)

	s.srv.AddResource(environmentGroupsResource, s.handleGetEnvironmentGroups())
	s.srv.AddTool(createEnvironmentGroupTool, s.handleCreateEnvironmentGroup())
	s.srv.AddTool(updateEnvironmentGroupTool, s.handleUpdateEnvironmentGroup())
}

func (s *PortainerMCPServer) handleGetEnvironmentGroups() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		edgeGroups, err := s.cli.GetEnvironmentGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment groups: %w", err)
		}

		data, err := json.Marshal(edgeGroups)
		if err != nil {
			return nil, err
		}

		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      "portainer://environment-groups",
					MIMEType: "application/json",
				},
				Text: string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleCreateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.Params.Arguments["name"].(string)
		environmentIdsStr := request.Params.Arguments["environmentIds"].(string)
		environmentIds := []int{}

		for _, idStr := range strings.Split(environmentIdsStr, ",") {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid environment ID: %v", err)), nil
			}
			environmentIds = append(environmentIds, id)
		}

		id, err := s.cli.CreateEnvironmentGroup(name, environmentIds)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error creating environment group: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment group created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := request.Params.Arguments["id"].(float64)
		name := request.Params.Arguments["name"].(string)
		environmentIdsStr := request.Params.Arguments["environmentIds"].(string)
		environmentIds := []int{}

		for _, idStr := range strings.Split(environmentIdsStr, ",") {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid environment ID: %v", err)), nil
			}
			environmentIds = append(environmentIds, id)
		}

		err := s.cli.UpdateEnvironmentGroup(int(id), name, environmentIds)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error updating environment group: %v", err)), nil
		}

		return mcp.NewToolResultText("Environment group updated successfully"), nil
	}
}
