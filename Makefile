UPSTREAM_REPO ?= git@github.com:portainer/portainer-api-docs.git
UPSTREAM_DIR  := spec/upstream

.PHONY: specs dev

# Refresh src/portainer_mcp/data/portainer-patched.yaml from upstream
# Usage: make specs VERSION=2.42.0
specs:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required, e.g. make specs VERSION=2.42.0" >&2; \
		exit 1; \
	fi
	@if [ -d $(UPSTREAM_DIR)/.git ]; then \
		git -C $(UPSTREAM_DIR) fetch --depth=1 origin HEAD && \
		git -C $(UPSTREAM_DIR) reset --hard FETCH_HEAD; \
	else \
		git clone --depth=1 --filter=blob:none --no-checkout \
			$(UPSTREAM_REPO) $(UPSTREAM_DIR); \
	fi
	git -C $(UPSTREAM_DIR) sparse-checkout set --no-cone /versions/ee/$(VERSION).yaml
	git -C $(UPSTREAM_DIR) checkout
	uv run python spec/patch_spec.py $(UPSTREAM_DIR)/versions/ee/$(VERSION).yaml

# Local dev server (HTTP transport = per-user passthrough). One-time setup:
#   1. cp .env.example .env, fill in PORTAINER_URL, and set
#      PORTAINER_MCP_AUTH_TOKEN to a fresh secret: `openssl rand -hex 32`.
#      Do NOT set PORTAINER_API_KEY — HTTP refuses to boot with it (it's
#      stdio-only); your Portainer key travels as a client header instead.
#   2. Register with Claude using the gate token + your own Portainer key:
#      claude mcp add portainer-dev --transport http http://127.0.0.1:17717/mcp \
#        --header "Authorization: Bearer <token>" \
#        --header "X-Portainer-API-Key: <ptr_key>"
# Then iterate: edit code, ctrl-c, make dev again. Claude reconnects automatically.
dev:
	PORTAINER_MCP_TRANSPORT=http uv run --env-file .env portainer-mcp
