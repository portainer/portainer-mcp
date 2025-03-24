package mcp

import (
	"context"
	"encoding/json"
	"fmt"

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
			mcp.Description("The IDs of the environments that are part of the group, separated by commas."+
				"Optional, provide this if you want to associate environments with the group based on their IDs."+
				"Specify either this parameter or the tagIds parameter, but not both."+
				"Must include all the environment IDs that are part of the group - this includes new environments and the existing environments that are already associated with the group."),
		),
		mcp.WithString("tagIds",
			mcp.Description("The IDs of the tags that are associated with the group, separated by commas."+
				"Optional, provide this if you want to associate environments with the group based on their tags."+
				"Specify either this parameter or the environmentIds parameter, but not both."+
				"Must include all the tag IDs that are associated with the group - this includes new tags and the existing tags that are already associated with the group."),
		),
	)

	s.srv.AddResource(environmentGroupsResource, s.handleGetEnvironmentGroups())
	s.srv.AddTool(createEnvironmentGroupTool, s.handleCreateEnvironmentGroup())
	s.srv.AddTool(updateEnvironmentGroupTool, s.handleUpdateEnvironmentGroup())
}

func (s *PortainerMCPServer) handleGetEnvironmentGroups() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		edgeGroups, err := s.cli.GetEnvironmentGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment groups: %w", err)
		}

		data, err := json.Marshal(edgeGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment groups: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://environment-groups",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleCreateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := request.Params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("environment group name is required")
		}

		environmentIdsStr, ok := request.Params.Arguments["environmentIds"].(string)
		if !ok {
			return nil, fmt.Errorf("environment IDs are required")
		}

		environmentIds, err := parseCommaSeparatedInts(environmentIdsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid environment IDs. Error: %w", err)
		}

		id, err := s.cli.CreateEnvironmentGroup(name, environmentIds)
		if err != nil {
			return nil, fmt.Errorf("error creating environment group. Error: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment group created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("environment group ID is required")
		}

		name, ok := request.Params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("environment group name is required")
		}

		environmentIdsStr := request.Params.Arguments["environmentIds"].(string)
		tagIdsStr := request.Params.Arguments["tagIds"].(string)

		environmentIds, err := parseCommaSeparatedInts(environmentIdsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid environment IDs. Error: %w", err)
		}

		tagIds := []int{}
		if tagIdsStr != "" {
			tagIds, err = parseCommaSeparatedInts(tagIdsStr)
			if err != nil {
				return nil, fmt.Errorf("invalid tag IDs. Error: %w", err)
			}
		}

		err = s.cli.UpdateEnvironmentGroup(int(id), name, environmentIds, tagIds)
		if err != nil {
			return nil, fmt.Errorf("error updating environment group. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment group updated successfully"), nil
	}
}
