package containers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/portainer/client-api-go/v2/pkg/client"
	"github.com/portainer/client-api-go/v2/pkg/client/auth"
	"github.com/portainer/client-api-go/v2/pkg/client/users"
	"github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPortainerImage = "portainer/portainer-ee:" + mcp.SupportedPortainerVersion
	defaultAPIPortTCP     = "9443/tcp"
	adminPassword         = "$2y$05$CiHrhW6R6whDVlu7Wdgl0eccb3rg1NWl/mMiO93vQiRIF1SHNFRsS" // Bcrypt hash of "adminpassword123"
	// Timeout for the container to start and be ready to use
	startupTimeout = time.Second * 5
	// Timeout for the requests to the API
	requestTimeout = time.Second * 3
)

// PortainerContainer represents a Portainer container for testing
type PortainerContainer struct {
	testcontainers.Container
	APIPort  nat.Port
	APIHost  string
	apiToken string
}

// portainerContainerConfig holds the configuration for creating a Portainer container
type portainerContainerConfig struct {
	Image            string
	BindDockerSocket bool
}

// PortainerContainerOption defines a function type for applying options to Portainer container configuration
type PortainerContainerOption func(*portainerContainerConfig)

// WithImage sets a custom Portainer image
func WithImage(image string) PortainerContainerOption {
	return func(cfg *portainerContainerConfig) {
		cfg.Image = image
	}
}

// WithDockerSocketBind configures the container to bind mount the Docker socket
func WithDockerSocketBind(bind bool) PortainerContainerOption {
	return func(cfg *portainerContainerConfig) {
		cfg.BindDockerSocket = bind
	}
}

