# 202602-1: Local Stack Support via Raw HTTP Client

**Date**: 24/02/2026

### Context

The official Portainer MCP server (`portainer/portainer-mcp`) only supports Edge Stacks — stacks distributed via Edge Groups to edge environments. However, the most common stack deployment model in Portainer is regular/standalone Docker Compose stacks deployed directly on environments.

The official SDK (`portainer/client-api-go`) does not expose any API methods for regular stacks — it only contains `edge_stack.go`. As a result, the existing `listStacks`, `getStackFile`, `createStack`, and `updateStack` tools all operate on Edge Stacks and return errors (503) when Edge features are disabled on the Portainer instance.

Users who deploy standard Docker Compose stacks (the majority of non-Edge setups) have no way to manage them through the MCP tools.

### Decision

Add 7 new MCP tools for managing local (standalone) Docker Compose stacks using direct HTTP requests to the Portainer REST API, bypassing the SDK entirely.

**New tools:**
- `listLocalStacks` — List all local stacks across environments
- `getLocalStackFile` — Retrieve the docker-compose.yml content for a stack
- `createLocalStack` — Create a new standalone Docker Compose stack
- `updateLocalStack` — Update compose file and environment variables
- `startLocalStack` — Start a stopped stack
- `stopLocalStack` — Stop a running stack
- `deleteLocalStack` — Permanently remove a stack

### Rationale

1. **SDK limitation**: The `client-api-go` SDK has no regular stack methods. Waiting for upstream SDK changes would block functionality indefinitely.

2. **Raw HTTP approach**: Adding an `apiRequest()` helper method to the existing `PortainerClient` struct allows direct HTTP communication with the Portainer REST API (`/api/stacks/*`) while maintaining the same client abstraction.

3. **Coexistence with Edge Stacks**: The new local stack tools use distinct tool names (prefixed with "Local") and separate handler functions, avoiding any conflict with existing Edge Stack tools.

4. **URL scheme normalization**: The `serverURL` configuration may not include a scheme (e.g., `192.168.0.40:31015`). The raw HTTP client normalizes this by defaulting to `https://`, matching the SDK's internal behavior.

5. **Read-only mode support**: Local stack write tools (create, update, start, stop, delete) respect the existing `readOnly` flag and are only registered when the flag is not set.

### Implementation Details

**Architecture layers:**
```
MCP Tool Handler (internal/mcp/local_stack.go)
    ↓ calls
PortainerClient interface (internal/mcp/server.go)
    ↓ implemented by
Raw HTTP methods (pkg/portainer/client/local_stack.go)
    ↓ uses apiRequest() helper
Portainer REST API (/api/stacks/*)
```

**Files added:**
- `pkg/portainer/client/local_stack.go` — `apiRequest()` helper + 7 client methods
- `internal/mcp/local_stack.go` — 7 MCP handler functions + `parseEnvVars()` helper + `AddLocalStackFeatures()` registration

**Files modified:**
- `pkg/portainer/models/stack.go` — `LocalStack`, `RawLocalStack`, `LocalStackEnvVar` types, enums, conversion function
- `pkg/portainer/client/client.go` — Added `serverURL`, `token`, `httpCli` fields; URL scheme normalization
- `internal/mcp/schema.go` — 7 new tool name constants
- `internal/mcp/server.go` — 7 new method signatures in `PortainerClient` interface
- `cmd/portainer-mcp/mcp.go` — `AddLocalStackFeatures()` call in server initialization
- `internal/tooldef/tools.yaml` — 7 new tool definitions

**Test files added:**
- `pkg/portainer/client/local_stack_test.go` — HTTP client tests using `httptest.NewServer`
- `internal/mcp/local_stack_test.go` — MCP handler tests using `MockPortainerClient`
- `pkg/portainer/models/stack_test.go` — Model conversion and enum tests (appended to existing file)
- `internal/mcp/mocks_test.go` — 7 new mock methods (appended to existing file)

### Trade-offs

**Benefits:**
- Enables management of the most common Portainer stack type (standalone Docker Compose)
- Does not modify the SDK, avoiding upstream dependency issues
- Follows existing patterns (tool registration, read-only gating, error handling)
- Comprehensive test coverage at all three layers

**Challenges:**
- Raw HTTP client is a separate code path from the SDK-based client, requiring its own testing approach (`httptest.NewServer` vs SDK mocks)
- If the Portainer REST API changes the `/api/stacks/` endpoints, the raw HTTP methods must be updated manually (no SDK versioning protection)
- The `PortainerClient` struct now serves dual roles: SDK wrapper and raw HTTP client
