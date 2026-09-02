---
id: 0072
title: Python projects get the scaffold and standing guidance Go has
status: Shipped
created: 2026-08-30
shipped: 2026-08-31
needs:
---

## Problem

Go is the only stack `kit init` has and `kit help go-checklist` the only standing guidance
the binary carries, so a Python project gets the docs and then whatever practice the agent
remembers. Two decisions in `Init` are Go's and written as neither the stack's nor the
map's: it writes `internal/.gitkeep` for any stack, and skips the entry point by matching
the filename `main.go.tmpl`. A Python project would get a stray `internal/` and a duplicate
entry point.

## Non-goals

- A checklist `audit` enforces. Prose is judgement; a rule a check could make would be one.
- Recording which stack a project is. The kit stores nothing about a project beyond its
  files, so the protocol names both checklists and the agent picks.
- Generalising for a third stack. Two separates Go's parts of `Init` from every stack's.
- Rewriting the Go checklist. Nothing in the Python review carries over: Go's `time.Time`
  is monotonic and Go has no `-O`.
- Deployment scaffolding. The same refusal the go stack makes.

## Design

A stack declares the two things `Init` knew about Go: the glob that means the project
already has an entry point, and the directory that gets a `.gitkeep`. Go's are
`cmd/*/main.go` and `internal/`, Python's `src/*/__main__.py` and `tests/`. Both `if`s
leave `Init`, so a third stack is templates and a map entry.

The checklist is a template and a help topic, as Go's is. The protocol names one per
language rather than one outright, a clause per stack, which is what not storing a
project's stack costs.

uv pins the interpreter, locks, and makes the virtualenv in one binary, which keeps the
Makefile as thin as Go's. Cost: a tool outside the stdlib and younger than pip-tools, so
`make check` reports a missing uv by name rather than failing obscurely. `check` is format,
lint, type and test in both languages, so what the protocol says about a project's own gate
stays true.

The checklist ships corrected against fifteen review findings; the rules themselves live in
`templates/python-checklist.md`. Two were decisions rather than fixes. "Parse, do not
validate" named no parser while the next section made the plain dataclass the default,
which validates nothing — so a validating parser sits at the boundary and the frozen
dataclass inside it. And the preamble promised to name no library above eight of them.

That promise comes out, and naming is tiered by reversibility: development tooling flatly,
because none of it ships; the boundary library as a default with its exit — pydantic, or
msgspec where deserialisation is on the hot path — which is what closes "parse, do not
validate"; a framework only in the tier that uses it, carrying the constraint that keeps it
an adapter. Nothing where the standard library is adequate. Names sit in the tier sections
and principles in Core, so a name that ages costs a paragraph rather than a rule.

No `needs:`. The rounds ahead edit `templates/AGENTS-section.md` at the numbered rules and
the planning route; this changes the closing paragraph. The dependency test is a shared
dispatch that merges cleanly and breaks at runtime, which disjoint prose is not.

## Tasks

- [x] `Init` takes the entry-point glob and the keep directory from the stack's own layout
- [x] `templates/python-checklist.md` carries the corrected list
- [x] `kit help python-checklist` prints it, and the protocol names a checklist per language
- [x] `kit init --stack python` scaffolds pyproject, Makefile, `src/{{pkg}}/`, README, ignore
- [x] A pipeline per forge, and none for a forge nothing names
- [x] `docs/TECH.md` says what a stack declares

## Open questions

- Does a clause per stack hold past two? Leaning: yes, and a third stack earns
  `kit help checklists`. A registry with one reader is not one.
- uv, or a venv and pip-tools? Leaning: uv. Reversible — one Makefile and one CI file name
  it.

## Log

- 2026-08-30 — the library-neutrality promise is dropped rather than kept. The boundary
  library is the only thing that closes "parse, do not validate", so naming became a rule
  with four tiers.
- 2026-08-31 — `needs:` dropped. The test is whether the work could start, not what it
  touches.
- 2026-08-31 — the checklists take a ceiling of their own. Python's is 286 lines against
  Go's 193, and one number for every topic hands the other six 93 lines of headroom.
- 2026-08-31 — the package directory is `{{pkg}}`, not the `{{name}}` the box said.
  `url-shortener` cannot hold a package of that name.
