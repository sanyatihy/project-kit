package main

// Conformance: `kit new`. IDs must be stable and unique, because branches and old
// commits cite them — a reused number makes history point at the wrong document.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// setUp returns a project that has already been initialised, with its vision
// answered: the stub `kit init` leaves is itself a finding, so a project still
// carrying it is not a quiet baseline.
func setUp(t *testing.T) *project {
	t.Helper()
	p := newProject(t)
	p.run("init").ok()
	writeVision(p)
	return p
}

// reservedIDs lists the ids the remote has handed out. The reservation is the ref and
// nothing else, so this is the only place a test can read one.
func (p *project) reservedIDs() []string {
	p.t.Helper()
	cmd := exec.Command("git", "ls-remote", "origin", "refs/kit/ids/*")
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		p.t.Fatalf("reading the remote's reservations: %v", err)
	}
	var ids []string
	for _, line := range strings.Split(string(out), "\n") {
		if _, ref, found := strings.Cut(strings.TrimSpace(line), "\t"); found {
			ids = append(ids, strings.TrimPrefix(ref, "refs/kit/ids/"))
		}
	}
	sort.Strings(ids)
	return ids
}

// reservationFor is the subject of the commit a reservation points at, fetched from the
// remote rather than read locally, so the test sees what another clone would.
func (p *project) reservationFor(id string) string {
	p.t.Helper()
	p.git("fetch", "-q", "origin", "refs/kit/ids/"+id)
	cmd := exec.Command("git", "log", "-1", "--format=%s", "FETCH_HEAD")
	cmd.Dir, cmd.Env = p.Root, gitEnv()
	out, err := cmd.Output()
	if err != nil {
		p.t.Fatalf("reading the reservation for %s: %v", id, err)
	}
	return strings.TrimSpace(string(out))
}

// The two agents fixture is a bare remote and two clones of it, both carrying the doc
// for 0007 — so the next free id on disk is 0008 in each of them.

func TestNewReservesTheIDOnTheRemote(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)

	a.run("new", "Eighth").ok().has("0008-eighth.md", "reserved  0008")

	// The doc says which id this agent took; only the ref says nobody else can be
	// handed it, and the ref is what a second clone reads.
	if got := a.reservedIDs(); !slices.Contains(got, "0008") {
		t.Errorf("the remote holds %v, none of which reserves 0008 — the next drafter "+
			"off this trunk is handed the same number", got)
	}
}

func TestNewSkipsAnIDTheRemoteHasAlreadyHandedOut(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	// The state every second drafter is in: disk says 0008 is free and the remote says
	// it is gone. What the ref points at is not the message — its existence is — so a
	// reservation made by hand out of a commit has to count too.
	a.git("push", "-q", "origin", "HEAD:refs/kit/ids/0008")

	a.run("new", "Ninth").ok().has("0009-ninth.md")

	if a.exists("docs/features/0008-ninth.md") {
		t.Error("new took 0008, which the remote had already handed out")
	}
	if got := a.reservedIDs(); !slices.Contains(got, "0009") {
		t.Errorf("the remote holds %v, none of which reserves the 0009 it just wrote", got)
	}
}

// The test this feature is for.
func TestTwoAgentsDraftingAtOnceGetDifferentIDs(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)

	first := a.run("new", "Mine").ok()
	// b cloned before a drafted, so its view is stale in exactly the way a second
	// agent's is: on disk 0007 is the highest doc it can see, and nothing local says
	// 0008 has been handed out. Six runs like this once produced four docs claiming
	// the same number.
	second := b.run("new", "Yours").ok()

	first.has("0008-mine.md")
	second.has("0009-yours.md")
	if b.exists("docs/features/0008-yours.md") {
		t.Fatal("both agents were handed 0008; every branch and commit citing it would " +
			"name two features")
	}

	// Each drafter reserved the number it took, so a third one skips both.
	if got := a.reservedIDs(); !slices.Equal(got, []string{"0008", "0009"}) {
		t.Errorf("the remote reserves %v, want 0008 and 0009", got)
	}
}

