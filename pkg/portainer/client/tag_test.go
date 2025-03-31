package client

import (
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	sdkmodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvironmentTags(t *testing.T) {
	tests := []struct {
		name          string
		mockTags      []*sdkmodels.PortainerTag
		mockError     error
		expectedTags  []models.EnvironmentTag
		expectedError bool
	}{
		{
			name: "successful retrieval",
			mockTags: []*sdkmodels.PortainerTag{
				{ID: 1, Name: "prod"},
				{ID: 2, Name: "dev"},
			},
			mockError: nil,
			expectedTags: []models.EnvironmentTag{
				{ID: 1, Name: "prod", EnvironmentIds: []int{}},
				{ID: 2, Name: "dev", EnvironmentIds: []int{}},
			},
			expectedError: false,
		},
		{
			name:          "empty tags list",
			mockTags:      []*sdkmodels.PortainerTag{},
			mockError:     nil,
			expectedTags:  []models.EnvironmentTag{},
			expectedError: false,
		},
		{
			name:          "api error",
			mockTags:      nil,
			mockError:     fmt.Errorf("api error"),
			expectedTags:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListTags").Return(tt.mockTags, tt.mockError)

			client := &PortainerClient{
				cli: mockAPI,
			}

			tags, err := client.GetEnvironmentTags()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTags, tags)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestCreateEnvironmentTag(t *testing.T) {
	tests := []struct {
		name          string
		tagName       string
		mockID        int64
		mockError     error
		expectedID    int
		expectedError bool
	}{
		{
			name:          "successful creation",
			tagName:       "prod",
			mockID:        1,
			mockError:     nil,
			expectedID:    1,
			expectedError: false,
		},
		{
			name:          "api error",
			tagName:       "dev",
			mockID:        0,
			mockError:     fmt.Errorf("api error"),
			expectedID:    0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("CreateTag", tt.tagName).Return(tt.mockID, tt.mockError)

			client := &PortainerClient{
				cli: mockAPI,
			}

			id, err := client.CreateEnvironmentTag(tt.tagName)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}
