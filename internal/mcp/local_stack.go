package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

// AddLocalStackFeatures registers the local (non-edge) stack tools with the MCP server.
func (s *PortainerMCPServer) AddLocalStackFeatures() {
	s.addToolIfExists(ToolListLocalStacks, s.HandleGetLocalStacks())
	s.addToolIfExists(ToolGetLocalStackFile, s.HandleGetLocalStackFile())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateLocalStack, s.HandleCreateLocalStack())
		s.addToolIfExists(ToolUpdateLocalStack, s.HandleUpdateLocalStack())
		s.addToolIfExists(ToolStartLocalStack, s.HandleStartLocalStack())
		s.addToolIfExists(ToolStopLocalStack, s.HandleStopLocalStack())
		s.addToolIfExists(ToolDeleteLocalStack, s.HandleDeleteLocalStack())
	}
}

func (s *PortainerMCPServer) HandleGetLocalStacks() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stacks, err := s.cli.GetLocalStacks()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get local stacks", err), nil
		}

		data, err := json.Marshal(stacks)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal local stacks", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleGetLocalStackFile() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		stackFile, err := s.cli.GetLocalStackFile(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get local stack file", err), nil
		}

		return mcp.NewToolResultText(stackFile), nil
	}
}

// parseEnvVars extracts environment variables from the request parameter
func parseEnvVars(parser *toolgen.ParameterParser) ([]models.LocalStackEnvVar, error) {
	rawEnv, err := parser.GetArrayOfObjects("env", false)
	if err != nil {
		return nil, err
	}

	if rawEnv == nil {
		return []models.LocalStackEnvVar{}, nil
	}

	env := make([]models.LocalStackEnvVar, 0, len(rawEnv))
	for _, item := range rawEnv {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid env variable format: expected object")
		}

		name, _ := obj["name"].(string)
		value, _ := obj["value"].(string)

		if name == "" {
			return nil, fmt.Errorf("env variable 'name' is required")
		}

		env = append(env, models.LocalStackEnvVar{Name: name, Value: value})
	}

	return env, nil
}

func (s *PortainerMCPServer) HandleCreateLocalStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		endpointId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		name, err := parser.GetString("name", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		file, err := parser.GetString("file", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid file parameter", err), nil
		}

		env, err := parseEnvVars(parser)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid env parameter", err), nil
		}

		id, err := s.cli.CreateLocalStack(endpointId, name, file, env)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("error creating local stack", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Local stack created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateLocalStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		endpointId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		file, err := parser.GetString("file", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid file parameter", err), nil
		}

		env, err := parseEnvVars(parser)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid env parameter", err), nil
		}

		prune, err := parser.GetBoolean("prune", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid prune parameter", err), nil
		}

		pullImage, err := parser.GetBoolean("pullImage", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid pullImage parameter", err), nil
		}

		err = s.cli.UpdateLocalStack(id, endpointId, file, env, prune, pullImage)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update local stack", err), nil
		}

		return mcp.NewToolResultText("Local stack updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleStartLocalStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		endpointId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		err = s.cli.StartLocalStack(id, endpointId)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to start local stack", err), nil
		}

		return mcp.NewToolResultText("Local stack started successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleStopLocalStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		endpointId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		err = s.cli.StopLocalStack(id, endpointId)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to stop local stack", err), nil
		}

		return mcp.NewToolResultText("Local stack stopped successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleDeleteLocalStack() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		endpointId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		err = s.cli.DeleteLocalStack(id, endpointId)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to delete local stack", err), nil
		}

		return mcp.NewToolResultText("Local stack deleted successfully"), nil
	}
}
