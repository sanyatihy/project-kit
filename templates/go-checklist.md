# Go project checklist

Review aid; not Tasks boxes and not checked by kit audit. Read Core, then only the
sections that apply to the project.

## Core (all projects)

### Structure
- Keep domain policy free of I/O. Adapters depend on domain; hide each external SDK
  behind one package.
- Wire dependencies in one `run()` called by thin `main`. Inject via constructors;
  no service locators, package globals or `init()` side effects.
- Organise by capability. Keep private code in `internal/`; packages own their types.
  Avoid common/utils dumping grounds; keep the import graph shallow and acyclic.
- Define small interfaces in consumers at real implementation or test seams.
  Accept interfaces, return concrete types; do not pre-abstract.

### Configuration and security
- Load one typed Config at startup. Define precedence (defaults → file → env where
  applicable), validate fields and cross-field invariants, then keep it immutable.
- Fail closed; insecure or empty states need explicit opt-in.
- Load secrets from env or a secret manager. Ignore env/state files; commit empty
  `.env.example` placeholders. Redact logs and errors, including URLs in `*url.Error`,
  while preserving `errors.Is/As` chains.
- Bind SQL values, use `html/template` escaping, and avoid shell-string construction.

### Errors and logging
- Wrap with `fmt.Errorf("context: %w", err)`; branch on sentinel/typed errors using
  `errors.Is/As`, never message strings. Use early validation and guard clauses.
- Handle errors as values. No panic for control flow; only unrecoverable startup faults.
  Keep `os.Exit`/`log.Fatal` in main/run; they bypass deferred cleanup.
- Use `log/slog` fields and meaningful levels. Pass the logger explicitly or via context
  with request fields. Add operation context at boundaries; log each error once where handled.

### Concurrency and resources
- Pass context first for I/O or blocking calls; propagate it, never store it in structs.
- Bound blocking operations with timeout, deadline or cancellation. Interactive sessions
  cancel on Ctrl-C. Every goroutine has an owner and exit; protect shared state and
  serialise correctness-critical sections.
- Inject clocks and randomness into policy. Every acquired resource has deterministic
  cleanup: defer close/stop at acquisition and expose `Close() error` when owned.

### Tests and tooling
- Prefer table-driven public-behavior tests and hand-written fakes. Names state behavior
  and condition. Isolate fixtures and environment; no shared globals or order dependence.
- Pin Go/toolchain in go.mod and manage dev tools reproducibly. `make check` covers
  format, vet, pinned lint and tests; run tests with `-race` and govulncheck in CI.
- Build reproducibly with CGO_ENABLED=0, -trimpath and stripped ldflags. Keep build/publish
  separate from CI gates; identify artifacts by immutable commit SHA.
- Prefer explicit code, useful zero values and value semantics. Use pointers for sharing
  or mutation; avoid hidden reflection and global mutable state.

## CLI tools

- Thin main calls run and maps failures to documented exit codes.
- stdout is data; stderr is diagnostics. Handle piped stdin and non-TTY use without prompts.

## Libraries

- Minimise exports; do not leak internal types. Preserve semver APIs and mark removals
  with `// Deprecated:` first.
- Leave policy to callers: no logging, Exit/Fatal/panic or global state. Document concurrency safety.

## Long-running services

- Use `signal.NotifyContext` for SIGINT/SIGTERM; cancel, drain and close within a grace window.
  Derive request and dependency deadlines from the request context.
- Use consistently named technical/business metrics. Separate `/healthz` liveness from
  `/readyz` dependency readiness. Build minimal/distroless multi-stage images, run non-root.
- Persist side-effecting work with atomic claims, visibility leases, timeouts and a terminal
  failure state. Retry transient errors with bounded backoff and server-requested delays;
  fail or skip permanent errors. Use deduplication keys for at-least-once paths.
- Treat callbacks as untrusted: re-fetch authoritative state, handle idempotently and
  reconcile missed events. Rate-limit untrusted ingress before expensive work.

## Datastore

- Apply versioned, ordered, idempotent migrations at startup under an advisory lock.
  Prefer additive, backward-compatible schema changes.
- Domain code uses repository interfaces; use cases own transactions.
- Bind every SQL value; dynamic SQL may concatenate constant fragments only.
- Enforce invariants with database constraints and locks or suitable isolation.
  Put triggered side effects in a transactional outbox with the state change.
