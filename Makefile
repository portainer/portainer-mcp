# Note: these can be overriden on the command line e.g. `make PLATFORM=<platform> ARCH=<arch>`
PLATFORM="$(shell go env GOOS)"
ARCH="$(shell go env GOARCH)"

.PHONY: pre build run

pre:
	mkdir -p dist

build: pre
	GOOS=$(PLATFORM) GOARCH=$(ARCH) CGO_ENABLED=0 go build -a --installsuffix cgo --ldflags '-s' -o dist/portainer cmd/portainer/portainer.go

push: PLATFORM=darwin
push: ARCH=arm64
push: build
	cp dist/portainer /share-tmp/portainer-mcp

run:
	go run cmd/portainer/portainer.go -server 1 -token 2

test:
	go test -v ./...
