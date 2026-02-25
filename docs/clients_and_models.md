# Portainer MCP Client and Model Usage Guide

This document clarifies the different client implementations and model structures used within the `portainer-mcp` project to prevent confusion and aid development.

## Overview

The project interacts with the Portainer API using two main client layers and involves two primary sets of data models:

1.  **Raw Client & Models:** Provided by the `portainer/client-api-go` library.
2.  **Wrapper Client & Local Models:** Defined within `portainer-mcp/pkg/portainer/`.
3.  **Raw HTTP Client (Local Stacks):** Direct HTTP requests for APIs not covered by the SDK.

Understanding the distinction and interaction between these layers is crucial.

## Clients

### 1. Raw Client (`portainer/client-api-go/v2`)

*   **Package:** `github.com/portainer/client-api-go/v2`
*   **Role:** This is the underlying library that directly communicates with the Portainer API.
*   **Usage:** It's instantiated within the Wrapper Client. It's also often used directly within **integration tests** (`tests/integration/`) to fetch the ground-truth state from Portainer for comparison against the MCP handler's output.
*   **Models Used:** Interacts primarily with the Raw Models defined in `github.com/portainer/client-api-go/v2/pkg/models`.

### 2. Wrapper Client (`portainer-mcp/pkg/portainer/client`)

*   **Package:** `github.com/portainer/portainer-mcp/pkg/portainer/client`
*   **Role:** This client acts as an **abstraction layer** on top of the Raw Client. Its primary purposes are:
    *   To simplify the interface exposed to the rest of the `portainer-mcp` application (specifically the MCP server handlers in `internal/mcp/`).
    *   To perform necessary **data transformations**, converting Raw Models from the API into the simpler, tailored Local Models.
    *   To encapsulate common logic or error handling related to Portainer API interactions.
*   **Usage:** This is the client used by the **MCP server handlers** (`internal/mcp/server.go` instantiates it and passes it to handlers).
*   **Models Used:** Takes Raw Models as input from the Raw Client but typically **returns Local Models** (`portainer-mcp/pkg/portainer/models`) after performing conversions.

### 3. Raw HTTP Client (Local Stacks) (`portainer-mcp/pkg/portainer/client/local_stack.go`)

*   **Role:** Provides direct HTTP access to Portainer REST API endpoints that are **not exposed by the SDK** (`client-api-go`). Currently used for regular/standalone Docker Compose stacks (as opposed to Edge Stacks).
*   **Why:** The official `portainer/client-api-go` SDK only contains Edge Stack methods (`edge_stack.go`). Regular stack endpoints (`/api/stacks/*`) are not available in the SDK, so direct HTTP requests are necessary.
*   **Implementation:**
    *   Uses `apiRequest()` helper method on `PortainerClient` that constructs HTTP requests with the `X-API-Key` authentication header.
    *   The `rawHTTPClient` struct (embedded in `PortainerClient` as `rawCli`) stores `serverURL`, `token`, and `httpCli` fields for this purpose.
    *   URL scheme normalization ensures `https://` is used by default when no scheme is provided.
*   **Models Used:** Defines its own `RawLocalStack` / `LocalStack` types in `pkg/portainer/models/stack.go` with a `ConvertRawLocalStackToLocalStack()` conversion function.
*   **Testing:** Uses `httptest.NewServer` to mock the Portainer REST API at the HTTP level, rather than mocking the SDK interface.

## Models

### 1. Raw Models (`portainer/client-api-go/v2/pkg/models`)

*   **Package:** `github.com/portainer/client-api-go/v2/pkg/models`
*   **Role:** These structs directly map to the data structures returned by the Portainer API.
*   **Characteristics:** Can be complex, may contain fields not relevant to MCP, and might use types (like numeric enums) that are less convenient for MCP's purposes.
*   **Examples:** `models.PortainereeSettings`, `models.PortainereeEndpoint`.
*   **Usage:** Returned by the Raw Client, used as input to the conversion functions within the Wrapper Client / Local Models package.
*   **Naming Convention:** To improve clarity, variables holding instances of these Raw Models are typically prefixed with `raw` (e.g., `rawSettings`, `rawEndpoint`).

### 2. Local Models (`portainer-mcp/pkg/portainer/models`)

*   **Package:** `github.com/portainer/portainer-mcp/pkg/portainer/models`
*   **Role:** These are simplified, tailored structs designed specifically for use within the `portainer-mcp` application and for exposure via the MCP tools.
*   **Characteristics:** Simpler structure, contain only relevant fields, often use more convenient types (like string enums).
*   **Examples:** `models.PortainerSettings`, `models.Environment`, `models.EnvironmentTag`.
*   **Usage:** Returned by the Wrapper Client, used within MCP server handlers, and ultimately determine the structure of data returned by MCP tools.

