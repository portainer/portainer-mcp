package models

import (
	"testing"
	"time"

	"reflect"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertEdgeStackToStack(t *testing.T) {
	tests := []struct {
		name      string
		edgeStack *models.PortainereeEdgeStack
		want      Stack
	}{
		{
			name: "basic edge stack conversion",
			edgeStack: &models.PortainereeEdgeStack{
				ID:           1,
				Name:         "Web Application Stack",
				CreationDate: 1609459200, // 2021-01-01 00:00:00 UTC
				EdgeGroups:   []int64{1, 2, 3},
			},
			want: Stack{
				ID:                  1,
				Name:                "Web Application Stack",
				CreatedAt:           "2021-01-01T00:00:00Z",
				EnvironmentGroupIds: []int{1, 2, 3},
			},
		},
		{
			name: "edge stack with no groups",
			edgeStack: &models.PortainereeEdgeStack{
				ID:           2,
				Name:         "Empty Stack",
				CreationDate: 1640995200, // 2022-01-01 00:00:00 UTC
				EdgeGroups:   []int64{},
			},
			want: Stack{
				ID:                  2,
				Name:                "Empty Stack",
				CreatedAt:           "2022-01-01T00:00:00Z",
				EnvironmentGroupIds: []int{},
			},
		},
		{
			name: "edge stack with single group",
			edgeStack: &models.PortainereeEdgeStack{
				ID:           3,
				Name:         "Single Group Stack",
				CreationDate: 1672531200, // 2023-01-01 00:00:00 UTC
				EdgeGroups:   []int64{4},
			},
			want: Stack{
				ID:                  3,
				Name:                "Single Group Stack",
				CreatedAt:           "2023-01-01T00:00:00Z",
				EnvironmentGroupIds: []int{4},
			},
		},
		{
			name: "edge stack with current timestamp",
			edgeStack: &models.PortainereeEdgeStack{
				ID:           4,
				Name:         "Recent Stack",
				CreationDate: time.Now().Add(-24 * time.Hour).Unix(), // Yesterday
				EdgeGroups:   []int64{1, 2},
			},
			want: Stack{
				ID:                  4,
				Name:                "Recent Stack",
				CreatedAt:           time.Unix(time.Now().Add(-24*time.Hour).Unix(), 0).Format(time.RFC3339),
				EnvironmentGroupIds: []int{1, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEdgeStackToStack(tt.edgeStack)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEdgeStackToStack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertRawLocalStackToLocalStack(t *testing.T) {
	tests := []struct {
		name string
		raw  RawLocalStack
		want LocalStack
	}{
		{
			name: "active compose stack",
			raw: RawLocalStack{
				ID:           1,
				Name:         "web-app",
				Type:         2,
				Status:       1,
				EndpointID:   3,
				CreationDate: 1700000000, // 2023-11-14T22:13:20Z
				UpdateDate:   1700000100,
				Env: []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}{
					{Name: "DB_HOST", Value: "localhost"},
				},
			},
			want: LocalStack{
				ID:         1,
				Name:       "web-app",
				Type:       "compose",
				Status:     "active",
				EndpointID: 3,
				CreatedAt:  time.Unix(1700000000, 0).Format(time.RFC3339),
				UpdatedAt:  time.Unix(1700000100, 0).Format(time.RFC3339),
				Env:        []LocalStackEnvVar{{Name: "DB_HOST", Value: "localhost"}},
			},
		},
		{
			name: "inactive swarm stack",
			raw: RawLocalStack{
				ID:           2,
				Name:         "monitoring",
				Type:         1,
				Status:       2,
				EndpointID:   3,
				CreationDate: 1700000000,
				UpdateDate:   0,
				Env:          nil,
			},
			want: LocalStack{
				ID:         2,
				Name:       "monitoring",
				Type:       "swarm",
				Status:     "inactive",
				EndpointID: 3,
				CreatedAt:  time.Unix(1700000000, 0).Format(time.RFC3339),
				UpdatedAt:  "",
				Env:        []LocalStackEnvVar{},
			},
		},
		{
			name: "stack with no timestamps",
			raw: RawLocalStack{
				ID:           3,
				Name:         "minimal",
				Type:         2,
				Status:       1,
				EndpointID:   3,
				CreationDate: 0,
				UpdateDate:   0,
				Env:          nil,
			},
			want: LocalStack{
				ID:         3,
				Name:       "minimal",
				Type:       "compose",
				Status:     "active",
				EndpointID: 3,
				CreatedAt:  "",
				UpdatedAt:  "",
				Env:        []LocalStackEnvVar{},
			},
		},
		{
			name: "stack with unknown type and status",
			raw: RawLocalStack{
				ID:           4,
				Name:         "unknown",
				Type:         99,
				Status:       99,
				EndpointID:   3,
				CreationDate: 1700000000,
				UpdateDate:   0,
				Env:          nil,
			},
			want: LocalStack{
				ID:         4,
				Name:       "unknown",
				Type:       "unknown",
				Status:     "unknown",
				EndpointID: 3,
				CreatedAt:  time.Unix(1700000000, 0).Format(time.RFC3339),
				UpdatedAt:  "",
				Env:        []LocalStackEnvVar{},
			},
		},
		{
			name: "stack with multiple env vars",
			raw: RawLocalStack{
				ID:           5,
				Name:         "multi-env",
				Type:         2,
				Status:       1,
				EndpointID:   3,
				CreationDate: 1700000000,
				UpdateDate:   0,
				Env: []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}{
					{Name: "KEY1", Value: "val1"},
					{Name: "KEY2", Value: "val2"},
					{Name: "KEY3", Value: "val3"},
				},
			},
			want: LocalStack{
				ID:         5,
				Name:       "multi-env",
				Type:       "compose",
				Status:     "active",
				EndpointID: 3,
				CreatedAt:  time.Unix(1700000000, 0).Format(time.RFC3339),
				UpdatedAt:  "",
				Env: []LocalStackEnvVar{
					{Name: "KEY1", Value: "val1"},
					{Name: "KEY2", Value: "val2"},
					{Name: "KEY3", Value: "val3"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertRawLocalStackToLocalStack(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertRawLocalStackToLocalStack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalStackStatusString(t *testing.T) {
	tests := []struct {
		name   string
		status LocalStackStatus
		want   string
	}{
		{"active", LocalStackStatusActive, "active"},
		{"inactive", LocalStackStatusInactive, "inactive"},
		{"unknown", LocalStackStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("LocalStackStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalStackTypeString(t *testing.T) {
	tests := []struct {
		name     string
		stackType LocalStackType
		want     string
	}{
		{"swarm", LocalStackTypeSwarm, "swarm"},
		{"compose", LocalStackTypeCompose, "compose"},
		{"unknown", LocalStackType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stackType.String(); got != tt.want {
				t.Errorf("LocalStackType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
