---
id: 0085
title: The settled scope of 0083 reaches the trunk so the feature can be claimed
status: Submitted
created: 2026-09-10
submitted: 2026-09-10
needs:
---

## Problem

0083 merged carrying an unticked box in its `## Open questions`, which is what 0082 made
block a claim. Only the operator could clear it, and the answer has to be on the trunk
before anyone can take the feature — `start` refuses a blocked doc, so the answer cannot
be recorded while working it.

## Non-goals

- Changing 0083's design. The answer confirmed it; nothing about the plan moved.

## Design

The operator chose silence over nagging: a free feature that names no path is not
reported. Recorded as the tick plus one `## Log` entry naming what was settled and its
cost — the warning reaches 35 of the 83 docs here and no more, by choice, because a
finding on the other 48 is the noise `audit` must not make.

Direct work rather than a plan of its own: there was no decision to approve before doing
it, and the decision it records is the operator's.

## Tasks

- [x] record the operator's answer to the open question on the collision warning

## Log

<!-- What the doing taught that a plan would not have predicted: a premise falsified, an
     approach abandoned, a measurement that changed the shape. -->
