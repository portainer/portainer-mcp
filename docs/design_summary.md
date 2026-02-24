# Design Documentation Summary

This document provides a summary of key design decisions for the Portainer MCP project. Each decision is documented in detail in its own file.

## Design Decisions

| ID | Title | Date | Description |
|----|-------|------|-------------|
| [202503-1](design/202503-1-external-tools-file.md) | Using an external tools file for tool definition | 29/03/2025 | Externalizes tool definitions into a YAML file for improved maintainability |
| [202503-2](design/202503-2-tools-vs-mcp-resources.md) | Using tools to get resources instead of MCP resources | 29/03/2025 | Prefers tool-based resource access over MCP resources for better model control |
| [202503-3](design/202503-3-specific-update-tools.md) | Specific tool for updates instead of a single update tool | 29/03/2025 | Splits update operations into specific tools for clearer parameter handling |
| [202504-1](design/202504-1-embedded-tools-yaml.md) | Embedding tools.yaml in the binary | 08/04/2025 | Embeds the tools configuration file in the binary for simplified distribution |
| [202504-2](design/202504-2-tools-yaml-versioning.md) | Strict versioning for tools.yaml file | 08/04/2025 | Implements versioning for tools.yaml to prevent compatibility issues |
| [202504-3](design/202504-3-portainer-version-compatibility.md) | Pinning compatibility to a specific Portainer version | 08/04/2025 | Binds each release to a specific Portainer version for guaranteed compatibility |
| [202504-4](design/202504-4-read-only-mode.md) | Read-only mode for enhanced security | 09/04/2025 | Provides a read-only mode to restrict modification capabilities for security |
| [202602-1](design/202602-1-local-stack-support.md) | Local stack support via raw HTTP client | 24/02/2026 | Adds standalone Docker Compose stack management using direct HTTP requests to the Portainer REST API |

## How to Add a New Design Decision

1. Create a new file in the `docs/design/` directory following the format:
   - Filename: `YYYYMM-N-short-description.md` (e.g., `202505-1-feature-toggles.md`)
   - Where `YYYYMM` is the date (year-month), and `N` is a sequence number for that date

2. Use the standard template structure:
   ```
   # YYYYMM-N: Title

   **Date**: DD/MM/YYYY

   ### Context
   [Background and reasons for this decision]

   ### Decision
   [The decision that was made]

   ### Rationale
   [Explanation of why this decision was made]

   ### Trade-offs
   [Benefits and challenges of this approach]
   ```

3. Add the decision to the table in this summary document