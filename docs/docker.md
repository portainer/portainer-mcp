# Docker

The container image lives at `docker.io/portainer/portainer-mcp`. Each `X.Y.Z`
release publishes two tags:

- `X.Y.Z` — exact patch.
- `X.Y` — rolling pointer to the latest patch on a Portainer minor.

No `latest` tag — pin a Portainer minor explicitly so a Portainer-side upgrade
doesn't slide under you. See [`versioning.md`](versioning.md).

Images are built for `linux/amd64` and `linux/arm64`.

## Minimal run

```bash
docker run --rm -it \
  -p 8000:8000 \
  -e PORTAINER_URL=https://portainer.example.com \
  -e PORTAINER_API_KEY=ptr_xxxxxxxxxxxxxxxx \
  -e PORTAINER_MCP_AUTH_TOKEN="$(openssl rand -hex 32)" \
  portainer/portainer-mcp:2.42
```

The container defaults to HTTP transport bound on `0.0.0.0:8000`, so it's
reachable from outside the container. `PORTAINER_MCP_AUTH_TOKEN` is **required**
in HTTP mode — startup fails loudly without it. Generate a fresh value per
deployment (`openssl rand -hex 32` is the recommended floor).

Clients connect to `http://<host>:8000/mcp` with
`Authorization: Bearer <token>`. Example for Claude Code:

```bash
claude mcp add portainer --transport http http://<host>:8000/mcp \
  --header "Authorization: Bearer <token>"
```

## Reverse-proxy / TLS

The container terminates HTTP, not HTTPS, and serves auth as a single shared
bearer. For anything reaching the public internet, put a TLS-terminating
reverse proxy (Caddy, Traefik, Cloudflare, nginx) in front:

- TLS termination at the proxy.
- Optional additional auth at the proxy (IP allowlist, mTLS) if you want
  defence in depth on top of the shared secret.
- Forward to the container on `127.0.0.1:8000` or a private network.

Per-user OIDC-gated auth is not shipped yet — the current image accepts a
single shared bearer only.

## Tunables

All the server env vars work in the container — see the top of
[`src/portainer_mcp/server.py`](../src/portainer_mcp/server.py) for the full
list. The image overrides three defaults:

| Var                          | Container default | Server default |
|------------------------------|-------------------|----------------|
| `PORTAINER_MCP_TRANSPORT`    | `http`            | `stdio`        |
| `PORTAINER_MCP_HTTP_HOST`    | `0.0.0.0`         | `127.0.0.1`    |
| `PORTAINER_MCP_HTTP_PORT`    | `8000`            | `8000`         |

Anything else (`PORTAINER_PROFILES`, `PORTAINER_READ_ONLY`,
`PORTAINER_NO_PROXY`, `PORTAINER_TLS_VERIFY`, `PORTAINER_MCP_LOG_LEVEL`,
`PORTAINER_TAGS_EXTRA`, `PORTAINER_MAX_RESPONSE_CHARS`) passes straight through.

## Logs

The server logs to stderr at INFO by default. `docker logs <container>` is the
intended consumption path. Bump verbosity with `-e PORTAINER_MCP_LOG_LEVEL=DEBUG`.

The startup line includes a masked fingerprint of the auth token
(`first4…last4`) so you can confirm the right secret loaded without exposing
it. The full value is never logged.
