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
check: fmt-check vet lint test

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
#
# Every budget test, not one of them: 0056 added the board with no trunk, and a target that
# named a single test by hand printed three ceilings of four while claiming to print them all.
.PHONY: gitcount
gitcount:
	@$(GO) test -count=1 -v -run 'TestABoard.*GitBudget' ./cmd/kit

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
	@test -z "$$(gofmt -s -l . | grep -v '^templates/')" \
		|| (echo "Run 'make fmt' to fix:" && gofmt -s -l . && exit 1)

.PHONY: vet
vet:
	$(GO) vet $(PKG)

# lint is skipped with a notice when golangci-lint is not installed, so a fresh
# clone goes green with only Go present. CI installs it, so there it really runs.
.PHONY: lint
lint:
	@which golangci-lint >/dev/null 2>&1 \
		&& golangci-lint run \
		|| echo "note: golangci-lint not installed, skipping (https://golangci-lint.run)"

# install puts `kit` on your PATH via the Go toolchain.
.PHONY: vuln
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest $(PKG)

# audit reads this repo's own board. `go run` builds it from the checkout, so it
# needs no credential for a private module — CI never installs the kit.
.PHONY: audit
audit:
	$(GO) run ./cmd/kit audit --strict

.PHONY: install
install:
	$(BUILDENV) $(GO) install $(BUILDFLAGS) -ldflags "$(LDFLAGS)" ./cmd/kit
	@echo "installed $(BIN) into $$($(GO) env GOBIN); if empty, $$($(GO) env GOPATH)/bin"

# release cuts the tag `go install ...@latest` resolves to, which is the only thing that
# changes what a fresh install gets. Annotated, not lightweight: the tag then carries who
# cut it and when, `git describe` reports it without --tags, and the message names the
# range rather than restating it — the feature docs are the changelog.
#
# The refusals come before the gate rather than after it, because each is instant and the
# gate is half a minute. Uncommitted work means the gate tested something the tag would not
# point at; a HEAD that is not what origin publishes means tagging unreviewed work as a
# release.
#
# The push is part of the target. `@latest` reads the remote, so a tag on one machine is
# not a release — the same state `audit` reports at HIGH for work that shipped and was
# never pushed.
.PHONY: release
release:
	@test -n "$(TAG)" || (echo "name the version: make release TAG=v0.3.0" && exit 1)
	@test -z "$$(git status --porcelain)" \
		|| (echo "uncommitted work: the gate would test what $(TAG) does not point at" && exit 1)
	@tip=$$(git ls-remote origin HEAD | cut -f1); \
		if [ -z "$$tip" ]; then echo "origin did not answer, so its trunk is unknown"; exit 1; fi; \
		if [ "$$(git rev-parse HEAD)" != "$$tip" ]; then \
			echo "HEAD is not the tip of origin's default branch: release from the trunk"; \
			exit 1; fi
	$(MAKE) check
	@prev=$$(git describe --tags --abbrev=0 2>/dev/null); \
		git tag -a $(TAG) -m "$(TAG)$${prev:+ — $$prev..$(TAG)}"
	git push origin refs/tags/$(TAG)
	@echo "released $(TAG): go install github.com/sanyatihy/project-kit/cmd/kit@$(TAG)"

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build       - compile ./$(BIN)"
	@echo "  install     - go install onto your PATH"
	@echo "  release     - gate, tag and push a version: make release TAG=v0.3.0"
	@echo "  check       - fmt-check + vet + lint + test (the gate)"
	@echo "  test        - unit tests + the conformance suite"
	@echo "  fmt         - gofmt -s -w ."
	@echo "  fmt-check   - fail if anything is unformatted"
	@echo "  vet         - go vet ./..."
	@echo "  lint        - golangci-lint run (skipped if not installed)"
	@echo "  vuln        - govulncheck against the module"
	@echo "  audit       - kit audit --strict on this repo (needs the remote)"
