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
