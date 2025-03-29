# Design documentation

This document summarizes design decisions for the software.

## Table of Contents
1. [202503-1: Using an external tools file for tool definition](#202503-1-using-an-external-tools-file-for-tool-definition)
2. [202503-2: Using tools to get resources instead of MCP resources](#202503-2-using-tools-to-get-resources-instead-of-mcp-resources)
3. [202503-3: Tools file versioning](#202503-3-tools-file-versioning)
4. [202503-4: Specific tool for updates instead of a single update tool](#202503-4-specific-tool-for-updates-instead-of-a-single-update-tool)

## 202503-1: Using an external tools file for tool definition

**Date**: 29/03/2025

### Context
The project needs to define and maintain a set of tools that interact with Portainer. Initially, these tool definitions could have been hardcoded within the application code.

### Decision
Tool definitions are externalized into a separate `tools.yaml` file instead of maintaining them in the source code.

### Rationale
1. **Improved Readability**
   - Tool definitions often contain multi-line descriptions and complex parameter structures
   - YAML format provides better readability and structure compared to in-code definitions
   - Separates concerns: tool definitions from implementation logic

2. **Dynamic Updates**
   - Allows modification of tool descriptions and parameters without rebuilding the binary
   - Enables rapid iteration on tool definitions
   - Particularly valuable when experimenting with LLM interactions, as descriptions can be optimized for AI comprehension without code changes

3. **Maintenance Benefits**
   - Single source of truth for tool definitions
   - Easier to review and validate changes to tool definitions
   - Simplified version control for documentation changes

4. **Version Management**
   - External file format may need versioning as schema evolves
   - Requires consideration of backward compatibility
   - Enables tracking of breaking changes in tool definitions

### Trade-offs

**Benefits**
- More flexible maintenance of tool definitions
- Better separation of concerns
- Easier experimentation with LLM-optimized descriptions
- Independent evolution of tool definitions and code

**Challenges**
- Need to handle file loading and validation
- Must ensure file distribution with the binary
- Additional complexity in version management

## 202503-2: Using tools to get resources instead of MCP resources

**Date**: 29/03/2025

### Context
Initially, listing Portainer resources (environments, environment groups, stacks, etc.) was implemented using MCP resources. The project needed to evaluate whether this was the optimal approach given the current usage patterns and client constraints.

### Decision
Replace MCP resources with tools for retrieving Portainer resources. For example, instead of exposing environments as MCP resources, provide a `listEnvironments` tool that the model can invoke.

### Rationale
1. **Client Compatibility**
   - Project currently relies on existing MCP clients (e.g., Claude Desktop)
   - MCP resources require manual selection in these clients
   - One-by-one resource selection creates friction in testing and iteration

2. **Protocol Design Alignment**
   - MCP resources are designed to be application-driven, requiring UI elements for selection
   - Tools are designed to be model-controlled, better matching current use case
   - Better alignment with the protocol's intended interaction patterns

3. **User Experience**
   - Models can directly request resource listings using natural language
   - No need for manual resource selection in the client
   - Faster iteration and testing cycles

4. **Model Control**
   - Tools provide a more direct interaction model for AI
   - Models can determine when and what resources to list
   - Approval flow is streamlined through tool invocation

### Trade-offs

**Benefits**
- Improved user experience through natural language requests
- Faster testing and iteration cycles
- Better alignment with existing client capabilities
- More direct model control over resource access

**Challenges**
- Potential loss of MCP resource-specific features
- Need to maintain tool definitions for resource access
- May need to reconsider if application-driven selection becomes necessary

### References
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/#user-interaction-model
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/#user-interaction-model

## 202503-3: Tools file versioning

**Date**: 29/03/2025

## 202503-4: Specific tool for updates instead of a single update tool

**Date**: 29/03/2025