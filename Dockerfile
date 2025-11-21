# Multi-stage Dockerfile for portainer-mcp with UPX compression
# Stage 1: Build the Go binary
FROM golang:1.24.2-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 for static binary
# -ldflags to reduce binary size
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s \
    -X main.Version=${VERSION} \
    -X main.Commit=${COMMIT} \
    -X main.BuildDate=${BUILD_DATE}" \
    -o portainer-mcp \
    ./cmd/portainer-mcp

# Stage 2: Compress with UPX
FROM ghcr.io/crazy-max/upx:latest AS compressor

COPY --from=builder /build/portainer-mcp /portainer-mcp

# Compress the binary with UPX
# --best for maximum compression
# --lzma for better compression ratio
RUN upx --best --lzma /portainer-mcp

# Stage 3: Final minimal image
FROM alpine:latest

# Add ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 portainer && \
    adduser -D -u 1000 -G portainer portainer

WORKDIR /app

# Copy compressed binary from compressor stage
COPY --from=compressor /portainer-mcp /app/portainer-mcp

# Change ownership
RUN chown -R portainer:portainer /app

# Switch to non-root user
USER portainer

# Expose default HTTP/SSE port
EXPOSE 3000

# Default to HTTP mode
ENTRYPOINT ["/app/portainer-mcp"]
CMD ["-http", "-addr", ":3000"]
