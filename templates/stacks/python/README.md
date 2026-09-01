# {{name}}

<!-- One or two sentences: what this is and who it is for. -->

## Development

```sh
make sync      # install the locked environment
make check     # fmt, lint, type, test — the gate CI runs
make run       # run it locally
make help      # every target
```

## How work is tracked

Plans live in [docs/features/](docs/features/) — one file per feature, the directory
listing is the board.

```sh
kit next       # what to work on
kit audit      # where the docs and the repo disagree
```

Agents follow [AGENTS.md](AGENTS.md). Questions only a human can answer go in
[docs/INBOX.md](docs/INBOX.md).
