---
id: 0016
title: Make an unfinished doc fail the audit
status: Draft        # Draft | Accepted | In progress | Shipped | Dropped | Superseded
created: 2026-08-14
shipped:             # fill the date when status flips to Shipped
check:               # optional — date to check whether this worked. Only for bets.
checked:             # date you checked; write the outcome in ## Log
needs:               # nothing — this is templates and audit.go, and it lands first
related: docs/features/0009-fix-what-the-first-real-run-found.md
---

## Problem

The audit currently rewards not doing the work. Run `kit new`, change one character —
`status: Draft` to `status: Accepted` — and touch nothing else:

```
kit next                       kit audit
  0001  0/2   Untouched          tmpl
  → …                              docs and repo agree
  kit claim 0001
```

An agent is handed `…` as its instruction, beside the command to claim it. Nothing
reports anything, because from the tool's point of view nothing is wrong.

Then delete the one placeholder line the template puts under `## Log`, which is what a
careful author does first, and *that* produces the only finding:

```
MEDIUM  0001: ## Log is missing or empty
        fix: fill it in, or set status: Draft until you can
```

Both halves come from the same root cause: `missing-section` tests whether a section is
*empty*, and the template fills every section with placeholder prose, two `- [ ] …`
boxes, and a fake dated `## Log` entry. So the untouched doc looks complete and the
tidied one looks broken. This is the failure 0005 named — a check that fires on work
done right trains everyone to stop reading it — except inverted, which is worse: the
lazy path is green.

Two smaller versions of the same thing, both found in the same pass:

**`--fyi` does not mean FYI to `audit`.** `BlockersFor` correctly skips an FYI line, but
`inbox-stale` does not look at the marker. It is HIGH, so it fails `audit --strict`,
which is the gate rule 12 puts in front of `kit ship`. Worse, an inbox line carries a
date and no time and `InboxStaleDays` is 1, so a question filed at 23:00 is HIGH an hour
later. A note explicitly marked "work can continue without this" blocks shipping.

**`BACKLOG.md` is not scanned for secrets.** It is one of the three places rule 6 sends
discovered work, so agents write to it, and it is committed like everything else. It is
the only agent-writable doc the scan does not open.

## Non-goals

- **A prose-quality heuristic.** This check answers "was this section written at all",
  never "is it any good". `TBD` passes, and that is correct: judging content is the
  human's job at review, and the clever version is what 0005 ruled out.
- **Making `needs:` explicitly answered.** An empty `needs:` still cannot be told apart
  from a forgotten one. The honest fix requires a token like `needs: none` and would put
  all fifteen shipped docs in breach, so it goes to BACKLOG.md rather than shipping as a
  cosmetic half-measure.
- **Repo-wide secret scanning.** The boundary 0008 drew holds: `audit` opens the
  documents the repo model already knows about. `BACKLOG.md` is one of them and was
  simply missed. `README.md` and source files are still not this tool's business.
- **Retuning `stalled`, `no-claim` or anything git-shaped.** That is 0017.

## Design

### The template teaches in comments, because comments are already stripped

`ParseSections` strips HTML comments before storing a section, and `forEachBox` never
sees them either. So guidance inside `<!-- -->` is invisible to every count and every
check while staying in front of the author who opens the file. The template's visible
placeholders move inside the comments they belong to:

- `## Tasks` loses `- [ ] …` twice. A fresh doc then genuinely has zero boxes, so
  `no-tasks` fires on an unplanned feature instead of `kit next` handing out `…`.
- `## Problem`, `## Design` and `## Non-goals` lose their instruction sentences.
- `## Log` loses `- YYYY-MM-DD — decided X over Y because … (#PR)`.

This one change is what makes the untouched doc read as empty rather than as filled in,
and it is the reason the check below can be exact rather than fuzzy.

### `unfilled-template`, at HIGH

A new check: a doc that is **not** `Draft` whose `## Problem`, `## Design`, or
`## Tasks` is empty, or still matches the embedded template's own text for that section.

HIGH, and the severity is the point. `kit claim` does not consult `audit`, so this has
to fail `--strict` in CI to stop an agent acting on a doc nobody wrote. It cannot
produce a false positive: the comparison is against the template the same binary
embeds, after `Normalise` and a trim, and `Draft` is exempt because an unfinished draft
is the normal state of a draft.

The template comparison stays even though the comment change already makes an untouched
doc empty. They catch different mistakes: empty catches `kit new` left alone, and the
text match catches a doc copied by hand or a section pasted back, which rule 1 of
bootstrapping already tells agents not to do and which nothing until now detected.

### `missing-section` gets one job

It keeps `## Log` and loses `## Problem` and `## Design` to the check above, so no
finding overlaps another. It also stops firing until `Shipped`: an `Accepted` feature
has nothing to log yet, and demanding a log entry before work starts is what produced
the backwards fix text quoted in the problem statement.

### Staleness reads the marker

Split in two, because one severity cannot be right for both:

| check | fires on | after | severity |
|---|---|---|---|
| `inbox-stale` | a BLOCKING question | 2 days | HIGH |
| `fyi-stale` | an FYI note | 14 days | LOW |

`InboxStaleDays` moves from 1 to 2 because the format has a date and no time: at 1 the
threshold can elapse an hour after filing, and a check that fires before a human could
plausibly have answered is a check nobody believes. FYI gets a long fuse and a severity
that never fails a build — enough to say "you are not draining these", never enough to
stop work that was explicitly cleared to continue.

### Two one-line corrections

`BACKLOG.md` joins the documents `secret-in-doc` opens, via a `Backlog` field on `Repo`
resolved from the features directory the project actually uses.

`done-not-shipped` currently advises "set status: Shipped and the shipped: date" — by
hand, which is exactly what `kit ship` exists to prevent and what `unticked-shipped` and
`shipped-no-date` are the wreckage of. It should name the command.

## Tasks

- [ ] Move every visible placeholder in `templates/features-TEMPLATE.md` into its HTML
      comment, and assert a fresh `kit new` doc reports `0` task boxes and empty
      Problem/Design/Log sections.
- [ ] Add `unfilled-template` at HIGH over Problem/Design/Tasks for non-Draft docs,
      comparing against the embedded template; narrow `missing-section` to `## Log` at
      `Shipped`. Conformance: fires on an accepted untouched doc, quiet on a filled one,
      quiet on a Draft.
- [ ] Split `inbox-stale` (BLOCKING, 2 days, HIGH) from `fyi-stale` (FYI, 14 days, LOW),
      with a test that an FYI line does not fail `--strict`.
- [ ] Add `BACKLOG.md` to the secret scan and point `done-not-shipped` at `kit ship`;
      update the check list in README.md.

## Log

<!-- Dated, append-only conclusions. The argument lives in the PR. -->
