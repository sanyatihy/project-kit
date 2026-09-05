# Python project checklist

Architectural principles and engineering practices for Python projects. Structure,
boundaries, and logic — not any specific domain.

**This is a review aid, not a task list.** Nothing here is a `## Tasks` box and nothing
should be ticked. Read it when starting a module, when designing a seam, or when reviewing
a change. `kit audit` does not and will not check any of it.

**It names libraries, tiered by how reversible the choice is.** Go's standard library
covers the places a checklist points at and Python's does not. Development tooling is named
outright, because none of it ships. The boundary library is named as a default with the
criterion for leaving it. A framework is named only in the section that uses it. Nothing is
named where the standard library is adequate, or "a dependency earns its place" is not a
rule anyone can be held to.

**Tiered by applicability.** Start with Core, which applies to every Python project. Add
the CLI, Libraries, Long-running services, Datastore, and Data and analysis sections only
when they apply. Applying a section that does not apply is how a checklist becomes noise.

## Core (all projects)

### Architecture and boundaries
- **Layered by dependency direction** — domain logic imports nothing outward; adapters
  (DB, HTTP, external APIs) import domain, never the reverse.
- **Pure-policy modules** — business rules (pricing, validation, state machines) contain no
  I/O. Arithmetic and decisions only, so they are trivially testable.
- **Adapters at the edges** — each external system behind one module; the rest of the app
  never imports its SDK.
- **Composition root** — one `run(argv) -> int`, called from a thin `main`, constructs and
  wires every component. Nothing else builds its own dependencies.
- **Inject through parameters**, not module globals, not a registry, and never by reaching
  back into a caller's module.

### Package design
- Organise by **domain or capability**, not by technical layer. Avoid `models.py`,
  `utils.py`, `helpers.py`.
- **`src/` layout**, so tests run against the installed package and an import that only
  works from the repo root fails loudly. Never mutate `sys.path`.
- `_`-prefixed for everything not public. Each module owns its types; no shared "common".
- **No import cycles.** Python defers them to runtime, where they surface as a
  half-initialised module. A cycle means a concept is in the wrong place, and
  `if TYPE_CHECKING` hides the symptom rather than fixing it.

### Protocols and abstraction
- **Define `typing.Protocol` in the consumer.** Structural typing means the implementation
  never imports the seam.
- Keep protocols **small** — one to three methods is typical.
- `isinstance` against a Protocol needs `@runtime_checkable`, and then it checks method
  **names only**. Reach for an ABC when the check has to mean more than that, or when there
  is shared implementation.
- Introduce an abstraction only where there is a real seam — a second implementation, or a
  test boundary. Do not pre-abstract.

### Typing and data
- **`mypy --strict` (or pyright strict) gates CI.** Static types are optional in Python, so
  the choice is made once, at the top, and not per-module.
- **Untyped third-party imports are where strict leaks.** Quarantine them in one stubs
  module, or a per-module override naming the package. Never a blanket
  `ignore_missing_imports`.
- **No `Any` at a boundary** — narrow `object`, or model the shape. **No bare
  `# type: ignore`**: an error code and a reason, or fix the type.
- **Parse, do not validate.** Untrusted input becomes a typed object at the edge, built by
  a validating parser — pydantic by default, msgspec where deserialisation is measured on
  the hot path. The interior never sees a raw dict.
- **A frozen dataclass is the value object inside that boundary.** `frozen=True` by
  default; `slots=True` only where it is measured, because it breaks `cached_property`,
  weakrefs and some inheritance.
- `Literal`, `Final`, `NewType` and `Enum` for domain scalars, so a currency code and an
  account id are not both `str`. **`Enum` for a closed set**, never bare strings compared
  by equality.
- **Never a mutable default.** `None` and a guard for a function parameter,
  `field(default_factory=...)` for a dataclass field.
- **`Decimal` or integer minor units for money.** `float` is for measured quantities where
  a rounding error is noise, never for an amount that must reconcile.
- **Type-check the tests too.** They are the largest body of code that calls the API.

### Configuration
- A single typed, frozen `Config` loaded once at startup, with **layered precedence** where
  multiple sources exist (defaults → file → env). File-only or env-only is fine for small
  tools.
- **Validate at load** — required fields, bounds, cross-field invariants. Refuse to start
  if invalid.
- **Fail closed.** Insecure or empty states require an explicit opt-in flag; never default
  to permissive.
- **No scattered `os.environ` reads.** The environment is read in one function.

### Security and secrets
- **Secrets never in the repo.** Load from env or a secret manager, gitignore env and state
  files, and commit a `.env.example` with empty placeholders.
- **Redact secrets from logs and errors.** A `repr` is a leak: exclude the field with
  `field(repr=False)`.
- **`assert` is not validation.** `python -O` removes every one, so a check that must hold
  at runtime is a `raise`. Asserts state internal invariants for a reader and a type
  checker.
- **Never deserialise untrusted input into code.** No `pickle`, no `yaml.load` without
  `SafeLoader`, no `eval`, no `exec`.
