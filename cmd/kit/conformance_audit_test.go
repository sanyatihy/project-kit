package main

// Conformance: `kit audit`. Two halves, and the second matters more — a check that
// fires when it should not trains people to ignore the tool.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	writeVision(p)
	feature(p, "0001", "thing", "A thing", "Draft", "- [ ] Do the thing")
	return p
}

// writeVision answers the stub `kit init` leaves behind. A project still carrying that
// stub is a real finding — unfilled-vision — so a fixture that keeps it is not a
// project with nothing wrong.
func writeVision(p *project) {
	p.t.Helper()
	p.write("docs/VISION.md", "# VISION\n\n## What this is\nA thing that does the thing.\n")
}

// visionStub puts back the stub `kit init` writes, rather than an imitation of it, so
// the cases below run against the template as it actually ships.
func visionStub(p *project) {
	p.t.Helper()
	if err := os.Remove(p.path("docs/VISION.md")); err != nil {
		p.t.Fatalf("removing VISION.md: %v", err)
	}
	p.run("init").ok()
}

func (p *project) auditCodes(flags ...string) []string {
	p.t.Helper()
	r := p.run(append([]string{"audit", "--json"}, flags...)...)
	if r.Code > 1 {
		p.t.Fatalf("audit --json failed: %s", r.context())
	}
	var payload audited
	if err := json.Unmarshal([]byte(r.Stdout), &payload); err != nil {
		p.t.Fatalf("audit --json emitted invalid JSON: %v\n%s", err, r.Stdout)
	}
	var codes []string
	for _, f := range payload.Findings {
		codes = append(codes, f.Code)
	}
	return codes
}

func daysAgo(n int) string {
	return time.Now().UTC().AddDate(0, 0, -n).Format("2006-01-02")
}

// commitAllAt commits with a specific date, for the checks that compare a doc's
// age in git against a date written in a doc.
// abandonedClaim is what `stalled` now looks for: a doc on the trunk, a claim branch
// carrying work beyond the doc, and no commit citing it since. The status says nothing —
// there is no longer one that could.
func abandonedClaim(p *project, id, slug string, daysOld int) {
	p.t.Helper()
	feature(p, id, slug, "Stuck", "Draft", "- [ ] Never finished")
	p.commitAllAt("the plan", daysAgo(daysOld))
	trunk := p.headBranch()
	p.git("checkout", "-q", "-b", id+"-"+slug)
	p.write("internal/"+slug+"/work.go", "package "+slug+"\n")
	p.commitAllAt(id+": started and left", daysAgo(daysOld))
	p.git("checkout", "-q", trunk)
}

// tagThenCommits releases the trunk and moves it on, which is what the distance between
// a tag and the trunk is made of. Empty commits: what the commits carry is not the
// measurement, and a doc in each of them would fire every other check as well.
func tagThenCommits(p *project, tag string, n int) {
	p.t.Helper()
	p.commitAll("the plan")
	p.git("tag", "-a", tag, "-m", tag)
	for i := 0; i < n; i++ {
		p.git("commit", "-q", "--allow-empty", "-m", fmt.Sprintf("work %d", i))
	}
}

