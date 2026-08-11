GO ?= go
PKG := ./...
BIN := kit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.Version=$(VERSION)

.PHONY: build
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kit

# check is the gate. The conformance suite is not in it: that file is currently
# missing (see docs/features/), and a gate that silently skips is worse than none.
.PHONY: check
check: fmt-check vet test

.PHONY: test
test:
	$(GO) test -race -cover $(PKG)

# conformance builds the binary and runs the behavioural suite against it.
.PHONY: conformance
conformance: build
	@test -f ./test || (echo "./test is missing — the conformance suite has not been restored" && exit 1)
	./test

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
	@echo "  test        - go unit tests"
	@echo "  conformance - build, then run ./test against the binary"
	@echo "  fmt         - gofmt -s -w ."
	@echo "  fmt-check   - fail if anything is unformatted"
	@echo "  vet         - go vet ./..."
	@echo "  lint        - golangci-lint run (skipped if not installed)"
