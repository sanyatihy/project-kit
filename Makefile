GO ?= go
PKG := ./...
BIN := kit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# -s -w drop the symbol table and DWARF; -trimpath keeps build paths out of the
# binary and CGO_ENABLED=0 keeps it static. Together they make the output depend on
# the source and nothing about the machine that built it. Module build info
# survives all of this, so `kit version` still resolves from it.
#
# CGO_ENABLED is set per-recipe rather than exported for the whole file: `test`
# runs with -race, and the race detector requires cgo.
LDFLAGS := -s -w -X main.Version=$(VERSION)
BUILDFLAGS := -trimpath
BUILDENV := CGO_ENABLED=0

.PHONY: build
build:
	$(BUILDENV) $(GO) build $(BUILDFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kit

# check is the gate, and it is whole: the conformance suite is a Go test in
# cmd/kit, so `test` runs it. Keeping it in-tree is the point — the previous suite
# was an external script outside the build, which is how it got deleted without
# anything noticing.
.PHONY: check
check: fmt-check vet test

# How many tests run at once. The suite is 190-odd independent tests, each with its own
# temporary directory, so this is pure wall clock: on a 12-core machine `go test -race`
# measured 32.8s serial, 22.2s at 2, 17.4s at 4, 12.0s at 6 and 8.8s uncapped.
#
# 4 is a third of that machine on purpose. The gate runs beside whatever else the operator
# is doing, and a check that takes the box with it is one people stop running. Raise it for
# a build server: make check PARALLEL=8
PARALLEL ?= 4

# What a board read costs in subprocesses, on a generated fixture. The ceilings are pinned
# in the test itself and `make check` fails on them; this target is for reading the numbers
# and the breakdown behind them when something has to be fixed.
.PHONY: gitcount
gitcount:
	@$(GO) test -count=1 -v -run TestABoardReadStaysWithinItsGitBudget ./cmd/kit

.PHONY: test
test:
	$(GO) test -race -cover -parallel $(PARALLEL) $(PKG)

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
.PHONY: vuln
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest $(PKG)

.PHONY: install
install:
	$(BUILDENV) $(GO) install $(BUILDFLAGS) -ldflags "$(LDFLAGS)" ./cmd/kit
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
	@echo "  vuln        - govulncheck against the module"
