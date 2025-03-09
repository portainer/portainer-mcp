package mcp

import (
	"github.com/deviantony/mcp-go/pkg/portainer/client"
	"github.com/mark3labs/mcp-go/server"
)

type PortainerMCPServer struct {
	srv *server.MCPServer
	cli *client.PortainerClient
}

func NewPortainerMCPServer(serverURL, token string) *PortainerMCPServer {
	return &PortainerMCPServer{
		srv: server.NewMCPServer(
			"Portainer MCP Server",
			"0.1.0",
			server.WithResourceCapabilities(true, true),
			server.WithLogging(),
		),
		cli: client.NewPortainerClient(serverURL, token, client.WithSkipTLSVerify(true)),
	}
}

func (s *PortainerMCPServer) Start() error {
	return server.ServeStdio(s.srv)
}
