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
The engine is already forge-agnostic and nobody knew. Measured against a real private
project on a self-hosted GitLab — HTTPS remote, corporate credentials, a features
directory written by hand before the kit existed — the read commands work with no
changes at all. Output redacted, shape intact:

```
$ kit audit …
  MEDIUM  0001 … unknown status "Implemented"
          0002 … unknown status "Implemented"
  LOW     no-inbox, no-vision
$ kit next --offline …
  nothing open, but there are drafts:
    0003  …
```

Every finding is true, and `Implemented` is a two-line migration rather than a forge
problem. Nothing in the Go code calls `gh` or any forge API; `claim` is a ref
compare-and-swap that every git host implements, and `Trunk()` asks `origin/HEAD` rather
than assuming. The host did not resolve during the measurement, and `audit` finished in
0.087s by falling back to the mirror — the degradation `remoteHeads` was built for, on
the case it was built for.

One thing breaks, and only one. `stackLayout` hardcodes
`{"check.yml", ".github/workflows/check.yml"}`, so `kit init --stack go` writes a GitHub
Actions workflow into a GitLab repo, next to a working `.gitlab-ci.yml` it cannot see. It
will never run, and nothing reports it, because `audit` has no check that could know.

Two smaller gaps travel with it. `secret-in-doc` knows `ghp_` and `github_pat_` and
nothing about `glpat-`, so the one HIGH check that exists to catch a published
credential is blind on every GitLab project. And the protocol says "pull request" six
times, which is not wrong so much as untranslated.

Now, because the first project to adopt the kit from outside is on a GitLab nobody here
controls, and it is not the last.

## Non-goals
- **Per-instance CI content.** The measured project's pipeline pulls its build image from
  an internal registry mirror and sets `GOPRIVATE` to its own host — values no template
  can guess, and the kind of thing that differs between two GitLab instances even before
  it differs between two forges. This is a self-solving problem in the case that matters:
  a project that already has CI keeps it, because `place()` never overwrites. For a
  genuinely new project the kit writes a minimal file and says what needs filling in.
  Runner tags and registries are the project's business.
- **A settings file, or per-host profiles.** `Repo` states that layout is discovered so
  there is no config to drift, and that is load-bearing rather than decorative. The forge
  is materialised as a file on disk at init and never consulted again, so there is
  nothing to store and nothing to go stale. If per-instance CI values ever earn a
  mechanism, that is a different feature and it has to argue against this line first.
- **Detecting the forge on a read path.** `next` and `audit` must stay config-free and
  offline-capable. Detection runs in `init` only.
- **Varying the protocol wording by forge.** `ProtocolIsStale` compares a project's
  section against the single string the binary would write. Make that string
  forge-dependent and either forge detection leaks into `audit`, or every GitLab project
  reports `protocol-stale` for ever. The wording is made neutral once, for everyone.
- **Reading merge-request state from any forge API.** Unchanged from 0018: the kit knows
  docs and refs.
- **An `audit` check for a CI file belonging to the wrong forge.** It is checkable, and it
  is exactly the mess this feature stops creating — but not one it cleans up, and the two
  are different jobs. A repo mirrored to both forges carries both files legitimately, so
  the check would fire on a healthy project, which the vision names as worse than no
  check. `init` declining to write the wrong file is the whole fix.

## Design

**A ladder of positive evidence, not a guess about a hostname.** `DetectForge(root)`
returns `github`, `gitlab`, or unknown, taking the first rung that answers:

1. `.gitlab-ci.yml` exists → `gitlab`. `.github/workflows/` exists → `github`.
   Certain, because it is the project's own answer.
2. The remote URL's host: `github.com` → `github`; a host containing `gitlab` → `gitlab`.
3. The directory convention, via the host segment `ModulePath` already extracts.
   `~/work/gitlab.example.com/infrastructure/thing` yields `gitlab.example.com`. If the
   kit is willing to bet `go.mod` on this convention, the CI filename is not a bigger bet.
4. Nothing → unknown.

Hostname alone cannot separate a self-hosted GitLab from Gitea or Forgejo, which is why
rung 1 outranks rung 2. When rung 1 fires, the correct action is usually to write no CI
file at all: the project already has a better one than any template.

**Every rung requires positive evidence, rung 3 above all.** It reads the host segment
only when that segment names a forge it recognises — a dot and `gitlab` or `github` in
it. A path that yields `example.com` or `internal.corp` falls through to unknown rather
than defaulting, because rung 3 is the rung with the least behind it and a silent wrong
guess about CI is the failure this whole feature exists to stop. It earns its place
regardless: it is the only rung that answers on a fresh `git init` with no remote, which
is exactly the state where the alternative is stopping to ask a human.

**`stackLayout` becomes keyed by forge**, so `check.yml` goes to
`.github/workflows/check.yml` or `.gitlab-ci.yml` accordingly, and the go stack gains a
GitLab template mirroring the existing one — `make check`, `make vuln`, nothing else.

**Unknown writes no CI file and closes with an instruction.** A missing CI file is a far
better failure than a wrong one: it is visible, and nobody has to work out why the
pipeline never fires. This is the state a fresh `kit init` reaches, since it runs
`git init` itself and there is no remote yet.

The closing line is written for an agent, per 0013 and 0014 — the human is the fallback,
not the first responder, so the agent must be able to finish this alone:

```
  no CI file — nothing here says which forge this project is on.
  after git remote add, run kit init again and it will detect one.
  or say so now: kit init . --stack go --forge gitlab
```

Both routes are the agent's to take. This deliberately does not go to `docs/INBOX.md`:
the inbox is for decisions only a human can make, and the forge is a discoverable fact.
Filing it there would idle an agent on something it could have answered by reading
`git remote -v`.

**`--forge` overrides every rung**, and is the escape when detection is wrong or absent.

**`glpat-` and GitLab deploy-token shapes join the secret scan**, alongside the existing
GitHub prefixes.

**The protocol says it once, neutrally**: *a feature is one pull request (merge request
on GitLab)*. One mention, no branching, `ProtocolIsStale` stays single-valued.

## Tasks
- [ ] `DetectForge` in `internal/kit`, one unit test per rung including the unknown case
- [ ] `stackLayout` keyed by forge, with a `.gitlab-ci.yml` template for the go stack
- [ ] `kit init --forge github|gitlab` overriding detection, rejecting an unknown value
- [ ] Unknown forge writes no CI file, and `kit init` closes with both ways to resolve it
- [ ] Conformance: init on a repo with `.gitlab-ci.yml` writes no `.github/`, init with a
      `github.com` remote writes the workflow, init with neither writes no CI file
- [ ] GitLab token shapes in `secret-in-doc`, with a fires case and a stays-quiet case
- [ ] Forge-neutral wording in `templates/AGENTS-section.md`, `help.go`, `main.go`,
      `claim.go`; confirm `ProtocolIsStale` still reports nothing on a clean project
- [ ] Document detection and `--forge` in README.md, and the `init` topic in `kit help`

## Log
- 2026-08-15 — measured against a private project on a self-hosted GitLab: the read
  commands need no changes, so this is scoped to `init` rather than a port.
- 2026-08-15 — kept rung 3, narrowed to host segments that name a recognised forge.
  Dropping it would send every fresh `git init` to the human, which is the wrong default
  for a tool agents run first; letting it default on any dotted segment would guess.
- 2026-08-15 — decided against an `audit` check for a wrong-forge CI file. Dual-mirrored
  repos make it fire on healthy projects, and `init` not writing the file is the fix.
