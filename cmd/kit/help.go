package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// overview is everything needed to use the kit without reading its source. An
// agent that only has the binary should be able to work from this.
const overview = `kit — bootstrap projects and run them with agents.

Commands:
  kit init  [PATH] [--stack STACK]    set a project up, or fill in what is missing
  kit new   <title...>                start a feature doc, ID and date filled in
  kit harvest <title...> --box "..."  give finished work a doc and a commit
  kit claim <id>                      take a feature: push the branch, win or lose
  kit release <id> [--force]          give a claim back: delete the branch, both sides
  kit block <id> "<question>"         park a decision only a human can make
  kit ship  <id>                      tick the last box: verify, stamp, submit for review
  kit drop  <id> "<why>"              retire a feature that will not be built
  kit reset <id> "<why>"              put a doc back to Draft: the one repair
  kit next  [--json] [--offline]      what should I work on?
  kit audit [--json] [--strict] [--offline]
                                      where do the docs disagree with the repo?
  kit help  [TOPIC]                   this, or a topic below
  kit version

How it works:
  Plans live in docs/features/, one markdown file per feature, named NNNN-name.md.
  The directory listing is the board. There is no database and no queue: the docs
  and the git branches are the state.

    Draft -> Shipped   (exits: Dropped)

  Three statuses, and only "kit ship", "kit drop" and "kit reset" write them. Work
  starts when a claim branch is pushed, so there is no status for it and nothing to
  set by hand.

  Two routes in. Plan first — draft a round, land it, claim from it — which is what
  agents that never speak to each other need, because "needs:" is what orders them.
  Or work first on a branch that claims nothing, and run "kit harvest" at the end: it
  takes an id, writes the doc and commits the tree in one commit. Both end in a Draft
  somebody else reviews.

  "kit next" and "kit audit" read the project you are in. For several projects:
    for d in ~/code/*; do (cd $d && kit next); done

The four rules:
  1. Lowest ID first, and the claim this checkout already holds before anything
     free. Finishing beats starting.
  2. Within a feature, do the topmost unticked "## Tasks" box. List order is the
     order of work, so ordering needs no extra syntax.
  3. One agent per feature, one feature per checkout. "kit claim <id>" pushes the
     branch that claims it, and creating that ref on the remote is what decides:
     exit 1 means another agent won it. Local branches, remote branches and
     worktrees all count as claims. A won claim makes the worktree to work in;
     go there, not here.
  4. A feature named by a BLOCKING line in docs/INBOX.md is blocked until it is
     answered, and so is one whose "needs:" has not shipped. "Named" means the id
     field of the line, not a mention in the question text. FYI lines never block.
     See "kit help inbox" for the format.

The claim refusals you will meet most: an unmet "needs:", a BLOCKING inbox line, an
unwritten ## Problem, ## Design or ## Tasks, an id: that disagrees with the filename,
a duplicate id, or a plan that has not reached the default branch. "kit claim --help"
names every one.

Blocked on something only a human can decide? Run "kit block <id> \"<question>\"",
which writes docs/INBOX.md in the format the tool parses, then move to another
feature. Never guess, and never idle.

Exit codes, so a script can tell these apart:
  0  it worked
  1  it ran and the answer is no: "kit claim" lost the feature to another agent,
     or "kit audit --strict" found something HIGH
  2  the command could not run, or refused: unknown command, bad flag, or a
     refusal named above

Topics:
  kit help protocol      the rules agents follow, as written into AGENTS.md
  kit help template      the feature doc format, field by field
  kit help checks        every audit finding and what it means
  kit help inbox         the docs/INBOX.md format, and what BLOCKING means
  kit help backlog       the docs/features/BACKLOG.md format, and what promotes an entry
  kit help vision        the docs/VISION.md shape, and what it is not for
  kit help as-built      the docs/TECH.md shape, section by section
  kit help go-checklist      engineering practices for Go projects, tiered
  kit help python-checklist  the same for Python, and the libraries it does name

Run "kit <command> --help" for a command's options.`

// topics map a help topic to embedded content, so the binary carries its own
// documentation and an agent never needs the kit's repository.
var topics = map[string]string{
	"protocol":         "templates/AGENTS-section.md",
	"template":         "templates/features-TEMPLATE.md",
	"go-checklist":     "templates/go-checklist.md",
	"python-checklist": "templates/python-checklist.md",
	"vision":           "templates/VISION.md",
	"as-built":         "templates/TECH.md",
	"inbox":            "templates/INBOX.md",
	"backlog":          "templates/BACKLOG.md",
}

