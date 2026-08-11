# Go project checklist

Architectural principles and engineering practices for Go projects. Structure,
boundaries, and logic — not any specific domain or library.

**This is a review aid, not a task list.** Nothing here is a `## Tasks` box and
nothing should be ticked. Read it when starting a package, when designing a seam, or
when reviewing a change. `kit audit` does not and will not check any of it.

**Tiered by applicability.** Start with Core, which applies to every Go project. Add
the CLI, Libraries, Long-running services, and Datastore sections only when they
apply. Applying a section that does not apply is how a checklist becomes noise.

## Core (all projects)

### Architecture and boundaries
- **Layered by dependency direction** — domain logic depends on nothing; adapters
  (DB, HTTP, external APIs) depend on domain, never the reverse.
- **Pure-policy packages** — business rules (pricing, validation, state machines)
  contain no I/O. Arithmetic and decisions only, so they are trivially testable.
- **Adapters at the edges** — each external system hidden behind one package; the
  rest of the app never imports its SDK.
- **Composition root** — one place (`run() error` called from `main`) constructs and
  wires every component. Nothing else news-up dependencies.
- **Inject dependencies via constructors** — no package-level globals, no service
  locator, no `init()` side effects.

### Package design
- Organise by **domain or capability**, not by technical layer. Avoid `models/`,
  `utils/`, `helpers/`.
- `internal/` for everything not meant as a public API; `cmd/<binary>/` stays a thin
  entry point.
- Each package owns its types. No shared "common" dumping ground.
- **No import cycles.** If two packages need each other, a concept is in the wrong
  place.

### Interfaces
- **Define interfaces in the consumer**, not alongside the implementation.
- Keep them **small** — one to three methods is typical. Accept interfaces, return
  concrete types.
- Introduce an interface only where there is a real seam (a second implementation, or
  a test boundary). Do not pre-abstract.

### Configuration
- A single typed `Config` struct loaded once at startup, with **layered precedence**
  where multiple sources exist (defaults → file → env). File-only or env-only is fine
  for small tools.
- **Validate at load** — required fields, bounds, cross-field invariants. Refuse to
  start if invalid.
- **Fail closed.** Insecure or empty states require an explicit opt-in flag; never
  default to permissive.
- Config is **immutable after load**. Changing it means a restart.

### Security and secrets
- **Secrets never in the repo.** Load from env or a secret manager, gitignore env and
  state files, and commit a `.env.example` with empty placeholders.
- **Redact secrets from logs and errors.** Tokens, passwords, and DSNs must never
  reach logs. Note that Go's `*url.Error` embeds the request URL and any credentials
  in it — scrub while preserving the `errors.Is/As` chain.
- **Context-aware output escaping.** Never hand-concatenate untrusted values into
  another language: bind SQL parameters, use `html/template` for HTML, avoid building
  shell strings.

### Error handling
- Wrap with `fmt.Errorf("context: %w", err)` so `errors.Is/As` work up the stack.
- **Sentinel or typed errors** for cases callers must branch on. Never string
  matching.
- Add context at each boundary (what operation failed), but log the error **once**, at
  the top.
- **Guard clauses** — validate and return early; keep the happy path unindented.
- **Errors are values.** Handle them; do not `panic` for control flow, and never
  `os.Exit`/`log.Fatal` outside `main`/`run` — they bypass deferred cleanup. Panic
  only for truly unrecoverable startup faults, such as a `mustLoad` of embedded
  assets.

### Concurrency
- `context.Context` is the **first parameter** of any call that does I/O or can block.
  Propagate it; never store it in a struct.
- Every blocking operation has a **timeout, deadline, or context cancellation**.
  Interactive sessions cancel via Ctrl-C, not a timer.
- Goroutines have a clear owner and exit condition. No fire-and-forget. Protect shared
  state with a mutex or a channel, and serialise correctness-critical sections.
- Run tests and CI with `-race`.

### Time and determinism
- **Inject the clock** — a `Now func() time.Time` or a small `Clock` interface. No
  `time.Now()` and no unseeded `rand` inside policy code, so tests are deterministic.

### Resource lifecycle
- Every acquired resource (file, connection, goroutine, ticker) has a **deterministic
  close or stop**. `defer` cleanup at the point of acquisition, and expose
  `Close() error` where you own one.

### Logging
- **Structured logging** (`log/slog`) with key-value pairs, not formatted strings.
- Thread a logger **explicitly** where you can, or via context, carrying
  request-scoped fields.
