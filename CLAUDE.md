# Portainer MCP Development Guide

## Build, Test & Run Commands
- Build: `make build`
- Run tests: `go test -v ./...`
- Run single test: `go test -v ./path/to/package -run TestName`
- Lint: `go vet ./...` and `golint ./...` (install golint if needed)
- Format code: `gofmt -s -w .`
- Run inspector: `make inspector`
- Build for specific platform: `make PLATFORM=<platform> ARCH=<arch> build`
- Integration tests: `make test-integration`
- Run all tests: `make test-all`

## Code Style Guidelines
- Use standard Go naming conventions: PascalCase for exported, camelCase for private
- Follow table-driven test pattern with descriptive test cases
- Error handling: return errors with context via `fmt.Errorf("failed to X: %w", err)`
- Imports: group standard library, external packages, and internal packages
- Function comments: document exported functions with Parameters/Returns sections
- Use functional options pattern for configurable clients
- Package structure: cmd/ for entry points, internal/ for implementation, pkg/ for reusable components
- Models belong in pkg/portainer/models, client implementations in pkg/portainer/client

## Design Documentation
- Design decisions are documented in individual files in `docs/design/` directory
- Follow the naming convention: `YYMMDD-N-short-description.md` where:
  - `YYMMDD` is the date (year-month-day)
  - `N` is a sequence number for that date
  - Example: `202505-1-feature-toggles.md`
- Use the standard template structure provided in `docs/design_summary.md`
- Add new decisions to the table in `docs/design_summary.md`
- Review existing decisions before making significant architectural changes

## Client and Model Guidelines

### Client Structure
1. **Raw Client** (`portainer/client-api-go/v2`)
   - Directly communicates with Portainer API
   - Used in integration tests for ground-truth comparisons
   - Works with raw models from `github.com/portainer/client-api-go/v2/pkg/models`

2. **Wrapper Client** (`pkg/portainer/client`)
   - Abstraction layer over the Raw Client
   - Simplifies interface for the MCP application
   - Handles data transformation between Raw and Local Models
   - Used by MCP server handlers

3. **Raw HTTP Client** (for local stacks, `pkg/portainer/client/local_stack.go`)
   - Direct HTTP requests via `apiRequest()` helper (not through the SDK)
   - Used because the SDK (`client-api-go`) has no regular/standalone stack API methods
   - Authenticates with `X-API-Key` header, talks to `/api/stacks/*` endpoints
   - Models defined locally in `pkg/portainer/models/stack.go` (LocalStack, RawLocalStack)
   - Tests use `httptest.NewServer` instead of mocking the SDK interface

### Model Structure
1. **Raw Models** (`portainer/client-api-go/v2/pkg/models`)
   - Direct mapping to Portainer API data structures
   - May contain fields not relevant to MCP
   - Prefix variables with `raw` (e.g., `rawSettings`, `rawEndpoint`)

2. **Local Models** (`pkg/portainer/models`)
   - Simplified structures tailored for the MCP application
   - Contain only relevant fields with convenient types
   - Define conversion functions to transform from Raw Models

### Import Conventions
```go
import (
    "github.com/portainer/portainer-mcp/pkg/portainer/models" // Default: models (Local MCP Models)
    apimodels "github.com/portainer/client-api-go/v2/pkg/models" // Alias: apimodels (Raw Client-API-Go Models)
)
```

### Testing Approach
- **Unit Tests**: Mock Raw Client interface, verify conversions and expected Local Model output
- **Integration Tests**: Call MCP handler and compare with ground-truth from Raw Client

## MCP Server Architecture

### Server Configuration
- Server is initialized in `cmd/portainer-mcp/mcp.go`
- Uses functional options pattern via `WithClient()` and `WithReadOnly()`
- Connects to Portainer API using token-based authentication
- Validates compatibility with specific Portainer version
- Loads tool definitions from YAML file

### Tool Definitions
- Tools are defined in `internal/tooldef/tools.yaml`
- File is embedded in binary at build time
- External file can override embedded definitions
- Version checking ensures compatibility
- Read-only mode restricts modification capabilities

### Handler Pattern
- Each tool has a corresponding handler in `internal/mcp/`
- Handlers follow ToolHandlerFunc signature
- Standard error handling with wrapped errors
- Parameter validation with required flag checks
- Response serialization to JSON

## Integration Testing Framework

### Test Environment Setup
- Uses Docker containers for Portainer instances
- `tests/integration/helpers/test_env.go` provides test environment utilities
- Creates isolated test environment for each test
- Configures both Raw Client and MCP Server for testing
- Automatically cleans up resources after tests

### Testing Conventions
- Tests verify both success and error conditions
- Use table-driven tests with descriptive case names
- Compare MCP handler results with direct API calls
- Validate correct error handling and parameter validation

## Version Compatibility

### Portainer Version Support
- Each release supports a specific Portainer version (defined in `server.go`)
- Version check at startup prevents compatibility issues
- Fail-fast approach with clear error messaging

### Tools File Versioning
- Strict versioning for tools.yaml file
- Version validation at startup
- Clear upgrade path for breaking changes

## Security Features

### Read-Only Mode
- Flag to enable read-only mode
- Only registers tools that don't modify resources
- Provides protection against accidental modifications
- Safe mode for monitoring and observation

### Error Handling
- Validate parameters before performing operations
- Proper error messages with context
- Fail-fast approach for invalid operations