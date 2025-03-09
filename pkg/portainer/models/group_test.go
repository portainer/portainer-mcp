package models

import (
	"reflect"
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertEdgeGroupToGroup(t *testing.T) {
	tests := []struct {
		name      string
		edgeGroup *models.EdgegroupsDecoratedEdgeGroup
		want      Group
	}{
		{
			name: "basic edge group conversion",
			edgeGroup: &models.EdgegroupsDecoratedEdgeGroup{
				ID:        1,
				Name:      "Production Servers",
				Endpoints: []int64{1, 2, 3},
			},
			want: Group{
				ID:             1,
				Name:           "Production Servers",
				EnvironmentIds: []int{1, 2, 3},
			},
		},
		{
			name: "edge group with no endpoints",
			edgeGroup: &models.EdgegroupsDecoratedEdgeGroup{
				ID:        2,
				Name:      "Empty Group",
				Endpoints: []int64{},
			},
			want: Group{
				ID:             2,
				Name:           "Empty Group",
				EnvironmentIds: []int{},
			},
		},
		{
			name: "edge group with single endpoint",
			edgeGroup: &models.EdgegroupsDecoratedEdgeGroup{
				ID:        3,
				Name:      "Single Server",
				Endpoints: []int64{4},
			},
			want: Group{
				ID:             3,
				Name:           "Single Server",
				EnvironmentIds: []int{4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEdgeGroupToGroup(tt.edgeGroup)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEdgeGroupToGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
