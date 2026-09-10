# {{name}}

<!-- One or two sentences: what this is and who it is for. -->

## Development

```sh
make sync      # create uv.lock on first use; install its environment
make check     # fmt, lint, type, test — the gate CI runs
make run       # run it locally
make help      # every target
```

Use Python 3.12 and uv 0.12.10. Commit `uv.lock` after the first `make sync`; CI
uses `uv sync --locked` and refuses a missing or stale lockfile. Review lockfile
changes with dependency changes. The generated test checks both installed entry
points from outside the source tree; update it when the command changes.

## How work is tracked

Plans live in [docs/features/](docs/features/) — one file per feature, the directory
listing is the board.

```sh
kit next       # what to work on
kit audit      # where the docs and the repo disagree
```

Agents follow [AGENTS.md](AGENTS.md). Questions only a human can answer go in the
feature's `## Open questions`, with `kit block <id> "<question>"`.
