package client

import (
	"github.com/portainer/client-api-go/v2/client"
)

type PortainerClient struct {
	sdkCli *client.PortainerClient
}

func NewPortainerClient(serverURL string, token string) *PortainerClient {
	return &PortainerClient{
		sdkCli: client.NewPortainerClient(serverURL, token, client.WithSkipTLSVerify(true)),
	}
}
