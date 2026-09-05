---
id: 0076
title: Operators can adopt kit v1.0 with reliable recovery and a stable upgrade contract
status: Draft
created: 2026-09-05
submitted:
needs: 0075
---


## Problem

Operators cannot rely on unattended work while an interrupted claim can strand its owner,
offline guidance invents remote state, and release retries need manual repair. Adopting
v1.0 also needs a stable compatibility contract and evidence that existing projects can
complete the whole workflow after upgrading, as required by VISION's autonomy and trust goals.

## Non-goals

A workflow service, identity database or new product capabilities. Markdown and git remain the state.
General source cleanup or clearing the backlog. Only misleading guidance and release blockers belong here.
Preventing raw git with prose. Human merge authority must be enforced by remote permissions.
Publishing from the implementation branch. Release follows human review and integration.

## Design

0075 supplies the lifecycle. This feature makes its recovery, upgrade and publication
contracts dependable enough to keep stable through v1.x. One implementation PR contains
the fixes and release evidence; the human approves this plan before implementation starts.

Claim acquisition uses a canonical branch derived only from the immutable feature id,
not its title or filename. A rename cannot create a second lock for the same feature.
Existing filename-based claims remain readable, revisable and releasable; any active
legacy claim for an id prevents a new acquisition. Migration requires upgrading all
writers sharing a remote before using the new claim namespace. Mixed old/new writers
cannot be promised mutual exclusion and must be excluded from the supported upgrade path.

A lost push acknowledgement means unknown ownership, not a lost race. Preserve durable
git evidence of the attempt and let retrying claim reconcile it without deleting another
agent's work. Evidence must distinguish competing attempts starting at the same commit;
matching the remote to HEAD alone cannot prove ownership. Failure before acceptance,
acceptance without acknowledgement, a competing winner and a failed worktree creation
must each have a safe command-driven recovery. Claim bookkeeping alone must not become
product work that prevents surrendering an otherwise untouched claim.

The board distinguishes known local changes, cached observations and unknown remote
state. Offline or failed reads cannot assert that work is unpushed, integrated or free.
The normal command sequence must remain usable without raw ref edits or false success.

v1.x keeps canonical commands, exit meanings, existing JSON fields and check identifiers,
and supported document fields compatible. Additive JSON fields are allowed; consumers
ignore unknown fields. New checks and stricter validation of unsafe states are documented
in release notes. Human-readable wording is not a machine interface. Legacy Shipped
history and ship/reset aliases remain supported through v1.x. The upgrade guide records
writer coordination, renamed flags and check identifiers from 0075, supported tool
versions, and how to verify the installed version before resuming work.

The proposed v1.0 support scope is macOS and Linux with the existing Go and Python stacks;
Windows is not a supported runtime in this release. Minimum tool versions are backed by
executed checks. Upgrade rehearsals preserve feature history, house rules, application
files and customized stack files. Restore this repo's truncated house rules from their
last intact version, retaining current gate commands. Protocol refresh recognizes the
actual heading, preserves the house-rule prefix, and tolerates quoted mentions of the
marker. Remove obsolete current guidance and align VISION with approval of planned work.

Release uses the complete CI gate, including vulnerability scanning and strict online
audit. Failed state reads refuse publication. Checks and tagging refer to the same clean,
approved trunk commit; a moving HEAD cannot silently change what is released. Retrying
a failed push may publish an existing local tag only when it identifies the intended
commit. Existing conflicting tags are never moved or overwritten. Successful publication
is verified against the remote, and installation by the exact version must report it.

Readiness evidence covers real command executions across separate clones, fresh Go and
Python gates on supported systems, and a disposable copy of financials for upgrade
rehearsal. The live financials checkout stays untouched and private project contents stay
out of this repo and its PR. Remote checks and agent permissions must demonstrably reserve
merge approval for the human. Missing credentials or remote access leave that release
condition explicitly pending; local fixtures do not certify live enforcement.

The implementation is ready for review when the tasks below pass and remaining external
release conditions are recorded in its PR. After the human merges, validate the merged
trunk, reinstall the shared binary, satisfy those conditions, publish v1.0.0 through the
release target, and verify installation. These are post-merge release steps, not unfinished
task boxes requiring an edit to a frozen feature. v1.0 is released only after publication
and installation verification; Submitted alone never claims that outcome.

## Tasks

- [ ] Restore and preserve house rules, remove obsolete workflow guidance, and align approval and documentation rules with regression coverage for protocol-section boundaries
- [ ] Make claim identity survive renames and recover interrupted acquisitions, with deterministic competing-clone tests and legacy-claim migration coverage
- [ ] Make offline and unavailable-remote handoffs report uncertainty without inventing progress or ownership, verified in text and JSON
- [ ] Document and verify the v1 compatibility and coordinated upgrade contract using legacy history, in-flight work and customized project files
- [ ] Make release run the complete gate, bind publication to the validated commit, and safely resume failed tag pushes under failure-injection tests
- [ ] Make fresh Go and Python projects pass their actual gates on the declared supported environments and document reproducible tool requirements
- [ ] Rehearse plan approval, independent claims, interruption and blocker recovery, review revisions, squash integration and dependency unlock across separate clones, plus initial MVP harvest and a disposable financials upgrade
- [ ] Record release-candidate evidence, remote enforcement verification or explicit pending conditions, and the exact post-merge v1.0.0 publication and installation procedure

## Log
