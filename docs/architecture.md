# Architecture

This MCP server bridges Portainer's REST API to MCP clients. It is
bootstrapped from the Portainer OpenAPI spec at startup — almost no
hand-written tool surface — with a small filtering and response-shaping
layer applied uniformly to every tool.

## Pipeline

```
spec/portainer-patched.yaml ──► FastMCP.from_openapi ──► tag filter ──┐
                                                                      │
                            hand-written proxy tools ─────────────────┤
                                                                      ▼
                                                  SelectArgTransform (per tool)
                                                                      │
                                                                      ▼
                                                  ResponseCapMiddleware (per call)
                                                                      │
                                                                      ▼
                                                                  MCP client
```

## Components

### Spec & tool generation — `server.py`

`build_server()` loads `spec/portainer-patched.yaml` (a locally-patched
copy of Portainer's EE OpenAPI spec) and hands it to
`FastMCP.from_openapi` with a shared `httpx.AsyncClient` carrying the
operator's `X-API-KEY`. Each operation becomes an MCP tool.

### Tag filter — `profiles.py`

The spec exposes ~380 operations across 40+ tags — too noisy for a model
to navigate. Profiles are named bundles of tags (e.g. `DOCKER`,
`KUBERNETES`). `PORTAINER_PROFILES` selects which to enable; the union
of their tags becomes a `RouteMap` per tag (FastMCP intersects multi-tag
`RouteMap`s, so we emit one per tag). `PORTAINER_READ_ONLY=1` further
restricts the surface to `GET`/`HEAD`.

### Proxy tools — `proxy.py`

Two hand-written tools — `docker_proxy` and `kubernetes_proxy` — forward
arbitrary paths under `/endpoints/{id}/docker/...` and
`/endpoints/{id}/kubernetes/...`. These paths aren't enumerated in the
OpenAPI spec (Portainer forwards them as a subpath to the underlying
daemon), so they can't be generated. Each tool validates the path (no
`..`, no `?#`) and blocks auth-bypass headers.

### Response shaping — `shaping.py`

Two cooperating layers, applied to every tool the server exposes:

1. **`SelectArgTransform`** injects an optional `select` parameter on
   every tool. The caller passes a JMESPath expression; the server
   projects the response before returning it. This is how the model
   trims noisy Portainer payloads (snapshots, K8s `managedFields`, etc.)
   server-side instead of dragging them into context.

2. **`ResponseCapMiddleware`** is the final safety valve. If a tool
   result exceeds `PORTAINER_MAX_RESPONSE_CHARS` (default 50 000), the
   middleware truncates and appends a hint that names `select` with a
   concrete example. The cap is sized to fire *before* Claude Code's
   own MCP output cap (~62k chars for dense JSON), so our hint reaches
   the model instead of Claude Code's generic "saved to file" message.

`select` narrows first (cheaper bodies); the cap catches whatever still
slips through.

## Why this shape

- **OpenAPI-driven, not hand-coded.** Portainer ships a spec; let
  FastMCP generate the tools so spec bumps are mostly a regen.
- **Filter at the spec layer.** Profiles reduce the visible surface
  without touching individual tools.
- **Universal shaping.** `select` and the cap apply to every tool —
  generated and hand-written alike — so the model learns one pattern
  that works everywhere.

## Pointers

- Profiles & tag bundles: [`profiles.md`](profiles.md)
- Versioning policy: [`versioning.md`](versioning.md)
- User-facing knobs & env vars: [`../README.md`](../README.md)
- Spec patcher: [`spec/patch_spec.py`](../spec/patch_spec.py)
