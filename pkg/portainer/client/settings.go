package client

import (
	"fmt"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
)

func (c *PortainerClient) GetSettings() (models.PortainerSettings, error) {
	settings, err := c.cli.GetSettings()
	if err != nil {
		return models.PortainerSettings{}, fmt.Errorf("failed to get settings: %w", err)
	}

	return models.ConvertSettingsToPortainerSettings(settings), nil
}