- Log levels mean something, and errors are logged once, at the boundary that handles
  them.

### Testing
- **Table-driven tests** as the default shape.
- **Fakes over mocks** — hand-written doubles that record calls, rather than
  mock-framework brittleness.
- Full **isolation** — no shared global state and no ordering dependence. Use
  `t.Setenv` and fresh fixtures.
- Test names state the behaviour and the condition.
- Test **behaviour at the public boundary**, not private internals.

### Build and tooling
- **Pin the Go version and `toolchain`** in `go.mod`, and manage dev tools
  reproducibly (`go.mod` tool directives or a `tools.go`).
- One command (`make check`) gated on fmt, vet, lint, and test.
- Pinned linter config (`golangci-lint`) and a vulnerability scan (`govulncheck`) in
  CI.
- Reproducible binaries — `CGO_ENABLED=0`, `-trimpath`, stripped
  `-ldflags "-s -w"`.
- CI gates (lint, test, vuln) separate from build and publish. Artifacts tagged by
  immutable commit SHA.

### General
- **Explicit over implicit** — no hidden magic, reflection, or global mutable state.
- **Make zero values useful.** Prefer value semantics; pointers only when sharing or
  mutating.
- Keep the **dependency graph shallow and acyclic**. A new engineer should trace a
  request top to bottom in one pass.

## CLI tools

For binaries a human or script invokes, which then exit.

- Thin `main` calling `run(...)`, and **map the error to a documented, categorised
  exit code**.
- **stdout is data, stderr is diagnostics.** Never interleave; it stays pipe-friendly.
- **Non-interactive-safe** — detect a non-TTY, never block on a prompt in a pipe, and
  read piped stdin when it is there.

## Libraries

For packages imported by other code.

- **Minimal exported surface** — unexported by default, and do not leak internal types
  through signatures.
- **API stability** — semver, no breaking exported signatures, and `// Deprecated:`
  before removal.
- **Do not impose policy on the caller** — no logging, no `os.Exit`/`Fatal`/`panic`,
  no global state.
- The concurrency contract is **documented**: safe for concurrent use, or explicitly
  not.

## Long-running services

For binaries that stay up and serve traffic — HTTP/gRPC servers, daemons, workers.

- Root context from `signal.NotifyContext` for SIGINT and SIGTERM.
- **Graceful shutdown** — cancel the root, drain in-flight work, close servers with a
  grace window.
- Per-request and per-dependency **deadlines** derived from the request context.
- Metrics (counters, histograms) for both technical and business events, with a
  consistent naming prefix.
- **Liveness and readiness split** — `/healthz` (process up) separate from `/readyz`
  (dependencies reachable).
- Multi-stage Docker to a minimal or distroless image, running non-root.
- **Durable work queue** for side-effecting work: claim with
  `FOR UPDATE SKIP LOCKED` or equivalent, a visibility lease and timeout, bounded
  retries with backoff honouring any server-requested delay, and a terminal or poison
  state so one bad item cannot wedge the loop.
- **Idempotent operations via dedup keys.** Any at-least-once path — enqueue, retry,
  callback — is safe to repeat, because a unique key makes duplicates no-ops.
- **Classify errors transient or permanent.** Retry transient with backoff, fail or
  skip permanent, and cap attempts. Never retry forever.
- **Treat inbound webhooks and callbacks as untrusted signals.** Verify authoritative
  state out of band by re-fetching by id, keep handlers idempotent, and add a
  reconciliation sweep for dropped events.
- **Rate-limit untrusted ingress** before doing real work.

## Datastore

When the project owns a database or other persistent store.

- **Versioned, ordered, idempotent migrations** applied at startup. Schema changes are
  additive and backward-compatible where possible.
- **Serialise migration runs** with a DB advisory lock, so concurrent instances or
  deploys cannot race to apply the same migration.
- Repository pattern — the domain talks to an interface, not the DB driver.
- **Transactions owned at the use-case layer**, not buried in repositories.
- **Parameterised queries only.** Never build SQL by string concatenation; for dynamic
  predicates, concatenate only constant SQL fragments and bind every value.
- **Enforce invariants in the DB**, not only in app code — constraints plus row locks
  (`SELECT … FOR UPDATE`) or appropriate isolation, to avoid TOCTOU under concurrency.
- **Transactional outbox** — enqueue a triggered side effect in the same transaction
  as the state change, so a crash cannot do one without the other.

