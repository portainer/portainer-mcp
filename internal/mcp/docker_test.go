package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createMockHttpResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// errorReader simulates an error during io.ReadAll
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func (r *errorReader) Close() error {
	return nil
}

func TestHandleDockerProxy_ParameterValidation(t *testing.T) {
	tests := []struct {
		name             string
		inputParams      map[string]any
		expectedErrorMsg string
	}{
		{
			name: "invalid body type (not a string)",
			inputParams: map[string]any{
				"environmentId": float64(2),
				"dockerAPIPath": "/containers/create",
				"method":        "POST",
				"body":          123.45, // Invalid type for body
			},
			expectedErrorMsg: "body must be a string",
		},
		{
			name: "missing environmentId",
			inputParams: map[string]any{
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
			},
			expectedErrorMsg: "environmentId is required",
		},
		{
			name: "missing dockerAPIPath",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"method":        "GET",
			},
			expectedErrorMsg: "dockerAPIPath is required",
		},
		{
			name: "missing method",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
			},
			expectedErrorMsg: "method is required",
		},
		{
			name: "invalid dockerAPIPath (no leading slash)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "containers/json",
				"method":        "GET",
			},
			expectedErrorMsg: "dockerAPIPath must start with a leading slash",
		},
		{
			name: "invalid HTTP method",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "INVALID",
			},
			expectedErrorMsg: "invalid method: INVALID",
		},
		{
			name: "invalid queryParams type (not an array)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
				"queryParams":   "not-an-array", // Invalid type
			},
			expectedErrorMsg: "queryParams must be an array",
		},
		{
			name: "invalid queryParams content (missing key)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
				"queryParams":   []any{map[string]any{"value": "true"}}, // Missing 'key'
			},
			expectedErrorMsg: "invalid query params: invalid key: <nil>",
		},
		{
			name: "invalid headers type (not an array)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
				"headers":       map[string]any{"key": "value"}, // Invalid type
			},
			expectedErrorMsg: "headers must be an array",
		},
		{
			name: "invalid headers content (value not string)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
				"headers":       []any{map[string]any{"key": "X-Custom", "value": 123}}, // Value not string
			},
			expectedErrorMsg: "invalid headers: invalid value: 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &PortainerMCPServer{}

			request := CreateMCPRequest(tt.inputParams)
			handler := server.HandleDockerProxy()
			result, err := handler(context.Background(), request)

			// All parameter/validation errors now return (result{IsError: true}, nil)
			assert.NoError(t, err)   // Handler now returns nil error
			assert.NotNil(t, result) // Handler returns a result object
			assert.True(t, result.IsError, "result.IsError should be true for parameter validation errors")
			assert.Len(t, result.Content, 1)                       // Expect one content item for the error message
			textContent, ok := result.Content[0].(mcp.TextContent) // Content should be TextContent
			assert.True(t, ok, "Result content should be mcp.TextContent for errors")
			assert.Contains(t, textContent.Text, tt.expectedErrorMsg, "Error message mismatch")
		})
	}
}

func TestHandleDockerProxy_ReadOnlyMode(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		expectError bool
	}{
		{
			name:        "GET allowed in read-only mode",
			method:      "GET",
			expectError: false,
		},
		{
			name:        "POST blocked in read-only mode",
			method:      "POST",
			expectError: true,
		},
		{
			name:        "PUT blocked in read-only mode",
			method:      "PUT",
			expectError: true,
		},
		{
			name:        "DELETE blocked in read-only mode",
			method:      "DELETE",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockPortainerClient)

			if !tt.expectError {
				mockClient.On("ProxyDockerRequest", mock.AnythingOfType("models.DockerProxyRequestOptions")).
					Return(createMockHttpResponse(http.StatusOK, `{}`), nil)
			}

			server := &PortainerMCPServer{
				cli:      mockClient,
				readOnly: true,
			}

			request := CreateMCPRequest(map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        tt.method,
			})

			handler := server.HandleDockerProxy()
			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.expectError {
				assert.True(t, result.IsError)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Contains(t, textContent.Text, "only GET requests are allowed in read-only mode")
			} else {
				assert.False(t, result.IsError)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleDockerProxy_ClientInteraction(t *testing.T) {
	type testCase struct {
		name  string
		input map[string]any // Parameters for the MCP request
		mock  struct {       // Details for mocking the client call
			response *http.Response
			err      error
		}
		expect struct { // Expected outcome
			errSubstring string // Check for error containing this text (if error expected)
			resultText   string // Expected text result (if success expected)
		}
	}

	tests := []testCase{
		{
			name: "successful GET request", // Query params are parsed by toolgen, but not yet passed by handler
			input: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
				"queryParams": []any{ //
					map[string]any{"key": "all", "value": "true"},
					map[string]any{"key": "filter", "value": "dangling"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusOK, `[{"Id":"123"}]`),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `[{"Id":"123"}]`,
			},
		},
		{
			name: "successful POST request with body",
			input: map[string]any{
				"environmentId": float64(2),
				"dockerAPIPath": "/containers/create",
				"method":        "POST",
				"body":          `{"name":"test"}`,
				"headers": []any{
					map[string]any{"key": "X-Custom", "value": "test-value"},
					map[string]any{"key": "Authorization", "value": "Bearer abc"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusCreated, `{"Id":"456"}`),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `{"Id":"456"}`,
			},
		},
		{
			name: "client API error",
			input: map[string]any{
				"environmentId": float64(3),
				"dockerAPIPath": "/version",
				"method":        "GET",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: nil,
				err:      errors.New("portainer api error"),
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to send Docker API request: portainer api error",
			},
		},
		{
			name: "error reading response body",
			input: map[string]any{
				"environmentId": float64(4),
				"dockerAPIPath": "/info",
				"method":        "GET",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &errorReader{}, // Simulate read error
				},
				err: nil, // No client error, but response body read fails
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to read Docker API response: simulated read error",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(MockPortainerClient)

			mockClient.On("ProxyDockerRequest", mock.AnythingOfType("models.DockerProxyRequestOptions")).
				Return(tc.mock.response, tc.mock.err)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(tc.input)
			handler := server.HandleDockerProxy()
			result, err := handler(context.Background(), request)

			if tc.expect.errSubstring != "" {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				assert.Contains(t, textContent.Text, tc.expect.errSubstring)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Equal(t, tc.expect.resultText, textContent.Text)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
