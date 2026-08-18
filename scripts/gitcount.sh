#!/bin/sh
# How many git processes does one board read cost?
#
# Builds a throwaway project with FEATURES features and CLAIMS claim branches, puts a
# counting wrapper ahead of git on PATH, and counts the processes one board read spawns.
#
# The regression it exists to catch has happened three times — an answer that cannot change
# inside one command, asked once per feature. Twice it was invisible until something took
# minutes; the third time it hid here, because this measured only `next --offline` and the
# duplication had moved to the checks that need the remote. So both paths are measured:
#
#   offline  `kit next --offline`, the number this script has always reported
#   online   `kit audit` against a local bare remote, every doc Shipped — the state that
#            makes the claim checks ask per feature. The remote is local so the round trip
#            stays one cheap process and what grows with the board is still the fan-out.
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
offline=$(wc -l < "$GITLOG" | tr -d ' ')

# The online half. A bare remote on disk answers ls-remote in a millisecond, so the only
# thing separating this number from the one above is what the checks ask per feature.
git init -q --bare "$work/origin"
git remote add origin "$work/origin"
git push -q origin HEAD
git fetch -q origin
sed -i.bak 's/^status: Accepted/status: Shipped/' docs/features/*.md
rm -f docs/features/*.bak
git add -A >/dev/null && git commit -qm "every feature shipped"
git push -q origin HEAD

: > "$GITLOG"
PATH="$work/bin:$PATH" "$KIT" audit >/dev/null
online=$(wc -l < "$GITLOG" | tr -d ' ')

printf '%s features, %s claims: %s git processes offline, %s online\n' \
	"$FEATURES" "$CLAIMS" "$offline" "$online"
awk '{print $3}' "$GITLOG" | sort | uniq -c | sort -rn | head -6
