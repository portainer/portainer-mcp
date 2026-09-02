# syntax=docker/dockerfile:1.7

FROM ghcr.io/astral-sh/uv:python3.13-bookworm-slim AS builder

ENV UV_LINK_MODE=copy \
    UV_COMPILE_BYTECODE=1 \
    UV_PROJECT_ENVIRONMENT=/app/.venv

WORKDIR /app

COPY pyproject.toml uv.lock ./
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-install-project --no-dev --no-editable

COPY README.md LICENSE ./
COPY src ./src
# Needed at build time: hatch force-include copies the hygiene skill into the
# wheel (see pyproject.toml [tool.hatch.build.targets.wheel.force-include]).
COPY skills ./skills
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-editable


FROM python:3.13-slim-bookworm AS runtime

RUN groupadd --system --gid 1000 portainer \
 && useradd --system --uid 1000 --gid portainer --no-create-home \
            --shell /usr/sbin/nologin portainer

WORKDIR /app

COPY --from=builder /app/.venv /app/.venv

ENV PATH="/app/.venv/bin:$PATH" \
    PYTHONUNBUFFERED=1 \
    PORTAINER_MCP_TRANSPORT=http \
    PORTAINER_MCP_HTTP_HOST=0.0.0.0 \
    PORTAINER_MCP_LOG_FORMAT=json

USER portainer
EXPOSE 17717

# The probe mirrors server.py's bind resolution (`or` fallbacks, so an empty
# value behaves like an unset one) and maps a wildcard bind to its loopback.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD python -c "import os, socket; h=os.environ.get('PORTAINER_MCP_HTTP_HOST') or '127.0.0.1'; h={'0.0.0.0': '127.0.0.1', '::': '::1'}.get(h, h); socket.create_connection((h, int(os.environ.get('PORTAINER_MCP_HTTP_PORT') or 17717)), timeout=2).close()"

ENTRYPOINT ["mcp-portainer"]
