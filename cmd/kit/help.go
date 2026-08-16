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
  kit accept <id>                     settle a Draft — the human's call, not an agent's
  kit claim <id>                      take a feature: push the branch, win or lose
  kit block <id> "<question>"         park a decision only a human can make
  kit ship  <id>                      tick the last box: verify, stamp, freeze
  kit drop  <id> "<why>"              retire a feature that will not be built
  kit next  [PATH ...] [--json] [--offline]
                                      what should I work on?
  kit audit [PATH ...] [--json] [--strict] [--offline]
                                      where do the docs disagree with the repo?
  kit help  [TOPIC]                   this, or a topic below
  kit version

How it works:
  Plans live in docs/features/, one markdown file per feature, named NNNN-name.md.
  The directory listing is the board and "status:" in the frontmatter is canonical.
  There is no database and no queue: the docs and the git branches are the state.

    Draft -> Accepted -> In progress -> Shipped   (exits: Dropped, Superseded)

  "status:" orders the board rather than gating it. Accepted is a human saying they
  have looked and blessed the design, so it is offered first; a Draft is invented
  and not yet blessed, so it is offered last and you should expect to settle its
  "## Open questions" before building. Neither is permission: what refuses a claim
  is a doc nobody wrote, and a BLOCKING line in docs/INBOX.md.

The four rules:
  1. In progress, then Accepted, then Draft, and lowest ID first within each.
     Finishing beats starting, and blessed beats unblessed.
  2. Within a feature, do the topmost unticked "## Tasks" box. List order is the
     order of work, so ordering needs no extra syntax.
  3. One agent per feature, one feature per checkout. "kit claim <id>" pushes the
     branch that claims it, and creating that ref on the remote is what decides:
     exit 1 means another agent won it. Local branches, remote branches and
     worktrees all count as claims. Work in the worktree it prints, not here.
  4. A feature named in an open docs/INBOX.md question is blocked until answered.
     "Named" means the id field of the line, not a mention in the question text.
     See "kit help inbox" for the format.

Blocked on something only a human can decide? Run "kit block <id> \"<question>\"",
which writes docs/INBOX.md in the format the tool parses, then move to another
feature. Never guess, and never idle.

Exit codes, so a script can tell these apart:
  0  it worked
  1  it ran and the answer is no: "kit claim" lost the feature to another agent,
     or "kit audit --strict" found something HIGH
  2  the command could not run: unknown command, bad flag, missing path

Topics:
  kit help protocol      the rules agents follow, as written into AGENTS.md
  kit help template      the feature doc format, field by field
  kit help conventions   how the features directory works
  kit help checks        every audit finding and what it means
  kit help inbox         the docs/INBOX.md format, and what BLOCKING means
  kit help vision        the docs/VISION.md shape, and what it is not for
  kit help as-built      the docs/TECH.md shape, section by section
  kit help go-checklist  engineering practices for Go projects, tiered
  kit help deployment    patterns for going to production, and their traps

Run "kit <command> --help" for a command's options.`

// topics map a help topic to embedded content, so the binary carries its own
// documentation and an agent never needs the kit's repository.
var topics = map[string]string{
	"protocol":     "templates/AGENTS-section.md",
	"template":     "templates/features-TEMPLATE.md",
	"conventions":  "templates/features-README.md",
	"go-checklist": "templates/go-checklist.md",
	"deployment":   "templates/deployment.md",
	"vision":       "templates/VISION.md",
	"as-built":     "templates/TECH.md",
	"inbox":        "templates/INBOX.md",
}

