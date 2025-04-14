package client

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyDockerRequest(t *testing.T) {
	tests := []struct {
		name             string
		environmentId    int
		dockerAPIPath    string
		method           string
		body             io.Reader // nil is a valid value for no body
		mockResponse     *http.Response
		mockError        error
		expectedError    bool
		expectedStatus   int
		expectedRespBody string // Add expected response body content
	}{
		{
			name:          "successful GET request",
			environmentId: 1,
			dockerAPIPath: "/containers/json",
			method:        "GET",
			body:          nil, // Explicitly nil for no body
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"Id":"123"}]`)), // Simulate a response body
			},
			mockError:        nil,
			expectedError:    false,
			expectedStatus:   http.StatusOK,
			expectedRespBody: `[{"Id":"123"}]`,
		},
		{
			name:          "POST request with body",
			environmentId: 2,
			dockerAPIPath: "/containers/create",
			method:        "POST",
			body:          bytes.NewBufferString(`{"Image": "nginx"}`),
			mockResponse: &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"Id": "456"}`)), // Simulate creation response
			},
			mockError:        nil,
			expectedError:    false,
			expectedStatus:   http.StatusCreated,
			expectedRespBody: `{"Id": "456"}`,
		},
		{
			name:             "API error",
			environmentId:    1,
			dockerAPIPath:    "/version",
			method:           "GET",
			body:             nil,
			mockResponse:     nil,
			mockError:        errors.New("failed to proxy request"),
			expectedError:    true,
			expectedStatus:   0,  // Not applicable
			expectedRespBody: "", // Not applicable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)

			// Pass tt.body directly. testify/mock should handle matching typed nil interfaces.
			mockAPI.On("ProxyDockerRequest", tt.environmentId, tt.dockerAPIPath, tt.method, tt.body).Return(tt.mockResponse, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			// Make a copy of the body if it's a bytes.Buffer or similar, as io.Reader reads can consume it.
			// For this test setup, since we control the mock, we pass the original tt.body.
			// If the function being tested *read* the body, we'd need to be more careful.
			resp, err := client.ProxyDockerRequest(tt.environmentId, tt.dockerAPIPath, tt.method, tt.body)

			if tt.expectedError {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.mockError.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)

				// Read and verify the response body
				if assert.NotNil(t, resp.Body) { // Ensure body is not nil before reading
					defer resp.Body.Close()
					bodyBytes, readErr := io.ReadAll(resp.Body)
					assert.NoError(t, readErr)
					assert.Equal(t, tt.expectedRespBody, string(bodyBytes))
				}
			}

			mockAPI.AssertExpectations(t)
		})
	}
}
