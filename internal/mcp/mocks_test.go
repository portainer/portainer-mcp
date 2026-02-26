package mcp

import (
	"net/http"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/mock"
)

// Mock Implementation Patterns:
//
// This file contains mock implementations of the PortainerClient interface.
// The following patterns are used throughout the mocks:
//
// 1. Methods returning (T, error):
//    - Uses m.Called() to record the method call and get mock behavior
//    - Includes nil check on first return value to avoid type assertion panics
//    - Example:
//      func (m *Mock) Method() (T, error) {
//          args := m.Called()
//          if args.Get(0) == nil {
//              return nil, args.Error(1)
//          }
//          return args.Get(0).(T), args.Error(1)
//      }
//
// 2. Methods returning only error:
//    - Uses m.Called() with any parameters
//    - Returns only the error value
//    - Example:
//      func (m *Mock) Method(param string) error {
//          args := m.Called(param)
//          return args.Error(0)
//      }
//
// Usage in Tests:
//   mock := new(MockPortainerClient)
//   mock.On("MethodName").Return(expectedValue, nil)
//   result, err := mock.MethodName()
//   mock.AssertExpectations(t)

// MockPortainerClient is a mock implementation of the PortainerClient interface
type MockPortainerClient struct {
	mock.Mock
}

// Tag methods

func (m *MockPortainerClient) GetEnvironmentTags() ([]models.EnvironmentTag, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.EnvironmentTag), args.Error(1)
}

func (m *MockPortainerClient) CreateEnvironmentTag(name string) (int, error) {
	args := m.Called(name)
	return args.Int(0), args.Error(1)
}

// Environment methods

func (m *MockPortainerClient) GetEnvironments() ([]models.Environment, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Environment), args.Error(1)
}

func (m *MockPortainerClient) UpdateEnvironmentTags(id int, tagIds []int) error {
	args := m.Called(id, tagIds)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateEnvironmentUserAccesses(id int, userAccesses map[int]string) error {
	args := m.Called(id, userAccesses)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateEnvironmentTeamAccesses(id int, teamAccesses map[int]string) error {
	args := m.Called(id, teamAccesses)
	return args.Error(0)
}

// Environment Group methods

func (m *MockPortainerClient) GetEnvironmentGroups() ([]models.Group, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Group), args.Error(1)
}

func (m *MockPortainerClient) CreateEnvironmentGroup(name string, environmentIds []int) (int, error) {
	args := m.Called(name, environmentIds)
	return args.Int(0), args.Error(1)
}

func (m *MockPortainerClient) UpdateEnvironmentGroupName(id int, name string) error {
	args := m.Called(id, name)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateEnvironmentGroupEnvironments(id int, environmentIds []int) error {
	args := m.Called(id, environmentIds)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateEnvironmentGroupTags(id int, tagIds []int) error {
	args := m.Called(id, tagIds)
	return args.Error(0)
}

// Access Group methods

func (m *MockPortainerClient) GetAccessGroups() ([]models.AccessGroup, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AccessGroup), args.Error(1)
}

func (m *MockPortainerClient) CreateAccessGroup(name string, environmentIds []int) (int, error) {
	args := m.Called(name, environmentIds)
	return args.Int(0), args.Error(1)
}

func (m *MockPortainerClient) UpdateAccessGroupName(id int, name string) error {
	args := m.Called(id, name)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateAccessGroupUserAccesses(id int, userAccesses map[int]string) error {
	args := m.Called(id, userAccesses)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateAccessGroupTeamAccesses(id int, teamAccesses map[int]string) error {
	args := m.Called(id, teamAccesses)
	return args.Error(0)
}

func (m *MockPortainerClient) AddEnvironmentToAccessGroup(id int, environmentId int) error {
	args := m.Called(id, environmentId)
	return args.Error(0)
}

func (m *MockPortainerClient) RemoveEnvironmentFromAccessGroup(id int, environmentId int) error {
	args := m.Called(id, environmentId)
	return args.Error(0)
}

// Stack methods

func (m *MockPortainerClient) GetStacks() ([]models.Stack, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Stack), args.Error(1)
}

func (m *MockPortainerClient) GetStackFile(id int) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockPortainerClient) CreateStack(name string, file string, environmentGroupIds []int) (int, error) {
	args := m.Called(name, file, environmentGroupIds)
	return args.Int(0), args.Error(1)
}

func (m *MockPortainerClient) UpdateStack(id int, file string, environmentGroupIds []int) error {
	args := m.Called(id, file, environmentGroupIds)
	return args.Error(0)
}

// Team methods

func (m *MockPortainerClient) CreateTeam(name string) (int, error) {
	args := m.Called(name)
	return args.Int(0), args.Error(1)
}

func (m *MockPortainerClient) GetTeams() ([]models.Team, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Team), args.Error(1)
}

func (m *MockPortainerClient) UpdateTeamName(id int, name string) error {
	args := m.Called(id, name)
	return args.Error(0)
}

func (m *MockPortainerClient) UpdateTeamMembers(id int, userIds []int) error {
	args := m.Called(id, userIds)
	return args.Error(0)
}

// User methods

func (m *MockPortainerClient) GetUsers() ([]models.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockPortainerClient) UpdateUserRole(id int, role string) error {
	args := m.Called(id, role)
	return args.Error(0)
}

// Settings methods

func (m *MockPortainerClient) GetSettings() (models.PortainerSettings, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return models.PortainerSettings{}, args.Error(1)
	}
	return args.Get(0).(models.PortainerSettings), args.Error(1)
}

func (m *MockPortainerClient) GetVersion() (string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

// Docker Proxy methods
func (m *MockPortainerClient) ProxyDockerRequest(opts models.DockerProxyRequestOptions) (*http.Response, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// Kubernetes Proxy methods
func (m *MockPortainerClient) ProxyKubernetesRequest(opts models.KubernetesProxyRequestOptions) (*http.Response, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// Local Stack methods

func (m *MockPortainerClient) GetLocalStacks() ([]models.LocalStack, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.LocalStack), args.Error(1)
}

func (m *MockPortainerClient) GetLocalStackFile(id int) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockPortainerClient) CreateLocalStack(endpointId int, name, file string, env []models.LocalStackEnvVar) (int, error) {
	args := m.Called(endpointId, name, file, env)
	return args.Int(0), args.Error(1)
}

func (m *MockPortainerClient) UpdateLocalStack(id, endpointId int, file string, env []models.LocalStackEnvVar, prune, pullImage bool) error {
	args := m.Called(id, endpointId, file, env, prune, pullImage)
	return args.Error(0)
}

func (m *MockPortainerClient) StartLocalStack(id, endpointId int) error {
	args := m.Called(id, endpointId)
	return args.Error(0)
}

func (m *MockPortainerClient) StopLocalStack(id, endpointId int) error {
	args := m.Called(id, endpointId)
	return args.Error(0)
}

func (m *MockPortainerClient) DeleteLocalStack(id, endpointId int) error {
	args := m.Called(id, endpointId)
	return args.Error(0)
}