// commandHelp is the description and examples printed by `kit <cmd> --help`,
// above the flag list.
var commandHelp = map[string]string{
	"init": `kit init [PATH] [--stack STACK] [--forge FORGE]

Set a project up, or fill in what an existing one is missing.

Creates the directory if it does not exist, makes it a git repo if it is not one,
and writes docs/VISION.md, docs/features/{README,TEMPLATE,BACKLOG}.md, docs/INBOX.md
and AGENTS.md.
With --stack it also scaffolds the code layout and one CI pipeline.

Which CI file that is depends on where the project is hosted, and it is worked out
from the first thing that actually says: a .gitlab-ci.yml or .github/workflows the
project already has, then the remote's host, then the directory layout when it
names a forge (~/code/gitlab.example.com/team/thing). When nothing says, no CI
file is written and the run tells you so — a missing pipeline is visible, and one
for the wrong forge is not. --forge settles it outright.

Your files are never overwritten or deleted, so it is safe to run on a populated
directory and safe to re-run. Re-running is how a project picks up protocol
changes: only the "## Work protocol" section of AGENTS.md is refreshed, and it is
also how a project gets its CI file once it has a remote.

Deployment scaffolding is deliberately not generated. Add it when the project has
something to deploy.

  kit init ~/code/github.com/you/thing --stack go
  kit init . --stack go --forge gitlab
  kit init .                     # fill in what is missing here`,

	"new": `kit new <title...>

Start a feature doc in the current project. Takes the next free four-digit ID,
fills in the title and today's date, and leaves status: Draft.

Fails rather than overwrites, so two agents drafting at once get an error instead
of one silently losing its work.

Then fill in Problem, Non-goals, Design, and Tasks. Anything unresolved goes in
## Open questions and the doc stays Draft; anything only a human can answer goes
in docs/INBOX.md. A human settles it with "kit accept"; never set the status
yourself.

  kit new "Shorten URLs"         # -> docs/features/0001-shorten-urls.md`,

	"accept": `kit accept <id>

Settle a Draft: set status to Accepted. This is the human's call, and it is a
signal rather than a gate — agents can work a Draft, and accepting says the design
has been looked at, which moves it up the board ahead of anything unblessed.

Refuses when there is nothing to settle, or when the doc says the design is not
settled yet:

  the status is not Draft          nothing to accept; the message names what it is
  ## Open questions is not empty   that section is what Draft is for
  a BLOCKING inbox line names it   a human already owes an answer, so answer that

Run it on the draft's own branch, so the acceptance lands in the same review
as the draft. Accepting after that request merges means a second one for a one-line
change.

No date is stamped. "created:" already records when the feature was proposed.

  kit accept 3                   # ids may be typed without leading zeros`,

	"claim": `kit claim <id>

Take a feature. Creates the branch named after its doc and pushes it, and the push
is what decides: creating the branch on the remote either succeeds or it does not,
so of any number of agents claiming at once exactly one wins.

  0  it is yours — a worktree command to start in is printed
  1  it is not yours: another agent got there first, a branch already claims it, an
     open INBOX question blocks it, something in its needs: has not shipped, the doc
     is closed, or nobody wrote it — an empty ## Problem, ## Design or ## Tasks is
     refused whatever the status
  2  it could not run: no such feature, no commits yet, the plan not landed on the
     default branch, several remotes and no origin

Every refusal here is one "kit next" already makes, bar the landing one. The board and
the commands read the same facts, so claiming by id without consulting it cannot get
you further.

The doc has to be on the default branch. Drafting and reading the board from a plan
branch are correct, so "next" goes on offering the feature there — taking it is what
waits. A claim made from an unmerged round pushes a branch whose parent is that plan
branch, so the only way to integrate the work is a merge into the branch its own author
coordinated. Land the round, then claim. A project with no trunk to land on — one
branch, no remote — is unaffected.

Losing is normal. Run "kit next" and take something else.

Nothing local is written unless the push wins, so a lost race leaves no branch
behind. On a project with no remote the branch is still created and the command
says so — that claim is invisible to everyone else.

This does not fetch first and does not edit the doc. A stale view only ever costs
you a clearer error, because the remote decides; set status: In progress yourself
once you start.

  kit claim 0007
  kit claim 7`,

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

Mark a feature Shipped. Refuses unless every "## Tasks" box is ticked, sets
status: Shipped, and stamps shipped: with today's date.

Two more refusals, both exit 2. Something in its needs: has not shipped — the plan
says this was built on a foundation that is not there, so the plan is wrong and you
have to look. Or the feature is claimed by a branch that is not this checkout's:
ship it from that worktree, because "Shipped" is history nobody may edit and doing
it from elsewhere freezes another agent's work mid-flight.

Doing this by hand is where two audit findings come from — unticked-shipped and
shipped-no-date — so the command removes the mistake rather than reporting it
afterwards.

A Shipped doc is frozen history: durable facts belong in docs/TECH.md from then
on, and the doc itself must not be edited again.

  kit ship 0007
  kit ship 7`,

	"drop": `kit drop <id> "<why>"

Retire a feature that will not be built. Sets status: Dropped and records the
reason in "## Log".

Feature docs live on the default branch at every status, so an unwanted one cannot
be disposed of by deleting a branch — left alone it sits on the board for ever and
every agent reads it as work nobody has started.

Refuses a Shipped doc: that is frozen history and the work is in the repo, so
retiring it would make the record say the feature was never built.

The branch is not this command's to delete. A leftover branch named after the doc
still reads as a claim, and dropping the doc says so rather than acting.

It also names any feature whose needs: cites the dropped one. A Dropped feature can
never ship, so those docs wait for ever until their needs: is edited — and the board
says so too, rather than promising a wait that will not end.

  kit drop 0007 "superseded by 0011's router"
  kit drop 7 "the metric it was chasing moved on its own"`,

	"next": `kit next [PATH ...] [--json]

What should I work on? Reads the feature docs and git, and names one task.

Ranks In progress before Accepted before Draft, then by lowest ID, and within a
feature returns the topmost unticked "## Tasks" box.

A Draft is offered like anything else — accepting promotes work rather than
unlocking it. A doc with an empty ## Problem or ## Design is reported as blocked
instead, because "kit claim" would refuse it.

Several paths survey several projects at once. One that cannot be read is reported
in its place and sets the exit code to 2; the others still report, in the order
given, because a tidied directory says nothing about the projects beside it.

Each feature shows its ticked-over-total, as "2/4", and --json carries the same as
"done" and "total". It is a count of boxes, not an estimate: boxes are not equal
sizes, so nothing here should be read as how much time is left.

A feature is reported as taken when a branch cites its id — local branches, remote
branches and worktrees all count, so a stale fetch hides someone else's claim. The
one exception is the branch this checkout is on: that claim is yours, and it is
reported as "yours" and offered as the work to do. A feature is reported as blocked
when an open docs/INBOX.md line carries its id in the id field, or when something in
its "needs:" has not shipped yet.

What is ready is therefore the answer to how much can happen at once: one ready
feature means the work is a chain, three means three agents can take one each. That
falls out of what the docs already declare rather than being planned separately.

Reads only. With no PATH it uses the repo you are in, from any subdirectory.

  kit next                       # this project
  kit next ~/code/*              # every project at once
  kit next --json                # for agents
  kit next --offline             # do not ask the remote which branches exist`,

	"audit": `kit audit [PATH ...] [--json] [--strict]

Where do the docs disagree with the repo? Names the file, the problem, and the fix.

Findings are ranked HIGH (a doc says something untrue, or work is stuck and nobody
noticed), MEDIUM (incomplete or drifting), LOW (hygiene). See "kit help checks"
for every finding and what it means.

Reads only, and never edits a file. With no PATH it uses the repo you are in.

Several paths audit several projects. One that cannot be read is reported in its
place and exits 2, which outranks --strict's 1 — a path you asked about and did not
get is a worse answer than findings you did get.

  kit audit                      # this project
  kit audit ~/code/*             # every project at once
  kit audit . --strict           # exit 1 on any HIGH, for CI
  kit audit --offline            # do not ask the remote which branches exist`,
}

// runHelp prints the overview, or one topic.
func runHelp(templates fs.FS, args []string) int {
	if len(args) == 0 {
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
		fmt.Printf("%s\n", severity)
		for _, c := range kit.Checks {
			if c.Severity == severity {
				fmt.Printf("  %-18s %s\n", c.Code, c.Means)
			}
		}
		fmt.Println()
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
