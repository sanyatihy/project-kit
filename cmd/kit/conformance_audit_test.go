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
	{"duplicate-id", func(p *project) {
		// Set up the way the race actually produces it: two agents off the same base
		// each take the next free ID, and the differing slugs mean the two files have
		// nothing for git to conflict on. The filename prefix matches the id in both,
		// so this is a clean duplicate and not an id-mismatch wearing a disguise.
		p.write("docs/features/0001-thing-as-well.md", doc("0001", "Thing as well",
			"Accepted", "", "- [ ] Something"))
	}},
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
	{"inbox-unknown", func(p *project) {
		p.write("docs/INBOX.md", fmt.Sprintf(
			"# Inbox\n\n- [ ] %s · 0099 · BLOCKING · About a feature that never existed?\n",
			daysAgo(1)))
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
	{"no-tasks", func(p *project) {
		feature(p, "0002", "empty", "Empty", "Accepted", "")
	}},
	{"no-claim", func(p *project) {
		feature(p, "0002", "working", "Working", "In progress", "- [ ] Something")
	}},
	{"split-claim", func(p *project) {
		feature(p, "0002", "contested", "Contested", "Accepted", "- [ ] Something")
		p.commitAll("the doc")
		p.git("branch", "0002-contested")
		p.git("branch", "0002-contested-again")
	}},
	{"orphan-claim", func(p *project) {
		p.commitAll("something")
		p.git("branch", "0099-long-gone")
	}},
	{"draft-claimed", func(p *project) {
		feature(p, "0002", "unsettled", "Unsettled", "Draft", "- [ ] Something")
		p.commitAll("the doc")
		p.git("branch", "0002-unsettled")
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
	{"secret-in-doc", func(p *project) {
		// The realistic path: an agent writes up a failed deploy and pastes what it
		// was holding into the Log.
		p.write("docs/features/0002-leaky.md", doc("0002", "Leaky thing", "Accepted", "",
			"- [ ] Do it\n\n## Log\n- 2026-01-02 — deploy failed with "+fakeSecrets[0].value))
	}},
	{"no-vision", func(p *project) {
		// A feature exists, so there is something that should have been decomposed
		// from a vision, and nothing to decompose it from.
		feature(p, "0002", "planned", "Planned thing", "Accepted", "- [ ] Do it")
		if err := os.Remove(p.path("docs/VISION.md")); err != nil {
			p.t.Fatalf("removing VISION.md: %v", err)
		}
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

// fakeSecrets are shape-correct and deliberately worthless: the payloads are runs of
// one character, so nothing here could ever be mistaken for a live credential.
var fakeSecrets = []struct {
	what  string
	value string
}{
	{"private key", "-----BEGIN RSA PRIVATE KEY-----"},
	{"github token", "ghp_" + strings.Repeat("A", 36)},
	{"github fine-grained", "github_pat_" + strings.Repeat("B", 30)},
	{"aws key id", "AKIA" + strings.Repeat("C", 16)},
	{"slack token", "xoxb-" + strings.Repeat("1", 12)},
	{"google api key", "AIza" + strings.Repeat("D", 35)},
	{"api key", "sk-ant-" + strings.Repeat("E", 24)},
	{"url with password", "postgres://app:hunter2@db.example.com/prod"},
}

func TestSecretInDocFiresOnEveryShape(t *testing.T) {
	for _, s := range fakeSecrets {
		t.Run(s.what, func(t *testing.T) {
			p := cleanProject(t)
			p.write("docs/INBOX.md", "- [ ] 2026-01-02 · 0001 · FYI · is "+s.value+" right?\n")
			if codes := p.auditCodes(); !slices.Contains(codes, "secret-in-doc") {
				t.Errorf("no finding for %s; audit reported %v", s.what, codes)
			}
		})
	}
}

func TestSecretInDocStaysQuietOnProseAboutSecrets(t *testing.T) {
	// Docs discuss credentials constantly — naming the variable is what the protocol
	// asks for. A check that fires on the correct behaviour would train people to
	// ignore it, and audit has nothing but being believed.
	p := cleanProject(t)
	p.write("docs/features/0002-talky.md", doc("0002", "Talky thing", "Accepted", "",
		"- [ ] Read GITHUB_TOKEN from the environment, never a ghp_ literal\n"+
			"- [ ] Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in CI, not here\n"+
			"- [ ] Connect with postgres://app@db.example.com/prod and a password from the vault\n"+
			"- [ ] Rotate the AIza-prefixed key and the sk-ant one after launch"))
	if codes := p.auditCodes(); slices.Contains(codes, "secret-in-doc") {
		t.Errorf("secret-in-doc fired on prose that only names credentials: %v", codes)
	}
}

func TestSecretInDocNeverEchoesTheValue(t *testing.T) {
	// Reporting the match would copy the secret into a terminal, a CI log, and
	// anything consuming --json — turning the finding into a second leak.
	p := cleanProject(t)
	secret := "ghp_" + strings.Repeat("A", 36)
	p.write("docs/INBOX.md", "- [ ] 2026-01-02 · 0001 · FYI · token "+secret+"\n")
	out := p.run("audit").Stdout + p.run("audit", "--json").Stdout
	if strings.Contains(out, secret) {
		t.Error("audit printed the credential it found")
	}
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

func TestNoVisionStaysQuietBeforeAnythingIsPlanned(t *testing.T) {
	// The baseline covers a project that has both. This pins the other half of the
	// rule: with no features there is nothing to decompose, so a fresh `kit init`
	// must not be greeted with a finding at the moment the tool is being judged.
	p := newProject(t)
	p.run("init").ok()
	if err := os.Remove(p.path("docs/VISION.md")); err != nil {
		t.Fatalf("removing VISION.md: %v", err)
	}
	if codes := p.auditCodes(); slices.Contains(codes, "no-vision") {
		t.Errorf("no-vision fired on a project with no features yet: %v", codes)
	}
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

// The claim checks read branch names, which is the one place this tool can
// generate noise about work that is perfectly fine. Each of these is a situation
// that must stay silent.
func TestClaimChecksDoNotFireOnOrdinaryWork(t *testing.T) {
	t.Run("an ordinary branch cites no feature", func(t *testing.T) {
		p := cleanProject(t)
		p.commitAll("something")
		p.git("branch", "fix-the-typo")
		p.git("branch", "spike/try-an-idea")
		if codes := p.auditCodes(); len(codes) > 0 {
			t.Errorf("ordinary branches were reported as claims: %v", codes)
		}
	})

	t.Run("a local branch and its remote are one claim", func(t *testing.T) {
		// origin/0001-thing and 0001-thing are the same claim. Counting them
		// separately would report split-claim on every pushed branch.
		p := cleanProject(t)
		p.commitAll("the doc")
		p.git("branch", "0001-thing")
		bare := p.path("remote.git")
		p.git("init", "--bare", "-q", bare)
		p.git("remote", "add", "origin", bare)
		p.git("push", "-q", "origin", "0001-thing")
		p.git("fetch", "-q", "origin")

		for _, code := range p.auditCodes() {
			if code == "split-claim" {
				t.Error("a pushed branch was reported as two claims")
			}
		}
	})

	t.Run("Accepted without a branch is not a missing claim", func(t *testing.T) {
		// Only In progress asserts that work is happening.
		p := cleanProject(t)
		for _, code := range p.auditCodes() {
			if code == "no-claim" {
				t.Error("an unstarted Accepted feature was reported as unclaimed")
			}
		}
	})
}

func TestSplitClaimIgnoresAShippedFeaturesLeftoverBranches(t *testing.T) {
	// Found by running the check against this repo: 0001-publish-the-kit and
	// 0001-ship both cite 0001, which is Shipped. Those are merged leftovers, not
	// two agents competing, and most projects never delete merged branches.
	p := cleanProject(t)
	p.write("docs/features/0002-done.md", doc("0002", "Done", "Shipped",
		"shipped: "+daysAgo(2), "- [x] Finished"))
	p.write("docs/TECH.md", "# As built\n")
	p.commitAll("ship it")
	p.git("branch", "0002-done")
	p.git("branch", "0002-done-follow-up")

	for _, code := range p.auditCodes() {
		if code == "split-claim" {
			t.Error("merged branches on a Shipped feature were reported as a split claim")
		}
	}
}

func TestTheThresholdsFireAtTheirBoundaryAndNotBefore(t *testing.T) {
	// Retuning a threshold is only real if the boundary moved with the constant.
	// Both are read from the package, so this keeps testing whatever they become.
	t.Run("inbox-stale", func(t *testing.T) {
		for _, tc := range []struct {
			age  int
			want bool
		}{{kit.InboxStaleDays - 1, false}, {kit.InboxStaleDays, true}} {
			p := cleanProject(t)
			p.write("docs/INBOX.md", fmt.Sprintf(
				"# Inbox\n\n- [ ] %s · 0001 · BLOCKING · Answer me?\n", daysAgo(tc.age)))

			if got := slices.Contains(p.auditCodes(), "inbox-stale"); got != tc.want {
				t.Errorf("a question %d day(s) unanswered: reported=%v, want %v",
					tc.age, got, tc.want)
			}
		}
	})

	t.Run("stalled", func(t *testing.T) {
		for _, tc := range []struct {
			idle int
			want bool
		}{{kit.StalledDays - 1, false}, {kit.StalledDays, true}} {
			p := cleanProject(t)
			feature(p, "0002", "stuck", "Stuck", "In progress", "- [ ] Never finished")
			// Midnight-aligned, because audit compares against today truncated to a
			// whole day. An mtime taken straight off the clock lands a few hours
			// short of the boundary and reads as a day younger than it is.
			when := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -tc.idle)
			if err := os.Chtimes(p.path("docs/features/0002-stuck.md"), when, when); err != nil {
				t.Fatalf("ageing the doc: %v", err)
			}

			if got := slices.Contains(p.auditCodes(), "stalled"); got != tc.want {
				t.Errorf("idle for %d day(s): reported=%v, want %v", tc.idle, got, tc.want)
			}
		}
	})
}

func TestDuplicateIDDoesNotFireOnDocsWithNoIDAtAll(t *testing.T) {
	// The quiet half. Docs missing an id: share the empty string, and grouping on
	// that would report them as duplicates of each other — a second, wrong finding
	// on top of the no-status one that is already telling the truth about them.
	p := cleanProject(t)
	for _, name := range []string{"0002-nameless", "0003-alsonameless"} {
		p.write("docs/features/"+name+".md",
			"---\ntitle: No id here\nstatus: Accepted\n---\n\n"+
				"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] y\n\n## Log\n- 2026-01-01 — x.\n")
	}

	for _, code := range p.auditCodes() {
		if code == "duplicate-id" {
			t.Error("two docs with no id: were reported as sharing one")
		}
	}
}

func TestDuplicateIDNamesBothDocs(t *testing.T) {
	// Naming one file is not actionable: the fix is to renumber a specific doc, and
	// you cannot pick which without seeing both.
	p := cleanProject(t)
	p.write("docs/features/0001-thing-as-well.md", doc("0001", "Thing as well",
		"Accepted", "", "- [ ] Something"))

	p.run("audit").ok().has("duplicate-id",
		"docs/features/0001-thing.md", "docs/features/0001-thing-as-well.md")
}

func TestInboxChecksStayQuietOnAWellFormedQuestion(t *testing.T) {
	// A question written by `kit block` against a real feature is the normal case
	// and must produce nothing.
	p := cleanProject(t)
	p.run("block", "0001", "Should this expire? Leaning: no.").ok()

	for _, code := range p.auditCodes() {
		if code == "inbox-unparsed" || code == "inbox-unknown" {
			t.Errorf("a well-formed question was reported as %q", code)
		}
	}
}
