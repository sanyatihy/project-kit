UV ?= uv
SRC := src tests

# check is the pre-push / CI gate. One command, same in every project.
.PHONY: check
check: tools fmt-check lint type test

# Everything runs through uv, so a missing uv is reported once and by name rather than as
# a command-not-found from whichever target happened to run first.
.PHONY: tools
tools:
	@which $(UV) >/dev/null 2>&1 || (echo "uv is not installed: https://docs.astral.sh/uv/" && exit 1)

.PHONY: sync
sync:
	$(UV) sync

.PHONY: test
test:
	$(UV) run pytest

.PHONY: fmt
fmt:
	$(UV) run ruff format $(SRC)
	$(UV) run ruff check --fix $(SRC)

# fmt-check fails (non-zero) if any file is unformatted, suitable for CI.
.PHONY: fmt-check
fmt-check:
	$(UV) run ruff format --check $(SRC)

.PHONY: lint
lint:
	$(UV) run ruff check $(SRC)

.PHONY: type
type:
	$(UV) run mypy

.PHONY: audit
audit:
	$(UV) run pip-audit

.PHONY: run
run:
	$(UV) run python -m {{pkg}}

.PHONY: lock
lock:
	$(UV) lock

.PHONY: help
help:
	@echo "Targets:"
	@echo "  check      - fmt-check + lint + type + test (pre-push/CI gate)"
	@echo "  sync       - install the locked environment"
	@echo "  test       - pytest, warnings as errors"
	@echo "  fmt        - ruff format, then ruff check --fix"
	@echo "  fmt-check  - fail if anything is unformatted"
	@echo "  lint       - ruff check"
	@echo "  type       - mypy --strict"
	@echo "  audit      - pip-audit"
	@echo "  run        - python -m {{pkg}}"
	@echo "  lock       - refresh uv.lock"
