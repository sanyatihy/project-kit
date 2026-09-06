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
const overview = `kit — run an agent's project workflow from markdown and git.

Start with kit next. Follow the operator's requested scope; an empty board is not
permission to invent work. Agents run the commands; humans approve plans and merge PRs.

Commands (kit help <command> gives usage, refusals and recovery):
  kit init [PATH] [--stack STACK]   create missing project files
  kit start [<id>|--plan <round>|--work <outcome>]  prepare requested work
  kit new <title...>               draft a feature on the planning branch
  kit harvest <title...> --box ... record completed exploratory work
  kit claim <id>                   acquire a feature and its worktree
  kit block <id> "<question>"       record a decision only the human can make
  kit release <id> [--discard]     surrender a claim; protects valuable work
  kit submit <id>                  stamp completed work for review
  kit reopen <id> "<why>"          revise an unmerged submission
  kit drop <id> "<why>"            retire an unbuilt plan
  kit next [--json] [--offline]    show the next task and checkout
  kit audit [--json] [--strict] [--offline]  report drift
  kit version                     show the installed version

Workflow:
  Plan: draft a round → push and open PR → human merges → update trunk → claim.
  Build: enter the claimed worktree → finish the first unticked Tasks box → repeat.
  Review: project gate → kit audit --strict → kit submit → push and open PR.
  Stop for human review. Never merge your own work. Use reopen for requested changes.
  Exploration: use a spike branch, then harvest and the same review sequence.
  Finish this checkout's claim first; otherwise take the lowest ready id.

State: Draft → Submitted; Dropped retires a plan. Submitted is integrated only on
trunk. Only submit, drop and reopen write status fields. Legacy ship/reset alias
submit/reopen. BLOCKING inbox questions and unmet needs: prevent claims.

Exit codes:
  0  success
  1  completed negative result
  2  refusal or incomplete operation
For claim, 0 means yours: enter its worktree; 1 means run next and choose other work;
2 means fix the reported cause. An unanswered remote is not evidence of absence.

Reference (read only what the task needs):
  kit help protocol       full workflow, also in AGENTS.md
  kit help template       feature format
  kit help checks         audit findings and limits
  kit help inbox          blocking questions and FYI format
  kit help compatibility  supported versions and upgrades
  kit help backlog        deferred ideas
  kit help vision         project direction
  kit help as-built       TECH format
  kit help go-checklist   Go review practices
  kit help python-checklist  Python review practices

Next and audit use the current repository. --offline reports reduced coverage.
kit <command> --help also lists flags.`

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
 "start": `kit start <id> | --plan "<round>" | --work "<outcome>"

Approved id: use the existing claim guards and enter the printed worktree.
Plan: record intent on plan/<round>, then use kit new and write the plans.
Direct work: record the operator's outcome on spike/<outcome>, then implement.
Start from clean, updated trunk. Refuses foreign work, conflicting branches,
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
  kit init . --stack go --forge gitlab`,

	"harvest": `kit harvest <title...> --box "<outcome>" [--box "<outcome>"]

Record human-directed exploration on a spike branch. Reserves an id, writes a Draft
with completed boxes, names the branch after the feature and commits the whole
working tree. Name observable outcomes with --box; history is not used to infer them.
Fill Problem and Design; keep Log for plan changes, which may be empty.

Refuses trunk, detached HEAD and existing claims, including branch names citing a
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

  kit claim 0007`,

	"release": `kit release <id> [--discard]

Surrender this clone's claim: delete its remote branch, then local branch and the
worktree kit created. Run from outside that worktree. Current or custom worktrees
may remain; the command reports required cleanup.

Exit 0 means released. Exit 2 refuses missing or multiple claims, foreign ownership,
unresolvable branches, a local copy missing remote work, unknown trunk/remote state,
ambiguous remotes, uncommitted files or unmerged work. Doc-only commits are work.

--discard permits deleting unmerged commits, never foreign claims or dirty files.
Deleted commits may be garbage-collected. Preserve valuable work by committing and
pushing it and opening an unfinished PR with the remaining work described.

Remote deletion leases the observed tip, even with --discard; a later push survives.

  kit release 0007
  kit release 0007 --discard`,

	"block": `kit block <id> "<question>" [--fyi]

Ask for a human decision by appending this line to docs/INBOX.md:
  - [ ] YYYY-MM-DD · <id> · BLOCKING|FYI · <question>

BLOCKING stops next offering the feature. --fyi records information without blocking
or aging. State a leaning. Use this command so the id field is parsed correctly;
malformed lines conservatively match their whole text and audit reports inbox-unparsed.

After the human answers, record the decision in the plan, remove its inbox line and
commit both. Integrate that commit into paused checkouts before resuming. Meanwhile,
take independent work from another trunk checkout, or report the blocker and stop.

  kit block 0007 "Should codes expire? Leaning: no, keep forever."`,

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
