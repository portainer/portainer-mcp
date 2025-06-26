package mcp

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleKubernetesProxy_ParameterValidation(t *testing.T) {
	tests := []struct {
		name             string
		inputParams      map[string]any
		expectedErrorMsg string
	}{
		{
			name: "invalid body type (not a string)",
			inputParams: map[string]any{
				"environmentId":     float64(2),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "POST",
				"body":              true, // Invalid type for body
			},
			expectedErrorMsg: "body must be a string",
		},
		{
			name: "missing environmentId",
			inputParams: map[string]any{
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
			},
			expectedErrorMsg: "environmentId is required",
		},
		{
			name: "missing kubernetesAPIPath",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"method":        "GET",
			},
			expectedErrorMsg: "kubernetesAPIPath is required",
		},
		{
			name: "missing method",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
			},
			expectedErrorMsg: "method is required",
		},
		{
			name: "invalid kubernetesAPIPath (no leading slash)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "api/v1/pods",
				"method":            "GET",
			},
			expectedErrorMsg: "kubernetesAPIPath must start with a leading slash",
		},
		{
			name: "invalid HTTP method",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "INVALID",
			},
			expectedErrorMsg: "invalid method: INVALID",
		},
		{
			name: "invalid queryParams type (not an array)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
				"queryParams":       "not-an-array",
			},
			expectedErrorMsg: "queryParams must be an array",
		},
		{
			name: "invalid queryParams content (value not string)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
				"queryParams":       []any{map[string]any{"key": "namespace", "value": false}},
			},
			expectedErrorMsg: "invalid query params: invalid value: false",
		},
		{
			name: "invalid headers type (not an array)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
				"headers":           "header-string",
			},
			expectedErrorMsg: "headers must be an array",
		},
		{
			name: "invalid headers content (missing value)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
				"headers":           []any{map[string]any{"key": "Content-Type"}},
			},
			expectedErrorMsg: "invalid headers: invalid value: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &PortainerMCPServer{} // No client needed for param validation

			request := CreateMCPRequest(tt.inputParams)
			handler := server.HandleKubernetesProxy()
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

func TestHandleKubernetesProxy_ClientInteraction(t *testing.T) {
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
			name: "successful GET request with query params",
			input: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"method":            "GET",
				"queryParams": []any{
					map[string]any{"key": "namespace", "value": "default"},
					map[string]any{"key": "labelSelector", "value": "app=myApp"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusOK, `{"kind":"PodList","items":[]}`),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `{"kind":"PodList","items":[]}`,
			},
		},
		{
			name: "successful POST request with body and headers",
			input: map[string]any{
				"environmentId":     float64(2),
				"kubernetesAPIPath": "/api/v1/namespaces/test/services",
				"method":            "POST",
				"body":              `{"apiVersion":"v1","kind":"Service","metadata":{"name":"my-service"}}`,
				"headers": []any{
					map[string]any{"key": "Content-Type", "value": "application/json"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusCreated, `{"metadata":{"name":"my-service"}}`),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `{"metadata":{"name":"my-service"}}`,
			},
		},
		{
			name: "client API error",
			input: map[string]any{
				"environmentId":     float64(3),
				"kubernetesAPIPath": "/version",
				"method":            "GET",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: nil,
				err:      errors.New("k8s api error"),
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to send Kubernetes API request: k8s api error",
			},
		},
		{
			name: "error reading response body",
			input: map[string]any{
				"environmentId":     float64(4),
				"kubernetesAPIPath": "/healthz",
				"method":            "GET",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &errorReader{}, // Simulate read error
				},
				err: nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to read Kubernetes API response: simulated read error",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(MockPortainerClient)

			mockClient.On("ProxyKubernetesRequest", mock.AnythingOfType("models.KubernetesProxyRequestOptions")).
				Return(tc.mock.response, tc.mock.err)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(tc.input)
			handler := server.HandleKubernetesProxy()
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

func TestHandleKubernetesProxyStripped_ParameterValidation(t *testing.T) {
	tests := []struct {
		name             string
		inputParams      map[string]any
		expectedErrorMsg string
	}{
		{
			name: "missing environmentId",
			inputParams: map[string]any{
				"kubernetesAPIPath": "/api/v1/pods",
			},
			expectedErrorMsg: "environmentId is required",
		},
		{
			name: "missing kubernetesAPIPath",
			inputParams: map[string]any{
				"environmentId": float64(1),
			},
			expectedErrorMsg: "kubernetesAPIPath is required",
		},
		{
			name: "invalid kubernetesAPIPath (no leading slash)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "api/v1/pods",
			},
			expectedErrorMsg: "kubernetesAPIPath must start with a leading slash",
		},
		{
			name: "invalid queryParams type (not an array)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"queryParams":       "not-an-array",
			},
			expectedErrorMsg: "queryParams must be an array",
		},
		{
			name: "invalid queryParams content (value not string)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"queryParams":       []any{map[string]any{"key": "namespace", "value": false}},
			},
			expectedErrorMsg: "invalid query params: invalid value: false",
		},
		{
			name: "invalid headers type (not an array)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"headers":           "header-string",
			},
			expectedErrorMsg: "headers must be an array",
		},
		{
			name: "invalid headers content (missing value)",
			inputParams: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"headers":           []any{map[string]any{"key": "Content-Type"}},
			},
			expectedErrorMsg: "invalid headers: invalid value: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &PortainerMCPServer{} // No client needed for param validation

			request := CreateMCPRequest(tt.inputParams)
			handler := server.HandleKubernetesProxyStripped()
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

