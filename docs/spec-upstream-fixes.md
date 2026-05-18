# Spec upstream-fix tracking (additional findings)

This document records defects in the Portainer OpenAPI spec
(`github.com/portainer/portainer-api-docs`) that the **portainer-mcp-fastmcp**
experiment encountered but are **not** already tracked in
[`portainer-go-sdk/docs/spec-upstream-fixes.md`](https://github.com/portainer/portainer-go-sdk/blob/main/docs/spec-upstream-fixes.md).

The Go SDK's toolchain (Redocly bundler + `gopkg.in/yaml.v3` + Go's
`encoding/json`) is permissive enough not to trip on these. Strict
consumers — PyYAML 1.1, jsonschema response validators — surface each one.

**Spec version this document tracks.** Portainer EE `2.41.1`
(`versions/ee/2.41.1/openapi.yaml`).

**Source of truth for the mitigations.** `spec/patch_spec.py` and
`src/portainer_mcp/server.py` in this repo.

---

## 1. Stray tab character in description scalar

| Location | Defect | Mitigation | Upstream fix |
| --- | --- | --- | --- |
| `components.schemas.KubectlShellImage.description` (line 25661 in the bundled YAML) | The description string contains a literal `\t` (`"Kubec\tl Shell Image Name/Tag"`). PyYAML rejects tabs inside plain scalars under strict YAML 1.1 scanning. | The patcher normalises *all* tab characters to single spaces on the raw YAML text before parsing — applied globally rather than at one site because this is exactly the kind of upstream defect that recurs. | Strip the stray tab from the swaggo `@Description` annotation on `KubectlShellImage` in `portainer/portainer-ee` so the swaggo step emits a clean description. |

## 2. Bare `=` enum value triggers YAML 1.1 value-tag resolution

| Location | Defect | Mitigation | Upstream fix |
| --- | --- | --- | --- |
| `components.schemas.portaineree.ConditionOperator.enum` | The enum lists comparison operators, with `>`, `>=`, `<=` correctly quoted but `<` and `=` left bare. PyYAML's YAML 1.1 scanner resolves the bare `=` to the `tag:yaml.org,2002:value` tag (the YAML 1.1 default-key indicator), and the safe loader has no constructor for it in a sequence context, so parsing fails. | The patcher registers a constructor for `tag:yaml.org,2002:value` on `yaml.SafeLoader` that returns `node.value` as a plain string. The enum then parses with `=` as the literal string `"="`. | In the swaggo `@Enums(...)` annotation for `ConditionOperator` (likely in `portainer/portainer-ee` under a conditions/policies package), quote `=` (and `<`, for symmetry) so the emitted YAML scalars are `"="` / `"<"` instead of bare tokens. |

## 3. Null-valued arrays in response payloads

| Location | Defect | Mitigation | Upstream fix |
| --- | --- | --- | --- |
| Many response objects (`Endpoint.AuthorizedUsers`, `Endpoint.AuthorizedTeams`, `Endpoint.Tags`, `DockerSnapshotRaw.{Containers,Networks,Images,Volumes.Volumes,…}`, and counterparts across the spec) | Fields are declared as `array` without `nullable: true`. The Go server returns `null` for uninitialized slices (`encoding/json`'s zero value for `nil`), so strict OAS 3.0 output validators reject the response. The Go SDK does not hit this because Go's JSON unmarshaler accepts `null` for `[]T`. Affects most Endpoint, Stack, and snapshot operations. | `FastMCP.from_openapi(..., validate_output=False)` in `server.py` — drops the post-response jsonschema gate. The advertised output schema is unchanged, so MCP clients still get structural guidance. | Either initialize slices to empty server-side before marshaling (changes wire output: `null` → `[]`, with risk for consumers that distinguish them), or mark every affected field `nullable: true` in the swaggo annotations (contract-only, zero wire change). |

---

This document can be deleted, or its rows moved into the SDK's catalogue,
once these are addressed upstream.
