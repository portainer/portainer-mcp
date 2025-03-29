# Design documentation

This document summarizes design decisions for the software.

## Using an external tools file for tool definition

Date: 29/03/2025

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

## Tools file versioning

Date: 29/03/2025

## Specific tool for updates instead of a single update tool

Date: 29/03/2025

## Using tools to get resources instead of MCP resources

Date: 29/03/2025

