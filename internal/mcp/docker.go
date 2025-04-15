package mcp

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddDockerProxyFeatures() {
	if !s.readOnly {
		s.addToolIfExists(ToolDockerProxy, s.HandleDockerProxy())
	}
}

func (s *PortainerMCPServer) HandleDockerProxy() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		environmentId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return nil, err
		}

		method, err := parser.GetString("method", true)
		if err != nil {
			return nil, err
		}
		if !isValidHTTPMethod(method) {
			return nil, fmt.Errorf("invalid method: %s", method)
		}

		dockerAPIPath, err := parser.GetString("dockerAPIPath", true)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(dockerAPIPath, "/") {
			return nil, fmt.Errorf("dockerAPIPath must start with a leading slash")
		}

		queryParams, err := parser.GetArrayOfObjects("queryParams", false)
		if err != nil {
			return nil, err
		}
		queryParamsMap, err := parseKeyValueMap(queryParams)
		if err != nil {
			return nil, fmt.Errorf("invalid query params: %w", err)
		}

		headers, err := parser.GetArrayOfObjects("headers", false)
		if err != nil {
			return nil, err
		}
		headersMap, err := parseKeyValueMap(headers)
		if err != nil {
			return nil, fmt.Errorf("invalid headers: %w", err)
		}

		body, err := parser.GetString("body", false)
		if err != nil {
			return nil, err
		}

		opts := models.DockerProxyRequestOptions{
			EnvironmentID: environmentId,
			Path:          dockerAPIPath,
			Method:        method,
			QueryParams:   queryParamsMap,
			Headers:       headersMap,
		}

		if body != "" {
			opts.Body = strings.NewReader(body)
		}

		response, err := s.cli.ProxyDockerRequest(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to send Docker API request: %w", err)
		}

		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read Docker API response: %w", err)
		}

		return mcp.NewToolResultText(string(responseBody)), nil
	}
}
