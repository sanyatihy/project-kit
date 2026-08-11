package main

// Conformance: `kit audit`. Two halves, and the second matters more — a check that
// fires when it should not trains people to ignore the tool.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// cleanProject is a set-up project with one honest feature in it. Audit must find
// nothing here, and every case below starts from it so that whatever a case
// asserts is the thing the case introduced.
func cleanProject(t *testing.T) *project {
	t.Helper()
	p := newProject(t)
	p.run("init").ok()
	feature(p, "0001", "thing", "A thing", "Accepted", "- [ ] Do the thing")
	return p
}

func (p *project) auditCodes() []string {
	p.t.Helper()
	r := p.run("audit", "--json")
	if r.Code > 1 {
		p.t.Fatalf("audit --json failed: %s", r.context())
	}
	var payload map[string][]kit.Finding
	if err := json.Unmarshal([]byte(r.Stdout), &payload); err != nil {
		p.t.Fatalf("audit --json emitted invalid JSON: %v\n%s", err, r.Stdout)
	}
	var codes []string
	for _, findings := range payload {
		for _, f := range findings {
			codes = append(codes, f.Code)
		}
	}
	return codes
}

func daysAgo(n int) string {
	return time.Now().UTC().AddDate(0, 0, -n).Format("2006-01-02")
}

// commitAllAt commits with a specific date, for the checks that compare a doc's
// age in git against a date written in a doc.
func (p *project) commitAllAt(message, isoDate string) {
	p.t.Helper()
	p.git("add", "-A")
	cmd := exec.Command("git", "commit", "-q", "-m", message)
	cmd.Dir = p.Root
	cmd.Env = gitEnv(
		"GIT_AUTHOR_DATE="+isoDate+"T12:00:00Z",
		"GIT_COMMITTER_DATE="+isoDate+"T12:00:00Z",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		p.t.Fatalf("dated commit: %v\n%s", err, out)
	}
}

// auditCases is one setup per check code. Every entry must make its own code
// appear; TestEveryRegisteredCheckIsExercised proves none is missing.
var auditCases = []struct {
	code  string
	setUp func(p *project)
}{
	{"stalled", func(p *project) {
		feature(p, "0002", "stuck", "Stuck", "In progress", "- [ ] Never finished")
		old := time.Now().AddDate(0, 0, -40)
		if err := os.Chtimes(p.path("docs/features/0002-stuck.md"), old, old); err != nil {
			p.t.Fatalf("ageing the doc: %v", err)
		}
	}},
	{"unticked-shipped", func(p *project) {
		p.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
			"shipped: "+daysAgo(1), "- [ ] Not actually done"))
	}},
	{"inbox-stale", func(p *project) {
		p.write("docs/INBOX.md", fmt.Sprintf(
			"# Inbox\n\n- [ ] %s · 0001 · BLOCKING · Answer me?\n", daysAgo(10)))
	}},
	{"inbox-unparsed", func(p *project) {
		p.write("docs/INBOX.md",
			"# Inbox\n\n- [ ] we should decide about 0001 at some point\n")
	}},
	{"unchecked", func(p *project) {
		p.write("docs/features/0002-bet.md", doc("0002", "A bet", "Accepted",
			"check: "+daysAgo(5), "- [ ] Do the thing"))
	}},
	{"no-plan", func(p *project) {
		// Commits but no feature doc: what adoption failing looks like.
		if err := os.Remove(p.path("docs/features/0001-thing.md")); err != nil {
			p.t.Fatalf("removing the feature: %v", err)
		}
		p.commitAll("work with nothing written down")
	}},
	{"done-not-shipped", func(p *project) {
		feature(p, "0002", "done", "Done", "Accepted", "- [x] All finished")
	}},
	{"tech-drift", func(p *project) {
		p.write("docs/TECH.md", "# As built\n\nThe old description.\n")
		p.commitAllAt("describe the system", daysAgo(60))
		p.write("docs/features/0002-newer.md", doc("0002", "Newer", "Shipped",
			"shipped: "+daysAgo(2), "- [x] Done"))
	}},
	{"shipped-no-date", func(p *project) {
		feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	}},
	{"missing-section", func(p *project) {
		p.write("docs/features/0002-hollow.md",
			"---\nid: 0002\ntitle: Hollow\nstatus: Accepted\n---\n\n"+
				"## Problem\nReal.\n\n## Design\n\n## Tasks\n- [ ] Something\n\n## Log\n- 2026-01-01 — x.\n")
	}},
	{"id-mismatch", func(p *project) {
		p.write("docs/features/0002-confused.md", doc("0009", "Confused", "Accepted",
			"", "- [ ] Something"))
	}},
	{"no-status", func(p *project) {
		p.write("docs/features/0002-statusless.md",
			"---\nid: 0002\ntitle: Statusless\n---\n\n"+
				"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] y\n\n## Log\n- 2026-01-01 — x.\n")
	}},
	{"bad-status", func(p *project) {
		feature(p, "0002", "odd", "Odd", "Marinating", "- [ ] Something")
	}},
	{"no-protocol", func(p *project) {
		if err := os.Remove(p.path("AGENTS.md")); err != nil {
			p.t.Fatalf("removing AGENTS.md: %v", err)
		}
	}},
	{"dead-link", func(p *project) {
		p.write("docs/features/0002-linky.md", doc("0002", "Linky", "Accepted", "",
			"- [ ] Read [the design](./nowhere.md)"))
	}},
	{"no-tech", func(p *project) {
		feature(p, "0002", "shipped", "Shipped thing", "Shipped", "- [x] Done")
		p.write("docs/features/0002-shipped.md", doc("0002", "Shipped thing", "Shipped",
			"shipped: "+daysAgo(1), "- [x] Done"))
	}},
	{"no-inbox", func(p *project) {
		if err := os.Remove(p.path("docs/INBOX.md")); err != nil {
			p.t.Fatalf("removing INBOX.md: %v", err)
		}
	}},
}