func TestADeletedDocsIDIsNeverHandedOutAgain(t *testing.T) {
	t.Parallel()
	// The bug this closes: the highest doc is deleted, disk offers its number again, and
	// every branch and commit citing it comes to name a different feature. The ref
	// outlives the file, so the number stays gone.
	a, _ := twoAgents(t)
	a.run("new", "Eighth").ok()
	if err := os.Remove(a.path("docs/features/0008-eighth.md")); err != nil {
		t.Fatalf("deleting the doc: %v", err)
	}

	a.run("new", "Ninth").ok().has("0009-ninth.md")

	if a.exists("docs/features/0008-ninth.md") {
		t.Error("0008 was handed out again once its doc was deleted, so every commit " +
			"citing it now names two features")
	}
}

func TestNewStartsAtOneAndCounts(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("new", "Shorten URLs").ok().has("0001-shorten-urls.md")
	p.run("new", "Rate limit").ok().has("0002-rate-limit.md")
}

func TestNewTakesTheHighestIDNotTheFirstGap(t *testing.T) {
	t.Parallel()
	// Filling a gap would reuse the number of a dropped feature, and every branch
	// and commit citing it would then point at the wrong doc.
	p := setUp(t)
	p.write("docs/features/0001-first.md", "---\nid: 0001\nstatus: Shipped\n---\n")
	p.write("docs/features/0003-third.md", "---\nid: 0003\nstatus: Shipped\n---\n")

	p.run("new", "Fourth").ok().has("0004-fourth.md")
	if p.exists("docs/features/0002-fourth.md") {
		t.Error("new filled the 0002 gap; IDs must only ever go up")
	}
}

func TestNewSlugsTheTitle(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	for _, tc := range []struct{ title, want string }{
		{"Shorten URLs", "0001-shorten-urls.md"},
		{"Rate-limit POST by IP!", "0002-rate-limit-post-by-ip.md"},
		{"  Leading and trailing  ", "0003-leading-and-trailing.md"},
	} {
		p.run("new", tc.title).ok().has(tc.want)
		if !p.exists("docs/features/" + tc.want) {
			t.Errorf("new %q did not write %s", tc.title, tc.want)
		}
	}
}

func TestNewFillsTheFrontmatter(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.run("new", "Shorten URLs").ok()
	doc := p.read("docs/features/0001-shorten-urls.md")

	// The binary still reads the real clock — only the library takes a date — so
	// today is the honest expectation here. It is exact rather than "some date"
	// because NewFeature no longer invents one.
	for _, want := range []string{
		"id: 0001",
		"title: Shorten URLs",
		"created: " + time.Now().UTC().Format("2006-01-02"),
		"status: Draft",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("the new doc is missing %q:\n%s", want, doc)
		}
	}
	// Draft is the only status a doc starts in, and the only one work starts from.
	if !strings.Contains(doc, "status: Draft") {
		t.Errorf("a new doc must start as a Draft:\n%s", doc)
	}
}

func TestANewDocIsEmptyToTheToolNotJustToTheReader(t *testing.T) {
	t.Parallel()
	// The template's guidance lives in HTML comments, which every parser here strips.
	// When it lived in visible prose and two `- [ ] …` boxes, a doc nobody had written
	// read as a plan: `next` counted 0/2 and handed out `…` as the task to build, and
	// `audit` said the docs and the repo agreed.
	p := setUp(t)
	p.run("new", "Untouched").ok()
	// Committed, because that is what the plan checks gate on: a doc nobody has written
	// is a defect once it reaches the tree others read, and a scratch file is not.
	p.commitAll("the unwritten doc")

	// Nothing to hand out, and the audit says why rather than nothing.
	r := p.run("next").ok()
	r.lacks("…")
	p.run("audit").ok().has("unfilled-template")

	// One defect, one finding: an unwritten doc used to report unfilled-template for its
	// empty sections and a second code for the missing boxes, two names for one plan
	// nobody wrote.
	if codes := p.auditCodes(); len(codes) != 3 {
		t.Errorf("an unwritten doc reported %v; three empty sections, one name", codes)
	}
}

func TestNewNeverClobbersAnExistingFile(t *testing.T) {
	t.Parallel()
	// Even a file that is not a valid feature doc counts toward the next ID, so
	// the name is skipped rather than the content replaced.
	p := setUp(t)
	p.write("docs/features/0001-shorten-urls.md", "not a feature doc at all\n")

	p.run("new", "Shorten URLs").ok()

	if got := p.read("docs/features/0001-shorten-urls.md"); got != "not a feature doc at all\n" {
		t.Errorf("new overwrote an existing file: %q", got)
	}
	if !p.exists("docs/features/0002-shorten-urls.md") {
		t.Error("new did not fall through to the next free ID")
	}
}

