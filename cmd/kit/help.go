package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// Legacy spellings route through the same guards as the canonical commands.
func canonicalCommand(name string) string {
	if name == "ship" {
		return "submit"
	}
	if name == "reset" {
		return "reopen"
	}
	return name
}

// overview is everything needed to use the kit without reading its source. An
// agent that only has the binary should be able to work from this.
const overview = `kit — bootstrap projects and run them with agents.

Commands:
  kit start [<id>|--plan <round>|--work <outcome>]  prepare requested work
  kit finish [--json] [-- <gate argv>]  validate and publish for review
  kit init  [PATH] [--stack STACK]    set a project up, or fill in what is missing
  kit new   <title...>                start a feature doc, ID and date filled in
  kit harvest <title...> --box "..."  give finished work a doc and a commit
  kit claim <id>                      take a feature: push the branch, win or lose
  kit release <id> [--discard]          give a claim back: delete the branch, both sides
  kit block <id> "<question>"         park a decision only a human can make
  kit submit  <id>                      verify completed boxes, stamp, submit for review
  kit drop  <id> "<why>"              retire a feature that will not be built
  kit reopen <id> "<why>"              reopen an unmerged submission for review changes
  kit next  [--json] [--offline]      what should I work on?
  kit audit [--json] [--strict] [--offline]
                                      where do the docs disagree with the repo?
  kit help  [TOPIC]                   this, or a topic below
  kit version

How it works:
  Plans live in docs/features/, one markdown file per feature, named NNNN-name.md.
  The directory listing is the board. There is no database and no queue: the docs
  and the git branches are the state.

    Draft -> Submitted -> integrated on trunk   (exit: Dropped)

  The stored statuses are Draft, Submitted and Dropped. Submitted is a review
  snapshot; git distinguishes submission from integration. Only submit, drop and reopen
  write status fields. A claim branch records who has started work.
  Legacy Shipped/shipped: documents remain readable; ship/reset are compatibility
  spellings for submit/reopen and enforce the same guards.

  Plan first: kit start --plan "<round>", draft with kit new, then kit finish and
  open a PR for human approval. After merge, update trunk and use kit start <id>.
  "needs:" orders work for agents that never speak to each other.
  Directly requested work: kit start --work "<outcome>", then implement and commit.
  kit finish -- <gate argv> harvests if needed, validates, submits and publishes.
  The agent opens the PR with the supplied title; humans approve and merge.
  Setup belongs on the first PR over an empty trunk. Never merge your own work.
  After handoff, continue independent approved work from an updated trunk checkout.

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
     answered, and so is one whose "needs:" has not merged. "Named" means the id
     field of the line, not a mention in the question text. FYI lines never block.
     See "kit help inbox" for the format.

The claim refusals you will meet most: an unmet "needs:", a BLOCKING inbox line, an
unwritten ## Problem, ## Design or ## Tasks, an id: that disagrees with the filename,
a duplicate id, or a plan that has not reached the default branch. "kit claim --help"
names every one.

Blocked on something only a human can decide? Run "kit block <id> \"<question>\"",
which writes docs/INBOX.md in the format the tool parses, then move to another
free feature from another trunk checkout. If none is free, report the blocker and stop.

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
  kit help compatibility supported use, stable interfaces and coordinated upgrades
  kit help backlog       the docs/features/BACKLOG.md format, and what promotes an entry
  kit help vision        the docs/VISION.md shape, and what it is not for
  kit help as-built      the docs/TECH.md shape, section by section
  kit help go-checklist      engineering practices for Go projects, tiered
  kit help python-checklist  the same for Python, and the libraries it does name

Run "kit <command> --help" for a command's options.`

