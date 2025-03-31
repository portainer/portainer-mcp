package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPortainerClient(t *testing.T) {
	tests := []struct {
		name        string
		serverURL   string
		token       string
		opts        []ClientOption
		expectError bool
	}{
		{
			name:      "creates client with default options",
			serverURL: "https://portainer.example.com",
			token:     "test-token",
			opts:      nil,
		},
		{
			name:      "creates client with skip TLS verify",
			serverURL: "https://portainer.example.com",
			token:     "test-token",
			opts:      []ClientOption{WithSkipTLSVerify(true)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client
			c := NewPortainerClient(tt.serverURL, tt.token, tt.opts...)

			// Assert client was created
			assert.NotNil(t, c)
			assert.NotNil(t, c.cli)
		})
	}
}

func TestWithSkipTLSVerify(t *testing.T) {
	tests := []struct {
		name     string
		skip     bool
		expected bool
	}{
		{
			name:     "enables TLS verification skip",
			skip:     true,
			expected: true,
		},
		{
			name:     "disables TLS verification skip",
			skip:     false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create options
			options := &clientOptions{}
			opt := WithSkipTLSVerify(tt.skip)
			opt(options)

			// Assert option was applied correctly
			assert.Equal(t, tt.expected, options.skipTLSVerify)
		})
	}
}