func (p *project) commitAllAt(message, isoDate string) {
	p.t.Helper()
	p.git("add", "-A")
	cmd := exec.Command("git", "commit", "-q", "-m", message)
	cmd.Dir = p.Root
	// Midnight, because `audit` compares against today truncated to a whole day: a commit
	// stamped at noon N days ago is N-and-a-half days from midnight today, which floors to
	// N-1 and reads a day younger than it is. Every threshold test would sit one day off.
	cmd.Env = gitEnv(
		"GIT_AUTHOR_DATE="+isoDate+"T00:00:00Z",
		"GIT_COMMITTER_DATE="+isoDate+"T00:00:00Z",
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
			"Draft", "", "- [ ] Something"))
	}},
	{"duplicate-key", func(p *project) {
		// One doc, two statuses. The parser keeps the last and every writer rewrote the
		// first, so the finding used to survive its own fix.
		p.write("docs/features/0002-twice.md",
			"---\nid: 0002\ntitle: Twice\nstatus: Shipped\nstatus: Draft\n"+
				"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n\n"+
				"## Tasks\n- [ ] Do it\n\n## Log\n- 2026-01-01 — started.\n")
	}},
	{"stalled", func(p *project) {
		abandonedClaim(p, "0002", "stuck", 40)
	}},
	{"shipped-not-pushed", func(p *project) {
		// The state four features reached in one session: Shipped in the doc, committed
		// here, and on no remote. A remote has to exist for the question to have an
		// answer at all — with none, the kit says nothing about where work is not.
		origin := p.t.TempDir()
		gitIn(p.t, origin, "init", "-q", "--bare", ".")
		p.git("remote", "add", "origin", origin)
		p.write("docs/features/0002-done.md", doc("0002", "Done", "Shipped",
			"shipped: "+daysAgo(1), "- [x] Built it"))
		p.commitAll("ship 0002 and tell nobody")
	}},
	{"unticked-shipped", func(p *project) {
		p.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
			"shipped: "+daysAgo(1), "- [ ] Not actually done"))
	}},
	{"inbox-stale", func(p *project) {
		p.write("docs/INBOX.md", fmt.Sprintf(
			"# Inbox\n\n- [ ] %s · 0001 · BLOCKING · Answer me?\n", daysAgo(10)))
	}},
	{"fyi-stale", func(p *project) {
		p.write("docs/INBOX.md", fmt.Sprintf(
			"# Inbox\n\n- [ ] %s · 0001 · FYI · Something you may want to know?\n",
			daysAgo(kit.FYIStaleDays+1)))
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
	{"no-plan", func(p *project) {
		// Commits but no feature doc: what adoption failing looks like.
		if err := os.Remove(p.path("docs/features/0001-thing.md")); err != nil {
			p.t.Fatalf("removing the feature: %v", err)
		}
		p.commitAll("work with nothing written down")
	}},
	{"done-not-shipped", func(p *project) {
		// Committed, because the gate is now whether HEAD carries the doc: a plan still
		// being written in a working tree is nobody's defect.
		feature(p, "0002", "done", "Done", "Draft", "- [x] All finished")
		p.commitAll("a feature with every box ticked")
	}},
	{"tech-drift", func(p *project) {
		p.write("docs/TECH.md", "# As built\n\nThe old description.\n")
		p.commitAllAt("describe the system", daysAgo(60))
		p.write("docs/features/0002-newer.md", doc("0002", "Newer", "Shipped",
			"shipped: "+daysAgo(2), "- [x] Done"))
	}},
	{"tag-stale", func(p *project) {
		// Released once and kept merging. An install still resolves to the tag, so the
		// distance is exactly the work it does not carry.
		tagThenCommits(p, "v0.1.0", kit.TagStaleCommits+1)
	}},
	{"shipped-no-date", func(p *project) {
		feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	}},
	{"shipped-bad-date", func(p *project) {
		// Nothing validated the field, so `tech-drift` compared against it: a day that
		// has not arrived is never older than the as-built doc, and that finding then
		// had no fix for as long as the date said.
		p.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
			"shipped: 2099-01-01", "- [x] Done"))
	}},
	{"id-mismatch", func(p *project) {
		p.write("docs/features/0002-confused.md", doc("0009", "Confused", "Draft",
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
		// Prose where the boxes should be. A ## Tasks section that is empty outright is
		// unfilled-template's, so this is the narrower case: somebody wrote about the
		// work without ever writing a box an agent can do.
		p.write("docs/features/0002-empty.md",
			"---\nid: 0002\ntitle: Empty\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
				"## Problem\nSomething needs doing.\n\n## Design\nThis way.\n\n"+
				"## Tasks\nWe will figure the boxes out later.\n\n## Log\n- 2026-01-01 — started.\n")
		p.commitAll("a plan with no boxes in it")
	}},
	{"unreadable-doc", func(p *project) {
		// A doc the kit cannot open. It used to be dropped on the floor by readFeature,
		// so no command mentioned it again and every check it should have failed was
		// skipped rather than reported.
		p.write("docs/features/0002-locked.md",
			doc("0002", "Locked", "Draft", "", "- [ ] Do it"))
		makeUnreadable(p, "docs/features/0002-locked.md")
	}},
	{"unfilled-template", func(p *project) {
		// `kit new` and then nothing, committed. The template's guidance is all comments,
		// so every section it did not write reads as empty; committing it is what turns a
		// doc somebody is still writing into a plan the repo now claims to have.
		p.run("new", "Never written").ok()
		p.commitAll("commit a doc nobody wrote")
	}},
	{"split-claim", func(p *project) {
		feature(p, "0002", "contested", "Contested", "Draft", "- [ ] Something")
		p.commitAll("the doc")
		p.git("branch", "0002-contested")
		p.git("branch", "0002-contested-again")
	}},
	{"orphan-claim", func(p *project) {
		p.commitAll("something")
		p.git("branch", "0099-long-gone")
	}},
	{"no-protocol", func(p *project) {
		if err := os.Remove(p.path("AGENTS.md")); err != nil {
			p.t.Fatalf("removing AGENTS.md: %v", err)
		}
	}},
	{"dead-link", func(p *project) {
		p.write("docs/features/0002-linky.md", doc("0002", "Linky", "Draft", "",
			"- [ ] Read [the design](./nowhere.md)"))
	}},
	{"no-tech", func(p *project) {
		feature(p, "0002", "shipped", "Shipped thing", "Shipped", "- [x] Done")
		p.write("docs/features/0002-shipped.md", doc("0002", "Shipped thing", "Shipped",
			"shipped: "+daysAgo(1), "- [x] Done"))
	}},
	{"needs-unknown", func(p *project) {
		p.write("docs/features/0002-hopeful.md",
			doc("0002", "Hopeful", "Draft", "needs: 0099", "- [ ] Do it"))
	}},
	{"needs-cycle", func(p *project) {
		p.write("docs/features/0002-chicken.md",
			doc("0002", "Chicken", "Draft", "needs: 0003", "- [ ] Do it"))
		p.write("docs/features/0003-egg.md",
			doc("0003", "Egg", "Draft", "needs: 0002", "- [ ] Do it"))
	}},
	{"shipped-early", func(p *project) {
		// 0001 from cleanProject is a Draft, so this shipped ahead of it.
		p.write("docs/features/0002-ahead.md",
			doc("0002", "Ahead", "Shipped", "needs: 0001\nshipped: "+daysAgo(1), "- [x] Done"))
	}},
	{"secret-in-doc", func(p *project) {
		// The realistic path: an agent writes up a failed deploy and pastes what it
		// was holding into the Log.
		p.write("docs/features/0002-leaky.md", doc("0002", "Leaky thing", "Draft", "",
			"- [ ] Do it\n\n## Log\n- 2026-01-02 — deploy failed with "+fakeSecrets[0].value))
	}},
	{"protocol-stale", func(p *project) {
		// A project scaffolded by an older kit: agents there follow superseded rules
		// and nothing inside the project can tell.
		p.write("AGENTS.md", "# House rules\n\n## Work protocol\n\nAn older protocol.\n")
	}},
	{"no-vision", func(p *project) {
		// A feature exists, so there is something that should have been decomposed
		// from a vision, and nothing to decompose it from.
		feature(p, "0002", "planned", "Planned thing", "Draft", "- [ ] Do it")
		if err := os.Remove(p.path("docs/VISION.md")); err != nil {
			p.t.Fatalf("removing VISION.md: %v", err)
		}
	}},
	{"unfilled-vision", func(p *project) {
		// A fresh `kit init` and nothing else. The template's guidance is all
		// comments, so every section under its headings reads as empty.
		visionStub(p)
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
	{"gitlab personal token", "glpat-" + strings.Repeat("F", 24)},
	{"gitlab deploy token", "gldt-" + strings.Repeat("G", 24)},
	{"gitlab runner token", "glrt-" + strings.Repeat("H", 24)},
	{"gitlab ci job token", "glcbt-" + strings.Repeat("I", 24)},
	{"aws key id", "AKIA" + strings.Repeat("C", 16)},
	{"slack token", "xoxb-" + strings.Repeat("1", 12)},
	{"google api key", "AIza" + strings.Repeat("D", 35)},
	{"api key", "sk-ant-" + strings.Repeat("E", 24)},
}

func TestNeedsChecksStayQuietOnAValidGraph(t *testing.T) {
	t.Parallel()
	// The shape every real project has: a chain, a diamond, and a feature depending
	// on nothing. None of it is a defect, and a check that fired here would fire on
	// almost every project that used the field at all.
	p := cleanProject(t)
	p.write("docs/features/0001-thing.md",
		doc("0001", "A thing", "Shipped", "shipped: "+daysAgo(2), "- [x] Done"))
	p.write("docs/features/0002-left.md",
		doc("0002", "Left", "Draft", "needs: 0001", "- [ ] Do it"))
	p.write("docs/features/0003-right.md",
		doc("0003", "Right", "Draft", "needs: 0001", "- [ ] Do it"))
	p.write("docs/features/0004-join.md",
		doc("0004", "Join", "Draft", "needs: 0002, 0003", "- [ ] Do it"))
	p.write("docs/features/0005-alone.md",
		doc("0005", "Alone", "Draft", "", "- [ ] Do it"))

	for _, code := range []string{"needs-unknown", "needs-cycle", "shipped-early"} {
		if codes := p.auditCodes(); slices.Contains(codes, code) {
			t.Errorf("%s fired on a valid graph: %v", code, codes)
		}
	}
}

func TestSecretInDocFiresOnEveryShape(t *testing.T) {
	t.Parallel()
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

func TestSecretInDocScansEveryDocAgentsWriteTo(t *testing.T) {
	t.Parallel()
	// Rule 6 sends discovered work to BACKLOG.md, so agents write to it as freely as to
	// a feature doc, and it is committed like everything else. It was the one
	// agent-writable document the scan did not open.
	for _, where := range []string{
		"docs/features/BACKLOG.md",
		"docs/INBOX.md",
		"docs/VISION.md",
		"docs/TECH.md",
	} {
		t.Run(where, func(t *testing.T) {
			p := cleanProject(t)
			p.write(where, "# Notes\n\nit failed with "+fakeSecrets[0].value+"\n")
			if codes := p.auditCodes(); !slices.Contains(codes, "secret-in-doc") {
				t.Errorf("a credential in %s went unreported; audit said %v", where, codes)
			}
		})
	}
}

func TestSecretInDocStaysQuietOnProseAboutSecrets(t *testing.T) {
	t.Parallel()
	// Docs discuss credentials constantly — naming the variable is what the protocol
	// asks for. A check that fires on the correct behaviour would train people to
	// ignore it, and audit has nothing but being believed.
	p := cleanProject(t)
	p.write("docs/features/0002-talky.md", doc("0002", "Talky thing", "Draft", "",
		"- [ ] Read GITHUB_TOKEN from the environment, never a ghp_ literal\n"+
			"- [ ] Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in CI, not here\n"+
			"- [ ] Connect with postgres://app@db.example.com/prod and a password from the vault\n"+
			// The doc that actually broke this: prose quoting a userinfo URL in order
			// to say such URLs are rejected. Correct documentation, failed at HIGH.
			"- [ ] Reject userinfo URLs (https://user:pw@example.com is a phishing shape)\n"+
			"- [ ] Rotate the AIza-prefixed key and the sk-ant one after launch"))
	if codes := p.auditCodes(); slices.Contains(codes, "secret-in-doc") {
		t.Errorf("secret-in-doc fired on prose that only names credentials: %v", codes)
	}
}

func TestSecretInDocNeverEchoesTheValue(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	// The half that matters. Every case above starts from this project, so a noisy
	// baseline would make all of them meaningless.
	p := cleanProject(t)
	if codes := p.auditCodes(); len(codes) > 0 {
		t.Errorf("audit reported %v on a project with nothing wrong", codes)
	}
	p.run("audit").ok().has("docs and repo agree")
}

func TestNoVisionStaysQuietBeforeAnythingIsPlanned(t *testing.T) {
	t.Parallel()
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

func TestUnfilledVisionStaysQuietOnAPartlyWrittenVision(t *testing.T) {
	t.Parallel()
	// One written section is somebody having started, and a check that nags at work in
	// progress is one people learn to ignore. Only the untouched stub is reported.
	p := cleanProject(t)
	visionStub(p)
	stub := p.read("docs/VISION.md")
	written := strings.Replace(stub, "## What done looks like\n",
		"## What done looks like\nSomebody can shorten a URL and follow it.\n", 1)
	if written == stub {
		t.Fatal("the stub's headings changed; this test no longer writes a section")
	}
	p.write("docs/VISION.md", written)

	if codes := p.auditCodes(); slices.Contains(codes, "unfilled-vision") {
		t.Errorf("unfilled-vision fired on a vision somebody had started: %v", codes)
	}
}

func TestUnfilledVisionStaysQuietWhenThereIsNoVisionAtAll(t *testing.T) {
	t.Parallel()
	// The missing-file case belongs to no-vision, which is LOW and unchanged. Reporting
	// both would make one gap produce two findings with two different fixes.
	p := cleanProject(t)
	if err := os.Remove(p.path("docs/VISION.md")); err != nil {
		t.Fatalf("removing VISION.md: %v", err)
	}

	codes := p.auditCodes()
	if slices.Contains(codes, "unfilled-vision") {
		t.Errorf("unfilled-vision fired on a project with no VISION.md: %v", codes)
	}
	if !slices.Contains(codes, "no-vision") {
		t.Errorf("no-vision stopped firing on a missing VISION.md: %v", codes)
	}
}

// makeUnreadable takes a file's permissions away, and gives up rather than asserting
// when the user running the suite can read it anyway — root ignores the mode bits, so on
// such a machine the state under test cannot be built at all.
func makeUnreadable(p *project, rel string) {
	p.t.Helper()
	if err := os.Chmod(p.path(rel), 0); err != nil {
		p.t.Fatalf("chmod %s: %v", rel, err)
	}
	if _, err := os.ReadFile(p.path(rel)); err == nil {
		p.t.Skip("this user reads a mode-000 file, so an unreadable doc cannot be set up")
	}
}

func TestAnUnreadableDocIsReportedOnceAndKeepsItsID(t *testing.T) {
	t.Parallel()
	p := cleanProject(t)
	p.write("docs/features/0002-locked.md", doc("0002", "Locked", "Draft", "", "- [ ] Do it"))
	p.commitAll("a doc that stops being readable")
	makeUnreadable(p, "docs/features/0002-locked.md")

	codes := p.auditCodes()
	if !slices.Contains(codes, "unreadable-doc") {
		t.Errorf("a doc nobody could read went unreported: %v", codes)
	}
	// Nothing else may speak for it. Every content check fires on the empty struct, and
	// not one of those findings is a fact about the file.
	for _, guess := range []string{"no-status", "unfilled-template", "done-not-shipped", "no-tasks"} {
		if slices.Contains(codes, guess) {
			t.Errorf("%s was reported on a doc whose content nobody has: %v", guess, codes)
		}
	}

	// The id stays taken. `kit new` reads the highest id from disk, so a doc it skipped
	// handed its number out a second time — the duplicate-id that audit reports at HIGH.
	p.run("new", "Something else").ok()
	if p.exists("docs/features/0002-something-else.md") {
		t.Error("kit new re-minted the id of a doc it could not read")
	}
}

func TestANonASCIIFilenameDoesNotDisableThePlanChecks(t *testing.T) {
	t.Parallel()
	// git C-quotes any path outside printable ASCII, so `ls-tree` never named the file
	// `os.ReadDir` had named and `tracked` answered false — the gate on every check that
	// reads a committed plan. The two docs below differ by one character and nothing else,
	// and the accented one used to audit clean and exit 0.
	for _, name := range []string{"0002-blank.md", "0002-blànk.md"} {
		t.Run(name, func(t *testing.T) {
			p := cleanProject(t)
			p.write("docs/features/"+name,
				"---\nid: 0002\ntitle: Blank\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
					"## Problem\n\n## Design\n\n## Tasks\n\n## Log\n")
			p.commitAll("a plan nobody wrote")
			p.run("audit", "--strict").fails(1).has("unfilled-template")
		})
	}
}

func TestTagStaleIsQuietUntilTheDistanceCouldMislead(t *testing.T) {
	t.Parallel()
	// The half that matters, and it is where the constant is decided: at the threshold an
	// install is a round behind at most, which is the ordinary state of a project between
	// releases. A check that fires there is one people learn to skip past.
	p := cleanProject(t)
	tagThenCommits(p, "v0.1.0", kit.TagStaleCommits)
	if codes := p.auditCodes(); slices.Contains(codes, "tag-stale") {
		t.Errorf("tag-stale fired %d commits past the tag, which is its own threshold: %v",
			kit.TagStaleCommits, codes)
	}

	// And a project that has never tagged. An install resolves to the trunk itself there,
	// so there is nothing for a tag to be behind and nothing to report.
	fresh := cleanProject(t)
	fresh.commitAll("the plan")
	if codes := fresh.auditCodes(); slices.Contains(codes, "tag-stale") {
		t.Errorf("tag-stale fired on a project that has never cut a tag: %v", codes)
	}
}

func TestProtocolStaleNeverOffersToOverwriteTheNewerSide(t *testing.T) {
	t.Parallel()
	// The finding read "older than the installed kit" whichever way the difference ran,
	// and offered `kit init .` — which replaces the section with the binary's copy. In
	// the one project that writes the protocol, that reverts it, and it did so here
	// silently, mid-edit, every time the protocol changed before make install.
	p := cleanProject(t)
	newer := "## Work protocol\n\nThe protocol as this checkout writes it.\n"
	p.write(kit.ProtocolTemplate, newer)
	p.write("AGENTS.md", "# The kit\n\n"+newer)

	r := p.run("audit").ok()
	r.has("protocol-stale", "make install")
	r.lacks("run: kit init .")

	// `next` is the command every session starts with, so it carries the same direction —
	// in both formats, since agents read the second.
	board := p.run("next").ok()
	board.has("make install")
	board.lacks("run kit init .")
	if plan := p.plan("next", "--json"); !plan.ProtocolAhead || !plan.ProtocolStale {
		t.Errorf("--json said stale=%v ahead=%v; agents read that format, not the prose",
			plan.ProtocolStale, plan.ProtocolAhead)
	}

	// The other direction is unchanged: an ordinary project's section is kit-owned, so
	// the older side is the project's and `kit init .` is the repair.
	behind := cleanProject(t)
	behind.write("AGENTS.md", "# House rules\n\n## Work protocol\n\nAn older protocol.\n")
	behind.run("audit").ok().has("protocol-stale", "run: kit init .")
	if plan := behind.plan("next", "--json"); plan.ProtocolAhead {
		t.Error("a project with no protocol template of its own was called the newer side")
	}
}

func TestNoProtocolFiresOnAFileWithNoProtocolSection(t *testing.T) {
	t.Parallel()
	// AGENTS.md present, its kit-owned section gone. Nothing looked for the heading, so
	// audit reported that the docs and the repo agree while every agent in the project
	// read a file with no protocol in it.
	p := cleanProject(t)
	p.write("AGENTS.md", "# House rules\n\nNever deploy on Friday.\n")

	codes := p.auditCodes()
	if !slices.Contains(codes, "no-protocol") {
		t.Errorf("an AGENTS.md with no protocol section went unreported: %v", codes)
	}
	// One defect, one finding: protocol-stale compares a section that is not there, and
	// both fixes are the same command anyway.
	if slices.Contains(codes, "protocol-stale") {
		t.Errorf("a missing section was reported as a stale one as well: %v", codes)
	}

	// The quiet half. `kit init` appends the section to the rules already there, and the
	// check has to accept what the kit itself writes.
	p.run("init").ok()
	if codes := p.auditCodes(); slices.Contains(codes, "no-protocol") {
		t.Errorf("no-protocol fired on the section kit init just wrote: %v", codes)
	}
}

func TestShippedNotPushedIsQuietWhereverTheWorkExists(t *testing.T) {
	t.Parallel()
	// Three ways for the work to be somewhere, and the check has to be silent in all of
	// them. A HIGH that fires on work already pushed is one nobody reads for long.
	a, _ := twoAgents(t)
	trunk := a.headBranch()
	a.run("claim", "0007").ok()
	a.git("checkout", "-q", "0007-widget")
	a.write("thing.txt", "the work\n")
	feature(a, "0007", "widget", "A widget", "Shipped", "- [x] Build it")
	a.commitAll("build and ship 0007")

	// One: shipped here and pushed to its own branch.
	a.git("push", "-q", "origin", "HEAD:refs/heads/0007-widget")
	if codes := a.auditCodes(); slices.Contains(codes, "shipped-not-pushed") {
		t.Errorf("fired on work pushed to its own branch: %v", codes)
	}

	// Two: merged, and the branch deleted afterwards — the remote no longer has a
	// branch naming the feature, and the work is on the trunk.
	a.git("checkout", "-q", trunk)
	a.git("merge", "-q", "--no-ff", "-m", "land 0007", "0007-widget")
	a.git("push", "-q", "origin", "HEAD:refs/heads/"+trunk)
	a.git("push", "-q", "origin", "--delete", "0007-widget")
	a.git("fetch", "-q", "--prune", "origin")
	if codes := a.auditCodes(); slices.Contains(codes, "shipped-not-pushed") {
		t.Errorf("fired on work that is on the trunk: %v", codes)
	}

	// Three: merged, and then the doc edited here — the compaction `kit ship` asks for,
	// or any later correction. The work landed weeks ago and the only thing this checkout
	// holds that the remote does not is a line of prose. Reading the age of the commit
	// that last wrote the doc reported this at HIGH, with a fix line that would recreate
	// the merged claim branch.
	a.write("docs/features/0007-widget.md", a.read("docs/features/0007-widget.md")+"\n")
	a.commitAll("compact 0007 after the fact")
	if codes := a.auditCodes(); slices.Contains(codes, "shipped-not-pushed") {
		t.Errorf("fired on a local edit to a doc whose work is on the trunk: %v", codes)
	}

	// Four: offline. The one fact this needs comes from the remote, so without it the
	// honest answer is nothing — the same bargain every other claim check makes.
	b := cleanProject(t)
	origin := t.TempDir()
	gitIn(t, origin, "init", "-q", "--bare", ".")
	b.git("remote", "add", "origin", origin)
	b.write("docs/features/0002-done.md", doc("0002", "Done", "Shipped",
		"shipped: "+daysAgo(1), "- [x] Built it"))
	b.commitAll("ship 0002 and tell nobody")
	if codes := b.auditCodes("--offline"); slices.Contains(codes, "shipped-not-pushed") {
		t.Errorf("fired with --offline, where nothing about the remote is known: %v", codes)
	}
	// And the same project, asked properly, still reports it — otherwise the case above
	// proves nothing.
	if codes := b.auditCodes(); !slices.Contains(codes, "shipped-not-pushed") {
		t.Errorf("stopped firing on work that is on no remote: %v", codes)
	}
}

func TestEveryRegisteredCheckIsExercised(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// A finding nobody can act on is noise.
	p := cleanProject(t)
	feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")

	r := p.run("audit").ok()
	r.has("shipped-no-date", "docs/features/0002-undated.md", "fix:")
}

func TestAuditNeverWrites(t *testing.T) {
	t.Parallel()
	// An auditor that fixes things stops being read.
	p := cleanProject(t)
	feature(p, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	before := p.read("docs/features/0002-undated.md")

	p.run("audit").ok()

	if after := p.read("docs/features/0002-undated.md"); after != before {
		t.Error("audit modified a document it was only meant to report on")
	}
}

func TestTheShippedDateIsReadAsWrittenOrReportedAsWrong(t *testing.T) {
	t.Parallel()
	// Three states of one field, and only the first was told apart from the others.
	for _, tc := range []struct {
		name, line, want string
	}{
		{"absent", "", "shipped-no-date"},
		// Reported as "Shipped with no shipped: date" on a doc that has one.
		{"unparseable", "shipped: last Tuesday", "shipped-bad-date"},
		{"in the future", "shipped: 2099-01-01", "shipped-bad-date"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := cleanProject(t)
			p.write("docs/features/0002-odd.md",
				doc("0002", "Odd", "Shipped", tc.line, "- [x] Done"))
			codes := p.auditCodes()
			if !slices.Contains(codes, tc.want) {
				t.Errorf("shipped: %q reported %v, want %s", tc.line, codes, tc.want)
			}
		})
	}

	// And a date that has not arrived no longer sets the bar tech-drift compares against,
	// which nothing could ever clear.
	p := cleanProject(t)
	p.write("docs/TECH.md", "# As built\n\nThe description.\n")
	p.write("docs/features/0002-early.md",
		doc("0002", "Early", "Shipped", "shipped: 2099-01-01", "- [x] Done"))
	p.commitAll("describe the system")
	if codes := p.auditCodes(); slices.Contains(codes, "tech-drift") {
		t.Errorf("tech-drift measured against a day that has not happened: %v", codes)
	}
}

func TestDoneNotShippedNeedsBoxesToCallDone(t *testing.T) {
	t.Parallel()
	// "every task is ticked" on a doc with no ## Tasks section at all, whose fix —
	// `kit ship` — refuses it for having no boxes. A zero-byte doc got the same finding
	// with `run: kit ship ` and no id in it.
	for _, tc := range []struct{ name, body string }{
		{"no Tasks section", "---\nid: 0002\ntitle: Sectionless\nstatus: Draft\n" +
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n"},
		{"empty file", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := cleanProject(t)
			p.write("docs/features/0002-sectionless.md", tc.body)
			p.commitAll("a doc with no boxes anywhere in it")

			r := p.run("audit").ok()
			r.lacks("done-not-shipped")
			// unfilled-template already says what is wrong with it, and its fix works.
			r.has("unfilled-template")
		})
	}

	// The quiet half: a doc whose boxes are all ticked is still reported, with the fix
	// that does the job.
	p := cleanProject(t)
	p.write("docs/features/0002-done.md", doc("0002", "Done", "Draft", "", "- [x] Finished"))
	p.commitAll("every box ticked")
	p.run("audit").ok().has("done-not-shipped", "kit ship 0002")

	// And with no id: there is no such command, so the fix names the edit instead of
	// printing `run: kit ship ` with nothing after it.
	nameless := cleanProject(t)
	nameless.write("docs/features/0002-nameless.md",
		"---\ntitle: Nameless\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- x\n")
	nameless.commitAll("a doc with no id in it")
	nameless.run("audit").ok().has("done-not-shipped").lacks("run: kit ship  ")
}

func TestAuditOnAnUnsetUpProjectExitsTwo(t *testing.T) {
	t.Parallel()
	// Nothing was read, so "no findings" is not a statement about this directory. Both
	// forms exited 0, which made `kit audit --strict` a green CI gate for a repository
	// carrying no docs at all — the one repository it most needed to fail.
	p := newProject(t)
	p.run("audit").fails(2).has("not set up yet", "kit init")
	p.run("audit", "--strict").fails(2)
	p.run("audit", "--json").fails(2).outHas("not set up yet")
}

func TestOfflineNamesTheCheckItDidNotRun(t *testing.T) {
	t.Parallel()
	// --offline drops the same HIGH check an unreachable remote does. The unreachable
	// path prints a notice and this one printed nothing, so --strict exited 0 where it
	// would have exited 1 with no line on the page to say a check had been skipped
	// rather than passed.
	p := cleanProject(t)
	r := p.run("audit", "--offline").ok()
	r.outHas("--offline", "shipped-not-pushed", "did not run")
	p.run("audit", "--json", "--offline").ok().outHas(`"skipped"`, "shipped-not-pushed")

	// And nothing of the sort when the kit was allowed to ask.
	p.run("audit").ok().lacks("--offline")
	p.run("audit", "--json").ok().lacks("skipped")
}

func TestAuditStrictExitsOnHighOnly(t *testing.T) {
	t.Parallel()
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

func TestIDMismatchFailsStrict(t *testing.T) {
	t.Parallel()
	// Raised from MEDIUM. A doc whose id: and filename disagree is not untidy: the
	// branch comes from the filename and the board looks for the id, so the feature's
	// own claim is invisible and it keeps being handed out. --strict is the gate in
	// front of `kit ship`, so this is what stops it being left sitting while agents run.
	mismatched := cleanProject(t)
	mismatched.write("docs/features/0002-confused.md",
		doc("0009", "Confused", "Draft", "", "- [ ] Something"))
	mismatched.run("audit", "--strict").fails(1).has("id-mismatch")

	// The quiet half: agreeing is the normal case and must not fail anything.
	agreed := cleanProject(t)
	agreed.write("docs/features/0002-agreed.md",
		doc("0002", "Agreed", "Draft", "", "- [ ] Something"))
	agreed.run("audit", "--strict").ok().lacks("id-mismatch")
}

func TestStalledIsVisibleFromTheTrunkCheckout(t *testing.T) {
	t.Parallel()
	// The failure this closes: a status set on the feature branch never reaches the trunk
	// checkout — the only one a human or a supervising agent audits — so it saw an open
	// doc and reported nothing. An agent that died holding a feature was indistinguishable
	// from one hard at work, permanently, and the board just said "taken".
	//
	// Liveness now comes from the claim branch, so what is being asserted is both
	// directions: active work is quiet, abandoned work is reported.
	for _, tc := range []struct {
		name string
		age  int
		want bool
	}{
		{"work committed today", 0, false},
		{"work abandoned", kit.StalledDays + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProject(t)
			p.run("init").ok()
			// Trunk carries the doc exactly as it looks after a draft PR merges.
			feature(p, "0002", "widget", "Widget", "Draft", "- [ ] Build it")
			p.commitAllAt("the plan", daysAgo(tc.age))
			trunk := p.headBranch()

			// The claim branch carries the work, and that is the whole signal.
			p.git("checkout", "-q", "-b", "0002-widget")
			p.write("internal/widget.go", "package widget\n")
			p.commitAllAt("0002: start building", daysAgo(tc.age))
			p.git("checkout", "-q", trunk)

			// Read from the trunk, where the doc still says Draft.
			if got := slices.Contains(p.auditCodes(), "stalled"); got != tc.want {
				t.Errorf("work %d day(s) old, seen from %s: stalled=%v, want %v",
					tc.age, trunk, got, tc.want)
			}
		})
	}
}

func TestAClaimCarryingNoWorkIsNeverCalledStalled(t *testing.T) {
	t.Parallel()
	// A branch cut from the trunk's tip is byte-identical to one whose pull request
	// merged, and neither carries a date the kit can read. Ageing it would invent a
	// finding, so `next` reports it as taken and audit says nothing.
	p := newProject(t)
	p.run("init").ok()
	feature(p, "0002", "widget", "Widget", "Draft", "- [ ] Build it")
	p.commitAllAt("the plan", daysAgo(60))
	p.git("branch", "0002-widget")

	if codes := p.auditCodes(); slices.Contains(codes, "stalled") {
		t.Errorf("an empty claim off an old trunk was aged as if it were work: %v", codes)
	}
}

func TestTheDefaultBranchIsAskedForRatherThanAssumed(t *testing.T) {
	t.Parallel()
	// The trunk used to be the first of `main`/`master` that resolved. A project whose
	// default branch is neither had no trunk to compare against, so BranchBuildsOn
	// answered "nothing changed" for every branch — and every check built on it went
	// quiet. `stalled` is the observable one: an agent that died holding a feature was
	// invisible for the whole class of repos that do not use those two names.
	// Both ways a project can have one: read from the remote, which is the case that
	// matters for anything hosted, and guessed from HEAD when there is no remote to ask.
	for _, withRemote := range []bool{true, false} {
		name := "local only"
		if withRemote {
			name = "origin/HEAD says develop"
		}
		t.Run(name, func(t *testing.T) {
			p := newProject(t)
			p.run("init").ok()
			p.git("checkout", "-q", "-b", "develop")
			feature(p, "0002", "widget", "Widget", "Draft", "- [ ] Build it")
			p.commitAllAt("the plan", daysAgo(kit.StalledDays+1))

			if withRemote {
				origin := filepath.Join(p.Root, "..", "origin.git")
				gitIn(t, p.Root, "init", "-q", "--bare", "-b", "develop", origin)
				p.git("remote", "add", "origin", origin)
				p.git("push", "-q", "origin", "develop")
				p.git("remote", "set-head", "origin", "develop")
			}

			// A branch carrying work beyond the doc, off a trunk called neither
			// main nor master, and nothing has touched it since.
			p.git("checkout", "-q", "-b", "0002-widget")
			p.write("internal/thing.go", "package thing\n")
			p.commitAllAt("0002: started and left", daysAgo(kit.StalledDays+1))
			p.git("checkout", "-q", "develop")

			if codes := p.auditCodes(); !slices.Contains(codes, "stalled") {
				t.Errorf("an abandoned claim went unreported because the trunk is called "+
					"develop; audit said %v", codes)
			}
		})
	}
}

func TestARemoteNeedNotBeCalledOriginForTheTrunkToResolve(t *testing.T) {
	t.Parallel()
	// The trunk comes from `<remote>/HEAD`, and only an `origin/` prefix was stripped off
	// the answer — so a clone whose remote is `upstream` resolved its trunk as
	// `upstream/develop`, which names no ref. Everything comparing against the trunk then
	// had nothing to compare with: `stalled` went quiet, and `release` refused every claim.
	p := newProject(t)
	p.run("init").ok()
	p.git("checkout", "-q", "-b", "develop")
	feature(p, "0002", "widget", "Widget", "Draft", "- [ ] Build it")
	p.commitAllAt("the plan", daysAgo(kit.StalledDays+1))

	remote := filepath.Join(p.Root, "..", "upstream.git")
	gitIn(t, p.Root, "init", "-q", "--bare", "-b", "develop", remote)
	p.git("remote", "add", "upstream", remote)
	p.git("push", "-q", "upstream", "develop")
	p.git("remote", "set-head", "upstream", "develop")

	p.git("checkout", "-q", "-b", "0002-widget")
	p.write("internal/thing.go", "package thing\n")
	p.commitAllAt("0002: started and left", daysAgo(kit.StalledDays+1))
	p.git("checkout", "-q", "develop")

	if codes := p.auditCodes(); !slices.Contains(codes, "stalled") {
		t.Errorf("an abandoned claim went unreported because the remote is called "+
			"upstream; audit said %v", codes)
	}
}

func TestAnFYINeverFailsTheShipGate(t *testing.T) {
	t.Parallel()
	// `kit block --fyi` promises the feature is not blocked and work continues.
	// BlockersFor kept that promise and inbox-stale broke it anyway: FYI was reported at
	// HIGH, so the `kit audit --strict` gate refused to let anything ship over a note
	// whose whole point was that it needed no answer.
	p := cleanProject(t)
	p.write("docs/INBOX.md", fmt.Sprintf(
		"# Inbox\n\n- [ ] %s · 0001 · FYI · Which host runs this? Leaning: decide later.\n",
		daysAgo(365)))

	r := p.run("audit", "--strict").ok()
	if !strings.Contains(r.Stdout, "fyi-stale") {
		t.Error("a year-old FYI was not reported at all; LOW still means visible")
	}

	// And the blocking case still does fail it, or the split bought nothing.
	blocking := cleanProject(t)
	blocking.write("docs/INBOX.md", fmt.Sprintf(
		"# Inbox\n\n- [ ] %s · 0001 · BLOCKING · Which host runs this?\n",
		daysAgo(kit.InboxStaleDays)))
	blocking.run("audit", "--strict").fails(1)
}

// The claim checks read branch names, which is the one place this tool can
// generate noise about work that is perfectly fine. Each of these is a situation
// that must stay silent.
func TestClaimChecksDoNotFireOnOrdinaryWork(t *testing.T) {
	t.Parallel()
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

	t.Run("a namespaced branch in a worktree is one claim", func(t *testing.T) {
		// One branch, one checkout. The worktree list gives `branch refs/heads/<name>`
		// and the branch list gives the name, so keeping only the last path segment
		// entered both `agents/0001-work` and `0001-work` — and the checkout reported
		// itself as two agents holding the feature, with a fix line that deletes the work.
		p := cleanProject(t)
		p.commitAll("the doc")
		p.git("checkout", "-q", "-b", "agents/0001-work")

		for _, code := range p.auditCodes() {
			if code == "split-claim" {
				t.Error("one namespaced branch was reported as two claims")
			}
		}
	})

	t.Run("two branches on one feature are still a split claim", func(t *testing.T) {
		// The other half: the case above must not be bought by making split-claim blind.
		p := cleanProject(t)
		p.commitAll("the doc")
		p.git("branch", "agents/0001-work")
		p.git("branch", "0001-thing")

		if codes := p.auditCodes(); !slices.Contains(codes, "split-claim") {
			t.Errorf("two branches claiming one feature went unreported: %v", codes)
		}
	})
}

func TestSplitClaimIgnoresAShippedFeaturesLeftoverBranches(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

	t.Run("fyi-stale", func(t *testing.T) {
		for _, tc := range []struct {
			age  int
			want bool
		}{{kit.FYIStaleDays - 1, false}, {kit.FYIStaleDays, true}} {
			p := cleanProject(t)
			p.write("docs/INBOX.md", fmt.Sprintf(
				"# Inbox\n\n- [ ] %s · 0001 · FYI · Just so you know?\n", daysAgo(tc.age)))

			if got := slices.Contains(p.auditCodes(), "fyi-stale"); got != tc.want {
				t.Errorf("an FYI %d day(s) undrained: reported=%v, want %v",
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
			abandonedClaim(p, "0002", "stuck", tc.idle)

			if got := slices.Contains(p.auditCodes(), "stalled"); got != tc.want {
				t.Errorf("idle for %d day(s): reported=%v, want %v", tc.idle, got, tc.want)
			}
		}
	})
}

func TestDuplicateIDDoesNotFireOnDocsWithNoIDAtAll(t *testing.T) {
	t.Parallel()
	// The quiet half. Docs missing an id: share the empty string, and grouping on
	// that would report them as duplicates of each other — a second, wrong finding
	// on top of the no-status one that is already telling the truth about them.
	p := cleanProject(t)
	for _, name := range []string{"0002-nameless", "0003-alsonameless"} {
		p.write("docs/features/"+name+".md",
			"---\ntitle: No id here\nstatus: Draft\n---\n\n"+
				"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [ ] y\n\n## Log\n- 2026-01-01 — x.\n")
	}

	for _, code := range p.auditCodes() {
		if code == "duplicate-id" {
			t.Error("two docs with no id: were reported as sharing one")
		}
	}
}

func TestDuplicateIDNamesBothDocs(t *testing.T) {
	t.Parallel()
	// Naming one file is not actionable: the fix is to renumber a specific doc, and
	// you cannot pick which without seeing both.
	p := cleanProject(t)
	p.write("docs/features/0001-thing-as-well.md", doc("0001", "Thing as well",
		"Draft", "", "- [ ] Something"))

	p.run("audit").ok().has("duplicate-id",
		"docs/features/0001-thing.md", "docs/features/0001-thing-as-well.md")
}

func TestInboxChecksStayQuietOnAWellFormedQuestion(t *testing.T) {
	t.Parallel()
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

func TestNextNeverHandsOverWorkThatIsNotYours(t *testing.T) {
	t.Parallel()
	// `task` means "the box to do now". Attaching one to a feature the board has
	// just reported as somebody else's is the payload contradicting the prose — and
	// --json is the format agents are told to read, so it is the one that matters.
	p := cleanProject(t)
	p.write("docs/features/0002-waiting.md",
		doc("0002", "Waiting", "Draft", "needs: 0001", "- [ ] Do it"))
	p.commitAll("two features")
	p.git("branch", "0001-thing")

	plan := p.plan("next", "--json")
	if plan.Guidance == "" {
		t.Error("--json carries no guidance, so a parsing agent gets a board and no conclusion")
	}
	for _, group := range []struct {
		name  string
		items []kit.Item
	}{{"taken", plan.Taken}, {"blocked", plan.Blocked}} {
		for _, item := range group.items {
			if item.Task != "" {
				t.Errorf("%s feature %s handed over a task: %q", group.name, item.ID, item.Task)
			}
		}
	}
}

func TestADraftOnItsOwnBranchIsNotAnOrphan(t *testing.T) {
	t.Parallel()
	// The protocol requires exactly this: a draft waits on a branch of its own and
	// nothing is committed to the trunk. Audited from the trunk, the doc is missing —
	// and the kit reported the protocol being followed as two MEDIUM defects, one
	// advising a fix (delete the branch) that destroys the draft.
	p := newProject(t)
	p.run("init").ok()
	p.commitAll("bootstrap")
	trunk := p.headBranch()

	p.git("checkout", "-q", "-b", "0001-a-feature")
	p.run("new", "A feature").ok()
	p.commitAll("Plan 0001")
	p.git("checkout", "-q", trunk)

	codes := p.auditCodes()
	for _, unwanted := range []string{"orphan-claim", "no-plan"} {
		if slices.Contains(codes, unwanted) {
			t.Errorf("following the protocol was reported as %q; audit said %v", unwanted, codes)
		}
	}
}

func TestOrphanClaimStillFiresOnABranchWithNoDoc(t *testing.T) {
	t.Parallel()
	// The check has to keep working, or standing it down for in-flight features would
	// have bought silence rather than accuracy. A branch citing an ID nobody wrote means
	// a doc was deleted or renamed and that ID is now unclaimable.
	p := cleanProject(t)
	p.commitAll("the plan")
	p.git("branch", "0099-never-written")

	if codes := p.auditCodes(); !slices.Contains(codes, "orphan-claim") {
		t.Errorf("a branch citing a feature nobody wrote went unreported: %v", codes)
	}
}

func TestABranchThatCannotBeResolvedIsNotJudged(t *testing.T) {
	t.Parallel()
	// A branch on a remote nobody has fetched cannot have its tree read, so nothing is
	// known about whether it carries a doc. Reporting it would be a guess, and the guess
	// that costs the most is the one that says "delete this".
	a, b := twoAgents(t)
	a.run("claim", "0007").ok()

	// Delete the doc in b, so 0007-widget cites an ID b knows nothing about and the
	// branch reaches the resolution path rather than being skipped as known.
	if err := os.Remove(b.path("docs/features/0007-widget.md")); err != nil {
		t.Fatalf("removing the doc: %v", err)
	}
	b.commitAll("drop the doc")

	// b sees the branch exists on the remote — 0017 made claims authoritative — but has
	// never fetched it, so there is no ref whose tree could be read.
	if b.resolves("refs/remotes/origin/0007-widget") {
		t.Skip("this clone already has the branch fetched, so it can be judged")
	}
	if codes := b.auditCodes(); slices.Contains(codes, "orphan-claim") {
		t.Errorf("an unfetched remote branch was judged as an orphan: %v", codes)
	}

	// Once fetched, its tree is readable and it does carry the doc — so still no finding,
	// now for the other reason.
	b.git("fetch", "origin")
	if codes := b.auditCodes(); slices.Contains(codes, "orphan-claim") {
		t.Errorf("a branch carrying its own doc was reported as an orphan: %v", codes)
	}
}

func TestADocLeftSayingInProgressIsReported(t *testing.T) {
	t.Parallel()
	// The migration path. `In progress` left the vocabulary because nothing could set it
	// without contradicting the claim branch, so a doc still carrying it is a leftover —
	// most likely a claim released without deleting the branch — and `bad-status` says so
	// rather than the board quietly ignoring the feature.
	p := cleanProject(t)
	p.write("docs/features/0002-working.md",
		doc("0002", "Working", "In progress", "", "- [ ] Something"))

	if codes := p.auditCodes(); !slices.Contains(codes, "bad-status") {
		t.Errorf("a doc saying In progress reported %v, want bad-status", codes)
	}
	// The feature stays visible and is reported as blocked, never dropped. Selecting the
	// board on a status allow-list would make every doc carrying a status a later kit
	// removed vanish silently — which is what dropping `Accepted` would have done to every
	// project already using the kit. Blocked, so `claim` cannot disagree with the board.
	if got := boardGroup(p.plan("next", "--json"), "0002"); got != "blocked" {
		t.Errorf("0002 is %s on the board; an unrecognised status is blocked, not hidden", got)
	}
	p.run("claim", "0002").fails(1).errHas("not open work")
}

func TestAuditRefusesAPathArgument(t *testing.T) {
	t.Parallel()
	// `audit` reads the project you are standing in and nothing else, so a path is a
	// broken invocation rather than a second project to report on. Exit 2, the reason
	// on stderr, and the shell loop that does what the argument was reaching for.
	p := cleanProject(t)
	p.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
		"shipped: "+daysAgo(1), "- [ ] Not actually done"))

	for _, arg := range []string{p.Root, filepath.Join(p.Root, "gone")} {
		r := p.run("audit", arg).fails(2)
		r.errHas("takes no arguments", "cd $d && kit audit")
		r.lacks("unticked-shipped")
	}
}

func TestTheAuditSummaryCountsBySeverity(t *testing.T) {
	t.Parallel()
	// The line used to read "N thing(s) to fix. HIGH first." for every board, including
	// one with no HIGH on it at all — so the instruction it gave was wrong more often
	// than it was right, and the count it gave was never singular.
	high := cleanProject(t)
	high.write("docs/features/0002-early.md", doc("0002", "Early", "Shipped",
		"shipped: "+daysAgo(1), "- [ ] Not actually done"))
	high.run("audit", "--strict").fails(1).outHas("1 HIGH finding, 0 MEDIUMs. Fix the HIGH first.")

	// No HIGH, so nothing is told to fix the HIGH first.
	medium := cleanProject(t)
	feature(medium, "0002", "undated", "Undated", "Shipped", "- [x] Done")
	r := medium.run("audit").ok()
	r.outHas("1 MEDIUM finding to fix.")
	r.lacks("HIGH first")

	// LOW is hygiene, and a summary that counted it would ask for a fix the severity
	// says can wait. The findings themselves are still printed.
	low := cleanProject(t)
	if err := os.Remove(low.path("docs/INBOX.md")); err != nil {
		t.Fatalf("removing the inbox: %v", err)
	}
	quiet := low.run("audit").ok()
	quiet.outHas("no-inbox")
	quiet.lacks("to fix.")
}
