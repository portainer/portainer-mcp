# Portainer MCP Development Guide

## Build, Test & Run Commands
- Build: `make build`
- Run tests: `go test -v ./...`
- Run single test: `go test -v ./path/to/package -run TestName`
- Lint: `go vet ./...` and `golint ./...` (install golint if needed)
- Format code: `gofmt -s -w .`
- Run inspector: `make inspector`
- Build for specific platform: `make PLATFORM=<platform> ARCH=<arch> build`

## Code Style Guidelines
- Use standard Go naming conventions: PascalCase for exported, camelCase for private
- Follow table-driven test pattern with descriptive test cases
- Error handling: return errors with context via `fmt.Errorf("failed to X: %w", err)`
- Imports: group standard library, external packages, and internal packages
- Function comments: document exported functions with Parameters/Returns sections
- Use functional options pattern for configurable clients
- Package structure: cmd/ for entry points, internal/ for implementation, pkg/ for reusable components
- Models belong in pkg/portainer/models, client implementations in pkg/portainer/client