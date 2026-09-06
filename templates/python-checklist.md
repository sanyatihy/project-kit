# Python project checklist

Review aid; not Tasks boxes and not checked by kit audit. Read Core, then applicable
sections. Named development tools are defaults; choose runtime dependencies only where
needed. The standard library comes first.

## Core (all projects)

### Structure
- Keep domain policy free of I/O. Adapters depend on domain; hide each external SDK
  behind one module.
- Wire dependencies in `run(argv) -> int` called by thin main. Inject through parameters;
  no registries, caller-module lookups or globals. Imports perform no I/O, config reads
  or registration side effects.
- Organise by capability; avoid common/utils/models dumping grounds. Modules own types;
  private names start with `_`. Keep imports shallow and acyclic; TYPE_CHECKING is not
  a fix for misplaced concepts.
- Use src layout and test the installed package; never modify sys.path.
- Define small `typing.Protocol` seams in consumers for real implementations or tests.
  `@runtime_checkable` checks names only; use an ABC for stronger contracts or shared
  implementation. Do not pre-abstract.

### Types and data
- Gate application and test types with mypy strict or pyright strict. Isolate untyped
  dependencies in stubs or named per-module overrides; no blanket ignore_missing_imports.
- No Any at boundaries: narrow object or model the shape. A type-ignore needs an error
  code and reason.
- Parse external input into typed objects: pydantic by default, msgspec for measured hot
  deserialisation paths. Keep raw dicts outside domain code.
- Use frozen dataclasses internally. Measure before slots=True; account for cached_property,
  weakrefs and inheritance limits. No mutable defaults: use None guards or default_factory.
- Use Literal, Final, NewType and Enum for domain scalars; Enum for closed sets.
  Money uses Decimal or integer minor units; float is for measured quantities.

### Configuration and security
- Load one typed frozen Config at startup with explicit precedence (defaults → file → env
  where applicable). Read os.environ in one place; validate bounds and cross-field invariants.
  Fail closed; insecure or empty states require opt-in.
- Load secrets from env or a secret manager. Ignore env/state files; commit empty
  .env.example placeholders. Redact logs, exceptions and repr fields (`repr=False`).
- Runtime validation raises: assert disappears under python -O. Reserve asserts for
  internal invariants.
- Never load untrusted code via pickle, unsafe yaml.load, eval or exec. Use secrets for
  unpredictable values. subprocess takes argv lists, never shell=True.
- Bind SQL values; dynamic predicates use constant fragments only. Use context-aware
  escaping in templates instead of concatenating untrusted values into another language.

### Errors and logging
- Use a project exception hierarchy for cases callers distinguish; never match messages.
  Chain with `raise NewError(...) from err`, add boundary context and use guard clauses.
- Catch narrow exceptions over narrow bodies. No bare except or swallowed Exception.
  sys.exit/SystemExit belong in main; libraries raise. Avoid cross-module exception control flow.
- Use logging.getLogger(__name__); configure handlers, levels and formatters only in the
  composition root. Defer formatting with `%s` arguments, put fields in extra, and call
  logger.exception once where the error is handled. print is CLI data, not diagnostics.

### Concurrency and time
- Add concurrency only for measured need: asyncio/threads for I/O, processes for CPU work
  under the GIL. Explicitly choose spawn or forkserver; fork can inherit locked state.
- Keep sync and async call graphs separate; use asyncio.to_thread for blocking work.
  Use TaskGroup for owned tasks and except* for grouped errors; never drop create_task results.
- Awaited I/O needs timeout or cancellation; interactive work cancels on Ctrl-C.
  Propagate CancelledError after cleanup. Protect shared state; use contextvars for request scope.
- Inject clocks and randomness into policy. Use aware datetimes (`datetime.now(UTC)`),
  zoneinfo for local rules and time.monotonic for durations.

### Resources
- Own files, connections, tasks and sockets through context managers; use ExitStack for
  dynamic counts. Do not rely on refcounting for cleanup.
- A generator's internal with stays open if consumption stops: wrap its consumer or
  return a list. Iteration, not creation, may raise its errors.
- Use pathlib and explicit UTF-8 for text I/O, including subprocess and Path.read_text.

### Tests and tooling
- Prefer pytest parametrization and hand-written fakes over import-path patching.
  Test public behavior; names state behavior and condition. Isolate fixtures with tmp_path
  and monkeypatch; no order dependence, shared globals or network.
- Use Hypothesis for large input spaces with exact invariants. Treat warnings as errors.
- Pin requires-python, use one pyproject.toml and a project virtualenv. Lock dependencies
  with uv and install with hashes in CI; no setup.py or requirements.txt source of truth.
- `make check` gates format/lint (ruff), types and tests; audit the lockfile with pip-audit.
  Keep CI gates separate from build/publish; identify artifacts by immutable commit SHA.
- Prefer explicit code over monkeypatching, metaclasses or module __getattr__. Dependencies
  must earn their cost; avoid global mutable state.

## CLI tools

- `__main__.py` calls `raise SystemExit(main())`; main calls run(argv) and maps failures
  to documented exit codes. Handle BrokenPipeError and KeyboardInterrupt at the top.
- stdout is data; stderr is diagnostics. Handle piped stdin and non-TTY use without prompts.

## Libraries

- Minimise public names, declare __all__, avoid leaking internals and ship py.typed.
- Preserve semver APIs; issue DeprecationWarning for at least one minor release before removal.
  Publish dependency ranges; applications own pins.
- Leave logging configuration, exit policy and globals to callers. No import side effects.
  Document thread and async safety explicitly.

## Long-running services

- Keep FastAPI/Litestar/Starlette as adapters: handlers parse, call domain use cases and
  map results. Keep framework dependency injection out of domain signatures.
- Install SIGINT/SIGTERM handlers in the composition root; cancel the root task group,
  stop accepting work, drain and close within a grace window. Derive dependency deadlines
  from each request's budget.
- Do not fork after starting threads; initialise worker pools and background tasks post-fork.
- Use consistently named technical/business metrics. Separate /healthz from /readyz.
  Use non-root, slim multi-stage images; install locked dependencies separately from source.
- Persist side-effecting work with atomic claims, leases, timeouts and terminal failures.
  Retry transient errors with bounded backoff and server-requested delays; fail or skip
  permanent errors. Deduplicate at-least-once operations.
- Treat callbacks as untrusted: re-fetch authoritative state, handle idempotently and
  reconcile missed events. Rate-limit untrusted ingress before expensive work.

## Datastore

- Apply versioned, ordered, idempotent migrations (Alembic) at startup under an advisory
  lock. Prefer additive, backward-compatible schema changes.
- Domain code uses repository protocols. Use cases own sessions and transactions per
  request/command; no global sessions. Rebuild connection pools after a fork.
- Enforce invariants with database constraints and locks or suitable isolation.
  Put triggered side effects in a transactional outbox with the state change.

## Data and analysis

- Move notebook discoveries into tested, importable modules before production use.
- Make output deterministic and diffable: stable order/format, preferably JSONL or CSV.
- Declare dtypes and schemas rather than inferring them from each batch.
- Reconcile against independent invariants (balances, control totals, row counts);
  fail instead of publishing inconsistent results.
