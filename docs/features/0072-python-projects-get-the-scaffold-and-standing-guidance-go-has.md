---
id: 0072
title: Python projects get the scaffold and standing guidance Go has
status: Draft
created: 2026-08-30
shipped:
needs: 0071
---

## Problem

Go is the only stack `kit init` has and `kit help go-checklist` the only standing
engineering guidance the binary carries. A Python project bootstrapped with the kit gets
the docs, the protocol and a language-neutral `.gitignore` — and is then held to whatever
practice the agent happens to remember. The kit says it bootstraps projects; it bootstraps
one language.

Nothing proves `stackLayout` is a map until a second stack reads it. Two decisions in
`Init` are Go's and are written as neither the stack's nor the map's: it writes
`internal/.gitkeep` for any stack, and it skips the entry point by matching the literal
filename `main.go.tmpl` against a glob for `main.go`. A Python project would get a stray
`internal/` and a duplicate entry point.

## Non-goals

- **A checklist `audit` enforces.** The Go checklist's preamble refuses this in a sentence
  and the Python one keeps it. Prose is judgement; a check that could be written would
  already be a check.
- **Recording which stack a project is.** The kit stores nothing about a project beyond
  its files, and `--stack` is an argument to one command rather than state. The protocol
  therefore names both checklists and lets the agent pick, which is the price of not
  keeping that fact anywhere.
- **Generalising for a third stack.** Two is what separates the parts of `Init` that are
  Go's from the parts that are every stack's. A third would be guessed at.
- **Rewriting the Go checklist.** The review that produced the Python one found nothing
  that carries over: Go's `time.Time` holds a monotonic reading, and Go has no `-O` that
  strips assertions.
- **Deployment scaffolding.** The same refusal the go stack makes, for the same reason.

## Design

**A stack declares what `Init` currently assumes.** Two fields join each stack's layout:
the glob that means the project already has an entry point, and the directory that gets a
`.gitkeep`. Go's are `cmd/*/main.go` and `internal/`; Python's are `src/*/__main__.py` and
nothing. Both `if`s leave `Init`, so adding a stack after this one is templates and a map
entry.

**The checklist is a template and a help topic, exactly as Go's is.** No new mechanism, and
`templates/python-checklist.md` sits beside `templates/go-checklist.md`.

**The protocol names a checklist per language rather than one.**
`templates/AGENTS-section.md` sends every agent to `kit help go-checklist`; it will name
both and say to read the one for the project's language. One clause per stack is affordable
at two and is what not storing the stack costs.

**The stack pins the interpreter and locks dependencies with uv.** The checklist requires a
lockfile and does not choose the tool, so the template does. uv gives the interpreter pin,
the lock and the virtualenv in one binary, which keeps the Makefile as thin as the Go one.
The cost is a tool younger than pip-tools and outside the stdlib, so `make check` reports a
missing uv as a missing tool rather than failing obscurely — the shape the go Makefile
already uses for `golangci-lint`.

**`check` is the same gate in a second language.** Format, lint, type and test behind one
`make check`, so the sentence the protocol writes about a project's own gate stays true
whatever it is written in.

**The checklist ships corrected, not as reviewed.** Fifteen findings against the draft, in
three groups. Six omissions where a rule was missing outright: `assert` stripped by `-O`,
text I/O without an explicit `encoding=`, warnings-as-errors in CI, a `with` held open
inside an unexhausted generator, the `multiprocessing` start method, and `time.monotonic`
for durations. Five claims that were wrong: `slots=True` as a blanket default breaks
`cached_property` and weakrefs; "parse, do not validate" named no parser while the next
section made the plain dataclass the default, which validates nothing; the mutable-default
rule gave the dataclass remedy and not the `None` sentinel; `Protocol` was implied to be
unusable with `isinstance`; and `--strict` was required without saying that untyped
third-party imports are where it leaks. Four structural: the Go preamble's refusal
sentence was dropped, `Enum` and SQL binding were each stated twice or more, and a promise
to name no library sits above eight of them.

**`needs: 0071` because both edit `templates/AGENTS-section.md`.** 0071 adds a second route
through the protocol's rules and this changes the line that sends an agent to a checklist.
Trivial to merge textually and serialised anyway, because that file has one owner at a time.

## Tasks

- [ ] `Init` takes the entry-point glob and the keep-directory from the stack's own layout,
      so no Go filename appears in it — with the conformance cases `--stack go` already has
      still passing
- [ ] `templates/python-checklist.md` carries the corrected list: the audit refusal
      restored, the six omissions added, the five wrong claims fixed, and `Enum`, SQL
      binding and the library-neutrality promise reconciled
- [ ] `kit help python-checklist` prints it, and the protocol and the help overview send an
      agent to the checklist for its language — this project's `AGENTS.md` regenerated in
      the same commit as the template
- [ ] `kit init --stack python` scaffolds `pyproject.toml`, a Makefile whose `check` is the
      same gate, `src/{{name}}/` with `__init__.py` and `__main__.py`, a README and the
      stack's `.gitignore`
- [ ] `kit init --stack python --forge github` and `--forge gitlab` each write that forge's
      pipeline, and an unknown forge gets none and is told so
- [ ] `docs/TECH.md` says what a stack declares, in one line

## Open questions

- **Does one clause per stack hold past two?** Leaning: yes at two, and a third stack is
  what earns a `kit help checklists` that lists them. Adding it now would be a registry
  with one reader.
- **uv, or a venv and pip-tools?** Leaning: uv, because the alternative needs three tools
  to reach the same lockfile. Reversible — it is one Makefile and one CI file, and no doc
  outside this one names it.

## Log

- 2026-08-30 — drafted against a Python checklist reviewed before this doc existed, so the
  fifteen corrections are a task rather than a thing to discover while writing the template.
