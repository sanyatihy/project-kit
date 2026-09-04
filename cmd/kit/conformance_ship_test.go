package main

// Conformance: `kit ship`. Its value is what it refuses — two of the audit's
// findings exist only because this is normally done by hand.

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// shipping commits what the fixture wrote, then ships. A tree with nothing uncommitted
// in it is the state the ship gate describes — the last box ticked in the commit that
// implemented it — and ship refuses anything else.
func shipping(p *project, args ...string) *result {
	p.t.Helper()
	p.commitAll("the work")
	return p.run(append([]string{"ship"}, args...)...)
}

func TestShipRefusesWhileABoxIsOpen(t *testing.T) {
	t.Parallel()
	// An open box is prevented here rather than reported afterwards: the audit finding
	// that once said so was deleted, because its only fix edited a Shipped doc.
	p := setUp(t)
	feature(p, "0007", "half", "Half done", "Draft", "- [x] One\n- [ ] Two")

	r := shipping(p, "0007").fails(2)
	r.errHas("unticked", "Two")
	if strings.Contains(p.read("docs/features/0007-half.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipNamesDebrisAsItIsOnDisk(t *testing.T) {
	t.Parallel()
	// `git status --porcelain` C-quotes a non-ASCII path, so the refusal named a file
	// that exists under no such name: the agent told to remove it cannot find it.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.commitAll("the work")
	p.write("probe-outpüt.json", "{}\n")

	r := p.run("ship", "0007").fails(2)
	r.errHas("probe-outpüt.json")
	r.lacks(`\303`)
}

func TestShipRefusesDebrisInTheWorktree(t *testing.T) {
	t.Parallel()
	// The scratch a feature generates is the agent's to clean up. Shipping over it
	// either commits it under a status change or leaves it for whoever works here next.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.commitAll("the work")
	p.write("probe-output.json", "{}\n")

	p.run("ship", "0007").fails(2).errHas("uncommitted or untracked", "probe-output.json")

	if strings.Contains(p.read("docs/features/0007-done.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipIgnoresWhatGitIgnores(t *testing.T) {
	t.Parallel()
	// Build output is not debris an agent has to explain, and a refusal that fired on
	// it would be one every project learns to work around.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.write(".gitignore", "build/\n")
	p.commitAll("the work")
	p.write("build/kit", "not source")

	p.run("ship", "0007").ok()
}

func TestShipStampsTheDate(t *testing.T) {
	t.Parallel()
	// shipped-no-date, prevented rather than reported.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One\n- [x] Two")

	shipping(p, "0007").ok().has("shipped")

	doc := p.read("docs/features/0007-done.md")
	if !strings.Contains(doc, "status: Shipped") {
		t.Errorf("status was not set:\n%s", doc)
	}
	if want := "shipped: " + time.Now().UTC().Format("2006-01-02"); !strings.Contains(doc, want) {
		t.Errorf("date was not stamped, want %q:\n%s", want, doc)
	}
	p.run("audit").ok().lacks("shipped-no-date")
}

func TestShipCommitsTheStampItWrites(t *testing.T) {
	t.Parallel()
	// A stamp left in the working tree is one session from being read as debris and
	// discarded, and `audit` reads the tree rather than HEAD, so nothing would say so.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")

	shipping(p, "0007").ok().has("Ship 0007")

	if dirty := p.gitOut(nil, "status", "--porcelain"); strings.TrimSpace(dirty) != "" {
		t.Errorf("ship left the tree dirty:\n%s", dirty)
	}
	if head := p.gitOut(nil, "show", "HEAD:docs/features/0007-done.md"); !strings.Contains(head, "status: Shipped") {
		t.Errorf("the committed doc is not the shipped one:\n%s", head)
	}
}

func TestShipReportsAStampItCouldNotCommit(t *testing.T) {
	t.Parallel()
	// The write has already happened when the commit fails, so silence here would
	// leave exactly the loss this closes. An index another process holds is one way in;
	// no identity configured and a rebase in progress are the others.
	p := setUp(t)
	feature(p, "0007", "done", "All done", "Draft", "- [x] One")
	p.commitAll("the work")
	p.write(".git/index.lock", "")

	p.run("ship", "0007").fails(2).errHas("stamped Shipped", "docs/features/0007-done.md")

	if doc := p.read("docs/features/0007-done.md"); !strings.Contains(doc, "status: Shipped") {
		t.Errorf("the stamp the error names is not on disk:\n%s", doc)
	}
}

func TestShipFillsAnExistingEmptyDateKey(t *testing.T) {
	t.Parallel()
	// The template ships a bare "shipped:" line, so this is the common path — and
	// it is a different branch from inserting one.
	p := setUp(t)
	p.write("docs/features/0007-templated.md",
		"---\nid: 0007\ntitle: Templated\nstatus: Draft\ncreated: 2026-01-01\nshipped:\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").ok()

	doc := p.read("docs/features/0007-templated.md")
	if want := "shipped: " + time.Now().UTC().Format("2006-01-02"); !strings.Contains(doc, want) {
		t.Errorf("the empty shipped: key was not filled, want %q:\n%s", want, doc)
	}
	if strings.Count(doc, "shipped:") != 1 {
		t.Errorf("ship produced a duplicate shipped: key:\n%s", doc)
	}
	p.run("audit").ok().lacks("shipped-no-date")
}

func TestShipAddsTheDateKeyWhenTheDocHasNone(t *testing.T) {
	t.Parallel()
	// An older doc may have no shipped: line at all. Stamping has to insert one
	// rather than silently do nothing.
	p := setUp(t)
	p.write("docs/features/0007-old.md",
		"---\nid: 0007\ntitle: Old\nstatus: Draft\n---\n\n"+
			"## Problem\nx\n\n## Design\nx\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").ok()
	if !strings.Contains(p.read("docs/features/0007-old.md"), "shipped: ") {
		t.Error("no shipped: key was added")
	}
}

func TestShipStampsInsideTheFrontmatterOnly(t *testing.T) {
	t.Parallel()
	// Docs for a tool whose subject is frontmatter quote frontmatter. The stamp for a doc
	// with no shipped: key was a document-wide strings.Replace on the first
	// `status: Shipped`, and a value carrying one comes before the status line — so the
	// date landed inside the title, and the fenced example in the body was one edit away.
	p := setUp(t)
	p.write("docs/features/0007-quoting.md",
		"---\nid: 0007\ntitle: Refuse status: Shipped twice\nstatus: Draft\n"+
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\n```\n"+
			"status: Shipped\n```\n\n## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").ok()

	doc := p.read("docs/features/0007-quoting.md")
	if !strings.Contains(doc, "title: Refuse status: Shipped twice\n") {
		t.Errorf("the stamp was written into the title:\n%s", doc)
	}
	if !strings.Contains(doc, "```\nstatus: Shipped\n```") {
		t.Errorf("the fenced example in the body was rewritten:\n%s", doc)
	}
	if got := strings.Count(doc, "shipped: "); got != 1 {
		t.Errorf("%d shipped: lines, want 1:\n%s", got, doc)
	}
}

func TestShipRefusesAFeatureWithNoBoxesAtAll(t *testing.T) {
	t.Parallel()
	// Nothing to have finished. Shipping it would assert work that was never planned.
	p := setUp(t)
	feature(p, "0007", "empty", "Empty", "Draft", "")
	shipping(p, "0007").fails(2).errHas("no ## Tasks boxes")
}

func TestShipRefusesARetiredFeature(t *testing.T) {
	t.Parallel()
	// The other direction of the same rule. `drop` refuses a Shipped doc as frozen history
	// (TestDropRefusesAShippedDoc); `ship` refused only Shipped, so a Dropped feature whose
	// boxes were ticked shipped, exit 0, reported as `(was Dropped)`.
	p := setUp(t)
	feature(p, "0007", "retired", "Retired", "Dropped", "- [x] One")

	shipping(p, "0007").fails(2).errHas("Dropped")
	if strings.Contains(p.read("docs/features/0007-retired.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipRefusesTwice(t *testing.T) {
	t.Parallel()
	// The second run would rewrite frozen history and restamp the date. It is run
	// directly rather than through `shipping`: the first ship committed its own stamp,
	// so there is nothing left for a second commit to make.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	shipping(p, "0007").ok()
	p.run("ship", "0007").fails(2).errHas("already Shipped")
}

func TestShipRefusesWhileItsNeedsHaveNotShipped(t *testing.T) {
	t.Parallel()
	// Prevented rather than reported: `ship` was minting a finding on a doc the protocol
	// then forbids anyone to edit, which is why that finding is gone and this refusal is
	// what is left. Exit 2 and not 1 — nobody else holds this, the work is done and the
	// plan is wrong, so the operator has to look.
	p := setUp(t)
	feature(p, "0001", "base", "Base", "Draft", "- [ ] Do it")
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [x] Done"))

	shipping(p, "0002").fails(2).errHas("0001", "not shipped")
	if strings.Contains(p.read("docs/features/0002-store.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipAllowsAFeatureWhoseNeedsHaveShipped(t *testing.T) {
	t.Parallel()
	// The quiet half: the foundation is there, so nothing is in the way.
	p := setUp(t)
	p.write("docs/features/0001-base.md",
		doc("0001", "Base", "Shipped", "shipped: 2026-01-02", "- [x] Do it"))
	p.write("docs/features/0002-store.md",
		doc("0002", "Store", "Draft", "needs: 0001", "- [x] Done"))

	shipping(p, "0002").ok().has("shipped")
}

func TestShipRefusesAFeatureAnotherCheckoutClaims(t *testing.T) {
	t.Parallel()
	// Measured from a checkout whose own board says the feature is taken: `ship` exited 0
	// and froze another agent's work mid-flight. `status: Shipped` is history the protocol
	// forbids anyone to edit, so the agent holding the branch could no longer touch its
	// own doc — and `audit` reported nothing at all.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok() // the branch is the claim; this checkout is not on it
	a.write("docs/features/0007-widget.md",
		doc("0007", "A widget", "Draft", "", "- [x] Build it"))

	r := shipping(a, "0007").fails(2)
	// Naming the branch and the way out: refusing without one is how an agent gets stuck.
	r.errHas("0007-widget", "kit release 0007")
	if strings.Contains(a.read("docs/features/0007-widget.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipFromTheCheckoutHoldingTheClaimIsFine(t *testing.T) {
	t.Parallel()
	// The quiet half, and the documented path: the worktree that holds the claim is
	// where the work was done and where it ships from.
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("0007-widget")
	w.write("docs/features/0007-widget.md",
		doc("0007", "A widget", "Draft", "", "- [x] Build it"))

	shipping(w, "0007").ok().has("shipped")
}

// Merged before the stamp: the branch is already in, so "push the branch and stop" names
// a push that changes nothing, and the commit ship is about to make is the part nobody has
// reviewed. Containment says this without asking who merged it.
func TestShipSaysWhenTheTrunkAlreadyHasTheWork(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	trunk := a.headBranch()
	a.run("claim", "0007").ok()
	w := a.claimWorktree("0007-widget")
	w.write("widget.go", "package widget\n")
	w.write("docs/features/0007-widget.md",
		doc("0007", "A widget", "Draft", "", "- [x] Build it"))
	w.commitAll("0007: build it")
	a.git("merge", "-q", "--no-ff", "-m", "land 0007", "0007-widget")

	// Not `shipping`, which makes a commit first: the tree is already clean here, and
	// the clean tree is what the merge-early case looks like.
	w.run("ship", "0007").ok().has("shipped", "already contains", trunk)
}

func TestShipRefusesADocThatDeclaresAKeyTwice(t *testing.T) {
	t.Parallel()
	// The parser keeps the last occurrence and SetField rewrites the first, so this
	// shipped: exit 0, a status line nothing reads, and `audit` repeating the same
	// finding with the same fix for ever.
	p := setUp(t)
	p.write("docs/features/0007-twice.md",
		"---\nid: 0007\ntitle: Twice\nstatus: Draft\nstatus: Draft\n"+
			"created: 2026-01-01\n---\n\n## Problem\nx\n\n## Design\nx\n\n"+
			"## Tasks\n- [x] Done\n\n## Log\n- 2026-01-01 — x.\n")

	shipping(p, "0007").fails(2).errHas("status", "twice")
	if strings.Contains(p.read("docs/features/0007-twice.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
	// And it is reported as well as refused, so a doc already in this state is findable.
	p.run("audit").has("duplicate-key")
}

func TestShipRefusesAnIDThatNamesTwoDocs(t *testing.T) {
	t.Parallel()
	// duplicate-id is HIGH because one id has to name one doc. `claim` counted the
	// matches and refused; this took the first and shipped it, so the other doc kept the
	// same id and nothing said which one had been shipped.
	p := setUp(t)
	p.write("docs/features/0007-one.md", doc("0007", "One", "Draft", "", "- [x] Done"))
	p.write("docs/features/0007-two.md", doc("0007", "Two", "Draft", "", "- [x] Done"))

	shipping(p, "0007").fails(2).errHas("0007-one.md", "0007-two.md")
	for _, rel := range []string{"docs/features/0007-one.md", "docs/features/0007-two.md"} {
		if strings.Contains(p.read(rel), "status: Shipped") {
			t.Errorf("%s was shipped despite the refusal", rel)
		}
	}
}

func TestShipRefusesAnUnknownFeature(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	shipping(p, "0099").fails(2).errHas("no feature")
}

func TestShipSaysTheWorkIsHandedOffNotFinished(t *testing.T) {
	t.Parallel()
	// The first real project to run the loop reached the end of ship, found no
	// instruction about what happens next, and merged its own work to main. The
	// protocol is read at the start of a session; this decision is made at the end
	// of one, so it has to be said here too.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")

	// No remote: there is nowhere to hand off to, which is the case that produced
	// the unreviewed merge — so it must say so rather than staying silent.
	shipping(p, "0007").ok().has("no remote", "merging is not yours")

	p.git("remote", "add", "origin", p.path(".git"))
	feature(p, "0008", "next", "Next", "Draft", "- [x] One")
	shipping(p, "0008").ok().has("push the branch and stop")
}

func TestShipPointsAtTheAsBuiltDoc(t *testing.T) {
	t.Parallel()
	// The protocol says move durable facts across; nothing enforced it, so the prompt
	// is the enforcement.
	// The creation case is TestShipCreatesTheAsBuiltDocOnce; this is the reminder
	// when the doc is already there.
	p := setUp(t)
	feature(p, "0007", "done", "Done", "Draft", "- [x] One")
	p.write("docs/TECH.md", "# As built\n")

	r := shipping(p, "0007").ok()
	r.has("move durable facts", "docs/TECH.md")
	r.lacks("created docs/TECH.md")
	// And the other two destinations, since this is where the routing is decided: a
	// rule stated only in the protocol is read at the start of a session, not here.
	r.has("only what changed the plan", "the PR")
}

func TestShipCreatesTheAsBuiltDocOnce(t *testing.T) {
	t.Parallel()
	// The fallback, not where the file comes from: `kit init` places the as-built doc
	// with the rest of the layout, so this is a project set up before it did — where
	// "move durable facts into docs/TECH.md" would otherwise name a file that is absent.
	p := setUp(t)
	if err := os.Remove(p.path("docs/TECH.md")); err != nil {
		t.Fatalf("removing TECH.md: %v", err)
	}
	feature(p, "0007", "first", "First", "Draft", "- [x] Done")

	shipping(p, "0007").ok().has("created", "docs/TECH.md")
	if !p.exists("docs/TECH.md") {
		t.Fatal("shipping did not create the as-built doc")
	}
	// It answers tech-drift, which is the finding it exists to resolve.
	p.run("audit").ok().lacks("tech-drift")

	// A second ship must leave what has been written into it alone. Note this is
	// enforced by the caller finding the doc already present, not by the create —
	// once it exists, createTechDoc is never reached.
	p.write("docs/TECH.md", "# TECH\n\nReal content someone wrote.\n")
	feature(p, "0008", "second", "Second", "Draft", "- [x] Done")
	shipping(p, "0008").ok()
	if got := p.read("docs/TECH.md"); !strings.Contains(got, "Real content someone wrote.") {
		t.Errorf("shipping overwrote the as-built doc:\n%s", got)
	}
}

// loggy is a doc whose ## Log carries the design: `entries` sentences of record against a
// plan of about thirty words.
func loggy(p *project, id string, entries int) {
	p.t.Helper()
	log := ""
	for i := 0; i < entries; i++ {
		log += fmt.Sprintf("- 2026-01-0%d — the currency is decided by the merchant "+
			"account and the commission is written as basis points.\n", i%9+1)
	}
	p.write("docs/features/"+id+"-loggy.md", fmt.Sprintf(
		"---\nid: %s\ntitle: Mostly log\nstatus: Draft\ncreated: 2026-01-01\n---\n\n"+
			"## Problem\nSomething needs doing.\n\n## Design\nThis way.\n\n"+
			"## Tasks\n- [x] One\n\n## Log\n%s", id, log))
}

func TestShipRefusesADocThatIsMostlyLog(t *testing.T) {
	t.Parallel()
	// The state a real doc reached: 79% Log, holding the facts the project runs on.
	// Shipping freezes it, so this is the last moment they can be moved at all.
	p := setUp(t)
	loggy(p, "0007", 6)

	r := shipping(p, "0007").fails(2)
	r.errHas("## Log", "docs/TECH.md", "one line each")
	if strings.Contains(p.read("docs/features/0007-loggy.md"), "status: Shipped") {
		t.Error("ship wrote the status despite refusing")
	}
}

func TestShipAllowsALongLogUnderALongerPlan(t *testing.T) {
	t.Parallel()
	// The ratio is not a word count: the same Log ships under a ## Design that needed
	// the room. A check that fired here would fire on the docs doing this right.
	p := setUp(t)
	loggy(p, "0007", 6)
	doc := p.read("docs/features/0007-loggy.md")
	p.write("docs/features/0007-loggy.md", strings.Replace(doc, "## Design\nThis way.",
		"## Design\n"+strings.Repeat("The design says what is built and why. ", 40), 1))

	shipping(p, "0007").ok()
}
