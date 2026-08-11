# Deployment patterns

Patterns for taking a project to production, and the traps behind them. Cloud-provider
agnostic — the specific droplets, buckets, and CLIs are yours to pick.

**The kit does not scaffold any of this**, on purpose: terraform, ansible, images, and
dashboards rot faster than anything else in a repo and are not needed to write code.
This is the knowledge, not the files.

**Read this when writing the feature doc that takes a project to production.** Going
to production is a feature like any other — it gets a doc, a design, and PR-sized
boxes. What follows is material for that doc's `## Design`, not a thing to do
directly.

## Three lifecycles, kept apart

Deployment is three jobs at three different rhythms, and merging them is the usual
mistake:

- **Provision** — create the infrastructure. Runs once, then rarely.
- **Configure** — install packages, firewall, runtimes on the host. Runs on change.
- **Deploy** — ship a new version of the app. Runs every release.

Each gets its own entry point. A deploy that can also re-provision is a deploy nobody
dares run on a Friday.

## Infrastructure state

- **A backend cannot create the bucket that holds its own state.** Break the cycle
  with a tiny separate bootstrap config that creates the state bucket, then `import`
  that bucket into the main config so the main config owns it. Bootstrap's local state
  becomes throwaway after the import, and losing it never means losing control of the
  bucket.
- **State is never committed.** Gitignore it. It contains resource ids and often
  secrets.
- **One tool owns each resource.** Resources created by hand in a web console are
  invisible to the tool that thinks it owns them, and will be clobbered or duplicated.
  If it is in the config, never touch it in the console.
- **Prefer a backend with native locking** over bolting on a separate lock table.

## Credentials

- **Scope every credential to what it does.** A key that writes to one bucket is a
  smaller incident than an account-wide token. Separate read from write where the
  provider allows it.
- **The host does not get repo access.** CI builds the artifact and pushes it; the
  host only pulls. A compromised host then cannot reach your source, and there is no
  deploy key to rotate on every machine.
- **Config in variables, secrets in secrets.** Region and instance size are not
  secret; conflating them means everything is write-only and nobody can see what is
  configured.
- Anything rendered onto a host — env files, compose files — is generated at deploy
  time from the secret store, never committed with values in it.

## Artifacts and rollback

- **Tag artifacts by immutable commit SHA**, not by `latest` or by branch. `latest` is
  a moving target that makes "what is actually running?" unanswerable.
- **Deploy takes a SHA as its input.** This is what makes rollback free: rolling back
  is deploying an older SHA through the identical path, not a special emergency
  procedure written under pressure.
- **Write the deploy logic once, as a reusable workflow**, and call it from both the
  post-merge path and the manual redeploy/rollback path. Two copies drift, and the
  copy you need in an incident is the one that was never exercised.

## CI shape

- **Gates separate from publish.** Lint, test, and vulnerability scan decide whether a
  commit is good; build and push decide what ships. Keep them distinct jobs so a
  publish cannot run on a red gate.
- **Filter jobs by what changed.** Compute the changed paths once, then run the Go
  gate only when Go changed and the infra plan only when infra changed. On a repo with
  terraform, ansible, compose, and dashboards, this is the difference between a
  two-minute CI and a twenty-minute one.
- **Infra changes are manual, not on push.** Provisioning runs on explicit dispatch
  with a plan step a human reads. Nothing that can destroy a database should trigger
  itself on merge.
- **Gate production behind an environment** with required reviewers, which also gives
  you a deployment history for free.
- **Never cancel an in-flight deploy.** Use a concurrency group so deploys queue
  rather than race, and do not cancel in progress — a half-applied deploy is worse
  than a slow one. Cancelling in-progress *gates* on a superseded PR is fine and
  saves money.

## Operations

- **Back up before you need to.** A scheduled dump to object storage, with rotation,
  set up on day one. Test the restore path at least once — an untested backup is a
  hypothesis.
- **Health endpoints split**: liveness (the process is up) separate from readiness
  (its dependencies are reachable). A single endpoint conflating them either restarts
  a healthy process or routes traffic to one that cannot serve.
- **Log and metric before scaling.** The instinct to add capacity usually precedes
  knowing what is slow.
