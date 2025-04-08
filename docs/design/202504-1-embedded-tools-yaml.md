# 202504-1: Embedding tools.yaml in the binary

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