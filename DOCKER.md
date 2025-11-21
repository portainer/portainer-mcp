# Docker Deployment Guide

This document describes how to build and run portainer-mcp using Docker.

## Quick Start

### Pull the pre-built image

```bash
docker pull ghcr.io/transform-ia/portainer-mcp:latest
```

### Run with Docker

```bash
docker run -d \
  --name portainer-mcp \
  -p 3000:3000 \
  ghcr.io/transform-ia/portainer-mcp:latest \
  -server https://portainer.example.com \
  -token YOUR_PORTAINER_TOKEN \
  -http \
  -addr :3000
```

## Building from Source

### Build the image

```bash
docker build -t portainer-mcp:local .
```

### Build with custom version tags

```bash
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t portainer-mcp:v1.0.0 \
  .
```

### Multi-architecture build

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t portainer-mcp:multi-arch \
  --push \
  .
```

## Configuration

### Environment Variables

While the application uses command-line flags, you can create wrapper scripts with environment variables:

```bash
#!/bin/sh
exec /app/portainer-mcp \
  -server "${PORTAINER_URL}" \
  -token "${PORTAINER_TOKEN}" \
  -http \
  -addr ":${PORT:-3000}" \
  ${PORTAINER_READ_ONLY:+-read-only} \
  ${PORTAINER_DISABLE_VERSION_CHECK:+-disable-version-check}
```

### Docker Compose

```yaml
version: '3.8'

services:
  portainer-mcp:
    image: ghcr.io/transform-ia/portainer-mcp:latest
    container_name: portainer-mcp
    restart: unless-stopped
    ports:
      - "3000:3000"
    command:
      - -server
      - https://portainer.example.com
      - -token
      - ${PORTAINER_TOKEN}
      - -http
      - -addr
      - :3000
      - -disable-version-check
    environment:
      - TZ=UTC
    security_opt:
      - no-new-privileges:true
    read_only: true
    user: "1000:1000"
```

## Kubernetes Deployment

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: portainer-mcp
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: portainer-mcp
  template:
    metadata:
      labels:
        app: portainer-mcp
    spec:
      containers:
      - name: portainer-mcp
        image: ghcr.io/transform-ia/portainer-mcp:latest
        ports:
        - containerPort: 3000
          name: http
        args:
          - -server
          - https://portainer.example.com
          - -token
          - $(PORTAINER_TOKEN)
          - -http
          - -addr
          - :3000
        env:
        - name: PORTAINER_TOKEN
          valueFrom:
            secretKeyRef:
              name: portainer-mcp-secret
              key: token
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
---
apiVersion: v1
kind: Service
metadata:
  name: portainer-mcp
  namespace: default
spec:
  selector:
    app: portainer-mcp
  ports:
  - port: 3000
    targetPort: 3000
    name: http
```

### Create Secret

```bash
kubectl create secret generic portainer-mcp-secret \
  --from-literal=token='YOUR_PORTAINER_TOKEN'
```

## Image Details

### Multi-Stage Build

The Dockerfile uses a three-stage build process:

1. **Builder stage**: Builds the Go binary with optimizations
2. **Compressor stage**: Compresses the binary with UPX for smaller image size
3. **Final stage**: Minimal Alpine-based image with only the compressed binary

### Image Size

The final image is approximately **15-20 MB** thanks to:
- Static binary compilation (CGO_ENABLED=0)
- Link-time optimizations (-w -s flags)
- UPX compression (--best --lzma)
- Minimal Alpine base image
- Multi-stage build (excludes build dependencies)

### Security Features

- Runs as non-root user (UID 1000)
- No unnecessary packages
- CA certificates included for HTTPS
- Read-only root filesystem compatible
- No privilege escalation

## Supported Architectures

- `linux/amd64`
- `linux/arm64`

## Health Check

Add a health check to your deployment:

```yaml
livenessProbe:
  httpGet:
    path: /
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 30
```

## Troubleshooting

### Check logs

```bash
docker logs portainer-mcp
```

### Interactive shell

```bash
docker exec -it portainer-mcp sh
```

### Test connectivity

```bash
docker run --rm ghcr.io/transform-ia/portainer-mcp:latest \
  -server https://portainer.example.com \
  -token YOUR_TOKEN \
  -http \
  -addr :3000
```

## CI/CD

The GitHub Actions workflow automatically:
- Builds images for AMD64 and ARM64
- Pushes to GitHub Container Registry
- Tags with version, SHA, and `latest`
- Runs basic tests on the built image

See `.github/workflows/docker-build.yml` for details.
