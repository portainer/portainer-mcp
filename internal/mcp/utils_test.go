package mcp

import (
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