func TestNewUnderConcurrencyLosesNothing(t *testing.T) {
	t.Parallel()
	// One agent per feature is the model, so several can run `new` at once. A
	// loser must report an error rather than silently overwrite the winner. The
	// interleaving is not deterministic, so assert the invariant instead: every
	// success left its own non-empty file behind.
	p := setUp(t)

	const racers = 4
	var wg sync.WaitGroup
	codes := make([]int, racers)
	for i := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = p.run("new", "Race").Code
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, c := range codes {
		if c == 0 {
			succeeded++
		}
	}
	if succeeded == 0 {
		t.Fatal("every concurrent new failed; at least one must win")
	}

	matches, err := filepath.Glob(p.path("docs/features/*-race.md"))
	if err != nil {
		t.Fatalf("globbing: %v", err)
	}
	if len(matches) != succeeded {
		t.Errorf("%d runs reported success but %d files exist — one overwrote another",
			succeeded, len(matches))
	}
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.Size() == 0 {
			t.Errorf("%s is empty; a doc was announced but not written", filepath.Base(m))
		}
	}
}

func TestNewWithNoRemoteNamesTheGuaranteeItCannotGive(t *testing.T) {
	t.Parallel()
	// A project with no remote still drafts: refusing would make the kit unusable
	// before the first push. What it cannot do is reserve, and the number it hands out
	// is then the highest one on disk — which is the state every project was in before
	// this, and it stayed invisible because nothing said so.
	p := setUp(t)

	r := p.run("new", "Alone").ok().has("0001-alone.md")
	r.has("no remote", "nothing reserved 0001", "handed the same id")
	r.lacks("reserved 0001 on")
}

func TestNewRefusesATitleItCannotUse(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	// The empty cases are caught by the CLI before NewFeature sees them, so these
	// pin the behaviour a user gets, not which layer produces it — NewFeature's own
	// empty-title guard is unreachable through the binary and defends direct
	// library callers.
	p.run("new").fails(2).errHas("title")
	p.run("new", "   ").fails(2).errHas("title")
	// Punctuation alone slugs to nothing, which would produce "0001-.md". This one
	// is NewFeature's.
	p.run("new", "!!!").fails(2).errHas("no usable characters")
}

func TestNewRefusesToDraftOnABranchThatClaimsAFeature(t *testing.T) {
	t.Parallel()
	// A branch citing a four-digit id claims that feature, wherever in the name it
	// sits, so a plan round drafted on one claims a feature it has nothing to do with:
	// `plan/fold-in-0007-fix` made the board report 0007 as taken while `audit` exited
	// 0 saying the docs and the repo agreed. Closed here rather than by narrowing what
	// counts as a claim, which would have made the guarantee smaller.
	a, _ := twoAgents(t)
	a.git("checkout", "-q", "-b", "plan/fold-in-0007-fix")

	a.run("new", "Later").fails(2).errHas("0007", "no four-digit id")

	if matches, err := filepath.Glob(a.path("docs/features/*-later.md")); err != nil ||
		len(matches) > 0 {
		t.Errorf("a refused draft wrote %v", matches)
	}
	if got := a.reservedIDs(); slices.Contains(got, "0008") {
		t.Errorf("a refused draft burned an id: the remote reserves %v", got)
	}

	// The quiet half: a round whose name cites nothing, which is what the protocol asks
	// for, and a feature's own claim branch, where work discovered mid-build is written.
	a.git("checkout", "-q", "-b", "plan/round-two")
	a.run("new", "Later").ok().has("0008-later.md")
	a.commitAll("the second round")
	a.git("checkout", "-q", "-b", "0008-later")
	a.run("new", "Discovered").ok().has("0009-discovered.md")
}

func TestNewOnAnUnsetUpProjectSaysHowToFixIt(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	r := p.run("new", "Anything").fails(2)
	r.errHas("kit init")
}

func TestAReservationSaysWhatItWasHandedOutFor(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)

	a.run("new", "Eighth").ok()

	// The ref pointed at the shared empty tree, so a number handed out for a draft that
	// was then cut recorded nothing — no title, no date, no author — and nobody could
	// tell it from one still being written.
	if got, want := a.reservationFor("0008"), "0008 Eighth"; got != want {
		t.Errorf("the reservation reads %q, want %q", got, want)
	}
}

