# Inbox

Questions an agent cannot answer alone. Agents append, a human drains.
Delete a line once it is resolved — git keeps the history.

Format, one per line:

```
- [ ] YYYY-MM-DD · <feature-id> · BLOCKING|FYI · <question, with your leaning>
```

State a leaning. It turns most questions into a yes/no.

`BLOCKING` stops `kit next` offering that feature until you answer. `FYI` is
recorded for you to see but does not block — use it for anything work can
continue without.

- [ ] 2026-08-11 · 0002 · FYI · The conformance suite will be rewritten as a Go test
      in `cmd/kit`, not restored as the old external Python script. Leaning: go
      ahead — an untracked script outside the build is what let it vanish unnoticed.
