package client

import (
	"errors"
	"testing"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetEnvironmentGroups(t *testing.T) {
	tests := []struct {
		name          string
		mockGroups    []*apimodels.EdgegroupsDecoratedEdgeGroup
		mockError     error
		expected      []models.Group
		expectedError bool
	}{
		{
			name: "successful retrieval",
			mockGroups: []*apimodels.EdgegroupsDecoratedEdgeGroup{
				{
					ID:        1,
					Name:      "group1",
					Endpoints: []int64{1, 2},
					TagIds:    []int64{1, 2},
				},
				{
					ID:        2,
					Name:      "group2",
					Endpoints: []int64{3},
					TagIds:    []int64{3},
				},
			},
			expected: []models.Group{
				{
					ID:             1,
					Name:           "group1",
					EnvironmentIds: []int{1, 2},
					TagIds:         []int{1, 2},
				},
				{
					ID:             2,
					Name:           "group2",
					EnvironmentIds: []int{3},
					TagIds:         []int{3},
				},
			},
		},
		{
			name:       "empty groups",
			mockGroups: []*apimodels.EdgegroupsDecoratedEdgeGroup{},
			expected:   []models.Group{},
		},
		{
			name:          "list error",
			mockError:     errors.New("failed to list edge groups"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListEdgeGroups").Return(tt.mockGroups, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			groups, err := client.GetEnvironmentGroups()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, groups)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestCreateEnvironmentGroup(t *testing.T) {
	tests := []struct {
		name           string
		groupName      string
		environmentIds []int
		mockID         int64
		mockError      error
		expectedID     int
		expectedError  bool
	}{
		{
			name:           "successful creation",
			groupName:      "new-group",
			environmentIds: []int{1, 2, 3},
			mockID:         1,
			expectedID:     1,
		},
		{
			name:           "creation error",
			groupName:      "error-group",
			environmentIds: []int{1},
			mockError:      errors.New("failed to create group"),
			expectedError:  true,
		},
		{
			name:           "empty environments",
			groupName:      "empty-group",
			environmentIds: []int{},
			mockID:         2,
			expectedID:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("CreateEdgeGroup", tt.groupName, mock.Anything).Return(tt.mockID, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			id, err := client.CreateEnvironmentGroup(tt.groupName, tt.environmentIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedID, id)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentGroupName(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		newName       string
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful update",
			groupID: 1,
			newName: "updated-group",
		},
		{
			name:          "update error",
			groupID:       1,
			newName:       "error-group",
			mockError:     errors.New("failed to update group name"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEdgeGroup", int64(tt.groupID), tt.newName, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentGroupName(tt.groupID, tt.newName)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentGroupEnvironments(t *testing.T) {
	tests := []struct {
		name           string
		groupID        int
		groupName      string
		environmentIds []int
		mockError      error
		expectedError  bool
	}{
		{
			name:           "successful update",
			groupID:        1,
			groupName:      "test-group",
			environmentIds: []int{1, 2, 3},
		},
		{
			name:           "update error",
			groupID:        1,
			groupName:      "error-group",
			environmentIds: []int{1},
			mockError:      errors.New("failed to update group environments"),
			expectedError:  true,
		},
		{
			name:           "empty environments",
			groupID:        1,
			groupName:      "empty-group",
			environmentIds: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEdgeGroup", int64(tt.groupID), tt.groupName, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentGroupEnvironments(tt.groupID, tt.groupName, tt.environmentIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentGroupTags(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		groupName     string
		tagIds        []int
		mockError     error
		expectedError bool
	}{
		{
			name:      "successful update",
			groupID:   1,
			groupName: "test-group",
			tagIds:    []int{1, 2, 3},
		},
		{
			name:          "update error",
			groupID:       1,
			groupName:     "error-group",
			tagIds:        []int{1},
			mockError:     errors.New("failed to update group tags"),
			expectedError: true,
		},
		{
			name:      "empty tags",
			groupID:   1,
			groupName: "empty-group",
			tagIds:    []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEdgeGroup", int64(tt.groupID), tt.groupName, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentGroupTags(tt.groupID, tt.groupName, tt.tagIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