// topics map a help topic to embedded content, so the binary carries its own
// documentation and an agent never needs the kit's repository.
var topics = map[string]string{
	"compatibility":    "templates/compatibility.md",
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
	"finish": `kit finish [--json] [-- <gate argv>]

Plans: validate Draft plans and untouched tasks, commit only setup/docs, audit, push.
Work: supply an executable and arguments after --. Harvest recorded direct work
if needed; fill the requested Problem/Design facts and commit before retrying.
Then require complete tasks and a clean tree, run the gate, audit, submit and push.
Failed checks or changed candidates stop. Retries revalidate without duplicate stamps.
The handoff supplies pr_title from the feature title (plans: Plan: <round>); set it
explicitly when creating the PR instead of using commit autofill. It also names
base, branch, exact candidate and verified publication. It never
claims a PR exists: the agent opens one through its forge interface. Humans approve
and merge. A review handoff does not stop independent approved work from trunk.
No remote: report the missing PR handoff. Add the intended remote and retry.
`,
	"start": `kit start <id> | --plan "<round>" | --work "<outcome>"

Approved id: use the existing claim guards and enter the printed worktree.
Plan: record intent on plan/<round>, then use kit new and write the plans.
Direct work: record the operator's outcome on spike/<outcome>, then implement.
Start from clean, updated trunk. New repos get a published empty main base;
staged and unstaged files stay on the first review branch. Refuses foreign work, conflicting branches,
unreadable remotes and direct work matching an existing plan. A pending plan
never becomes a spike to avoid approval. Retrying the same recorded start is safe.
Run kit finish for review; humans approve plans and merge implementations.
`,
	"init": `kit init [PATH] [--stack STACK] [--forge FORGE]

Create missing project files; never clear the target first. Makes a directory and
git repository if needed, then AGENTS.md, its CLAUDE.md symlink, .gitignore,
docs/VISION.md, docs/INBOX.md, docs/TECH.md and docs/features/BACKLOG.md.
--stack adds code layout, stack gitignore and CI. Deployment is not scaffolded.

Preserves user files. Re-running refreshes only AGENTS.md's Work protocol section;
keep house rules outside it. It also backfills id reservations and missing CI.

Forge: --forge overrides detection; otherwise existing CI, remote host, then project
path. Unknown forge produces no CI and reports it. Invalid stack/forge, a non-directory
target or failed file/git operations refuse.

  kit init ~/code/github.com/you/thing --stack go
  kit init . --stack go --forge gitlab
  kit init .                     # fill in missing files`,

	"harvest": `kit harvest <title...> --box "<outcome>" [--box "<outcome>"]

Record human-directed exploration on a spike branch. Reserves an id, writes a Draft
with completed boxes, names the branch after the feature and commits the whole
working tree. Name observable outcomes with --box; history is not used to infer them.
Fill Problem and Design; keep Log for plan changes, which may be empty.
Without --box, the refusal lists the working tree to help name outcomes.

Refuses trunk, planning branches, detached HEAD and existing claims, including branch names citing a
four-digit id. Use a spike branch; never bypass a pending plan's approval. With no
commits, first creates an empty trunk review base; product files stay on the feature
branch. Publish that base before the first PR.

Then run the project gate, kit audit --strict and kit submit. Push the branch and
open a PR for human review; never merge it yourself.

--nothing "<lesson>" records a discarded spike in BACKLOG.md, commits only that file
and takes no id.

  kit harvest "Shorten URLs" --box "a long URL comes back short"
  kit harvest --nothing "a bloom filter costs more than the lookups it saves"`,

	"new": `kit new <title...>

Create docs/features/NNNN-name.md with id, title, date and status: Draft.
Use a plan/<round> branch with no four-digit id. Fill Problem, Non-goals, Design
and Tasks; unresolved choices go in Open questions and kit block when human-owned.
Push the planning branch and open a PR. The human approves through merge before
implementation. Do not set status or submitted fields by hand.

The remote reserves refs/kit/ids/<id> before writing; deleting a doc does not free
its id. No remote means local allocation without cross-clone guarantees.

Refuses an empty/unusable title, missing setup, a foreign claim branch, ambiguous
remotes, failed reservation or file creation. It never overwrites an existing doc.
Remote id allocation also refuses exhaustion of the four-digit range. A feature's
own claim branch may draft discovered work.

  kit new "Shorten URLs"`,

	"claim": `kit claim <id>

Acquire work/<id> and enter the printed worktree. Filename changes do not change
the claim. Start from an updated trunk checkout carrying the human-approved plan.
HEAD must equal the published trunk tip (local trunk with no remote).

Exit 0: yours, including a recovered acquisition. Work in .worktrees/<branch>.
Exit 1: not available: another claim, non-Draft/missing status, BLOCKING question,
unmet needs:, id/filename mismatch, ambiguous fields, invalid dependencies or an
unwritten Problem, Design or Tasks. Run kit next and choose other work.
Exit 2: cannot run: missing/duplicate id, no history, uncommitted or unlanded plan,
checkout already holding a claim, stale/unreadable trunk, ambiguous remotes or an
unconfirmed push result. Fix the reported cause before retrying.

Exclusive remote branch creation chooses the winner. No remote means a local claim
invisible to independent clones. Next and claim share eligibility checks.

A lost push acknowledgement preserves a unique receipt; retry from the same clone
when the remote answers. Confirmed ownership survives rebases. A worktree failure
does not undo a won claim: preserve obstructing files and retry. Existing matching
worktrees are reused. Empty claim receipts are bookkeeping, not product work.

Legacy filename-based claims still block acquisition. Upgrade all writers together
(kit help compatibility). A won claim excludes .worktrees/ in .git/info/exclude.
Kit does not pull into the checkout for you. Never delete another agent's claim.

  kit claim 0007
  kit claim 7                    # ids accept the short spelling`,

	"release": `kit release <id> [--discard]

Surrender this clone's claim: delete its remote branch, then local branch and the
worktree kit created. Run from outside that worktree. Current and custom worktrees are preserved as detached snapshots. A cleanup
failure reports the preserved checkout and asks for a retry through kit release.

Refuses while the branch carries any unmerged commit or uncommitted file. Stopping covers two
situations and only one of them is this: work worth nothing, and work worth keeping.
For the second, push the branch and open the pull request unfinished, saying why there
— the doc travels with the branch, so the record lands when it merges.

  0  the claim is gone and the feature is free for anyone to claim
  2  it could not run: no such feature, nothing claims it, two branches claim it,
     a branch this clone never created, a branch this checkout cannot resolve, a
     copy of it here that is not the remote's, no default branch to read work
     against, several remotes and no origin, the remote did not answer, or the
     branch carries work and --discard was not given

--discard explicitly permits deleting unmerged commits from this clone's claim.
It never overrides ownership or uncommitted-work protection. Preserve valuable work
before using it; deleted commits may eventually be garbage-collected.

Release removes an unused worktree the claim made. Current and custom checkouts stay
as detached snapshots. Incomplete cleanup names what remains and a kit release retry.

Uncommitted work in its worktree also refuses release. A commit changing only the doc is work.
The remote delete uses the observed tip as a lease, even with --discard; a later push
cannot be deleted by an earlier decision.

  kit release 0007
  kit release 7 --discard`,

	"block": `kit block <id> "<question>" [--fyi]

Blocked on something only a human can decide? Write it here and move on. Appends
one line to docs/INBOX.md in the format the tool parses:

  - [ ] YYYY-MM-DD · <id> · BLOCKING|FYI · <question>

"kit next" then stops offering that feature until a human answers and deletes the
line. Record the answer in the affected plan, commit it with the inbox change, and
integrate that commit into any paused checkout before resuming. With --fyi it is recorded without blocking, for anything work can continue
without.

Use this rather than editing INBOX.md by hand. A line that misses the format is
matched by prose instead of by its id field, so it blocks whatever it happens to
mention — "kit audit" reports those as inbox-unparsed.

State a leaning. It turns most questions into a yes/no.

  kit block 0007 "Should codes expire? Leaning: no, keep forever."
  kit block 7 "Using base62 for slugs." --fyi`,

	"submit": `kit submit <id>

On the feature branch, stamp status: Submitted and today's submitted: date, then
commit the stamp. This requests review; integration requires human merge into trunk.

Before submit: move durable facts to docs/TECH.md; keep only plan changes in Log
(empty is allowed); run the project gate, then kit audit --strict. Submit checks
the document, not whether tests ran or passed.

Exit 2 refuses non-Draft/missing status, no Tasks boxes or unfinished boxes,
unwritten Problem/Design, blockers, ambiguous fields/ids, invalid or unintegrated
dependencies, foreign claims, trunk/detached checkout, uncommitted/untracked files
or unreadable git status. If the stamp was written but its commit failed, commit
the named file as instructed.

Push and open the PR; never merge your own work. For review changes, run kit reopen,
revise tasks, implement, then repeat gate, audit, submit and push. Submitted docs
on trunk are frozen; further changes need a new feature.

  kit submit 0007`,

	"drop": `kit drop <id> "<why>" [--reservation]

Retire unbuilt work: set Dropped, record the reason in Log and commit only that doc
when the repo has history. Keep other files untouched. Deleting a planning branch
does not retire its doc on trunk.

Refuses Submitted/legacy Shipped, already Dropped, foreign claims, ambiguous docs or
missing status fields. A failed stamp commit exits 2 and names the file to commit.
Drop does not release claim branches; use kit release. It reports dependent features
whose needs: must be revised because Dropped can never satisfy them.

--reservation releases an unused refs/kit/ids/<id> instead of changing a document.
Refuses a doc holding the id on any branch tree, a reservation younger than the
stalled horizon, or unreadable remote/reservation/branch evidence. Fetch missing
objects before retrying. Deletion leases the observed ref. With no doc, record the
reason in the PR.

  kit drop 0007 "superseded by 0011's router"`,

	"reopen": `kit reopen <id> "<why>" [--repair]

Revise an unmerged submission: set Draft, clear submitted:, add the reason to Log
and commit the doc. Unknown or missing statuses require --repair. Dropped stays
retired; draft a new feature if the operator wants that work again.

Refuses already Draft, foreign claims, ambiguous docs or missing frontmatter.
Reopening Submitted requires an unmerged working branch and readable published
trunk (local trunk with no remote). Fetch missing objects before retrying.
Submitted on trunk is frozen history and needs a new feature.

If writing succeeded but committing failed, commit the named file. After review
fixes, repeat the project gate, audit, submit and push to the existing PR.

  kit reopen 0007 "review requested another case"
  kit reopen 0007 --repair "In Progress is not a supported status"`,

	"next": `kit next [--json] [--offline]

Read this repository and name the next task. Finish this checkout's claim first;
otherwise lowest ready id first. Within a feature, take the first unticked Tasks box.
An incomplete or unlanded plan, BLOCKING question or unmet needs: prevents claiming.

Groups: yours (this checkout), ready, taken and blocked; published submissions not
yet integrated appear under awaiting review. Branch names citing an id are claims,
including local branches and worktrees. Entries you built are marked for human review.

Progress is completed/total boxes, not estimated effort. Other claims use the advertised
remote tip and its last-moved date; missing objects give no count. Concurrent agents
can take independent ready features, one feature and checkout each.

Remote reads are bounded at 15 seconds. A failed read labels cached claims and unknown
review state. --offline skips remote reads and labels reduced coverage; it does not
prove remote availability or absence. Missing docs/features exits 2.

Reads only, from any subdirectory. --json includes the same closing guidance.
JSON progress fields are done and total. For several repositories:
  for d in ~/code/*; do (cd "$d" && kit next); done

  kit next --json
  kit next --offline`,

	"audit": `kit audit [--json] [--strict] [--offline]

Report document/repository drift with file, finding and fix. See kit help checks for
all codes, severity and limits. HIGH is serious drift; MEDIUM needs attention; LOW
is advisory. Fix the cause before continuing.

--strict exits 1 for HIGH findings, or 2 when the remote does not answer. --offline
explicitly limits coverage and lists skipped checks. Missing docs/features exits 2;
unread state is not a clean audit.

Leaves project files unchanged; online reservation reads may refresh git refs.
Uses this repository from any subdirectory. --json provides structured findings.

  kit audit --strict
  kit audit --offline`,
}

// runHelp prints the overview, or one topic.
func runHelp(templates fs.FS, args []string) int {
	// `kit help -h` asked for the overview and got "no topic \"-h\"" at exit 2, which is
	// the one command where being turned away for asking is absurd.
	if len(args) == 0 || isHelpFlag(args[0]) {
		fmt.Println(overview)
		return 0
	}
	topic := canonicalCommand(args[0])
	if text, ok := commandHelp[topic]; ok {
		fmt.Println(text)
		return 0
	}
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
