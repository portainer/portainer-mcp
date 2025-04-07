# Note: these can be overriden on the command line e.g. `make PLATFORM=<platform> ARCH=<arch>`
PLATFORM="$(shell go env GOOS)"
ARCH="$(shell go env GOARCH)"

.PHONY: pre build run test test-integration test-all

pre:
	mkdir -p dist

build: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags '-s' -o dist/portainer cmd/portainer/portainer.go

push: PLATFORM=darwin
push: ARCH=arm64
push: build
	rm -f /share-tmp/portainer-mcp
	cp dist/portainer /share-tmp/portainer-mcp

inspector: build
	npx @modelcontextprotocol/inspector dist/portainer

test:
	go test -v $(shell go list ./... | grep -v /tests/)

test-integration:
	go test -v ./tests/...

test-all: test test-integration
