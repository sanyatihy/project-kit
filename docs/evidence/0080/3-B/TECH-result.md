# TECH

Current system facts and constraints. [features/](features/) holds plans and history.

## What it is
A Go CLI that stores bookmarks in a single SQLite file. `bm list` is the local read command.

## The model
Bookmarks live in one SQLite database file; imported titles are trimmed at the import boundary, so stored titles carry no trailing space.

## Decisions and why
- SQLite over JSONL: atomic updates required; the accepted cost is a native dependency.

## Gotchas
- A trailing space in an imported title previously broke exact lookup; the import boundary trims it.

## Verifying
- `make check` passes.

## State of play
Cloud sync is absent.
