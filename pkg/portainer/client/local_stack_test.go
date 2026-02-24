package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
)

// newTestClient creates a PortainerClient pointing at a httptest.Server
func newTestClient(t *testing.T, serverURL string) *PortainerClient {
	t.Helper()
	return &PortainerClient{
		serverURL: serverURL,
		token:     "test-token",
		httpCli:   &http.Client{},
	}
}

func TestGetLocalStacks(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "successful retrieval",
			statusCode: http.StatusOK,
			body: `[
				{"Id":1,"Name":"stack1","Type":2,"Status":1,"EndpointId":3,"CreationDate":1700000000,"UpdateDate":0,"Env":[]},
				{"Id":2,"Name":"stack2","Type":2,"Status":2,"EndpointId":3,"CreationDate":1700000100,"UpdateDate":1700000200,"Env":[{"name":"KEY","value":"val"}]}
			]`,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `[]`,
			wantCount:  0,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"internal error"}`,
			wantErr:    true,
		},
		{
			name:       "invalid json",
			statusCode: http.StatusOK,
			body:       `invalid`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/stacks", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			stacks, err := client.GetLocalStacks()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, stacks, tt.wantCount)
				if tt.wantCount > 0 {
					assert.Equal(t, "stack1", stacks[0].Name)
					assert.Equal(t, "compose", stacks[0].Type)
					assert.Equal(t, "active", stacks[0].Status)
				}
			}
		})
	}
}

func TestGetLocalStackFile(t *testing.T) {
	tests := []struct {
		name       string
		stackID    int
		statusCode int
		body       string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "successful retrieval",
			stackID:    1,
			statusCode: http.StatusOK,
			body:       `{"StackFileContent":"services:\n  web:\n    image: nginx"}`,
			wantFile:   "services:\n  web:\n    image: nginx",
			wantErr:    false,
		},
		{
			name:       "not found",
			stackID:    999,
			statusCode: http.StatusNotFound,
			body:       `{"message":"stack not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/")
				assert.Contains(t, r.URL.Path, "/file")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			content, err := client.GetLocalStackFile(tt.stackID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantFile, content)
			}
		})
	}
}