func TestHandleKubernetesProxyStripped_ClientInteraction(t *testing.T) {
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
			name: "successful GET request with managedFields stripped",
			input: map[string]any{
				"environmentId":     float64(1),
				"kubernetesAPIPath": "/api/v1/pods",
				"queryParams": []any{
					map[string]any{"key": "namespace", "value": "default"},
					map[string]any{"key": "labelSelector", "value": "app=myApp"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusOK, `{
					"apiVersion": "v1",
					"kind": "PodList",
					"items": [
						{
							"apiVersion": "v1",
							"kind": "Pod",
							"metadata": {
								"name": "test-pod-1",
								"namespace": "default",
								"managedFields": [
									{
										"manager": "kubectl-client-side-apply",
										"operation": "Update",
										"apiVersion": "v1",
										"time": "2023-01-01T00:00:00Z"
									}
								]
							},
							"spec": {
								"containers": [
									{
										"name": "test-container",
										"image": "nginx"
									}
								]
							}
						}
					]
				}`),
				err: nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod-1","namespace":"default"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}],"kind":"PodList"}`,
			},
		},
		{
			name: "successful GET request with headers",
			input: map[string]any{
				"environmentId":     float64(2),
				"kubernetesAPIPath": "/api/v1/namespaces/default/pods",
				"headers": []any{
					map[string]any{"key": "X-Custom-Header", "value": "test-value"},
					map[string]any{"key": "Authorization", "value": "Bearer abc"},
				},
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusOK, `{
					"apiVersion": "v1",
					"kind": "Pod",
					"metadata": {
						"name": "test-pod",
						"namespace": "default",
						"managedFields": [
							{
								"manager": "kubectl-client-side-apply",
								"operation": "Update",
								"apiVersion": "v1",
								"time": "2023-01-01T00:00:00Z"
							}
						]
					},
					"spec": {
						"containers": [
							{
								"name": "test-container",
								"image": "nginx"
							}
						]
					}
				}`),
				err: nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod","namespace":"default"},"spec":{"containers":[{"image":"nginx","name":"test-container"}]}}`,
			},
		},
		{
			name: "client API error",
			input: map[string]any{
				"environmentId":     float64(3),
				"kubernetesAPIPath": "/version",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: nil,
				err:      errors.New("k8s api error"),
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to send Kubernetes API request: k8s api error",
			},
		},
		{
			name: "error processing response body",
			input: map[string]any{
				"environmentId":     float64(4),
				"kubernetesAPIPath": "/healthz",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       &errorReader{}, // Simulate read error
				},
				err: nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to process Kubernetes API response: failed to read response body: simulated read error",
			},
		},
		{
			name: "empty response body",
			input: map[string]any{
				"environmentId":     float64(5),
				"kubernetesAPIPath": "/api/v1/namespaces",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusNoContent, ""),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				resultText: "",
			},
		},
		{
			name: "invalid JSON response",
			input: map[string]any{
				"environmentId":     float64(6),
				"kubernetesAPIPath": "/api/v1/pods",
			},
			mock: struct {
				response *http.Response
				err      error
			}{
				response: createMockHttpResponse(http.StatusOK, "invalid json"),
				err:      nil,
			},
			expect: struct {
				errSubstring string
				resultText   string
			}{
				errSubstring: "failed to process Kubernetes API response: failed to unmarshal JSON into Unstructured",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(MockPortainerClient)

			mockClient.On("ProxyKubernetesRequest", mock.AnythingOfType("models.KubernetesProxyRequestOptions")).
				Return(tc.mock.response, tc.mock.err)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(tc.input)
			handler := server.HandleKubernetesProxyStripped()
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
				if tc.expect.resultText == "" {
					assert.Equal(t, tc.expect.resultText, textContent.Text)
				} else {
					assert.JSONEq(t, tc.expect.resultText, textContent.Text)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}