### 3. Conversion Functions

*   **Location:** Typically reside within `portainer-mcp/pkg/portainer/models`.
*   **Role:** Bridge the gap between Raw Models and Local Models.
*   **Examples:** `ConvertSettingsToPortainerSettings`, `ConvertEndpointToEnvironment`.
*   **Usage:** Called by the Wrapper Client methods to transform data before returning it. The function parameters accepting Raw Models typically follow the `raw` prefix naming convention (e.g., `func ConvertSettingsToPortainerSettings(rawSettings *apimodels.PortainereeSettings)`).

## Typical Workflow Example (`GetSettings`)

1.  **MCP Handler (`internal/mcp/settings.go`)**: Receives a tool call.
2.  Calls `s.cli.GetSettings()`. Here, `s.cli` is an instance of the **Wrapper Client** (`PortainerClient`).
3.  **Wrapper Client (`pkg/portainer/client/settings.go`)**: Its `GetSettings` method is executed.
4.  Calls the **Raw Client**'s `GetSettings` method (e.g., `c.cli.GetSettings()`).
5.  Raw Client interacts with the Portainer API and returns a **Raw Model** (`*portainermodels.PortainereeSettings`).
6.  Wrapper Client calls the **Conversion Function** (`models.ConvertSettingsToPortainerSettings`) with the Raw Model.
7.  Conversion Function returns a **Local Model** (`models.PortainerSettings`).
8.  Wrapper Client returns the Local Model to the MCP Handler.
9.  MCP Handler marshals the **Local Model** (`models.PortainerSettings`) into JSON and returns it as the tool result.

## Typical Workflow Example (Local Stacks — `GetLocalStacks`)

Unlike the SDK-based workflow above, local stack operations use direct HTTP requests:

1.  **MCP Handler (`internal/mcp/local_stack.go`)**: Receives a tool call for `listLocalStacks`.
2.  Calls `s.cli.GetLocalStacks()`. Here, `s.cli` is the **Wrapper Client** (`PortainerClient`).
3.  **Wrapper Client (`pkg/portainer/client/local_stack.go`)**: Its `GetLocalStacks` method uses `apiRequest()` to make a direct `GET /api/stacks` HTTP request.
4.  The Portainer REST API returns a JSON array of raw stack objects.
5.  The method decodes the JSON into `[]models.RawLocalStack` and calls `models.ConvertRawLocalStackToLocalStack()` for each entry.
6.  Returns `[]models.LocalStack` (Local Models) to the MCP Handler.
7.  MCP Handler marshals the Local Models into JSON and returns them as the tool result.

## Import Conventions

To improve clarity, especially in files where both model types might appear (like tests), consider using consistent import aliases. Leaving the local `portainer-mcp/pkg/portainer/models` package as the default `models` and aliasing the external library is recommended:

```go
import (
    "github.com/portainer/portainer-mcp/pkg/portainer/models" // Default: models (Local MCP Models)
    apimodels "github.com/portainer/client-api-go/v2/pkg/models"      // Alias: apimodels (Raw Client-API-Go Models)
)
```

This approach keeps code cleaner for the more frequently used local models while clearly indicating when the raw API models are involved.

## Testing Implications

*   **Unit Tests** (like `pkg/portainer/client/settings_test.go`): Should mock the Raw Client interface and verify that the Wrapper Client correctly calls the Raw Client and performs the necessary conversions, returning the expected Local Model.
*   **Unit Tests for Local Stacks** (like `pkg/portainer/client/local_stack_test.go`): Use `httptest.NewServer` to mock the Portainer REST API at the HTTP level, since these methods bypass the SDK. Create a test client with `serverURL` pointing to the test server.
*   **Integration Tests** (like `tests/integration/settings_test.go`): 
    *   Call the MCP handler, which uses the Wrapper Client internally and returns JSON representing a Local Model.
    *   Often need to *also* call the Raw Client directly to get the ground-truth state from the live Portainer instance (variables holding this state should follow the `raw` prefix convention, e.g., `rawEndpoint`).
    *   May need to manually apply the same Conversion Function to the Raw Model obtained from the Raw Client to create an expected Local Model for comparison against the handler's result.

By understanding these distinct layers and their interactions, development and testing within `portainer-mcp` should be clearer. 