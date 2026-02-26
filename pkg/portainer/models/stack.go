package models

import (
	"time"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
)

type Stack struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	CreatedAt           string `json:"created_at"`
	EnvironmentGroupIds []int  `json:"group_ids"`
}

func ConvertEdgeStackToStack(rawEdgeStack *apimodels.PortainereeEdgeStack) Stack {
	createdAt := time.Unix(rawEdgeStack.CreationDate, 0).Format(time.RFC3339)

	return Stack{
		ID:                  int(rawEdgeStack.ID),
		Name:                rawEdgeStack.Name,
		CreatedAt:           createdAt,
		EnvironmentGroupIds: utils.Int64ToIntSlice(rawEdgeStack.EdgeGroups),
	}
}

// LocalStackStatus represents the status of a local (non-edge) stack
type LocalStackStatus int

const (
	LocalStackStatusActive   LocalStackStatus = 1
	LocalStackStatusInactive LocalStackStatus = 2
)

func (s LocalStackStatus) String() string {
	switch s {
	case LocalStackStatusActive:
		return "active"
	case LocalStackStatusInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

// LocalStackType represents the type of a local stack
type LocalStackType int

const (
	LocalStackTypeSwarm   LocalStackType = 1
	LocalStackTypeCompose LocalStackType = 2
)

func (t LocalStackType) String() string {
	switch t {
	case LocalStackTypeSwarm:
		return "swarm"
	case LocalStackTypeCompose:
		return "compose"
	default:
		return "unknown"
	}
}

// LocalStackEnvVar represents an environment variable in a local stack
type LocalStackEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// LocalStack represents a regular (non-edge) Docker Compose or Swarm stack in Portainer
type LocalStack struct {
	ID            int              `json:"id"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	EndpointID    int              `json:"endpoint_id"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at,omitempty"`
	Env           []LocalStackEnvVar `json:"env,omitempty"`
}

// RawLocalStack is the raw API response structure for a local stack
type RawLocalStack struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	Type         int    `json:"Type"`
	Status       int    `json:"Status"`
	EndpointID   int    `json:"EndpointId"`
	CreationDate int64  `json:"CreationDate"`
	UpdateDate   int64  `json:"UpdateDate"`
	Env          []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"Env"`
}

// RawLocalStackFile is the raw API response for a stack file
type RawLocalStackFile struct {
	StackFileContent string `json:"StackFileContent"`
}

// ConvertRawLocalStackToLocalStack converts a raw API stack to a LocalStack model
func ConvertRawLocalStackToLocalStack(raw RawLocalStack) LocalStack {
	createdAt := ""
	if raw.CreationDate > 0 {
		createdAt = time.Unix(raw.CreationDate, 0).Format(time.RFC3339)
	}

	updatedAt := ""
	if raw.UpdateDate > 0 {
		updatedAt = time.Unix(raw.UpdateDate, 0).Format(time.RFC3339)
	}

	env := make([]LocalStackEnvVar, len(raw.Env))
	for i, e := range raw.Env {
		env[i] = LocalStackEnvVar{Name: e.Name, Value: e.Value}
	}

	return LocalStack{
		ID:         raw.ID,
		Name:       raw.Name,
		Type:       LocalStackType(raw.Type).String(),
		Status:     LocalStackStatus(raw.Status).String(),
		EndpointID: raw.EndpointID,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Env:        env,
	}
}
