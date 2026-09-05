# Publishing kit

The human reviews and merges the implementation PR before publication. A Submitted
feature records readiness for that review; it does not certify a released version.
The [compatibility contract](../templates/compatibility.md) defines the supported
upgrade path. All agents sharing a remote must upgrade together.

Before publishing v1.0.0, the release operator must have evidence that the remote
requires the project's check job and reserves merge authority for the human. Verify
both required checks and agent credentials, including any bypass permissions. Shared
human credentials cannot enforce this separation. An unavailable settings API or an
unauthenticated CLI leaves the condition pending. Record the verification in the PR;
a local bare-remote test cannot certify hosted permissions.

After the human merges, run these commands in the kit's trunk checkout:

```sh
git switch main
git pull --ff-only
git status --short
make check
make vuln
make audit
make install
kit version
make release TAG=v1.0.0
```

Require a clean tree and successful gates. `make install` updates the shared binary
from merged source. `make release` deliberately runs the complete CI gate again at the
publication boundary: check, vulnerability scan and strict online audit. It requires
HEAD to equal the advertised default-branch tip before and after validation, and tags
that captured commit. A failing or incomplete gate stops publication. The pre-release
`tag-stale` advisory is expected; it does not waive a HIGH finding.

The target accepts stable vX.Y.Z tags. A failed push retains its annotated local tag;
retry the same command from the same intended commit. It accepts an already published
tag only when the remote tag object is identical. A conflicting tag is never moved.
If trunk advances before a retry, stop and establish which commit and version the human
intends to release; do not delete a tag to make the command pass.

After publication, verify the distribution path using the exact tag:

```sh
go install github.com/sanyatihy/project-kit/cmd/kit@v1.0.0
command -v kit
kit version
test "$(kit version)" = "kit v1.0.0"
```

The private-module authentication setup is in [README](../README.md#install). Failure
to fetch or install the tagged module leaves installation verification pending even
if the tag was pushed. Fix the reported cause and retry; do not substitute an untagged
local build as evidence that installation by version works.

Record the published tag object, commit, gate results and exact installed version in
the release handoff. Refresh projects with `kit init`, review the protocol diff and run
strict online audit before resuming their writers. Customized Makefiles and CI need an
explicit comparison with the new scaffold; init preserves them.
