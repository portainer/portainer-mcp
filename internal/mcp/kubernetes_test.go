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

			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.expectedErrorMsg)
			assert.Nil(t, result)
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
				assert.Error(t, err)
				assert.ErrorContains(t, err, tc.expect.errSubstring)
				assert.Nil(t, result)
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
