# TECH

Current system facts and constraints. [features/](features/) holds plans and history.

## What it is
A Go CLI that stores bookmarks in a single SQLite file. `bm list` is the local read command for listing stored bookmarks.

## The model
Bookmarks are rows in one SQLite database file. Imported titles are trimmed of trailing whitespace at the import boundary before storage, so lookups match on the trimmed value.

## Decisions and why
- **SQLite over JSONL.** Chosen for atomic updates to the bookmark store; the accepted cost is a native dependency instead of plain files.
- **Trim trailing space in imported titles at the import boundary.** A trailing space in an imported title previously broke exact lookup, so the import boundary now removes it before storage.

## Gotchas
- Exact lookup fails on titles whose imported form carried a trailing space; the import boundary trims it, so stored values never do.

## Verifying
`make check` passes. `bm list` reads bookmarks locally.

## State of play
Cloud sync is absent.