// A locked-in session leaves a working tree, not a log: the project that prompted this
// had 2,090 lines, 82 tests and no commits at all. Harvest has to make the history rather
// than read it, and a repository with no commits is where that is provable.
func TestHarvestMakesTheCommitOnARepoWithNoCommits(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("thing.py", "print('hi')\n")

	r := p.run("harvest", "Shorten URLs",
		"--box", "a long URL comes back short",
		"--box", "an unknown code is a 404").ok()
	r.has("created", "committed")

	// The doc, with the operator's boxes ticked and the questions left to answer.
	doc := p.read("docs/features/0001-shorten-urls.md")
	for _, want := range []string{
		"status: Draft",
		"- [x] a long URL comes back short",
		"- [x] an unknown code is a 404",
		"## Problem", "## Design", "## Log",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("the harvested doc does not carry %q:\n%s", want, doc)
		}
	}

	// The commit is the point: the work and the doc arrive together, and the tree is
	// clean afterwards, which is what `kit ship` refuses to run without.
	if dirty := p.run("audit").out(); strings.Contains(dirty, "status-uncommitted") {
		t.Errorf("harvest left the doc uncommitted:\n%s", dirty)
	}
	out := p.gitOut(nil, "status", "--porcelain")
	if strings.TrimSpace(out) != "" {
		t.Errorf("harvest left the tree dirty:\n%s", out)
	}
	if log := p.gitOut(nil, "log", "--oneline"); !strings.Contains(log, "Harvest 0001") {
		t.Errorf("no harvest commit:\n%s", log)
	}
	if files := p.gitOut(nil, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(files, "thing.py") {
		t.Errorf("the commit does not carry the work:\n%s", files)
	}
}

// The boxes are the operator's, because no order of work can be recovered from a tree.
// The refusal hands back the one mechanical half there is: what the tree holds.
func TestHarvestRefusesToInventTheBoxes(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("thing.py", "print('hi')\n")

	p.run("harvest", "Shorten URLs").fails(2).errHas("--box", "thing.py")

	if p.exists("docs/features/0001-shorten-urls.md") {
		t.Error("a refused harvest wrote a doc anyway")
	}
}

// The spike that did not survive. Throwing work away has to stay cheap, so this takes no
// id and writes no doc — the entry is the one durable thing it leaves behind.
func TestHarvestNothingTakesNoID(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("thing.py", "print('hi')\n")

	p.run("harvest", "--nothing", "a bloom filter costs more than the lookups it saves").
		ok().has("appended", "no id taken")

	backlog := p.read("docs/features/BACKLOG.md")
	if !strings.Contains(backlog, "a bloom filter costs more than the lookups it saves") {
		t.Errorf("the entry did not reach the backlog:\n%s", backlog)
	}
	if docs, _ := filepath.Glob(p.path("docs/features/0*.md")); len(docs) != 0 {
		t.Errorf("--nothing wrote %v, and it takes no id", docs)
	}
	// The discarded work stays discarded: only the entry is committed.
	if out := p.gitOut(nil, "status", "--porcelain"); !strings.Contains(out, "thing.py") {
		t.Errorf("--nothing committed the work it was throwing away:\n%s", out)
	}

	// A title or a box contradicts it: those are the shape for work that survived.
	p.run("harvest", "--nothing", "something", "A title").fails(2).errHas("no title")
}

// The handoff: a spike ends where the operator knows what is worth doing next and the
// fleet has nothing to claim, so harvest names the parked entries there. Named, not
// promoted — which of them is worth building is not the kit's decision.
func TestHarvestOffersTheParkedEntries(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	p.write("docs/features/BACKLOG.md", "# Backlog\n\n"+
		"- **A bloom filter in front of the store** — one line. Promote when: reads cost.\n\n"+
		"## Batching the writes\n\nProse about it.\n")
	p.write("thing.py", "print('hi')\n")

	r := p.run("harvest", "Shorten URLs", "--box", "a long URL comes back short").ok()
	r.has("2 entries", `kit new "A bloom filter in front of the store"`,
		`kit new "Batching the writes"`)

	// Offered, not taken: nothing was drafted for them.
	if docs, _ := filepath.Glob(p.path("docs/features/0002-*.md")); len(docs) != 0 {
		t.Errorf("harvest promoted %v on the operator's behalf", docs)
	}
}
