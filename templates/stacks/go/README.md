# {{name}}

<!-- One or two sentences: what this is and who it is for. -->

## Development

```sh
make check     # fmt, lint (including vet), test — the gate CI runs
make run       # run it locally
make help      # every target
```

Use the Go version recorded in `go.mod` and install golangci-lint v2.12.2
before running the gate. CI installs that same linter version:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

Put the Go binary directory on PATH so `make check` can find the linter.

## How work is tracked

Plans live in [docs/features/](docs/features/) — one file per feature, the directory
listing is the board.

```sh
kit next       # what to work on
kit audit      # where the docs and the repo disagree
```

Agents follow [AGENTS.md](AGENTS.md). Questions only a human can answer go in
[docs/INBOX.md](docs/INBOX.md).