// commandHelp is the description and examples printed by `kit <cmd> --help`,
// above the flag list.
var commandHelp = map[string]string{
	"init": `kit init [PATH] [--stack STACK] [--forge FORGE]

Set a project up, or fill in what an existing one is missing.

Creates the directory if it does not exist, makes it a git repo if it is not one,
and writes AGENTS.md, a CLAUDE.md symlink to it, docs/VISION.md, docs/INBOX.md,
docs/TECH.md, docs/features/BACKLOG.md and a .gitignore for what an editor and an OS
leave behind.
Nothing else creates a project file, so there is one answer to where a file came
from. With --stack it also scaffolds the code layout, one CI pipeline, and that
stack's .gitignore in place of the default.

Which CI file that is depends on where the project is hosted, and it is worked out
from the first thing that actually says: a .gitlab-ci.yml or .github/workflows the
project already has, then the remote's host, then the directory layout when it
names a forge (~/code/gitlab.example.com/team/thing). When nothing says, no CI
file is written and the run tells you so — a missing pipeline is visible, and one
for the wrong forge is not. --forge settles it outright.

Nothing you wrote is overwritten or deleted, so it is safe on a populated directory
and safe to re-run. One kit-owned thing is refreshed on every run: the
"## Work protocol" section of AGENTS.md. Every document is yours the moment it is
placed, and house rules go above the marker comment in AGENTS.md.
Re-running is also how a project gets its CI file once it has a remote, and how the
IDs its docs already hold are reserved — after which deleting a doc cannot free its
number for a later "kit new" to hand out again.

Deployment scaffolding is deliberately not generated. Add it when the project has
something to deploy.

  kit init ~/code/github.com/you/thing --stack go
  kit init . --stack go --forge gitlab
  kit init .                     # fill in what is missing here`,

	"harvest": `kit harvest <title...> --box "<outcome>" [--box "<outcome>"]

Give finished work a doc and a commit. This is the route for a session that explored
first, on a branch that is neither a claim nor a plan round — the board does not route
one, and "kit next" says so rather than refusing you.

Harvest does not read history. It takes the next id, writes the doc, and commits the
whole working tree with it, so a session that produced two thousand lines and no commits
still becomes one reviewable change. That commit is also what leaves the clean working
tree "kit ship" refuses to run without.

Every box it writes is ticked: work that is finished has no unticked box to offer. The
boxes are yours to name, because no order of work can be recovered from a tree and a
commit message is not an outcome. Run it without --box and the refusal lists what the
tree holds, which is the raw material for naming them.

## Problem, ## Design and ## Log arrive as questions. They carry what the code cannot
say, and the kit will not invent them: "kit audit --strict" refuses the doc until they
are answered. The status is Draft, so the work still ends in a pull request somebody
else merges.

Refused on the trunk, where committing the whole tree is a change nobody could review,
and on a branch whose name cites a four-digit id, which would claim that feature.

--nothing is the spike that did not survive: it appends one BACKLOG.md entry, commits
that file alone, and takes no id. Trying something and throwing it away has to stay as
cheap as it is, and the entry is the one durable thing it leaves.

In an established repo, harvest renames the spike branch after the new feature,
so audit and the review board can follow its submission. An existing claim refuses
harvest. Push with git push -u <remote> HEAD after the gate and ship.

  kit harvest "Shorten URLs" --box "a long URL comes back short"
  kit harvest --nothing "a bloom filter costs more than the lookups it saves"`,

	"new": `kit new <title...>

Start a feature doc in the current project. Takes the next free four-digit ID,
fills in the title and today's date, and leaves status: Draft.

The ID is reserved before the doc is written, by pushing refs/kit/ids/<id>: that
either creates the ref or it does not, so of any number of agents drafting at
once no two are handed the same number. The ref outlives the doc, so deleting a
doc does not free its ID either.

With no remote there is nothing to compare and swap against. The doc is still
written, under the next free ID on disk, and the run says which guarantee it
could not give.

Fails rather than overwrites, so two agents drafting at once get an error instead
of one silently losing its work.

Refuses on a branch whose name cites a four-digit id that is not its own claim. Any
branch citing an id claims that feature, so a round drafted on one claims a feature
it has nothing to do with. Put the round on a "plan/" branch with no id in the name.

Then fill in Problem, Non-goals, Design, and Tasks. Anything unresolved goes in
## Open questions; anything only a human can answer goes in docs/INBOX.md via
"kit block". Never set status: or shipped: yourself — "kit ship", "kit drop" and
"kit reset" are the only things that write them.

  kit new "Shorten URLs"         # -> docs/features/0001-shorten-urls.md`,

	"claim": `kit claim <id>

Take a feature. Creates the branch named after its doc and pushes it, and the push
is what decides: creating the branch on the remote either succeeds or it does not,
so of any number of agents claiming at once exactly one wins.

  0  it is yours — the worktree to work in is made and named
  1  it is not yours: another agent got there first, a branch already claims it,
     an open INBOX question blocks it, something in its needs: has not shipped,
     the doc is closed or has no status:, the id: disagrees with the filename,
     or nobody wrote it — an empty ## Problem, ## Design or ## Tasks is refused
     whatever the status
  2  it could not run: no such feature, two docs share the id, no commits yet,
     the plan not committed here, the plan not landed on the default branch,
     this checkout already claims something or is ahead of the default branch,
     several remotes and no origin, or the push said nothing about the ref

Every exit 1 refusal is one "kit next" already makes, bar the race it cannot see: the
board and the commands read the same facts, so claiming by id without consulting it
cannot get you further. The exit 2 ones are about the repository rather than the plan,
and the board reports none of them.

The doc has to be on the default branch. Drafting and reading the board from a plan
branch are correct, and "next" reports the feature as blocked there rather than offering
it. A claim made from an unmerged round pushes a branch whose parent is that plan
branch, so the only way to integrate the work is a merge into the branch its own author
coordinated. Land the round, then claim. A project with no trunk to land on — one
branch, no remote — is unaffected.

Losing is normal. Run "kit next" and take something else.

Nothing local is written unless the push wins, so a lost race leaves no branch
behind. On a project with no remote the branch is still created and the command
says so — that claim is invisible to everyone else.

The worktree is created at .worktrees/<branch>. A path already holding the claim branch is
success and anything else there is left alone; making it cannot fail a claim the push has
won. A won claim also writes .worktrees/ to .git/info/exclude — git clone does not carry it.

This does not fetch first and does not edit the doc. A stale view only ever costs you
a clearer error, because the remote decides. Nothing to set afterwards either: the
branch is what says the work started.
Malformed needs:, dependency cycles and duplicate keys also refuse the claim.
  kit claim 0007
  kit claim 7`,

	"release": `kit release <id> [--force]

Stopping for good? Give the claim back. Deletes the branch that claims the feature on
the remote first, because that ref is the claim, and then locally.

Refuses while the branch carries anything beyond the feature doc. Stopping covers two
situations and only one of them is this: work worth nothing, and work worth keeping.
For the second, push the branch and open the pull request unfinished, saying why there
— the doc travels with the branch, so the record lands when it merges.

  0  the claim is gone and the feature is free for anyone to claim
  2  it could not run: no such feature, nothing claims it, two branches claim it,
     a branch this clone never created, a branch this checkout cannot resolve, a
     copy of it here that is not the remote's, no default branch to read work
     against, several remotes and no origin, the remote did not answer, or the
     branch carries work and --force was not given

--force deletes it anyway. Reach for it when the commits are genuinely worth nothing;
git keeps them reachable only through the reflog afterwards.

Release removes the worktree the claim made, which is the other half of creating it. It
cannot remove the one you are standing in, and a worktree you put elsewhere is yours: the
claim is released either way and the command says what is left, because until the branch
goes this checkout still reads the feature as taken.

Uncommitted work in its worktree also refuses release. A commit changing only the doc is work.
The remote delete uses the observed tip as a lease, even with --force; a later push
cannot be deleted by an earlier decision.

  kit release 0007
  kit release 7 --force`,

	"block": `kit block <id> "<question>" [--fyi]

Blocked on something only a human can decide? Write it here and move on. Appends
one line to docs/INBOX.md in the format the tool parses:

  - [ ] YYYY-MM-DD · <id> · BLOCKING|FYI · <question>

"kit next" then stops offering that feature until a human answers and deletes the
line. With --fyi it is recorded without blocking, for anything work can continue
without.

Use this rather than editing INBOX.md by hand. A line that misses the format is
matched by prose instead of by its id field, so it blocks whatever it happens to
mention — "kit audit" reports those as inbox-unparsed.

State a leaning. It turns most questions into a yes/no.

  kit block 0007 "Should codes expire? Leaning: no, keep forever."
  kit block 7 "Using base62 for slugs." --fyi`,

	"ship": `kit ship <id>

Submit finished work for review. Refuses unless every "## Tasks" box is ticked,
sets status: Shipped, stamps shipped: with today's date, and commits that stamp.

Exit 2 when the doc is already Shipped or Dropped, has an unknown status or
no status: line, no ## Tasks boxes, an unwritten Problem or Design, a BLOCKING
question, ambiguous fields or ids, or invalid dependencies. If needs: has not
shipped, finish the prerequisites first. If it is claimed by a branch other than
this checkout's, use the owning worktree.

The command also refuses uncommitted or untracked files and an unreadable git
status. A failure is not proof of a clean tree. If committing it failed after the
stamp was written, the error names the file to commit.

Before this command, move durable facts to docs/TECH.md, keep only changes to the
plan in Log (an empty Log is fine), run the project gate, and run kit audit --strict.
Ship verifies the document; it does not run or certify the project's tests.

Push the branch and open the pull request. Do not merge your own work.
For review changes before merge, use kit reset <id> "<why>", revise the tasks,
then repeat the gate and ship. A Shipped document on the trunk is frozen history.

  kit ship 0007
  kit ship 7`,

	"drop": `kit drop <id> "<why>"

Retire a feature that will not be built. Sets status: Dropped, records the reason in
"## Log", and commits that doc — a status left in the working tree is one session
from being read as debris. Only the doc: whatever else the tree carries is yours.
Refuses at exit 2 when the commit fails, naming the file the status is already on.

Feature docs live on the default branch at every status, so an unwanted one cannot
be disposed of by deleting a branch — left alone it sits on the board for ever and
every agent reads it as work nobody has started.

Refuses a Shipped doc: that is frozen history and the work is in the repo, so
retiring it would make the record say the feature was never built.

The branch is not this command's to delete: that is "kit release". A leftover branch
named after the doc still reads as a claim, and dropping the doc says so rather than
acting.

It also names any feature whose needs: cites the dropped one. A Dropped feature can
never ship, so those docs wait for ever until their needs: is edited — and the board
says so too, rather than promising a wait that will not end.

Unreadable branch objects refuse id release: fetch before trying again.
Reservation deletion leases the observed ref, so a concurrent replacement survives.

Given an id no doc holds, it releases that id's reservation instead, deleting
refs/kit/ids/<id> so the number goes back into circulation. A cut draft leaves one
behind and nothing else can reach it. Three refusals: a doc for that id on any
branch's tree, a reservation younger than the stalled horizon, and a list of
reservations the remote could not be asked for. There is no doc to record the reason
in, so say it in the pull request.

  kit drop 0007 "superseded by 0011's router"
  kit drop 7 "the metric it was chasing moved on its own"`,

	"reset": `kit reset <id> "<why>"

Put a doc back to status: Draft, clear shipped:, record the reason in Log, and
commit it. This repairs unknown statuses, revives Dropped plans and lets review
feedback revise a Shipped feature before it merges.

Refuses if already Draft, claimed by a branch other than this checkout's, or
carrying no frontmatter. If committing it failed after writing, commit the named
file to preserve the repair.

Reopening it from Shipped requires an unmerged working branch and a readable trunk.
With a remote, reset checks its published trunk; fetch first if its objects are
missing. A document already Shipped on the trunk is frozen history: create a new
feature for further work.

  kit reset 0007 "review requested another case"
  kit reset 7 "In Progress was never a status the kit writes"`,

	"next": `kit next [--json] [--offline]

What should I work on? Reads the feature docs and git, and names one task.

Ranks by lowest ID, and within a feature returns the topmost unticked "## Tasks"
box. What this checkout already claims comes first of all.

Blocked rather than offered: a doc with an empty ## Problem, ## Design or ## Tasks, and
a feature whose plan round is not on the trunk yet. "kit claim" refuses both.

Each feature shows ticked-over-total, "2/4", carried in --json as "done" and "total".
For a feature somebody else holds the count comes from the branch the remote has,
where the boxes are actually ticked, and the entry says when that branch last moved.
A claim this checkout has never fetched shows no count rather than a stale one. A
count of boxes and not an estimate: boxes are not equal sizes.

A feature is reported as taken when a branch cites its id — local branches, remote
branches and worktrees all count, so a stale fetch hides someone else's claim. The
one exception is the branch this checkout is on: that claim is yours, and it is
reported as "yours" and offered as the work to do. A feature is reported as blocked
when a BLOCKING docs/INBOX.md line carries its id in the id field, or when something
in its "needs:" has not shipped yet.

Below the board, "awaiting review" is the far side of ship: a branch the remote has,
carrying a doc that says Shipped, not yet on the trunk, and off the board above while
it waits. The status is read from the branch, because on the trunk it still says what
it said when the plan round landed. Entries you built are marked; where reviewing is
all there is, the closing guidance says so instead of telling you to stop.

What is ready is how much can happen at once: three ready features means three agents
can take one each, which falls out of what the docs declare rather than being planned.

The board asks the remote which branches exist, bounded at 15 seconds. If that does not
answer the page says so: claims fall back to the last fetch, and anything awaiting review
is missing rather than absent. --offline skips the question and stays quiet.

A directory with no docs/features exits 2: nothing was surveyed, so an empty board is
not a report that there is nothing to do.

Reads only. Uses the repo you are in, from any subdirectory. For several projects:
  for d in ~/code/*; do (cd $d && kit next); done

  kit next                       # this project
  kit next --json                # for agents
  kit next --offline             # do not ask the remote which branches exist`,

	"audit": `kit audit [--json] [--strict] [--offline]

Where do the docs disagree with the repo? Names the file, the problem, and the fix.

Findings are ranked HIGH (a doc says something untrue, or work is stuck and nobody
noticed) and MEDIUM (incomplete or drifting). See "kit help checks" for every
finding and what it means.

--offline names the checks it did not run, because a check that was skipped and one
that passed otherwise print the same page. A directory with no docs/features exits 2:
nothing was read, so there is no finding and no clean bill either.
With --strict, an unanswered remote also exits 2. --offline explicitly limits the audit.

Leaves project files unchanged; online reservation checks refresh git refs. Uses the repo you are in. For several projects:
  for d in ~/code/*; do (cd $d && kit audit); done

  kit audit                      # this project
  kit audit --strict             # exit 1 on any HIGH, for CI
  kit audit --offline            # do not ask the remote which branches exist`,
}

