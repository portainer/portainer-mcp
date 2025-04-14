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

// Helper function to create a mock http.Response
func createMockHttpResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// Define TestHandleDockerProxyData struct locally for setupMock type safety
type TestHandleDockerProxyData struct {
	name               string
	inputParams        map[string]any
	mockEnvID          int
	mockAPIPath        string
	mockMethod         string
	mockBodyStr        string
	mockClientResponse *http.Response
	mockClientError    error
	expectError        bool
	expectedErrorMsg   string
	expectedResultText string
	setupMock          func(mockClient *MockPortainerClient, tc TestHandleDockerProxyData)
}

// Helper function to create a reader matcher for mock arguments
func readerMatcher(expectedContent string) interface{} {
	return mock.MatchedBy(func(r io.Reader) bool {
		if r == nil {
			return expectedContent == ""
		}
		b, err := io.ReadAll(r)
		if err != nil {
			return false
		}
		return string(b) == expectedContent
	})
}

func TestHandleDockerProxy(t *testing.T) {
	tests := []struct {
		name               string
		inputParams        map[string]any
		mockEnvID          int
		mockAPIPath        string
		mockMethod         string
		mockBodyStr        string // The string expected to be in the reader passed to the client
		mockClientResponse *http.Response
		mockClientError    error
		expectError        bool
		expectedErrorMsg   string
		expectedResultText string
		setupMock          func(mockClient *MockPortainerClient, tc TestHandleDockerProxyData)
	}{
		{
			name: "successful GET request",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
			},
			mockEnvID:          1,
			mockAPIPath:        "/containers/json",
			mockMethod:         "GET",
			mockBodyStr:        "", // No body for GET
			mockClientResponse: createMockHttpResponse(http.StatusOK, `[{"Id":"123"}]`),
			mockClientError:    nil,
			expectError:        false,
			expectedResultText: `[{"Id":"123"}]`,
		},
		{
			name: "successful POST request with body",
			inputParams: map[string]any{
				"environmentId": float64(2),
				"dockerAPIPath": "/containers/create",
				"method":        "POST",
				"body":          `{"name":"test"}`,
			},
			mockEnvID:          2,
			mockAPIPath:        "/containers/create",
			mockMethod:         "POST",
			mockBodyStr:        `{"name":"test"}`,
			mockClientResponse: createMockHttpResponse(http.StatusCreated, `{"Id":"456"}`),
			mockClientError:    nil,
			expectError:        false,
			expectedResultText: `{"Id":"456"}`,
		},
		{
			name: "invalid body type (not a string)",
			inputParams: map[string]any{
				"environmentId": float64(2),
				"dockerAPIPath": "/containers/create",
				"method":        "POST",
				"body":          123.45, // Invalid type for body
			},
			expectError:      true,
			expectedErrorMsg: "body must be a string",
		},
		{
			name: "missing environmentId",
			inputParams: map[string]any{
				"dockerAPIPath": "/containers/json",
				"method":        "GET",
			},
			expectError:      true,
			expectedErrorMsg: "environmentId is required",
		},
		{
			name: "missing dockerAPIPath",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"method":        "GET",
			},
			expectError:      true,
			expectedErrorMsg: "dockerAPIPath is required",
		},
		{
			name: "missing method",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
			},
			expectError:      true,
			expectedErrorMsg: "method is required",
		},
		{
			name: "invalid dockerAPIPath (no leading slash)",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "containers/json",
				"method":        "GET",
			},
			expectError:      true,
			expectedErrorMsg: "dockerAPIPath must start with a leading slash",
		},
		{
			name: "invalid HTTP method",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/containers/json",
				"method":        "INVALID",
			},
			expectError:      true,
			expectedErrorMsg: "invalid method: INVALID",
		},
		{
			name: "client API error",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/version",
				"method":        "GET",
			},
			mockEnvID:        1,
			mockAPIPath:      "/version",
			mockMethod:       "GET",
			mockBodyStr:      "",
			mockClientError:  errors.New("portainer api error"),
			expectError:      true,
			expectedErrorMsg: "failed to send Docker API request: portainer api error",
		},
		{
			name: "error reading response body",
			inputParams: map[string]any{
				"environmentId": float64(1),
				"dockerAPIPath": "/info",
				"method":        "GET",
			},
			mockEnvID:   1,
			mockAPIPath: "/info",
			mockMethod:  "GET",
			mockBodyStr: "",
			mockClientResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       &errorReader{}, // Simulate read error
			},
			mockClientError:  nil,
			expectError:      true,
			expectedErrorMsg: "failed to read Docker API response: simulated read error",
		},
	}

	for _, tt := range tests {
		// Reassign to loop variable tc of the correct type
		tc := TestHandleDockerProxyData(tt)
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(MockPortainerClient)

			// Setup mock only if no parameter validation error is expected
			if !tc.expectError || tc.mockClientError != nil || tc.mockClientResponse != nil {
				bodyMatcher := readerMatcher(tc.mockBodyStr)
				mockClient.On("ProxyDockerRequest",
					tc.mockEnvID,
					tc.mockAPIPath,
					tc.mockMethod,
					bodyMatcher,
				).Return(tc.mockClientResponse, tc.mockClientError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(tc.inputParams) // Use helper from another test file

			handler := server.handleDockerProxy() // Get the handler func
			result, err := handler(context.Background(), request)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErrorMsg != "" {
					assert.ErrorContains(t, err, tc.expectedErrorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Equal(t, tc.expectedResultText, textContent.Text)
			}

			mockClient.AssertExpectations(t)
		})
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
