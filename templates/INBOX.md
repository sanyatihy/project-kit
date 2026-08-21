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

The marker decides what `kit audit` says, too. An unanswered `BLOCKING` question
is HIGH after two days, which fails `kit audit --strict` and so stops anything
shipping. An `FYI` is never aged: it was cleared to wait, so nothing owes it an
answer.
