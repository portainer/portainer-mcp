# Design documentation

This document summarizes design decisions for the software.

## Table of Contents
1. [202503-1: Using an external tools file for tool definition](#202503-1-using-an-external-tools-file-for-tool-definition)
2. [202503-2: Using tools to get resources instead of MCP resources](#202503-2-using-tools-to-get-resources-instead-of-mcp-resources)
3. [202503-3: Specific tool for updates instead of a single update tool](#202503-3-specific-tool-for-updates-instead-of-a-single-update-tool)
4. [202504-1: Embedding tools.yaml in the binary](#202504-1-embedding-toolsyaml-in-the-binary)
5. [202504-2: Strict versioning for tools.yaml file](#202504-2-strict-versioning-for-toolsyaml-file)

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
- Improved visibility and security through externalized tool definitions, making it easier for users to audit and understand potential prompt injection risks

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
- May need to reconsider if application-driven selection becomes necessary or when we'll need to build our own client

### References
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/#user-interaction-model
- https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/#user-interaction-model

## 202503-3: Specific tool for updates instead of a single update tool

**Date**: 29/03/2025

### Context
Initially, resource updates (such as access groups, environments, etc.) were handled through single, multi-purpose update tools that could modify multiple properties at once. This approach led to complex parameter handling and unclear behavior around optional values.

### Decision
Split update operations into multiple specific tools, each responsible for updating a single property or related set of properties. For example, instead of a single `updateAccessGroup` tool, create separate tools like:
- `updateAccessGroupName`
- `updateAccessGroupUserAccesses`
- `updateAccessGroupTeamAccesses`

### Rationale
1. **Parameter Clarity**
   - Each tool has clear, required parameters
   - No ambiguity between undefined parameters and empty values
   - Eliminates need for complex optional parameter handling

2. **Code Simplification**
   - Removes need for pointer types in parameter handling
   - Clearer validation of required parameters
   - Simpler implementation of each specific update operation

3. **Maintenance Benefits**
   - Each tool has a single responsibility
   - Easier to test individual update operations
   - Clearer documentation of available operations

4. **Model Interaction**
   - Models can clearly understand which property they're updating
   - More explicit about the changes being made
   - Better alignment with natural language commands

### Trade-offs

**Benefits**
- Clearer parameter requirements and validation
- Simpler code without pointer logic
- Better separation of concerns
- More explicit and focused tools
- Easier testing and maintenance

**Challenges**
- Multiple API calls needed for updating multiple properties
- Slightly increased network traffic for multi-property updates
- More tool definitions to maintain
- No atomic updates across multiple properties
- More tools might clutter the context of the model
- Some clients have a hard limit on the number of tools that can be used/enabled

### Notes
Performance impact of multiple API calls is considered acceptable given:
- Non-performance-critical context
- Relatively low frequency of update operations
- Benefits of simpler code and clearer behavior outweigh the overhead

## 202504-1: Embedding tools.yaml in the binary

**Date**: 08/04/2025

### Context
After deciding to use an external tools.yaml file for tool definitions (see 202503-1), there was a need to determine the best distribution method for this file. Questions arose about how to ensure the file is available when the application runs.

### Decision
Embed the tools.yaml file directly in the binary during the build process, while also checking for and using a user-provided version at runtime if available.

### Rationale
1. **Simplified Distribution**
   - Single binary contains everything needed to run the application
   - No need to manage separate file distribution
   - Eliminates file path configuration issues

2. **User Customization**
   - Application checks for external tools.yaml at startup
   - If found, uses the external file for tool definitions
   - If not found, creates it using the embedded version as reference

3. **Default Configuration**
   - Provides sensible defaults out of the box
   - Ensures application can always run even without external configuration
   - Serves as a reference for users who want to customize

4. **Version Control**
   - Embedded file serves as the official version for each release
   - External file allows for hotfixes without binary updates
   - Clear separation between default and custom configurations

### Trade-offs

**Benefits**
- Simpler distribution process
- Self-contained application
- Ability to run without configuration
- Support for user customization
- Clear fallback mechanism

**Challenges**
- Slightly larger binary size
- Need for embedding logic in the build process
- Managing differences between embedded and external versions
- Ensuring proper precedence between versions

## 202504-2: Strict versioning for tools.yaml file

**Date**: 08/04/2025

### Context
With tools.yaml being externalized and allowing user customization, there's a risk of incompatibility between the tool definitions and the application code. Changes to the schema or expected tool definitions could lead to runtime errors that are difficult to diagnose.

### Decision
Implement strict versioning for the tools.yaml file with version validation at startup. The application will define a required/current version, check if the provided tools.yaml file uses this version, and fail fast if there's a version mismatch.

### Rationale
1. **Compatibility Assurance**
   - Prevents runtime errors caused by incompatible tool definitions
   - Clearly communicates version requirements to users
   - Makes version mismatches immediately apparent

2. **Error Handling**
   - Provides clear error messages about version mismatches
   - Fails fast instead of letting subtle errors occur during operation
   - Guides users toward proper resolution

3. **Recovery Path**
   - Users can update their tools.yaml file manually to match the required version
   - Alternatively, users can simply delete their customized file and let the application regenerate it
   - Regeneration uses the embedded version which is guaranteed to be compatible

4. **Upgrade Management**
   - Clear versioning creates explicit upgrade paths
   - Version checks provide a mechanism to enforce schema migrations
   - Makes breaking changes in tool definitions more manageable

### Trade-offs

**Benefits**
- Prevents subtle runtime errors
- Provides clear error messages
- Offers straightforward recovery options
- Makes version incompatibilities immediately apparent
- Simplifies upgrade paths

**Challenges**
- Need to manage version numbers across releases
- Must communicate version changes to users
- Requires additional validation logic at startup
- Necessitates documentation of version compatibility

