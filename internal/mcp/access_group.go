package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddAccessGroupFeatures() {
	accessGroupResource := mcp.NewResource("portainer://access-groups",
		"Portainer Access Groups",
		mcp.WithResourceDescription("Lists all available access groups"),
		mcp.WithMIMEType("application/json"),
	)

	// createAccessGroupTool := mcp.NewTool("createAccessGroup",
	// 	mcp.WithDescription("Create a new access group."+
	// 		"Use this tool when you want to define accesses on more than one environment."+
	// 		"Otherwise, define the accesses on the environment level."),
	// 	mcp.WithString("name",
	// 		mcp.Required(),
	// 		mcp.Description("The name of the access group"),
	// 	),
	// 	mcp.WithString("environmentIds",
	// 		mcp.Description("The IDs of the environments that are part of the access group, separated by commas."+
	// 			"Must include all the environment IDs that are part of the group - this includes new environments and the existing environments that are already associated with the group."),
	// 	),
	// )

	// updateAccessGroupTool := mcp.NewTool("updateAccessGroup",
	// 	mcp.WithDescription("Update an existing access group."),
	// 	mcp.WithString("id",
	// 		mcp.Required(),
	// 		mcp.Description("The ID of the access group to update"),
	// 	),
	// 	mcp.WithString("environmentIds",
	// 		mcp.Description("The IDs of the environments that are part of the access group, separated by commas."+
	// 			"Must include all the environment IDs that are part of the group - this includes new environments and the existing environments that are already associated with the group."),
	// 	),
	// )

	s.srv.AddResource(accessGroupResource, s.handleGetAccessGroups())
	// s.srv.AddTool(createAccessGroupTool, s.handleCreateAccessGroup())
	// s.srv.AddTool(updateAccessGroupTool, s.handleUpdateAccessGroup())
}

func (s *PortainerMCPServer) handleGetAccessGroups() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		accessGroups, err := s.cli.GetAccessGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get access groups: %w", err)
		}

		data, err := json.Marshal(accessGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal access groups: %w", err)
		}

		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      "portainer://access-groups",
					MIMEType: "application/json",
				},
				Text: string(data),
			},
		}, nil
	}
}
