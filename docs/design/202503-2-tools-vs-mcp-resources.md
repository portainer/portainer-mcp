# 202503-2: Using tools to get resources instead of MCP resources

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
- May need to reconsider if application-driven selection becomes necessary or when we'll need to build our own client

### References
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/#user-interaction-model
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/#user-interaction-model