- **`subprocess` takes a list argv and never `shell=True`.**
- **Never build another language by concatenation.** Bind every SQL value, and for a
  dynamic predicate concatenate only constant fragments. Use a templating engine that
  escapes by context.
- `secrets`, not `random`, for anything an attacker should not predict.

### Error handling
- **One exception hierarchy** rooted at a project base class, with a type per case a caller
  must branch on. Never match on a message string.
- **`raise NewError(...) from err`** so the chain survives. Add context at each boundary,
  and log the error **once**, at the top.
- **Catch the narrowest type over the narrowest body.** No bare `except:`, and no
  `except Exception: pass` — a swallowed error is a bug with the evidence deleted.
- **Guard clauses** — validate and return early; keep the happy path unindented.
- **`sys.exit` and `SystemExit` belong in `main` alone.** A library raises and lets the
  caller decide. Exceptions are not control flow across a module boundary.

### Concurrency
- **Prefer none.** Add it when a measurement asks for it, not in anticipation.
- **Know the bound.** I/O-bound work goes to `asyncio` or threads; CPU-bound work goes to
  processes, because the GIL makes threads the wrong answer for compute.
- **Set the `multiprocessing` start method explicitly** — `spawn` or `forkserver`. `fork`
  inherits locks held by other threads and deadlocks the child that waits on one.
- **Do not mix sync and async in one call graph.** Blocking work called from a coroutine
  goes through `asyncio.to_thread`, or it stalls the loop for everyone.
- **Structured concurrency** — `asyncio.TaskGroup`, not a bare `create_task` whose
  reference is dropped and whose exception nobody sees. `except*` for the group's errors.
- Every awaited I/O has a **timeout** (`asyncio.timeout`) or an explicit cancellation path.
  Interactive sessions cancel on Ctrl-C, not a timer.
- **Cancellation is cooperative.** `CancelledError` propagates; catching it to clean up
  means re-raising it.
- Shared mutable state is protected or absent. `contextvars` for request scope, not
  thread-locals and not globals.

### Time and determinism
- **Inject the clock** — a `now: Callable[[], datetime]`. No `datetime.now()` and no
  module-level `random` inside policy code, so tests are deterministic.
- **Timezone-aware datetimes only.** `datetime.now(UTC)`, `zoneinfo` for local rules, and
  never a naive datetime crossing a boundary.
- **`time.monotonic` for a duration.** A timeout measured on the wall clock moves when the
  clock is corrected.

### Resource lifecycle
- Every acquired resource (file, connection, task, socket) is held by a **context
  manager**. `with` at the point of acquisition, `@contextmanager` for your own,
  `ExitStack` when the count is dynamic.
- **A generator holding a `with` is not covered by that.** One left unexhausted keeps the
  resource until it is closed or collected, and an exception raised inside it surfaces at
  iteration rather than at the call. Wrap the consumer, or return a list.
- **Never rely on refcounting to close a file.** It is a CPython implementation detail.
- **Text I/O states its encoding.** `encoding="utf-8"` on `open`, `subprocess` and
  `Path.read_text`; the default is the locale's, so the same code reads different bytes on
  another machine.
- `pathlib` for paths; a path is a `Path`, not a string that happens to look like one.

### Logging
- **`logging.getLogger(__name__)` per module.** Handlers, levels and formatters are
  configured **only** in the composition root — a library that configures logging has taken
  a decision that was not its to take.
- **Never an f-string in a log call.** `logger.info("...%s", value)` defers the formatting,
  and fields go in `extra=` so logs stay queryable.
- `logger.exception` inside the handler that handles the error, once.
- `print` is for a CLI's data output, never for diagnostics.

### Testing
- **`@pytest.mark.parametrize` is the default shape** — one test, a table of cases.
- **Fakes over patching.** A hand-written double records calls; `unittest.mock.patch`
  couples the test to an import path and survives the refactor it should have caught.
- Full **isolation** — `tmp_path`, `monkeypatch`, fresh fixtures, no ordering dependence,
  no network, no shared global state.
- Test **behaviour at the public API**, not private internals. Names state the behaviour
  and the condition.
- **Property-based tests** (Hypothesis) where the input space is large and the invariant is
  exact: parsers, money arithmetic, round-trips.
- **Warnings are errors in CI** (`-W error`). It is what turns a deprecation into a failure
  one release before it becomes a breakage.

### Build and tooling
- **Pin the interpreter** (`requires-python`) and lock dependencies with **uv**, installed
  with hashes in CI. A reproducible environment is the precondition for a reproducible
  failure.
- One `pyproject.toml`. No `setup.py`, and no `requirements.txt` as the source of truth.
- One command (`make check`) gated on format, lint, type and test, with **ruff** for the
  first two and **pip-audit** over the lockfile.
- **A virtualenv per project.** Never the system interpreter.
- CI gates separate from build and publish. Artifacts tagged by immutable commit SHA.

