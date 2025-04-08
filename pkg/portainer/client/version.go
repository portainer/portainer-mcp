package client

import "fmt"

func (c *PortainerClient) GetVersion() (string, error) {
	version, err := c.cli.GetVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	return version, nil
}