func TestCreateLocalStack(t *testing.T) {
	tests := []struct {
		name       string
		endpointId int
		stackName  string
		file       string
		env        []models.LocalStackEnvVar
		statusCode int
		body       string
		wantID     int
		wantErr    bool
	}{
		{
			name:       "successful creation",
			endpointId: 3,
			stackName:  "test-stack",
			file:       "services:\n  web:\n    image: nginx",
			env:        []models.LocalStackEnvVar{},
			statusCode: http.StatusOK,
			body:       `{"Id":10,"Name":"test-stack","Type":2,"Status":1,"EndpointId":3}`,
			wantID:     10,
			wantErr:    false,
		},
		{
			name:       "creation with env vars",
			endpointId: 3,
			stackName:  "test-stack",
			file:       "services:\n  web:\n    image: nginx",
			env:        []models.LocalStackEnvVar{{Name: "DB_HOST", Value: "localhost"}},
			statusCode: http.StatusOK,
			body:       `{"Id":11,"Name":"test-stack","Type":2,"Status":1,"EndpointId":3}`,
			wantID:     11,
			wantErr:    false,
		},
		{
			name:       "server error",
			endpointId: 3,
			stackName:  "test-stack",
			file:       "services:\n  web:\n    image: nginx",
			env:        []models.LocalStackEnvVar{},
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"failed"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/create/standalone/string")
				assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				bodyBytes, _ := io.ReadAll(r.Body)
				var reqBody map[string]interface{}
				json.Unmarshal(bodyBytes, &reqBody)
				assert.Equal(t, tt.stackName, reqBody["name"])
				assert.Equal(t, tt.file, reqBody["stackFileContent"])

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			id, err := client.CreateLocalStack(tt.endpointId, tt.stackName, tt.file, tt.env)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestUpdateLocalStack(t *testing.T) {
	tests := []struct {
		name       string
		stackID    int
		endpointId int
		file       string
		env        []models.LocalStackEnvVar
		prune      bool
		pullImage  bool
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful update",
			stackID:    1,
			endpointId: 3,
			file:       "services:\n  web:\n    image: nginx:latest",
			env:        []models.LocalStackEnvVar{},
			prune:      false,
			pullImage:  false,
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:       "update with options",
			stackID:    1,
			endpointId: 3,
			file:       "services:\n  web:\n    image: nginx:latest",
			env:        []models.LocalStackEnvVar{{Name: "KEY", Value: "val"}},
			prune:      true,
			pullImage:  true,
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			stackID:    1,
			endpointId: 3,
			file:       "services:\n  web:\n    image: nginx",
			env:        []models.LocalStackEnvVar{},
			prune:      false,
			pullImage:  false,
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"failed"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/")
				assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))

				bodyBytes, _ := io.ReadAll(r.Body)
				var reqBody map[string]interface{}
				json.Unmarshal(bodyBytes, &reqBody)
				assert.Equal(t, tt.file, reqBody["stackFileContent"])
				assert.Equal(t, tt.prune, reqBody["prune"])
				assert.Equal(t, tt.pullImage, reqBody["pullImage"])

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			err := client.UpdateLocalStack(tt.stackID, tt.endpointId, tt.file, tt.env, tt.prune, tt.pullImage)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartLocalStack(t *testing.T) {
	tests := []struct {
		name       string
		stackID    int
		endpointId int
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful start",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"failed"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/")
				assert.Contains(t, r.URL.Path, "/start")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			err := client.StartLocalStack(tt.stackID, tt.endpointId)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStopLocalStack(t *testing.T) {
	tests := []struct {
		name       string
		stackID    int
		endpointId int
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful stop",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"failed"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/")
				assert.Contains(t, r.URL.Path, "/stop")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			err := client.StopLocalStack(tt.stackID, tt.endpointId)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteLocalStack(t *testing.T) {
	tests := []struct {
		name       string
		stackID    int
		endpointId int
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "successful delete with 204",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusNoContent,
			body:       ``,
			wantErr:    false,
		},
		{
			name:       "successful delete with 200",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			stackID:    1,
			endpointId: 3,
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"failed"}`,
			wantErr:    true,
		},
		{
			name:       "not found",
			stackID:    999,
			endpointId: 3,
			statusCode: http.StatusNotFound,
			body:       `{"message":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Contains(t, r.URL.Path, "/api/stacks/")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			err := client.DeleteLocalStack(tt.stackID, tt.endpointId)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIRequestSetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	resp, err := client.apiRequest(http.MethodPost, "/api/test", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestAPIRequestWithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-token", r.Header.Get("X-API-Key"))
		assert.Empty(t, r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	resp, err := client.apiRequest(http.MethodGet, "/api/test", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestURLSchemeNormalization(t *testing.T) {
	tests := []struct {
		name       string
		serverURL  string
		wantScheme string
	}{
		{
			name:       "no scheme adds https",
			serverURL:  "example.com:9443",
			wantScheme: "https://example.com:9443",
		},
		{
			name:       "http preserved",
			serverURL:  "http://example.com:8080",
			wantScheme: "http://example.com:8080",
		},
		{
			name:       "https preserved",
			serverURL:  "https://example.com:9443",
			wantScheme: "https://example.com:9443",
		},
		{
			name:       "trailing slash removed",
			serverURL:  "https://example.com:9443/",
			wantScheme: "https://example.com:9443",
		},
		{
			name:       "no scheme with trailing slash",
			serverURL:  "example.com:9443/",
			wantScheme: "https://example.com:9443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewPortainerClient(tt.serverURL, "test-token", WithSkipTLSVerify(true))
			assert.Equal(t, tt.wantScheme, c.serverURL)
		})
	}
}
