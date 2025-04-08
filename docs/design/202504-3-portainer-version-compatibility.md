# 202504-3: Pinning compatibility to a specific Portainer version

**Date**: 08/04/2025

### Context
Portainer server does not implement API versioning, making it challenging to ensure compatibility between our software and different Portainer server versions. Each version of Portainer may have different API behaviors and endpoints, which could cause runtime errors or unexpected behavior if not properly managed.

### Decision
Maintain independent versioning for this software while explicitly pinning compatibility to a specific Portainer server version. The software will validate the Portainer server version at startup and fail fast if the detected version does not match the required version exactly. Documentation will clearly indicate which exact Portainer version is supported by each software release.

### Rationale
1. **Independent Release Cycle**
   - Software can be updated outside of the Portainer release lifecycle
   - Allows for bug fixes and features without waiting for Portainer releases
   - Enables more frequent iterations and improvements

2. **Exact Compatibility**
   - Each release will document the specific Portainer version it supports
   - Strict version checking at startup prevents compatibility issues
   - Ensures 100% compatibility with the supported API endpoints

3. **SDK Alignment**
   - Software will use a Go SDK version that matches exactly the supported Portainer version
   - Creates a precise binding between SDK capabilities and software functionality
   - Eliminates ambiguity about supported functionality

4. **Error Prevention**
   - Early validation of the exact Portainer version prevents any API compatibility issues
   - Users receive clear error messages when the version doesn't match
   - Completely eliminates support requests related to API incompatibilities

### Trade-offs

**Benefits**
- Flexible release schedule independent of Portainer
- Absolute certainty about compatibility requirements
- Fail-fast behavior for unsupported versions
- Predictable behavior with supported Portainer version
- Simplified testing against a single Portainer version

**Challenges**
- Users must upgrade/downgrade Portainer to the exact supported version
- Each software release requires a new version when supporting a new Portainer version
- More restrictive for users who can't easily change their Portainer version
- Overhead of version validation at startup
- Need to clearly communicate the exact supported version in all documentation