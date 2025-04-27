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

func (s *PortainerMCPServer) AddKubernetesProxyFeatures() {
	if !s.readOnly {
		s.addToolIfExists(ToolKubernetesProxy, s.HandleKubernetesProxy())
	}
}

func (s *PortainerMCPServer) HandleKubernetesProxy() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		environmentId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid environmentId parameter", err), nil
		}

		method, err := parser.GetString("method", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid method parameter", err), nil
		}
		if !isValidHTTPMethod(method) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid method: %s", method)), nil
		}

		kubernetesAPIPath, err := parser.GetString("kubernetesAPIPath", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid kubernetesAPIPath parameter", err), nil
		}
		if !strings.HasPrefix(kubernetesAPIPath, "/") {
			return mcp.NewToolResultError("kubernetesAPIPath must start with a leading slash"), nil
		}

		queryParams, err := parser.GetArrayOfObjects("queryParams", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid queryParams parameter", err), nil
		}
		queryParamsMap, err := parseKeyValueMap(queryParams)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid query params", err), nil
		}

		headers, err := parser.GetArrayOfObjects("headers", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid headers parameter", err), nil
		}
		headersMap, err := parseKeyValueMap(headers)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid headers", err), nil
		}

		body, err := parser.GetString("body", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid body parameter", err), nil
		}

		opts := models.KubernetesProxyRequestOptions{
			EnvironmentID: environmentId,
			Path:          kubernetesAPIPath,
			Method:        method,
			QueryParams:   queryParamsMap,
			Headers:       headersMap,
		}

		if body != "" {
			opts.Body = strings.NewReader(body)
		}

		response, err := s.cli.ProxyKubernetesRequest(opts)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to send Kubernetes API request", err), nil
		}

		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read Kubernetes API response", err), nil
		}

		return mcp.NewToolResultText(string(responseBody)), nil
	}
}
