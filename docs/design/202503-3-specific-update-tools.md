# 202503-3: Specific tool for updates instead of a single update tool

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