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
