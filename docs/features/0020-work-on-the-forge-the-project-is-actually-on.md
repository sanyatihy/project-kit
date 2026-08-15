---
id: 0020
title: Work on the forge the project is actually on
status: Draft
created: 2026-08-15
shipped:
check:
checked:
needs:
related: 0012-kit-init-refreshes-the-docs-it-owns.md
---

## Problem
`stackLayout` hardcodes `{"check.yml", ".github/workflows/check.yml"}`, so
`kit init --stack go` writes a GitHub Actions workflow into any project — including a
GitLab one that already has a working `.gitlab-ci.yml`. It never runs, and no check
reports it.

Everything else already works. Measured against a private project on a self-hosted
GitLab: `audit` and `next` were correct with no changes, because nothing calls a forge
API, `claim` is a ref compare-and-swap every git host implements, and `Trunk()` asks
`origin/HEAD`. The host did not resolve during the measurement and `audit` finished in
0.087s on the mirror fallback.

Two smaller gaps travel with it: `secret-in-doc` knows `ghp_` and `github_pat_` but not
`glpat-`, so the one HIGH credential check is blind on GitLab; and the protocol says
"pull request" untranslated.

## Non-goals
- **Per-instance CI content.** Internal registry mirrors and `GOPRIVATE` hosts differ
  between two GitLab instances, let alone two forges, and no template can guess them. A
  project that already has CI keeps it — `place()` never overwrites.
- **A settings file or per-host profiles.** `Repo` states that layout is discovered so
  there is no config to drift. The forge is materialised as a file at init and never
  consulted again, so there is nothing to store.
- **Detecting the forge on a read path.** `next` and `audit` stay config-free and
  offline-capable. Detection runs in `init` only.
- **Varying the protocol wording by forge.** `ProtocolIsStale` compares against the single
  string the binary would write; making it forge-dependent either leaks detection into
  `audit` or reports every GitLab project stale for ever.
- **An `audit` check for a wrong-forge CI file.** A repo mirrored to both forges carries
  both files legitimately, so it would fire on healthy projects. `init` declining to write
  the file is the fix.
- **Reading merge-request state from a forge API.** Unchanged from 0018.

## Design
`DetectForge(root)` returns `github`, `gitlab` or unknown, first rung that answers:

1. `.gitlab-ci.yml` or `.github/workflows/` exists — the project's own answer.
2. Remote host: `github.com`, or a host containing `gitlab`.
3. The host segment `ModulePath` already extracts from the directory layout, but only
   when it names a recognised forge. `internal.corp` falls through rather than defaulting.
4. Nothing → unknown.

Rung 1 outranks rung 2 because a hostname cannot separate self-hosted GitLab from Gitea.
Rung 3 is the only one that answers on a fresh `git init` with no remote — the state where
the alternative is stopping to ask a human.

`stackLayout` becomes keyed by forge, and the go stack gains a `.gitlab-ci.yml` template
mirroring the existing one. `--forge` overrides every rung.

**Unknown writes no CI file** and closes with an instruction the agent can act on, per
0013 and 0014 — a missing file is visible, a wrong one is not:

```
  no CI file — nothing here says which forge this project is on.
  after git remote add, run kit init again and it will detect one.
  or say so now: kit init . --stack go --forge gitlab
```

Not an inbox line: the inbox is for decisions only a human can make, and the forge is a
discoverable fact.

`glpat-` and GitLab deploy-token shapes join the secret scan. The protocol says it once,
neutrally: *a feature is one pull request (merge request on GitLab)*.

## Tasks
- [ ] `DetectForge` in `internal/kit`, one unit test per rung including unknown
- [ ] `stackLayout` keyed by forge, with a `.gitlab-ci.yml` template for the go stack
- [ ] `kit init --forge github|gitlab` overriding detection, rejecting an unknown value
- [ ] Unknown forge writes no CI file and closes with both ways to resolve it
- [ ] Conformance: `.gitlab-ci.yml` present writes no `.github/`; a `github.com` remote
      writes the workflow; neither writes no CI file
- [ ] GitLab token shapes in `secret-in-doc`, with a fires case and a stays-quiet case
- [ ] Forge-neutral wording in `AGENTS-section.md`, `help.go`, `main.go`, `claim.go`;
      confirm `ProtocolIsStale` still reports nothing on a clean project
- [ ] Document detection and `--forge` in README.md and the `init` help topic

## Log
- 2026-08-15 — measured against a private project on a self-hosted GitLab: the read
  commands need no changes, so this is scoped to `init` rather than a port.
- 2026-08-15 — kept rung 3, narrowed to host segments naming a recognised forge. Dropping
  it sends every fresh `git init` to a human; letting it default on any dotted segment
  guesses.
- 2026-08-15 — no `audit` check for a wrong-forge CI file: dual-mirrored repos make it
  fire on healthy projects.
