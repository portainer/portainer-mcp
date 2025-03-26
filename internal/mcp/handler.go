package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Handler is a generic interface for both resource and tool handlers
type Handler interface {
	// GetKind returns the kind of handler (resource or tool)
	GetKind() string
	// GetName returns the name of the handler
	GetName() string
}

// ResourceHandler represents a handler for resource operations
type ResourceHandler struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	handlerFunc server.ResourceHandlerFunc
}

// GetKind returns "resource" for ResourceHandler
func (h *ResourceHandler) GetKind() string {
	return "resource"
}

// GetName returns the name of the resource
func (h *ResourceHandler) GetName() string {
	return h.Name
}

// GetHandlerFunc returns the resource handler function
func (h *ResourceHandler) GetHandlerFunc() server.ResourceHandlerFunc {
	return h.handlerFunc
}

// NewResourceHandler creates a new resource handler
func NewResourceHandler(uri, name, description string, handlerFunc server.ResourceHandlerFunc) *ResourceHandler {
	return &ResourceHandler{
		URI:         uri,
		Name:        name,
		Description: description,
		MIMEType:    MIMETypeJSON,
		handlerFunc: handlerFunc,
	}
}

// GetResource returns the mcp.Resource for this handler
func (h *ResourceHandler) GetResource() mcp.Resource {
	return mcp.NewResource(h.URI, h.Name,
		mcp.WithResourceDescription(h.Description),
		mcp.WithMIMEType(h.MIMEType),
	)
}

// ResourceHandlerCreator is a function that creates a resource handler
type ResourceHandlerCreator func(s *PortainerMCPServer) *ResourceHandler

// ToolHandler represents a handler for tool operations
type ToolHandler struct {
	Name        string
	Description string
	ToolDef     mcp.Tool
	handlerFunc server.ToolHandlerFunc
}

// GetKind returns "tool" for ToolHandler
func (h *ToolHandler) GetKind() string {
	return "tool"
}

// GetName returns the name of the tool
func (h *ToolHandler) GetName() string {
	return h.Name
}

// GetHandlerFunc returns the tool handler function
func (h *ToolHandler) GetHandlerFunc() server.ToolHandlerFunc {
	return h.handlerFunc
}

// NewToolHandler creates a new tool handler with a predefined tool definition
func NewToolHandler(tool mcp.Tool, handlerFunc server.ToolHandlerFunc) *ToolHandler {
	return &ToolHandler{
		Name:        tool.GetName(),
		Description: tool.GetDescription(),
		ToolDef:     tool,
		handlerFunc: handlerFunc,
	}
}

// GetTool returns the mcp.Tool for this handler
func (h *ToolHandler) GetTool() mcp.Tool {
	return h.ToolDef
}

// ToolHandlerCreator is a function that creates a tool handler
type ToolHandlerCreator func(s *PortainerMCPServer) *ToolHandler

// ResourceResponse is a standardized resource response
type ResourceResponse struct {
	URI      string
	MIMEType string
	Data     interface{}
}

// CreateResourceContents creates resource contents from a response
func CreateResourceContents(response ResourceResponse) ([]mcp.ResourceContents, error) {
	data, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      response.URI,
			MIMEType: response.MIMEType,
			Text:     string(data),
		},
	}, nil
}

// CreateResourceHandler is a helper function to create a resource handler with a standard pattern
func CreateResourceHandler(
	name string,
	description string,
	uri string,
	handler func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error),
) server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		data, err := handler(ctx, request)
		if err != nil {
			return nil, err
		}

		return CreateResourceContents(ResourceResponse{
			URI:      uri,
			MIMEType: MIMETypeJSON,
			Data:     data,
		})
	}
}

// CreateErrorResponse creates a standardized error response for tools
func CreateErrorResponse(err error) *mcp.CallToolResult {
	var message string
	var code string

	// Check for domain error
	if domainErr, ok := err.(*DomainError); ok {
		message = domainErr.Message
		code = string(domainErr.Code)
	} else {
		message = err.Error()
		code = string(ErrCodeServerError)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Error: %s (%s)", message, code))
}

// CreateSuccessResponse creates a standardized success response for tools
func CreateSuccessResponse(message string) *mcp.CallToolResult {
	return mcp.NewToolResultText(message)
}