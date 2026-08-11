GO ?= go
PKG := ./...
BIN := bin/{{name}}
CMD := ./cmd/{{name}}
FMT_DIRS := cmd internal

# build runs the full gate (format, vet, lint, tests) before compiling.
.PHONY: build
build: check
	$(GO) build -o $(BIN) $(CMD)

# release builds the static, stripped binary an image would ship. Override
# GOOS/GOARCH to cross-compile.
GOOS ?= linux
GOARCH ?= amd64
RELEASE_BIN ?= {{name}}
.PHONY: release
release:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags "-s -w" -o $(RELEASE_BIN) $(CMD)

# check is the pre-push / CI gate. One command, same in every project.
.PHONY: check
check: fmt-check vet lint test

.PHONY: test
test:
	$(GO) test -race -cover $(PKG)

.PHONY: fmt
fmt:
	gofmt -s -w $(FMT_DIRS)

# fmt-check fails (non-zero) if any file is not gofmt-clean, suitable for CI.
.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -s -l $(FMT_DIRS))" || (echo "Run 'make fmt' to fix:" && gofmt -s -l $(FMT_DIRS) && exit 1)

.PHONY: vet
vet:
	$(GO) vet $(PKG)

# lint is skipped with a warning when golangci-lint is not installed, so a fresh
# clone goes green with only Go present. CI installs it, so there it really runs.
.PHONY: lint
lint:
	@which golangci-lint >/dev/null 2>&1 \
		&& golangci-lint run \
		|| echo "note: golangci-lint not installed, skipping (https://golangci-lint.run)"

.PHONY: vuln
vuln:
	govulncheck $(PKG)

.PHONY: run
run:
	$(GO) run $(CMD)

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build      - check + compile $(BIN)"
	@echo "  release    - static stripped $(RELEASE_BIN) (GOOS/GOARCH overridable)"
	@echo "  check      - fmt-check + vet + lint + test (pre-push/CI gate)"
	@echo "  test       - go test -race -cover ./..."
	@echo "  fmt        - gofmt -s -w"
	@echo "  fmt-check  - fail if anything is unformatted"
	@echo "  vet        - go vet ./..."
	@echo "  lint       - golangci-lint run (skipped if not installed)"
	@echo "  vuln       - govulncheck ./..."
	@echo "  run        - go run ./cmd/{{name}}"
	@echo "  tidy       - go mod tidy"
