package models

import (
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertTagToEnvironmentTag(t *testing.T) {
	tests := []struct {
		name         string
		portainerTag *models.PortainerTag
		want         EnvironmentTag
	}{
		{
			name: "basic tag conversion",
			portainerTag: &models.PortainerTag{
				ID:   1,
				Name: "Production",
				Endpoints: map[string]bool{
					"1": true,
					"2": true,
					"3": true,
				},
			},
			want: EnvironmentTag{
				ID:             1,
				Name:           "Production",
				EnvironmentIds: []int{1, 2, 3},
			},
		},
		{
			name: "tag with no endpoints",
			portainerTag: &models.PortainerTag{
				ID:        2,
				Name:      "Empty Tag",
				Endpoints: map[string]bool{},
			},
			want: EnvironmentTag{
				ID:             2,
				Name:           "Empty Tag",
				EnvironmentIds: []int{},
			},
		},
		{
			name: "tag with invalid endpoint ID",
			portainerTag: &models.PortainerTag{
				ID:   3,
				Name: "Mixed IDs",
				Endpoints: map[string]bool{
					"42":      true,
					"abc":     true, // Invalid ID, should be skipped
					"99":      true,
					"invalid": true, // Invalid ID, should be skipped
				},
			},
			want: EnvironmentTag{
				ID:             3,
				Name:           "Mixed IDs",
				EnvironmentIds: []int{42, 99},
			},
		},
		{
			name: "tag with single endpoint",
			portainerTag: &models.PortainerTag{
				ID:   4,
				Name: "Single Server",
				Endpoints: map[string]bool{
					"5": true,
				},
			},
			want: EnvironmentTag{
				ID:             4,
				Name:           "Single Server",
				EnvironmentIds: []int{5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertTagToEnvironmentTag(tt.portainerTag)

			// Since the order of EnvironmentIds is not guaranteed due to map iteration,
			// we need to sort both slices before comparison
			if !compareEnvironmentTags(got, tt.want) {
				t.Errorf("ConvertTagToEnvironmentTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// compareEnvironmentTags compares two EnvironmentTag structs, handling the
// unordered nature of the EnvironmentIds slice
func compareEnvironmentTags(a, b EnvironmentTag) bool {
	if a.ID != b.ID || a.Name != b.Name || len(a.EnvironmentIds) != len(b.EnvironmentIds) {
		return false
	}

	// Create maps to check if all IDs exist in both slices
	aMap := make(map[int]bool)
	bMap := make(map[int]bool)

	for _, id := range a.EnvironmentIds {
		aMap[id] = true
	}

	for _, id := range b.EnvironmentIds {
		bMap[id] = true
		if !aMap[id] {
			return false
		}
	}

	// Check if all IDs in a exist in b
	for id := range aMap {
		if !bMap[id] {
			return false
		}
	}

	return true
}
