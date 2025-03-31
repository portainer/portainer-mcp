package client

import (
	"errors"
	"testing"
	"time"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/pkg/portainer/utils"
	sdkmodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetStacks(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name          string
		mockStacks    []*sdkmodels.PortainereeEdgeStack
		mockError     error
		expected      []models.Stack
		expectedError bool
	}{
		{
			name: "successful retrieval",
			mockStacks: []*sdkmodels.PortainereeEdgeStack{
				{
					ID:           1,
					Name:         "stack1",
					CreationDate: now,
					EdgeGroups:   []int64{1, 2},
				},
				{
					ID:           2,
					Name:         "stack2",
					CreationDate: now,
					EdgeGroups:   []int64{3},
				},
			},
			expected: []models.Stack{
				{
					ID:                  1,
					Name:                "stack1",
					CreatedAt:           time.Unix(now, 0).Format(time.RFC3339),
					EnvironmentGroupIds: []int{1, 2},
				},
				{
					ID:                  2,
					Name:                "stack2",
					CreatedAt:           time.Unix(now, 0).Format(time.RFC3339),
					EnvironmentGroupIds: []int{3},
				},
			},
		},
		{
			name:       "empty stacks",
			mockStacks: []*sdkmodels.PortainereeEdgeStack{},
			expected:   []models.Stack{},
		},
		{
			name:          "list error",
			mockError:     errors.New("failed to list stacks"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListEdgeStacks").Return(tt.mockStacks, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			stacks, err := client.GetStacks()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, stacks)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestGetStackFile(t *testing.T) {
	tests := []struct {
		name          string
		stackID       int
		mockFile      string
		mockError     error
		expected      string
		expectedError bool
	}{
		{
			name:     "successful retrieval",
			stackID:  1,
			mockFile: "version: '3'\nservices:\n  web:\n    image: nginx",
			expected: "version: '3'\nservices:\n  web:\n    image: nginx",
		},
		{
			name:          "get file error",
			stackID:       2,
			mockError:     errors.New("failed to get stack file"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("GetEdgeStackFile", int64(tt.stackID)).Return(tt.mockFile, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			file, err := client.GetStackFile(tt.stackID)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, file)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestCreateStack(t *testing.T) {
	tests := []struct {
		name                string
		stackName           string
		stackFile           string
		environmentGroupIds []int
		mockID              int64
		mockError           error
		expected            int
		expectedError       bool
	}{
		{
			name:                "successful creation",
			stackName:           "test-stack",
			stackFile:           "version: '3'\nservices:\n  web:\n    image: nginx",
			environmentGroupIds: []int{1, 2},
			mockID:              1,
			expected:            1,
		},
		{
			name:                "create error",
			stackName:           "test-stack",
			stackFile:           "version: '3'\nservices:\n  web:\n    image: nginx",
			environmentGroupIds: []int{1},
			mockError:           errors.New("failed to create stack"),
			expectedError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("CreateEdgeStack", tt.stackName, tt.stackFile, utils.IntToInt64Slice(tt.environmentGroupIds)).Return(tt.mockID, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			id, err := client.CreateStack(tt.stackName, tt.stackFile, tt.environmentGroupIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, id)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateStack(t *testing.T) {
	tests := []struct {
		name                string
		stackID             int
		stackFile           string
		environmentGroupIds []int
		mockError           error
		expectedError       bool
	}{
		{
			name:                "successful update",
			stackID:             1,
			stackFile:           "version: '3'\nservices:\n  web:\n    image: nginx:latest",
			environmentGroupIds: []int{1, 2},
		},
		{
			name:                "update error",
			stackID:             2,
			stackFile:           "version: '3'\nservices:\n  web:\n    image: nginx:latest",
			environmentGroupIds: []int{1},
			mockError:           errors.New("failed to update stack"),
			expectedError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEdgeStack", int64(tt.stackID), tt.stackFile, utils.IntToInt64Slice(tt.environmentGroupIds)).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateStack(tt.stackID, tt.stackFile, tt.environmentGroupIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
