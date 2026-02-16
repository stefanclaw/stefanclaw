.PHONY: build test lint clean release-dry-run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o stefanclaw ./cmd/stefanclaw

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f stefanclaw

release-dry-run:
	goreleaser release --snapshot --clean
