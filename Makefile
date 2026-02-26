# Note: these can be overriden on the command line e.g. `make PLATFORM=<platform> ARCH=<arch>`
PLATFORM="$(shell go env GOOS)"
ARCH="$(shell go env GOARCH)"

VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS_STRING = -s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}

.PHONY: clean pre build run test test-integration test-all lint

clean:
	rm -rf dist

pre:
	mkdir -p dist

build: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) CGO_ENABLED=0 go build --ldflags '$(LDFLAGS_STRING)' -o dist/portainer-mcp ./cmd/portainer-mcp

release: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) CGO_ENABLED=0 go build --ldflags '$(LDFLAGS_STRING)' -o dist/portainer-mcp ./cmd/portainer-mcp

inspector: build
	npx @modelcontextprotocol/inspector dist/portainer-mcp

test:
	go test -v $(shell go list ./... | grep -v /tests/integration)

test-coverage:
	go test -v $(shell go list ./... | grep -v /tests/integration) -coverprofile=./coverage.out

test-integration:
	go test -v ./tests/...

test-all: test test-integration

lint:
	golangci-lint run ./...

# Include custom make targets
-include $(wildcard .dev/*.make)