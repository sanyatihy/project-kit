#!/bin/sh
# Publish only the commit validated by the complete gate; retries preserve tag identity.
set -eu

die() { printf '%s\n' "$*" >&2; exit 1; }
tag=${TAG:-}
printf '%s\n' "$tag" | LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' || die 'TAG must be a stable version, e.g. v1.0.0'
[ "$#" -gt 0 ] || die 'release requires the CI gate list'
commit=$(git rev-parse --verify HEAD) || die 'cannot read HEAD'
ref=refs/tags/$tag

verify_source() {
    current=$(git rev-parse --verify HEAD) || die 'cannot read HEAD'
    [ "$current" = "$commit" ] || die 'HEAD moved during release; start again from the intended trunk commit'
    dirty=$(git status --porcelain) || die 'cannot read working tree state'
    [ -z "$dirty" ] || die 'uncommitted work: release requires a clean tree'
    advertised=$(git ls-remote origin HEAD) || die 'cannot read the published trunk'
    tip=$(printf '%s\n' "$advertised" | awk '$2 == "HEAD" {print $1}')
    [ "$tip" = "$commit" ] || die 'HEAD is not the published default-branch tip; release from the updated trunk'
}

verify_source
for gate in "$@"; do
    make "$gate"
done
verify_source

local_tag=$(git for-each-ref --format='%(objectname)' "$ref") || die 'cannot inspect local tags'
if [ -n "$local_tag" ]; then
    kind=$(git cat-file -t "$ref") || die 'cannot read existing tag'
    target=$(git rev-parse "$ref^{commit}") || die 'existing tag does not identify a commit'
    [ "$kind" = tag ] && [ "$target" = "$commit" ] || die 'existing tag conflicts with this release; it was not changed'
else
    git tag -a "$tag" "$commit" -m "$tag" || die 'tag creation failed; inspect the existing tag before retrying'
    local_tag=$(git rev-parse "$ref") || die 'cannot read the created tag'
fi

read_remote_tag() {
    advertised=$(git ls-remote origin "$ref") || die 'cannot verify remote tag; local tag retained for retry'
    remote_tag=$(printf '%s\n' "$advertised" | awk -v ref="$ref" '$2 == ref {print $1}')
}

read_remote_tag
if [ -n "$remote_tag" ]; then
    [ "$remote_tag" = "$local_tag" ] || die 'remote tag conflicts with this release; it was not changed'
else
    # A lost acknowledgement may still mean the exact tag was published.
    git push --force-with-lease="$ref:" origin "$local_tag:$ref" || :
    read_remote_tag
    [ "$remote_tag" = "$local_tag" ] || die 'publication was not confirmed; local tag retained, rerun the same release to retry'
fi
printf 'released %s at %s; verify: go install github.com/sanyatihy/project-kit/cmd/kit@%s\n' "$tag" "$commit" "$tag"
