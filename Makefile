# Version baked into the binary. Falls back to "dev" outside a git checkout.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/neoighodaro/blvckhole/cmd.version=$(VERSION)

.PHONY: build install test vet fmt clean

# Build the ./blvckhole artifact that ~/.local/bin/blvckhole symlinks to.
build:
	go build -ldflags "$(LDFLAGS)" -o blvckhole .

# Stripped release-style build.
install:
	go build -ldflags "-s -w $(LDFLAGS)" -o blvckhole .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f blvckhole
