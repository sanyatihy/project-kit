GO ?= go
PKG := ./...
BIN := kit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.Version=$(VERSION)

.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kit

# check is the gate, and it is whole: the conformance suite is a Go test in
# cmd/kit, so `test` runs it. Keeping it in-tree is the point — the previous suite
# was an external script outside the build, which is how it got deleted without
# anything noticing.
.PHONY: check
check: fmt-check vet test

.PHONY: test
test:
	$(GO) test -race -cover $(PKG)

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -s -l . | grep -v '^templates/')" \
		|| (echo "Run 'make fmt' to fix:" && gofmt -s -l . && exit 1)

.PHONY: vet
vet:
	$(GO) vet $(PKG)

.PHONY: lint
lint:
	@which golangci-lint >/dev/null 2>&1 \
		&& golangci-lint run \
		|| echo "note: golangci-lint not installed, skipping"

# install puts `kit` on your PATH via the Go toolchain.
.PHONY: install
install:
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/kit
	@echo "installed $(BIN) into $$($(GO) env GOBIN); if empty, $$($(GO) env GOPATH)/bin"

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build       - compile ./$(BIN)"
	@echo "  install     - go install onto your PATH"
	@echo "  check       - fmt-check + vet + test (the gate)"
	@echo "  test        - unit tests + the conformance suite"
	@echo "  fmt         - gofmt -s -w ."
	@echo "  fmt-check   - fail if anything is unformatted"
	@echo "  vet         - go vet ./..."
	@echo "  lint        - golangci-lint run (skipped if not installed)"
