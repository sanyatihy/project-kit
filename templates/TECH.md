# TECH

The system as built. Feature docs in [features/](features/) describe a change and
then freeze; this describes what exists now, and must always be true.

<!-- When a feature ships, its durable facts move here and the feature doc becomes
     history. This file has no lifecycle and no status: it is either accurate or it
     is a bug. `kit audit` reports tech-drift when a feature ships and this file has
     not been touched since. -->

## What it is

<!-- Two or three sentences: what the system does, and the shape of it. Then the
     directory layout, one line each — enough that someone can find the code that
     handles a request without asking. -->

## The model

<!-- The nouns and their relationships: data model, state machine, the lifecycle of
     the main thing. Whatever someone has to hold in their head before a change is
     safe to make. -->

## Decisions and why

<!-- Recorded so they are not re-litigated. One bullet each: what was chosen, what
     it was chosen over, and the cost that was accepted. A decision without its cost
     reads as free and gets reversed by the next person who meets the cost. -->

## Gotchas

<!-- Each of these should have already caused a bug. Things that are surprising,
     invisible, or that look wrong and are not. Resist filling this in early — a
     gotcha nobody has hit yet is a guess. -->

## Verifying

<!-- The commands that prove it works: make check, how to run it locally, how to
     tell it is healthy in production. -->

## State of play

<!-- What is true today and what the next gap is. This section goes stale fastest,
     so keep it short and update it when a feature ships. -->