// NewPortainerContainer creates and starts a new Portainer container with the specified options
func NewPortainerContainer(ctx context.Context, opts ...PortainerContainerOption) (*PortainerContainer, error) {
	// Default configuration
	cfg := &portainerContainerConfig{
		Image:            defaultPortainerImage,
		BindDockerSocket: false,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(cfg)
	}

	// Container request configuration
	req := testcontainers.ContainerRequest{
		Image:        cfg.Image,
		ExposedPorts: []string{defaultAPIPortTCP},
		WaitingFor: wait.ForAll(
			// Wait for the HTTPS server to start
			wait.ForLog("starting HTTPS server").
				WithStartupTimeout(startupTimeout),
			// Then wait for the API to be responsive
			wait.ForHTTP("/api/system/status").
				WithTLS(true, nil).
				WithAllowInsecure(true).
				WithPort(defaultAPIPortTCP).
				WithStatusCodeMatcher(
					func(status int) bool {
						return status == http.StatusOK
					},
				).
				WithStartupTimeout(startupTimeout),
		),
		Cmd: []string{
			"--admin-password",
			adminPassword,
			"--log-level",
			"DEBUG",
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			if cfg.BindDockerSocket {
				hostConfig.Binds = append(hostConfig.Binds, "/var/run/docker.sock:/var/run/docker.sock")
			}
		},
	}

	// Create and start the container
	cntr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Portainer container: %w", err)
	}

	// Get the host and port mapping
	host, err := cntr.Host(ctx)
	if err != nil {
		cntr.Terminate(ctx) // Clean up if we fail post-start
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := cntr.MappedPort(ctx, nat.Port(defaultAPIPortTCP))
	if err != nil {
		cntr.Terminate(ctx) // Clean up if we fail post-start
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	pc := &PortainerContainer{
		Container: cntr,
		APIPort:   mappedPort,
		APIHost:   host,
	}

	// Register API token after successful container start and port mapping
	if err := pc.registerAPIToken(); err != nil {
		// Attempt to clean up the container if token registration fails
		cntr.Terminate(ctx)
		return nil, fmt.Errorf("failed to register API token: %w", err)
	}

	return pc, nil
}

// GetAPIBaseURL returns the base URL for the Portainer API
func (pc *PortainerContainer) GetAPIBaseURL() string {
	return fmt.Sprintf("https://%s:%s", pc.APIHost, pc.APIPort.Port())
}

// GetHostAndPort returns the host and port for the Portainer API
func (pc *PortainerContainer) GetHostAndPort() (string, string) {
	return pc.APIHost, pc.APIPort.Port()
}

func (pc *PortainerContainer) GetAPIToken() string {
	return pc.apiToken
}

// registerAPIToken registers an API token for the admin user
func (pc *PortainerContainer) registerAPIToken() error {
	// SDK implementation kept as reference - doesn't work currently because of an issue with the client-api-go
	// See: https://github.com/portainer/portainer-suite/pull/604
	// Once this PR is merged and a new version of the client-api-go is released, we can use it again
	transport := httptransport.New(
		fmt.Sprintf("%s:%s", pc.APIHost, pc.APIPort.Port()),
		"/api",
		[]string{"https"},
	)

	transport.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	portainerClient := client.New(transport, strfmt.Default)

	username := "admin"
	password := "adminpassword123"
	params := auth.NewAuthenticateUserParams().WithBody(&models.AuthAuthenticatePayload{
		Username: &username,
		Password: &password,
	})

	authResp, err := portainerClient.Auth.AuthenticateUser(params)
	if err != nil {
		return fmt.Errorf("failed to authenticate user: %w", err)
	}

	token := authResp.Payload.Jwt

	// Setup JWT authentication
	jwtAuth := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		return r.SetHeaderParam("Authorization", fmt.Sprintf("Bearer %s", token))
	})
	transport.DefaultAuthentication = jwtAuth

	description := "test-api-key"
	createTokenParams := users.NewUserGenerateAPIKeyParams().WithID(1).WithBody(&models.UsersUserAccessTokenCreatePayload{
		Description: &description,
		Password:    &password,
	})

	createTokenResp, err := portainerClient.Users.UserGenerateAPIKey(createTokenParams, nil)
	// Because of the issue with the client-api-go, this will return an error even though the API key is created
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	pc.apiToken = createTokenResp.Payload.RawAPIKey

	// Direct HTTP implementation
	// alternative to the SDK implementation above
	// httpClient := &http.Client{
	// 	Timeout: requestTimeout,
	// 	Transport: &http.Transport{
	// 		TLSClientConfig: &tls.Config{
	// 			InsecureSkipVerify: true,
	// 		},
	// 	},
	// }

	// baseURL := pc.GetAPIBaseURL()

	// // Step 1: Authenticate admin user
	// authPayload := map[string]string{
	// 	"username": "admin",
	// 	"password": "adminpassword123",
	// }
	// authBody, err := json.Marshal(authPayload)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal auth payload: %w", err)
	// }

	// authReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/auth", bytes.NewBuffer(authBody))
	// if err != nil {
	// 	return fmt.Errorf("failed to create auth request: %w", err)
	// }
	// authReq.Header.Set("Content-Type", "application/json")

	// authResp, err := httpClient.Do(authReq)
	// if err != nil {
	// 	return fmt.Errorf("failed to send auth request: %w", err)
	// }
	// defer authResp.Body.Close()

	// if authResp.StatusCode != http.StatusOK {
	// 	body, _ := io.ReadAll(authResp.Body)
	// 	return fmt.Errorf("auth request failed with status %d: %s", authResp.StatusCode, string(body))
	// }

	// var authResult struct {
	// 	Jwt string `json:"jwt"`
	// }
	// if err := json.NewDecoder(authResp.Body).Decode(&authResult); err != nil {
	// 	return fmt.Errorf("failed to decode auth response: %w", err)
	// }

	// // Step 2: Generate API key
	// apiKeyPayload := map[string]string{
	// 	"description": "test-api-key",
	// 	"password":    "adminpassword123",
	// }
	// apiKeyBody, err := json.Marshal(apiKeyPayload)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal API key payload: %w", err)
	// }

	// apiKeyReq, err := http.NewRequest(http.MethodPost, baseURL+"/api/users/1/tokens", bytes.NewBuffer(apiKeyBody))
	// if err != nil {
	// 	return fmt.Errorf("failed to create API key request: %w", err)
	// }
	// apiKeyReq.Header.Set("Content-Type", "application/json")
	// apiKeyReq.Header.Set("Authorization", "Bearer "+authResult.Jwt)

	// apiKeyResp, err := httpClient.Do(apiKeyReq)
	// if err != nil {
	// 	return fmt.Errorf("failed to send API key request: %w", err)
	// }
	// defer apiKeyResp.Body.Close()

	// if apiKeyResp.StatusCode != http.StatusCreated && apiKeyResp.StatusCode != http.StatusOK {
	// 	body, _ := io.ReadAll(apiKeyResp.Body)
	// 	return fmt.Errorf("API key request failed with status %d: %s", apiKeyResp.StatusCode, string(body))
	// }

	// var apiKeyResult struct {
	// 	RawAPIKey string `json:"rawAPIKey"`
	// }
	// if err := json.NewDecoder(apiKeyResp.Body).Decode(&apiKeyResult); err != nil {
	// 	return fmt.Errorf("failed to decode API key response: %w", err)
	// }

	// pc.apiToken = apiKeyResult.RawAPIKey
	return nil
}
