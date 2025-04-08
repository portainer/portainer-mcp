package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name           string
		mockVersion    string
		mockError      error
		expectedResult string
		expectedError  bool
	}{
		{
			name:           "successful retrieval",
			mockVersion:    "2.19.0",
			mockError:      nil,
			expectedResult: "2.19.0",
			expectedError:  false,
		},
		{
			name:           "api error",
			mockVersion:    "",
			mockError:      fmt.Errorf("api error"),
			expectedResult: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("GetVersion").Return(tt.mockVersion, tt.mockError)

			client := &PortainerClient{
				cli: mockAPI,
			}

			version, err := client.GetVersion()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, "", version)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, version)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}
