# Docker

The container image lives at `docker.io/portainer/portainer-mcp`. Each `X.Y.Z` release publishes two tags:

- `X.Y.Z` — exact patch.
- `X.Y` — rolling pointer to the latest patch on a Portainer minor.

No `latest` tag — pin a Portainer minor explicitly so a Portainer-side upgrade doesn't slide under you. See [`versioning.md`](versioning.md).

Images are built for `linux/amd64`.
