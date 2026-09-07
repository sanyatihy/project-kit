# TECH

The system as built: the facts that outlive the feature that produced them. Feature
docs in [features/](features/) describe a change and then freeze; this describes what
exists now, and must always be true.

## What it is

A Go CLI for managing bookmarks. Bookmarks live in one SQLite file; `bm list` reads them
locally. The layout is the usual Go module: command code at the root, no separate packages
to point at yet.

## The model

A bookmark is a row in the single SQLite database file. Titles are stored as given, except
the import boundary trims a trailing space before storing (see Gotchas). There is no other
state; nothing persists between runs except that one file.

## Decisions and why

- **SQLite over JSONL.** Atomic updates of individual rows without rewriting the file; the
  cost accepted is a native dependency in the toolchain.

## Gotchas

- **A trailing space in an imported title broke exact lookup.** The import boundary now trims
  it, so stored titles match the way lookups compare them.

## Verifying

`make check` passes and is the gate for this project. `bm list` is the local read command:
run it to see the bookmarks as stored.

## State of play

Cloud sync is absent; all reads and writes are local to the SQLite file. The next gap is
whatever work picks it up — nothing else is in flight.
