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

# The whole of CI, in order: .github/workflows/check.yml runs these and nothing
# else, and cmd/kit/ci_test.go fails when the two lists drift. The list lives here
# because this is the half a developer can run.
CI_STEPS := check vuln audit

# check is the gate, and it is whole: the conformance suite is a Go test in
# cmd/kit, so `test` runs it. Keeping it in-tree is the point — the previous suite
# was an external script outside the build, which is how it got deleted without
# anything noticing.
#
# What is not in it needs the network: `vuln` downloads the vulnerability database
# and `audit` reads the remote's claim branches.
.PHONY: check
check: fmt-check lint test

# Bound concurrent fixture work on a developer machine; build servers can raise it.
PARALLEL ?= 4

# Print git-process growth across board sizes; the test also runs in the gate.
.PHONY: gitcount
gitcount:
	@$(GO) test -count=1 -v -run 'TestABoardReadKeepsGitWorkBounded' ./cmd/kit

# No -cover. The conformance suite drives a separately built binary, so the number
# counted the test process and not the kit: 2.9% for the package the suite exercises
# end to end. docs/TECH.md says what measuring it for real would take.
.PHONY: test
test:
	$(GO) test -race -parallel $(PARALLEL) $(PKG)

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: fmt-check
fmt-check:
	@unformatted=$$(gofmt -s -l .) || exit $$?; \
		if [ -n "$$unformatted" ]; then \
			printf 'Run make fmt to fix:\n%s\n' "$$unformatted"; exit 1; \
		fi

.PHONY: vet
vet:
	$(GO) vet $(PKG)

# A missing or failing linter fails the gate. CI installs the pinned version.
.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is required: https://golangci-lint.run"; exit 1; \
	fi

# install puts `kit` on your PATH via the Go toolchain.
.PHONY: vuln
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.7.0 $(PKG)

# audit reads this repo's own board. `go run` builds it from the checkout, so it
# needs no credential for a private module — CI never installs the kit.
.PHONY: audit
audit:
	$(GO) run ./cmd/kit audit --strict

.PHONY: install
install:
	$(BUILDENV) $(GO) install $(BUILDFLAGS) -ldflags "$(LDFLAGS)" ./cmd/kit
	@echo "installed $(BIN) into $$($(GO) env GOBIN); if empty, $$($(GO) env GOPATH)/bin"

# Pass TAG through the environment, never interpolate it into shell source.
export TAG
.PHONY: release
release:
	@sh scripts/release.sh $(CI_STEPS)

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build       - compile ./$(BIN)"
	@echo "  install     - go install onto your PATH"
	@echo "  release     - gate, tag and push a version: make release TAG=v0.3.0"
	@echo "  check       - fmt-check + lint + test (the gate)"
	@echo "  test        - unit tests + the conformance suite"
	@echo "  fmt         - gofmt -s -w ."
	@echo "  fmt-check   - fail if anything is unformatted"
	@echo "  vet         - go vet ./..."
	@echo "  lint        - golangci-lint run (required)"
	@echo "  vuln        - govulncheck against the module"
	@echo "  audit       - kit audit --strict on this repo (needs the remote)"
