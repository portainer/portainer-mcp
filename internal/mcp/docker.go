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
	// The Docker proxy tool is registered in both read-only and read-write modes
	// because it supports GET requests which are useful in read-only mode.
	// Write operations (non-GET methods) are enforced at runtime inside HandleDockerProxy.
	s.addToolIfExists(ToolDockerProxy, s.HandleDockerProxy())
}

func (s *PortainerMCPServer) HandleDockerProxy() server.ToolHandlerFunc {
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

		if s.readOnly && method != "GET" {
			return mcp.NewToolResultError("only GET requests are allowed in read-only mode"), nil
		}

		dockerAPIPath, err := parser.GetString("dockerAPIPath", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid dockerAPIPath parameter", err), nil
		}
		if !strings.HasPrefix(dockerAPIPath, "/") {
			return mcp.NewToolResultError("dockerAPIPath must start with a leading slash"), nil
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

		applyDockerDefaultFilters(dockerAPIPath, queryParamsMap)

		if isDangerousDockerPath(dockerAPIPath) {
			return mcp.NewToolResultError("access to this Docker API endpoint is not permitted"), nil
		}
		if isPrivilegedDockerContainerCreate(dockerAPIPath, body) {
			return mcp.NewToolResultError("creating privileged containers or containers with sensitive host mounts is not permitted"), nil
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
			return mcp.NewToolResultErrorFromErr("failed to send Docker API request", err), nil
		}
		defer response.Body.Close()

		responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxProxyResponseBytes))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read Docker API response", err), nil
		}
		if int64(len(responseBody)) >= maxProxyResponseBytes {
			return mcp.NewToolResultError("Docker API response exceeded the maximum allowed size and was truncated; try a more specific request"), nil
		}

		// Compact responses from endpoints known to return very verbose JSON.
		// This strips fields that are rarely useful in an LLM context (network
		// settings details, mount propagation, host config internals) to keep
		// the response within a size that can be reasoned about.
		compacted := compactDockerResponse(dockerAPIPath, responseBody)

		return mcp.NewToolResultText(string(compacted)), nil
	}
}
