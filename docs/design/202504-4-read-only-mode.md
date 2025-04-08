# 202504-4: Read-only mode for enhanced security

**Date**: 09/04/2025

### Context
Model Context Protocol (MCP) is a relatively new technology with varying levels of trust among infrastructure operators. There are significant concerns about potential security risks when allowing AI models to modify production resources. These concerns are heightened by the growing awareness of prompt injection attacks, model hallucinations, and other LLM-specific vulnerabilities that could be exploited to trigger unintended operations on critical infrastructure. Portainer often manages production container environments, making the security implications particularly serious.

### Decision
Implement a read-only flag that can be specified at application startup. When this flag is enabled, the application will only register and expose read-oriented tools, completely omitting any tools capable of modifying Portainer resources.

### Rationale
1. **Security Enhancement**
   - Eliminates risk of accidental or unauthorized modifications to production environments
   - Provides a safe mode for users to explore and monitor without modification capabilities
   - Creates a clear separation between monitoring and management use cases

2. **Operational Safety**
   - Enables safe usage in sensitive production environments
   - Reduces potential impact of prompt injection or model hallucination issues
   - Provides an additional layer of protection for critical infrastructure

3. **User Trust**
   - Addresses concerns of security-conscious users about potential write implications
   - Creates confidence that the application cannot modify resources when in read-only mode
   - Offers a path for skeptical users to start with limited capabilities before enabling full functionality

4. **Use Case Alignment**
   - Matches common use case of "explore first, modify later" workflow
   - Supports read-only scenarios like monitoring, auditing, and documentation
   - Creates a clear distinction between observability and management roles

### Trade-offs

**Benefits**
- Enhanced security posture for sensitive environments
- Reduced risk surface for production deployments
- Builds user trust through clear capability boundaries
- Better alignment with specific read-only use cases
- Allows progressive adoption starting with read-only mode

**Challenges**
- Need to categorize tools as read or write operations
- Additional startup mode to test and maintain
- Potential user confusion about available capabilities in each mode
- May require switching between modes for different workflows
- Reduced functionality in read-only mode may limit some complex scenarios