// runHelp prints the overview, or one topic.
func runHelp(templates fs.FS, args []string) int {
	// `kit help -h` asked for the overview and got "no topic \"-h\"" at exit 2, which is
	// the one command where being turned away for asking is absurd.
	if len(args) == 0 || isHelpFlag(args[0]) {
		fmt.Println(overview)
		return 0
	}
	topic := args[0]
	if topic == "checks" {
		printChecks()
		return 0
	}
	path, ok := topics[topic]
	if !ok {
		fmt.Fprintf(os.Stderr, "kit help: no topic %q\n\nTopics: %s\n",
			topic, strings.Join(topicNames(), ", "))
		return 2
	}
	raw, err := fs.ReadFile(templates, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit help: %v\n", err)
		return 2
	}
	fmt.Println(strings.TrimRight(string(raw), "\n"))
	return 0
}

// printChecks lists every finding audit can report, straight from the registry
// audit itself uses, so this can never drift from the tool's behaviour.
func printChecks() {
	fmt.Print("Every finding kit audit can report.\n\n")
	for _, severity := range kit.Severities {
		// A level with nothing in it is not printed. Every check that was hygiene has
		// been deleted or merged into one that is not, so LOW has no members — and a
		// heading over an empty list advertises findings the tool cannot report.
		printed := false
		for _, c := range kit.Checks {
			if c.Severity != severity {
				continue
			}
			if !printed {
				fmt.Printf("%s\n", severity)
				printed = true
			}
			fmt.Printf("  %-18s %s\n", c.Code, c.Means)
		}
		if printed {
			fmt.Println()
		}
	}
	// The thresholds used to be restated here and drifted from the constants the
	// moment they were retuned. Each check now carries its own in the line above.
}

func topicNames() []string {
	names := []string{"checks"}
	for name := range topics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