// doc writes a feature doc with an extra frontmatter line, for the fields feature()
// does not carry.
func doc(id, title, status, extra, tasks string) string {
	if extra != "" {
		extra += "\n"
	}
	return fmt.Sprintf(`---
id: %s
title: %s
status: %s
created: 2026-01-01
%s---

## Problem
Something needs doing.

## Design
This way.

## Tasks
%s

## Log
- 2026-01-01 — started.
`, id, title, status, extra, tasks)
}

func TestAuditEveryCheckFires(t *testing.T) {
	for _, tc := range auditCases {
		t.Run(tc.code, func(t *testing.T) {
			p := cleanProject(t)
			tc.setUp(p)
			if codes := p.auditCodes(); !slices.Contains(codes, tc.code) {
				t.Errorf("audit did not report %q; it reported %v", tc.code, codes)
			}
		})
	}
}

func TestAuditIsQuietOnACleanProject(t *testing.T) {
	// The half that matters. Every case above starts from this project, so a noisy
	// baseline would make all of them meaningless.
	p := cleanProject(t)
	if codes := p.auditCodes(); len(codes) > 0 {
		t.Errorf("audit reported %v on a project with nothing wrong", codes)
	}
	p.run("audit").ok().has("docs and repo agree")
}

func TestEveryRegisteredCheckIsExercised(t *testing.T) {
	// AGENTS.md requires a test that a new check fires and one that it stays quiet.
	// This is what makes that enforceable rather than aspirational.
	covered := map[string]bool{}
	for _, tc := range auditCases {
		covered[tc.code] = true
	}
	for _, c := range kit.Checks {
		if !covered[c.Code] {
			t.Errorf("check %q is in the registry but no conformance case fires it", c.Code)
		}
	}
	for code := range covered {
		if !slices.ContainsFunc(kit.Checks, func(c kit.Check) bool { return c.Code == code }) {
			t.Errorf("conformance case %q is not a real check", code)
		}
	}
}

func TestAuditNamesTheFileTheProblemAndTheFix(t *testing.T) {
	// A finding nobody can act on is noise.
	p := cleanProject(t)
	feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")

	r := p.run("audit").ok()
	r.has("shipped-no-date", "docs/features/0002-undated.md", "fix:")
}

func TestAuditNeverWrites(t *testing.T) {
	// An auditor that fixes things stops being read.
	p := cleanProject(t)
	feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	before := p.read("docs/features/0002-undated.md")

	p.run("audit").ok()

	if after := p.read("docs/features/0002-undated.md"); after != before {
		t.Error("audit modified a document it was only meant to report on")
	}
}

func TestAuditStrictExitsOnHighOnly(t *testing.T) {
	// --strict is the CI gate, so the boundary between severities is a contract.
	high := cleanProject(t)
	high.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
		"shipped: "+daysAgo(1), "- [ ] Not actually done"))
	high.run("audit", "--strict").fails(1)

	// shipped-no-date is MEDIUM: reported, but not a build failure.
	medium := cleanProject(t)
	feature(medium, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	r := medium.run("audit", "--strict").ok()
	if !strings.Contains(r.Stdout, "shipped-no-date") {
		t.Error("--strict swallowed a MEDIUM finding instead of reporting it")
	}

	cleanProject(t).run("audit", "--strict").ok()
}
