package main

// Conformance: `kit next`. This is the command an agent runs first in every
// session, so what it hands back — and what it refuses to hand back — is the whole
// contract.

import (
	"fmt"
	"strings"
	"testing"
)

// feature writes a minimal but valid feature doc. The template's full shape is
// covered by the `new` suite; what matters here is status, ID, and task list.
func feature(p *project, id, slug, title, status, tasks string) {
	p.t.Helper()
	p.write("docs/features/"+id+"-"+slug+".md", fmt.Sprintf(
		`---
id: %s
title: %s
status: %s
created: 2026-01-01
---

## Problem
Something needs doing.

## Design
This way.

## Tasks
%s

## Log
- 2026-01-01 — started.
`, id, title, status, tasks))
}

func TestNextHandsBackTheTopmostUntickedBox(t *testing.T) {
	// List order is the order of work, which is what lets ordering need no syntax.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted",
		"- [x] Schema\n- [ ] The POST handler\n- [ ] The redirect")

	r := p.run("next").ok()
	r.has("start here", "0001", "The POST handler")
	r.lacks("The redirect")
}

func TestNextOffersInProgressBeforeAccepted(t *testing.T) {
	// Finishing beats starting, so a half-done feature outranks a fresh one even
	// though its ID is higher.
	p := setUp(t)
	feature(p, "0001", "first", "First", "Accepted", "- [ ] Do the first thing")
	feature(p, "0002", "second", "Second", "In progress", "- [ ] Do the second thing")

	out := p.run("next").ok().Stdout
	inProgress := strings.Index(out, "0002")
	accepted := strings.Index(out, "0001")
	if inProgress == -1 || accepted == -1 {
		t.Fatalf("both features should be listed:\n%s", out)
	}
	if inProgress > accepted {
		t.Errorf("Accepted 0001 was offered before In progress 0002:\n%s", out)
	}
}

func TestNextTreatsABranchAsAClaim(t *testing.T) {
	// The claim is a branch name and nothing else — no lock file, no state to sync.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.commitAll("add the feature doc")
	p.git("branch", "0001-shorten")

	r := p.run("next").ok()
	r.has("taken", "0001-shorten")
	// A taken feature must not also be offered as the thing to start.
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a claimed feature was still offered as work:\n%s", r.Stdout)
	}
}

func TestNextIsBlockedByAnOpenBlockingQuestion(t *testing.T) {
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md",
		"# Inbox\n\n- [ ] 2026-01-01 · 0001 · BLOCKING · Should codes expire?\n")

	r := p.run("next").ok()
	r.has("blocked", "waiting on", "Should codes expire?")
	if strings.Contains(r.Stdout, "start here") {
		t.Errorf("a blocked feature was offered as work:\n%s", r.Stdout)
	}
}

func TestNextIsNotBlockedByAnFYIQuestion(t *testing.T) {
	// FYI is recorded for a human without stopping the work, or every note would
	// halt a feature.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")
	p.write("docs/INBOX.md",
		"# Inbox\n\n- [ ] 2026-01-01 · 0001 · FYI · Using base62 for slugs.\n")

	p.run("next").ok().has("start here", "The POST handler")
}

func TestNextBlocksAFeatureWithEveryBoxTicked(t *testing.T) {
	// Nothing left to do is not the same as ready to work: it needs shipping or a
	// next task, and silently offering nothing would hide that.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [x] Done\n- [x] Also done")

	p.run("next").ok().has("blocked", "every task is ticked")
}

func TestNextExplainsDraftsRatherThanSayingNothingIsOpen(t *testing.T) {
	// `next` never offers a Draft — that is where a human settles the design — but
	// silence would look like there is no work at all.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Draft", "- [ ] The POST handler")

	r := p.run("next").ok()
	r.has("drafts", "0001", "Accepted")
	r.lacks("start here")
}

func TestNextWarnsOnlyAboveTheWIPLimit(t *testing.T) {
	const warning = "in progress at once"

	atLimit := setUp(t)
	feature(atLimit, "0001", "a", "A", "In progress", "- [ ] Task a")
	feature(atLimit, "0002", "b", "B", "In progress", "- [ ] Task b")
	atLimit.run("next").ok().lacks(warning)

	over := setUp(t)
	feature(over, "0001", "a", "A", "In progress", "- [ ] Task a")
	feature(over, "0002", "b", "B", "In progress", "- [ ] Task b")
	feature(over, "0003", "c", "C", "In progress", "- [ ] Task c")
	over.run("next").ok().has(warning)
}

func TestNextPrintsTheCommandThatStartsTheWork(t *testing.T) {
	// The output is meant to be actionable without reading the docs first.
	p := setUp(t)
	feature(p, "0001", "shorten", "Shorten URLs", "Accepted", "- [ ] The POST handler")

	p.run("next").ok().has(
		"git worktree add",
		"-b 0001-shorten",
		"docs/features/0001-shorten.md",
	)
}

func TestNextOnAnUnsetUpProjectSaysHowToFixIt(t *testing.T) {
	p := newProject(t)
	p.run("next").ok().has("not set up yet", "kit init")
}
