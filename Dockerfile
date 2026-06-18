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

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD python -c "import socket; s=socket.socket(); s.settimeout(2); s.connect(('127.0.0.1', 17717)); s.close()"

ENTRYPOINT ["mcp-portainer"]
