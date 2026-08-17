#!/bin/sh
# How many git processes does one board read cost?
#
# Builds a throwaway project with FEATURES features and CLAIMS claim branches, puts a
# counting wrapper ahead of git on PATH, and runs `kit next --offline` against it. Offline
# on purpose: the network call is one process whatever happens, and what this measures is
# the fan-out that grows with the number of features.
#
# The regression it exists to catch has happened twice — an answer that cannot change
# inside one command, asked once per feature. Both times it was invisible until something
# took minutes.
set -eu

FEATURES=${FEATURES:-30}
CLAIMS=${CLAIMS:-10}
KIT=$(cd "$(dirname "$0")/.." && pwd)/kit
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

"$KIT" init "$work/proj" >/dev/null
cd "$work/proj"
i=1
while [ "$i" -le "$FEATURES" ]; do
	id=$(printf '%04d' "$i")
	cat > "docs/features/$id-feature-$i.md" <<DOC
---
id: $id
title: Feature $i
status: Accepted
created: 2026-01-01
needs:
---

## Problem
x

## Design
y

## Tasks
- [ ] One
- [ ] Two

## Log
- 2026-01-01 — x.
DOC
	i=$((i + 1))
done
git add -A >/dev/null && git commit -qm "$FEATURES features"
i=1
while [ "$i" -le "$CLAIMS" ]; do
	git branch "$(printf '%04d' "$i")-feature-$i"
	i=$((i + 1))
done

mkdir -p "$work/bin"
cat > "$work/bin/git" <<'WRAP'
#!/bin/sh
echo "$@" >> "$GITLOG"
exec /usr/bin/git "$@"
WRAP
chmod +x "$work/bin/git"

GITLOG="$work/git.log"
export GITLOG
: > "$GITLOG"
PATH="$work/bin:$PATH" "$KIT" next --offline >/dev/null

printf '%s features, %s claims: %s git processes for one board read\n' \
	"$FEATURES" "$CLAIMS" "$(wc -l < "$GITLOG" | tr -d ' ')"
awk '{print $3}' "$GITLOG" | sort | uniq -c | sort -rn | head -6
