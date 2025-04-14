package client

import (
	"io"
	"net/http"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/mock"
)

// Mock Implementation Patterns:
//
// This file contains mock implementations of the PortainerAPIClient interface.
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
// 3. Methods with primitive return types:
//    - Uses type-specific getters (e.g., Int64, String)
//    - Example:
//      func (m *Mock) Method() (int64, error) {
//          args := m.Called()
//          return args.Get(0).(int64), args.Error(1)
//      }
//
// Usage in Tests:
//   mock := new(MockPortainerAPI)
//   mock.On("MethodName").Return(expectedValue, nil)
//   result, err := mock.MethodName()
//   mock.AssertExpectations(t)

// MockPortainerAPI is a mock of the PortainerAPIClient interface
type MockPortainerAPI struct {
	mock.Mock
}

// ListEdgeGroups mocks the ListEdgeGroups method
func (m *MockPortainerAPI) ListEdgeGroups() ([]*apimodels.EdgegroupsDecoratedEdgeGroup, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.EdgegroupsDecoratedEdgeGroup), args.Error(1)
}

// CreateEdgeGroup mocks the CreateEdgeGroup method
func (m *MockPortainerAPI) CreateEdgeGroup(name string, environmentIds []int64) (int64, error) {
	args := m.Called(name, environmentIds)
	return args.Get(0).(int64), args.Error(1)
}

// UpdateEdgeGroup mocks the UpdateEdgeGroup method
func (m *MockPortainerAPI) UpdateEdgeGroup(id int64, name string, environmentIds *[]int64, tagIds *[]int64) error {
	args := m.Called(id, name, environmentIds, tagIds)
	return args.Error(0)
}

// ListEdgeStacks mocks the ListEdgeStacks method
func (m *MockPortainerAPI) ListEdgeStacks() ([]*apimodels.PortainereeEdgeStack, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainereeEdgeStack), args.Error(1)
}

// CreateEdgeStack mocks the CreateEdgeStack method
func (m *MockPortainerAPI) CreateEdgeStack(name string, file string, environmentGroupIds []int64) (int64, error) {
	args := m.Called(name, file, environmentGroupIds)
	return args.Get(0).(int64), args.Error(1)
}

// UpdateEdgeStack mocks the UpdateEdgeStack method
func (m *MockPortainerAPI) UpdateEdgeStack(id int64, file string, environmentGroupIds []int64) error {
	args := m.Called(id, file, environmentGroupIds)
	return args.Error(0)
}

// GetEdgeStackFile mocks the GetEdgeStackFile method
func (m *MockPortainerAPI) GetEdgeStackFile(id int64) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

// ListEndpointGroups mocks the ListEndpointGroups method
func (m *MockPortainerAPI) ListEndpointGroups() ([]*apimodels.PortainerEndpointGroup, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainerEndpointGroup), args.Error(1)
}

// CreateEndpointGroup mocks the CreateEndpointGroup method
func (m *MockPortainerAPI) CreateEndpointGroup(name string, associatedEndpoints []int64) (int64, error) {
	args := m.Called(name, associatedEndpoints)
	return args.Get(0).(int64), args.Error(1)
}

// UpdateEndpointGroup mocks the UpdateEndpointGroup method
func (m *MockPortainerAPI) UpdateEndpointGroup(id int64, name *string, userAccesses *map[int64]string, teamAccesses *map[int64]string) error {
	args := m.Called(id, name, userAccesses, teamAccesses)
	return args.Error(0)
}

// AddEnvironmentToEndpointGroup mocks the AddEnvironmentToEndpointGroup method
func (m *MockPortainerAPI) AddEnvironmentToEndpointGroup(groupId int64, environmentId int64) error {
	args := m.Called(groupId, environmentId)
	return args.Error(0)
}

// RemoveEnvironmentFromEndpointGroup mocks the RemoveEnvironmentFromEndpointGroup method
func (m *MockPortainerAPI) RemoveEnvironmentFromEndpointGroup(groupId int64, environmentId int64) error {
	args := m.Called(groupId, environmentId)
	return args.Error(0)
}

// ListEndpoints mocks the ListEndpoints method
func (m *MockPortainerAPI) ListEndpoints() ([]*apimodels.PortainereeEndpoint, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainereeEndpoint), args.Error(1)
}

// GetEndpoint mocks the GetEndpoint method
func (m *MockPortainerAPI) GetEndpoint(id int64) (*apimodels.PortainereeEndpoint, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apimodels.PortainereeEndpoint), args.Error(1)
}

// UpdateEndpoint mocks the UpdateEndpoint method
func (m *MockPortainerAPI) UpdateEndpoint(id int64, tagIds *[]int64, userAccesses *map[int64]string, teamAccesses *map[int64]string) error {
	args := m.Called(id, tagIds, userAccesses, teamAccesses)
	return args.Error(0)
}

// GetSettings mocks the GetSettings method
func (m *MockPortainerAPI) GetSettings() (*apimodels.PortainereeSettings, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apimodels.PortainereeSettings), args.Error(1)
}

// ListTags mocks the ListTags method
func (m *MockPortainerAPI) ListTags() ([]*apimodels.PortainerTag, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainerTag), args.Error(1)
}

// CreateTag mocks the CreateTag method
func (m *MockPortainerAPI) CreateTag(name string) (int64, error) {
	args := m.Called(name)
	return args.Get(0).(int64), args.Error(1)
}

// ListTeams mocks the ListTeams method
func (m *MockPortainerAPI) ListTeams() ([]*apimodels.PortainerTeam, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainerTeam), args.Error(1)
}

// ListTeamMemberships mocks the ListTeamMemberships method
func (m *MockPortainerAPI) ListTeamMemberships() ([]*apimodels.PortainerTeamMembership, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainerTeamMembership), args.Error(1)
}

// CreateTeam mocks the CreateTeam method
func (m *MockPortainerAPI) CreateTeam(name string) (int64, error) {
	args := m.Called(name)
	return args.Get(0).(int64), args.Error(1)
}

// UpdateTeamName mocks the UpdateTeamName method
func (m *MockPortainerAPI) UpdateTeamName(id int, name string) error {
	args := m.Called(id, name)
	return args.Error(0)
}

// DeleteTeamMembership mocks the DeleteTeamMembership method
func (m *MockPortainerAPI) DeleteTeamMembership(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// CreateTeamMembership mocks the CreateTeamMembership method
func (m *MockPortainerAPI) CreateTeamMembership(teamId int, userId int) error {
	args := m.Called(teamId, userId)
	return args.Error(0)
}

// ListUsers mocks the ListUsers method
func (m *MockPortainerAPI) ListUsers() ([]*apimodels.PortainereeUser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apimodels.PortainereeUser), args.Error(1)
}

// UpdateUserRole mocks the UpdateUserRole method
func (m *MockPortainerAPI) UpdateUserRole(id int, role int64) error {
	args := m.Called(id, role)
	return args.Error(0)
}

// GetVersion mocks the GetVersion method
func (m *MockPortainerAPI) GetVersion() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// ProxyDockerRequest mocks the ProxyDockerRequest method
func (m *MockPortainerAPI) ProxyDockerRequest(environmentId int, dockerAPIPath string, method string, body io.Reader) (*http.Response, error) {
	args := m.Called(environmentId, dockerAPIPath, method, body)
	return args.Get(0).(*http.Response), args.Error(1)
}
