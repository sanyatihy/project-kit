# Backlog

Ideas that are not features yet. Nothing here is scheduled or prioritised — the
default state is **stays frozen**.

Brainstorm output lands here. Promote one to its own `NNNN-*.md` with `kit new`
only on a real signal: someone asked, a metric moved, something else needs it.

Keep each entry to a few lines: what it is, and what would make it worth doing.

Everything below came from reviewing the kit against `kit help go-checklist`
(2026-08-11, feature 0003). Findings, not fixes.

- **Inject the clock into `NewFeature`** — `internal/kit/newdoc.go` calls
  `time.Now()` inline while filling `created:`, so no test can assert which date a
  doc gets. `audit` already does this correctly: `cmd/kit/main.go` computes `today`
  and passes it to `kit.Audit`. The inconsistency is the argument. Promote when: 0002
  writes a `new` test that needs a fixed date — which it will.
- **Document and categorise the exit codes** — 0, 1, and 2 are in use, but only
  `--strict`'s exit 1 is documented. Everything else, bad usage and genuine failure
  alike, is 2. An agent scripting the kit cannot tell "you typed it wrong" from
  "audit found something". `kit help` should state the contract. Promote when: anyone
  wraps `kit` in a script, or 0002 pins exit codes.
- **Add operation context to errors** — no `fmt.Errorf(...%w...)` anywhere, and about
  ten bare `return err` in `scaffold.go`, `newdoc.go`, and `repo.go`. A failed
  `kit init` says `permission denied` without naming the file it could not write.
  Promote when: someone hits one of these and has to read the source to find out
  what failed.
- **Harden the build** — three related gaps: `go.mod` pins `go 1.26` but no
  `toolchain`; CI runs `golangci-lint` but no `govulncheck`; and the build sets only
  `-X main.Version`, without `CGO_ENABLED=0`, `-trimpath`, or `-ldflags "-s -w"`. One
  small feature, not three. Promote when: the kit is distributed anywhere its build
  provenance matters, or a CVE lands in a dependency.

Considered and deliberately not filed: the package-level lookup tables and compiled
regexes across `internal/kit` are immutable data, not the mutable global state the
checklist warns about, and `var Version` is the `-ldflags` idiom with no alternative.
`var templates = assets.FS` is a dependency held as a global, which is a real if
minor smell — threading it through every constructor is ceremony for a single-binary
CLI, so it stays until there is a second reason to move it.