### General
- **Nothing runs on import but definitions.** No I/O, no config read, no registration side
  effect — this is what makes the composition root enforceable at all.
- **Explicit over implicit** — no monkeypatching, no metaclasses, no module-level
  `__getattr__`, no global mutable state.
- **A dependency earns its place.** The stdlib first; a package that saves ten lines costs
  a supply chain.
- Keep the **import graph shallow and acyclic**. A new engineer should trace a request top
  to bottom in one pass.

## CLI tools

For programs a human or script invokes, which then exit.

- **`raise SystemExit(main())` in `__main__.py`** — the one file where running something on
  import is the point — with `main()` calling `run(argv) -> int`, and **every failure
  mapped to a documented, named exit code**.
- **stdout is data, stderr is diagnostics.** Never interleave; it stays pipe-friendly.
- **Non-interactive-safe** — check `isatty`, never block on a prompt in a pipe, and read
  piped stdin when it is there.
- Handle `BrokenPipeError` and `KeyboardInterrupt` at the top rather than tracebacking.

## Libraries

For packages imported by other code.

- **Minimal public surface** — `_`-prefixed by default, an explicit `__all__`, and no
  internal types leaking through signatures.
- **Submit `py.typed`** (PEP 561), or the types you wrote are invisible to every caller.
- **Do not impose policy on the caller** — no logging configuration, no `sys.exit`, no
  global state, no import-time side effects.
- **Dependency ranges, not pins.** Pinning is an application's job; a pinned library makes
  its callers unresolvable.
- **API stability** — semver, and a `DeprecationWarning` for at least one minor release
  before removal.
- The concurrency contract is **documented**: thread-safe, async-safe, or explicitly
  neither.

## Long-running services

For processes that stay up and serve traffic — HTTP servers, daemons, workers.

- **The framework is an adapter.** FastAPI, Litestar or bare Starlette owns routing and
  serialisation; the handler parses, calls a use case that has never heard of the framework,
  and maps the result back. `Depends()` is a service locator — keep it at the edge, out of
  domain signatures.
- Signal handlers for SIGINT and SIGTERM installed in the composition root, cancelling a
  root task group.
- **Graceful shutdown** — stop accepting, drain in-flight work, close connections within a
  grace window.
- Per-request and per-dependency **deadlines**, derived from the request's own budget.
- **Do not fork after starting threads.** A worker model that forks builds connection pools
  and background tasks post-fork, in the worker.
- Metrics (counters, histograms) for both technical and business events, with a consistent
  naming prefix.
- **Liveness and readiness split** — `/healthz` (process up) separate from `/readyz`
  (dependencies reachable).
- Multi-stage Docker to a slim base, running non-root, with dependencies installed from the
  lockfile in a layer separate from the source.
- **Durable work queue** for side-effecting work: claim with `FOR UPDATE SKIP LOCKED` or
  equivalent, a visibility lease and timeout, bounded retries with backoff honouring any
  server-requested delay, and a terminal or poison state so one bad item cannot wedge the
  loop.
- **Idempotent operations via dedup keys.** Any at-least-once path — enqueue, retry,
  callback — is safe to repeat, because a unique key makes duplicates no-ops.
- **Classify errors transient or permanent.** Retry transient with backoff, fail or skip
  permanent, and cap attempts. Never retry forever.
- **Treat inbound webhooks and callbacks as untrusted signals.** Verify authoritative state
  out of band by re-fetching by id, keep handlers idempotent, and add a reconciliation
  sweep for dropped events.
- **Rate-limit untrusted ingress** before doing real work.

## Datastore

When the project owns a database or other persistent store.

- **Versioned, ordered, idempotent migrations** (Alembic) applied at startup. Schema changes
  are additive and backward-compatible where possible.
- **Serialise migration runs** with a DB advisory lock, so concurrent instances or deploys
  cannot race to apply the same migration.
- Repository pattern — the domain talks to a protocol, not the driver or the ORM.
- **Sessions and transactions owned at the use-case layer**, one per request or per command,
  never a module-level session and never buried in a repository.
- **Enforce invariants in the DB**, not only in app code — constraints plus row locks
  (`SELECT … FOR UPDATE`) or appropriate isolation, to avoid TOCTOU under concurrency.
- **Transactional outbox** — enqueue a triggered side effect in the same transaction as the
  state change, so a crash cannot do one without the other.
- Connection pools are **not fork-safe**. Dispose and rebuild in the child.

## Data and analysis

When the project transforms data rather than serving requests.

- **Notebooks are not production code.** They explore; the logic they find moves into an
  importable, tested module before anything depends on it.
- **Deterministic and diffable outputs** — stable ordering, stable formatting, and a
  line-oriented format (JSONL, CSV) rather than a binary blob, so a change to a number shows
  up in review.
- **Declare dtypes and schemas; never infer them.** An inferred column type is a silent
  behaviour change when the next batch arrives.
- **Reconcile against an independent invariant** — a balance chain, a control total, a row
  count — and fail rather than publish a total that does not add up.
