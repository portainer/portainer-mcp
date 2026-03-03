package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseAccessMap(t *testing.T) {
	tests := []struct {
		name    string
		entries []any
		want    map[int]string
		wantErr bool
	}{
		{
			name: "Valid single entry",
			entries: []any{
				map[string]any{
					"id":     float64(1),
					"access": AccessLevelEnvironmentAdmin,
				},
			},
			want: map[int]string{
				1: AccessLevelEnvironmentAdmin,
			},
			wantErr: false,
		},
		{
			name: "Valid multiple entries",
			entries: []any{
				map[string]any{
					"id":     float64(1),
					"access": AccessLevelEnvironmentAdmin,
				},
				map[string]any{
					"id":     float64(2),
					"access": AccessLevelReadonlyUser,
				},
			},
			want: map[int]string{
				1: AccessLevelEnvironmentAdmin,
				2: AccessLevelReadonlyUser,
			},
			wantErr: false,
		},
		{
			name: "Invalid entry type",
			entries: []any{
				"not a map",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid ID type",
			entries: []any{
				map[string]any{
					"id":     "string-id",
					"access": AccessLevelEnvironmentAdmin,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid access type",
			entries: []any{
				map[string]any{
					"id":     float64(1),
					"access": 123,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid access level",
			entries: []any{
				map[string]any{
					"id":     float64(1),
					"access": "invalid_access_level",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty entries",
			entries: []any{},
			want:    map[int]string{},
			wantErr: false,
		},
		{
			name: "Missing ID field",
			entries: []any{
				map[string]any{
					"access": AccessLevelEnvironmentAdmin,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing access field",
			entries: []any{
				map[string]any{
					"id": float64(1),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAccessMap(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAccessMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAccessMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidHTTPMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		expect bool
	}{
		{"Valid GET", "GET", true},
		{"Valid POST", "POST", true},
		{"Valid PUT", "PUT", true},
		{"Valid DELETE", "DELETE", true},
		{"Valid HEAD", "HEAD", true},
		{"Invalid lowercase get", "get", false},
		{"Invalid PATCH", "PATCH", false},
		{"Invalid OPTIONS", "OPTIONS", false},
		{"Invalid Empty", "", false},
		{"Invalid Random", "RANDOM", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHTTPMethod(tt.method)
			if got != tt.expect {
				t.Errorf("isValidHTTPMethod(%q) = %v, want %v", tt.method, got, tt.expect)
			}
		})
	}
}

func TestParseKeyValueMap(t *testing.T) {
	tests := []struct {
		name    string
		items   []any
		want    map[string]string
		wantErr bool
	}{
		{
			name: "Valid single entry",
			items: []any{
				map[string]any{"key": "k1", "value": "v1"},
			},
			want: map[string]string{
				"k1": "v1",
			},
			wantErr: false,
		},
		{
			name: "Valid multiple entries",
			items: []any{
				map[string]any{"key": "k1", "value": "v1"},
				map[string]any{"key": "k2", "value": "v2"},
			},
			want: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			wantErr: false,
		},
		{
			name:    "Empty items",
			items:   []any{},
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name: "Invalid item type",
			items: []any{
				"not a map",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid key type",
			items: []any{
				map[string]any{"key": 123, "value": "v1"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid value type",
			items: []any{
				map[string]any{"key": "k1", "value": 123},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing key field",
			items: []any{
				map[string]any{"value": "v1"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing value field",
			items: []any{
				map[string]any{"key": "k1"},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKeyValueMap(tt.items)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValueMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyValueMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyDockerDefaultFilters(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		queryParams map[string]string
		wantParams  map[string]string
	}{
		{
			name:        "containers/json gets default limit",
			path:        "/containers/json",
			queryParams: map[string]string{},
			wantParams:  map[string]string{"limit": "10"},
		},
		{
			name:        "containers/json explicit limit not overridden",
			path:        "/containers/json",
			queryParams: map[string]string{"limit": "5"},
			wantParams:  map[string]string{"limit": "5"},
		},
		{
			name:        "images/json gets default filter",
			path:        "/images/json",
			queryParams: map[string]string{},
			wantParams:  map[string]string{"filters": `{"dangling":["false"]}`},
		},
		{
			name:        "images/json explicit filter not overridden",
			path:        "/images/json",
			queryParams: map[string]string{"filters": `{"reference":["nginx"]}`},
			wantParams:  map[string]string{"filters": `{"reference":["nginx"]}`},
		},
		{
			name:        "other path not modified",
			path:        "/version",
			queryParams: map[string]string{},
			wantParams:  map[string]string{},
		},
		{
			name:        "other path with params not modified",
			path:        "/networks",
			queryParams: map[string]string{"filters": `{"driver":["bridge"]}`},
			wantParams:  map[string]string{"filters": `{"driver":["bridge"]}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyDockerDefaultFilters(tt.path, tt.queryParams)
			if !reflect.DeepEqual(tt.queryParams, tt.wantParams) {
				t.Errorf("applyDockerDefaultFilters(%q) params = %v, want %v", tt.path, tt.queryParams, tt.wantParams)
			}
		})
	}
}

func TestCompactDockerResponse(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		input     string
		checkFunc func(t *testing.T, output []byte)
	}{
		{
			name: "containers/json strips verbose fields",
			path: "/containers/json",
			input: `[{"Id":"abc123","Names":["/mycontainer"],"Image":"nginx:latest","State":"running",` +
				`"Status":"Up 2 hours","NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2"}}},"HostConfig":{"NetworkMode":"bridge"},` +
				`"Mounts":[{"Type":"volume","Name":"data","Destination":"/data"}],"GraphDriver":{"Name":"overlay2"},` +
				`"Labels":{"com.docker.compose.project":"mystack","build_version":"v1.0"},"ImageID":"sha256:abc123def"}]`,
			checkFunc: func(t *testing.T, output []byte) {
				var containers []map[string]any
				if err := json.Unmarshal(output, &containers); err != nil {
					t.Fatalf("failed to parse output: %v", err)
				}
				if len(containers) != 1 {
					t.Fatalf("expected 1 container, got %d", len(containers))
				}
				c := containers[0]
				// Essential fields preserved
				if c["Id"] != "abc123" {
					t.Error("Id field missing")
				}
				if c["Image"] != "nginx:latest" {
					t.Error("Image field missing")
				}
				if c["State"] != "running" {
					t.Error("State field missing")
				}
				// Verbose fields stripped
				for _, field := range verboseContainerFields {
					if _, exists := c[field]; exists {
						t.Errorf("verbose field %q should have been removed", field)
					}
				}
			},
		},
		{
			name:  "non-containers path returns unchanged",
			path:  "/images/json",
			input: `[{"Id":"sha256:abc","RepoTags":["nginx:latest"]}]`,
			checkFunc: func(t *testing.T, output []byte) {
				if string(output) != `[{"Id":"sha256:abc","RepoTags":["nginx:latest"]}]` {
					t.Errorf("non-containers path should not be modified")
				}
			},
		},
		{
			name:  "invalid JSON returns unchanged",
			path:  "/containers/json",
			input: `not valid json`,
			checkFunc: func(t *testing.T, output []byte) {
				if string(output) != "not valid json" {
					t.Error("invalid JSON should be returned unchanged")
				}
			},
		},
		{
			name:  "output is smaller than input",
			path:  "/containers/json",
			input: `[{"Id":"abc","Names":["/test"],"Image":"nginx","State":"running","NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2","Gateway":"172.17.0.1","MacAddress":"02:42:ac:11:00:02"}}},"HostConfig":{"NetworkMode":"bridge","RestartPolicy":{"Name":"always"}},"Mounts":[{"Type":"volume","Source":"/var/lib/docker/volumes/data/_data","Destination":"/data","Driver":"local","RW":true}],"GraphDriver":{"Name":"overlay2","Data":{"LowerDir":"/var/lib/docker/overlay2/abc/diff"}}}]`,
			checkFunc: func(t *testing.T, output []byte) {
				var containers []map[string]any
				if err := json.Unmarshal(output, &containers); err != nil {
					t.Fatalf("failed to parse: %v", err)
				}
				// The output should be meaningfully smaller
				if len(output) >= len(`[{"Id":"abc","Names":["/test"],"Image":"nginx","State":"running","NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2","Gateway":"172.17.0.1","MacAddress":"02:42:ac:11:00:02"}}},"HostConfig":{"NetworkMode":"bridge","RestartPolicy":{"Name":"always"}},"Mounts":[{"Type":"volume","Source":"/var/lib/docker/volumes/data/_data","Destination":"/data","Driver":"local","RW":true}],"GraphDriver":{"Name":"overlay2","Data":{"LowerDir":"/var/lib/docker/overlay2/abc/diff"}}}]`) {
					t.Errorf("compacted output (%d bytes) should be smaller than input", len(output))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := compactDockerResponse(tt.path, []byte(tt.input))
			tt.checkFunc(t, output)
		})
	}
}

func TestIsDangerousDockerPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{"exec endpoint blocked", "/containers/abc123/exec", true},
		{"exec endpoint uppercase", "/containers/abc123/EXEC", true},
		{"build endpoint blocked", "/build", true},
		{"build endpoint with query", "/build?dockerfile=Dockerfile", true},
		{"containers list allowed", "/containers/json", false},
		{"containers create allowed", "/containers/create", false},
		{"images list allowed", "/images/json", false},
		{"version allowed", "/version", false},
		{"info allowed", "/info", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDangerousDockerPath(tt.path)
			if got != tt.expect {
				t.Errorf("isDangerousDockerPath(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestIsPrivilegedDockerContainerCreate(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		body   string
		expect bool
	}{
		{
			name:   "privileged flag true",
			path:   "/containers/create",
			body:   `{"Image":"alpine","HostConfig":{"Privileged":true}}`,
			expect: true,
		},
		{
			name:   "privileged flag true with space after colon",
			path:   "/containers/create",
			body:   `{"Image":"alpine","HostConfig":{"Privileged": true}}`,
			expect: true,
		},
		{
			name:   "sensitive host mount /etc",
			path:   "/containers/create",
			body:   `{"HostConfig":{"Binds":["/etc:/etc"]}}`,
			expect: true,
		},
		{
			name:   "root host mount",
			path:   "/containers/create",
			body:   `{"HostConfig":{"Binds":["/:/ "]}}`,
			expect: true,
		},
		{
			name:   "normal create request allowed",
			path:   "/containers/create",
			body:   `{"Image":"nginx","HostConfig":{"Privileged":false}}`,
			expect: false,
		},
		{
			name:   "non-create path not checked",
			path:   "/containers/json",
			body:   `{"Privileged":true}`,
			expect: false,
		},
		{
			name:   "empty body on create allowed",
			path:   "/containers/create",
			body:   `{}`,
			expect: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrivilegedDockerContainerCreate(tt.path, tt.body)
			if got != tt.expect {
				t.Errorf("isPrivilegedDockerContainerCreate(%q, %q) = %v, want %v", tt.path, tt.body, got, tt.expect)
			}
		})
	}
}

func TestIsDangerousKubernetesPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		expect bool
	}{
		{"secrets GET allowed", "/api/v1/namespaces/default/secrets", "GET", false},
		{"secrets POST blocked", "/api/v1/namespaces/default/secrets", "POST", true},
		{"secrets PUT blocked", "/api/v1/namespaces/default/secrets", "PUT", true},
		{"secrets DELETE blocked", "/api/v1/namespaces/default/secrets", "DELETE", true},
		{"pods GET allowed", "/api/v1/pods", "GET", false},
		{"pods POST allowed", "/api/v1/pods", "POST", false},
		{"services PUT allowed", "/api/v1/namespaces/default/services/my-svc", "PUT", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDangerousKubernetesPath(tt.path, tt.method)
			if got != tt.expect {
				t.Errorf("isDangerousKubernetesPath(%q, %q) = %v, want %v", tt.path, tt.method, got, tt.expect)
			}
		})
	}
}

func TestHasWatchQueryParam(t *testing.T) {
	tests := []struct {
		name        string
		queryParams map[string]string
		expect      bool
	}{
		{"watch=true blocked", map[string]string{"watch": "true"}, true},
		{"watch=True blocked", map[string]string{"watch": "True"}, true},
		{"watch=TRUE blocked", map[string]string{"watch": "TRUE"}, true},
		{"watch=false allowed", map[string]string{"watch": "false"}, false},
		{"no watch param allowed", map[string]string{"labelSelector": "app=foo"}, false},
		{"empty params allowed", map[string]string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasWatchQueryParam(tt.queryParams)
			if got != tt.expect {
				t.Errorf("hasWatchQueryParam(%v) = %v, want %v", tt.queryParams, got, tt.expect)
			}
		})
	}
}
