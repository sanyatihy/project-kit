// Command kit bootstraps projects and tells you what to work on.
//
//	kit init  [PATH] [--stack go]   set a project up, or fill in what is missing
//	kit next  [PATH ...]            what should I work on?
//	kit audit [PATH ...]            where do the docs disagree with the repo?
//
// Plans live in markdown in git. There is no database and no queue — the feature
// docs and the git branches are the state.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	assets "github.com/sanyatihy/project-kit"
	"github.com/sanyatihy/project-kit/internal/kit"
)

var templates = assets.FS

// Version is stamped at build time: -ldflags "-X main.Version=$(git describe)".
// `go install` does not apply the Makefile's ldflags, so a binary installed over
// the network reports whatever Go embedded in its build info instead — see
// resolveVersion.
var Version = "dev"

// resolveVersion decides what `kit version` prints.
//
// The ldflags stamp wins when it is there: `git describe` carries the commit and a
// -dirty marker, which the module version cannot. Otherwise fall back to the module
// version Go embeds, which is the tag for `@v0.1.0`, a pseudo-version for `@latest`
// off an untagged branch, and "(devel)" for a plain `go build` — that last one is
// noise, so it degrades to "dev".
func resolveVersion(stamped, embedded string) string {
	if stamped != "" && stamped != "dev" {
		return stamped
	}
	if embedded != "" && embedded != "(devel)" {
		return embedded
	}
	return "dev"
}

func version() string {
	embedded := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		embedded = info.Main.Version
	}
	return resolveVersion(Version, embedded)
}

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) < 2 {
		fmt.Println(overview)
		return 0
	}
	switch cmd := os.Args[1]; cmd {
	case "-h", "--help", "help":
		return runHelp(templates, os.Args[2:])
	case "version", "--version":
		fmt.Println("kit " + version())
		return 0
	case "init":
		return runInit(os.Args[2:])
	case "new":
		return runNew(os.Args[2:])
	case "accept":
		return runAccept(os.Args[2:])
	case "claim":
		return runClaim(os.Args[2:])
	case "drop":
		return runDrop(os.Args[2:])
	case "block":
		return runBlock(os.Args[2:])
	case "ship":
		return runShip(os.Args[2:])
	case "next":
		return runNext(os.Args[2:])
	case "audit":
		return runAudit(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "kit: unknown command %q\n\n%s\n", cmd, overview)
		return 2
	}
}

// ---------------------------------------------------------------- init

func runInit(args []string) int {
	stacks := kit.Stacks(templates)
	fs := newFlagSet("init")
	stack := fs.String("stack", "", "also scaffold a language stack ("+
		strings.Join(stacks, ", ")+")")
	forge := fs.String("forge", "", "where the project is hosted ("+
		strings.Join(kit.Forges, ", ")+"); detected when not given")
	paths := parseArgs(fs, args)

	target := "."
	if len(paths) > 0 {
		target = paths[0]
	}
	root, err := filepath.Abs(expand(target))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit init: %v\n", err)
		return 2
	}
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		fmt.Fprintf(os.Stderr, "kit init: not a directory: %s\n", target)
		return 2
	}
	if *stack != "" && !contains(stacks, *stack) {
		fmt.Fprintf(os.Stderr, "kit init: unknown stack %q (have: %s)\n",
			*stack, strings.Join(stacks, ", "))
		return 2
	}
	if *forge != "" && !contains(kit.Forges, *forge) {
		fmt.Fprintf(os.Stderr, "kit init: unknown forge %q (have: %s)\n",
			*forge, strings.Join(kit.Forges, ", "))
		return 2
	}

	res, err := kit.Init(templates, root, *stack, *forge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit init: %v\n", err)
		return 2
	}

	fmt.Printf("\n%s\n", filepath.Base(root))
	for _, p := range res.Created {
		fmt.Printf("  created    %s\n", p)
	}
	for _, p := range res.Refreshed {
		fmt.Printf("  refreshed  %s\n", p)
	}
	for _, p := range res.Kept {
		fmt.Printf("  kept       %s\n", p)
	}

	// Both routes out are the agent's own, because the forge is a discoverable fact and
	// not a decision only a human can make. Saying nothing here is what would leave a
	// project with no pipeline and no way to know it.
	if res.CINotWritten {
		fmt.Printf("\n  no CI file — nothing here says which forge this project is on.\n")
		fmt.Printf("  after git remote add, run kit init again and it will detect one,\n")
		fmt.Printf("  or say so now: kit init %s --stack %s --forge %s\n",
			target, *stack, strings.Join(kit.Forges, "|"))
	}

	// Singular here is what produced a plan with one feature covering a whole
	// project: the tool asked for a feature and the agent wrote a feature. Bootstrap
	// means the whole set, and accepting them is not the agent's to do.
	fmt.Printf("\nnext:\n")
	fmt.Printf("  write docs/VISION.md — what this is for, and what it is not\n")
	fmt.Printf("  then decompose it: one kit new per feature the vision describes,\n")
	fmt.Printf("  each one a change a human can review in one sitting\n")
	fmt.Printf("  leave them Draft — accepting them is the human's call\n")
	// Said here because this is where a bootstrap run reads what to do next, and the
	// omission is what let `kit claim` win a branch carrying no plan: a claim pushes
	// HEAD, so a plan that is only in the working tree does not travel with it.
	fmt.Printf("  then land all of it on the default branch — a pull or merge request,\n")
	fmt.Printf("  if that branch is protected. A claim pushes HEAD, so nothing uncommitted\n")
	fmt.Printf("  reaches the branch an agent is sent to work in\n\n")
	return 0
}

// ---------------------------------------------------------------- new

func runNew(args []string) int {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "-help" {
			fmt.Println(commandHelp["new"])
			return 0
		}
		if strings.HasPrefix(a, "-") {
			fmt.Fprintf(os.Stderr, "kit new: unknown option %q\n\n%s\n", a, commandHelp["new"])
			return 2
		}
	}
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		fmt.Fprintln(os.Stderr, "kit new: give the feature a title, e.g."+
			` kit new "Shorten URLs"`)
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit new: %v\n", err)
		return 2
	}
	path, err := kit.NewFeature(templates, repo, title, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit new: %v\n", err)
		return 2
	}
	fmt.Printf("\n  created  %s\n", repo.Rel(path))
	fmt.Printf("\n  fill in Problem, Non-goals, Design, and Tasks.\n")
	fmt.Printf("  leave status: Draft while anything is unresolved — put those in\n")
	fmt.Printf("  ## Open questions, or docs/INBOX.md if only a human can answer.\n")
	fmt.Printf("  leave status: Draft — accepting it is the human's call, and\n")
	fmt.Printf("  kit next will offer it once they have.\n\n")
	return 0
}

// ---------------------------------------------------------------- accept

func runAccept(args []string) int {
	fs := newFlagSet("accept")
	review := fs.Bool("review", false, "a human merges this one, whatever the trunk allows")
	rest := parseArgs(fs, args)
	if len(rest) != 1 {
		fmt.Fprintf(os.Stderr, "kit accept: need one feature id\n\n%s\n", commandHelp["accept"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit accept: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit accept: %v\n", err)
		return 2
	}
	res, err := kit.Accept(repo, id, *review)
	if err != nil {
		// Exit 2 throughout: unlike `claim`, none of these is another agent holding the
		// feature, so "take something else" would be the wrong instruction.
		fmt.Fprintf(os.Stderr, "kit accept: %v\n", err)
		return 2
	}

	fmt.Printf("\n  accepted  %s  (was Draft)\n", res.Doc)
	if res.NeedsHuman {
		fmt.Printf("  review: human — a person merges this one, whatever the trunk allows\n")
	}
	fmt.Printf("\nnext:\n")
	// Where the edit lands is the whole reason this prints anything. Accepting inside the
	// draft's own pull request is one review and one merge per feature; accepting after it
	// merges is a second pull request for a one-line change, which is what the protocol
	// left undefined and a human had to guess at every time.
	fmt.Printf("  commit this on %s so it lands in the same review as the draft\n",
		res.Branch)
	fmt.Printf("  the branch is deleted on merge — its name is the claim name\n")
	fmt.Printf("  kit next\n\n")
	return 0
}

// ---------------------------------------------------------------- drop

func runDrop(args []string) int {
	fs := newFlagSet("drop")
	rest := parseArgs(fs, args)
	if len(rest) < 1 {
		fmt.Fprintf(os.Stderr, "kit drop: need one feature id\n\n%s\n", commandHelp["drop"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit drop: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit drop: %v\n", err)
		return 2
	}
	why := strings.TrimSpace(strings.Join(rest[1:], " "))
	res, err := kit.Drop(repo, id, why, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit drop: %v\n", err)
		return 2
	}

	fmt.Printf("\n  dropped  %s  (was %s)\n", res.Doc, res.Was)
	if why == "" {
		fmt.Printf("\n  no reason recorded — say why in ## Log, or the next person\n")
		fmt.Printf("  cannot tell a rejected plan from a forgotten one\n")
	}
	// Dropping strands whoever declared this in needs:, and said nothing about it: the
	// board went on promising them a wait that can never end, and audit reported
	// needs-unknown once it was too late to ask why.
	if len(res.Dependants) > 0 {
		fmt.Printf("\n  still needed by %s — a Dropped feature can never ship, so\n",
			strings.Join(res.Dependants, ", "))
		fmt.Printf("  those docs need their needs: edited or they wait for ever\n")
	}
	// The doc is retired and the branch is not, and only one of those is this command's
	// to touch. A leftover branch named after the doc still reads as a claim.
	if len(res.Claims) > 0 {
		fmt.Printf("\n  still claimed by %s — delete it, or the board shows work\n",
			strings.Join(res.Claims, " and "))
		fmt.Printf("  nobody is doing\n")
	}
	fmt.Printf("\nnext:\n  kit next\n\n")
	return 0
}

// ---------------------------------------------------------------- claim

func runClaim(args []string) int {
	fs := newFlagSet("claim")
	rest := parseArgs(fs, args)
	if len(rest) != 1 {
		fmt.Fprintf(os.Stderr, "kit claim: need one feature id\n\n%s\n", commandHelp["claim"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit claim: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit claim: %v\n", err)
		return 2
	}

	claimed, err := kit.ClaimFeature(repo, id)
	if err != nil {
		// Losing is an answer, not a breakage, and it exits 1 so a script can tell
		// "this one is not yours, take another" from "the command could not run".
		var taken *kit.NotYours
		if errors.As(err, &taken) {
			fmt.Fprintf(os.Stderr, "\n  not claimed: %s\n", taken.Why)
			fmt.Fprintf(os.Stderr, "  %s\n\n", taken.Then)
			return 1
		}
		fmt.Fprintf(os.Stderr, "kit claim: %v\n", err)
		return 2
	}

	fmt.Printf("\n  claimed  %s\n", claimed.Branch)
	if claimed.Remote == "" {
		fmt.Printf("\n  this project has no remote, so no other agent can see it.\n")
		fmt.Printf("  The claim exists only here — add one before running a second agent.\n")
	} else {
		fmt.Printf("  pushed   %s/%s — the push is the claim, and you won it\n",
			claimed.Remote, claimed.Branch)
	}
	fmt.Printf("\nnext:\n")
	fmt.Printf("  git worktree add ../%s-%s %s\n", repo.Name, id, claimed.Branch)
	// No status edit to instruct: the branch just pushed is what says this is started,
	// and asking for a second copy of that in the doc is what rule 1 forbade all along.
	fmt.Printf("  work there, not here — the branch is the claim, so nothing else to set\n\n")
	return 0
}

// ---------------------------------------------------------------- block

func runBlock(args []string) int {
	fs := newFlagSet("block")
	fyi := fs.Bool("fyi", false, "record it without blocking the feature")
	rest := parseArgs(fs, args)

	if len(rest) < 2 {
		fmt.Fprintf(os.Stderr, "kit block: need a feature id and a question\n\n%s\n",
			commandHelp["block"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit block: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit block: %v\n", err)
		return 2
	}
	line, err := kit.AppendQuestion(repo, id, strings.Join(rest[1:], " "), *fyi, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit block: %v\n", err)
		return 2
	}

	fmt.Printf("\n  %s\n", repo.Rel(repo.Inbox))
	fmt.Printf("  %s\n", line)
	if *fyi {
		fmt.Printf("\n  recorded. %s is not blocked — carry on.\n\n", id)
	} else {
		fmt.Printf("\n  %s is blocked until a human answers. Move to another feature.\n\n", id)
	}
	return 0
}

// ---------------------------------------------------------------- ship

func runShip(args []string) int {
	fs := newFlagSet("ship")
	rest := parseArgs(fs, args)
	if len(rest) != 1 {
		fmt.Fprintf(os.Stderr, "kit ship: need one feature id\n\n%s\n", commandHelp["ship"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit ship: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit ship: %v\n", err)
		return 2
	}
	res, err := kit.Ship(templates, repo, id, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit ship: %v\n", err)
		return 2
	}

	fmt.Printf("\n  shipped  %s  (was %s)\n", res.Doc, res.Was)
	fmt.Printf("\nnext:\n")
	if res.TechCreated {
		fmt.Printf("  created %s — the as-built doc. See: kit help as-built\n", res.TechDoc)
	}
	if res.TechDoc != "" {
		fmt.Printf("  move durable facts into %s — the feature doc is frozen history now\n", res.TechDoc)
	}
	// The moment a working document becomes agent context that every later session
	// carries. Worded to lead with what to keep, because the risk is an agent reading
	// "compact" as "delete": the doc is the only record of why the code is like this.
	fmt.Printf("  keep every decision, constraint and measurement — cut only what the code now says\n")
	if res.OpenLogs {
		fmt.Printf("  ## Log is empty; record what was decided and why\n")
	}
	// Said here because this decision is made at the end of a session and the
	// protocol is read at the start of one. An agent that reaches this point with
	// no instruction fills the gap by merging its own work, which is what happened
	// the first time a real project ran the loop.
	if res.HasRemote {
		fmt.Printf("  push the branch and stop — reviewing and merging are not yours\n")
	} else {
		fmt.Printf("  this project has no remote, so there is nowhere to hand this off.\n")
		fmt.Printf("  say so rather than merging it yourself — merging is not yours\n")
	}
	fmt.Printf("  kit next\n\n")
	return 0
}

// ---------------------------------------------------------------- next

func runNext(args []string) int {
	fs := newFlagSet("next")
	asJSON := fs.Bool("json", false, "machine-readable output")
	offline := fs.Bool("offline", false, "do not ask the remote which branches exist")
	paths := parseArgs(fs, args)

	all := survey(paths, *offline)

	// planFor keeps the staleness check next to the plan it belongs to, so both output
	// formats carry the same fact and cannot drift.
	protocol := currentProtocol()
	planFor := func(r *kit.Repo) kit.Plan {
		plan := kit.Survey(r)
		plan.ProtocolStale = kit.ProtocolIsStale(r, protocol)
		return plan
	}

	if *asJSON {
		// Keyed by name for a project and by the path as given for one that could not
		// be opened, so a fleet payload lists everything the caller asked about. The
		// failure rides in the plan itself: an agent parsing this reads one entry per
		// path either way, and cannot be handed four projects with a fifth missing.
		plans := inParallel(all, func(s surveyed) kit.Plan {
			if s.Err != "" {
				return kit.Plan{Error: s.Err}
			}
			return planFor(s.Repo)
		})
		payload := map[string]kit.Plan{}
		for i, s := range all {
			payload[s.Name] = plans[i]
		}
		if code := emit(payload); code != 0 {
			return code
		}
	} else {
		color := isTTY()
		// Read before anything is printed, so a fan-out cannot interleave two projects'
		// output. One project sees no difference; a fleet gets its whole board when the
		// slowest remote answers instead of one project at a time.
		plans := inParallel(all, func(s surveyed) kit.Plan {
			if s.Err != "" || s.Repo.FeaturesDir == "" {
				return kit.Plan{}
			}
			return planFor(s.Repo)
		})
		for i, s := range all {
			fmt.Print(paint("\n"+s.Name+"\n", "1", color))
			switch {
			case s.Err != "":
				fmt.Printf("  %s — nothing to report for this path\n\n", s.Err)
			case s.Repo.FeaturesDir == "":
				fmt.Printf("  not set up yet — run:  kit init %s\n\n", s.Repo.Root)
			default:
				printPlan(s.Repo, plans[i], color)
			}
		}
	}

	if reportUnreadable(all, "kit next") {
		return 2
	}
	return 0
}

// listItems prints features as a plain list: the ID and title, and the one task
// that would be next in each.
func listItems(items []kit.Item) {
	for _, item := range items {
		fmt.Printf("    %s  %-5s %s\n", item.ID, item.Progress(), kit.Show(item.Title, 54))
		fmt.Printf("          %s\n", kit.Show(item.Task, 64))
	}
}

func printPlan(r *kit.Repo, plan kit.Plan, color bool) {
	// Above the board, because an agent that has read the plan and started working has
	// already walked past anything printed after it. Never changes the exit code: the plan
	// is still right, it is the rules around it that drifted.
	if plan.ProtocolStale {
		fmt.Println(paint("  ! AGENTS.md is older than this kit, so the protocol below it "+
			"is superseded — run kit init . before following it", "33", color))
	}
	if n := plan.InProgress(); n > kit.WIPLimit {
		fmt.Println(paint(fmt.Sprintf("  %d features in progress at once — "+
			"finish one before starting another", n), "33", color))
	}

	// What this checkout already holds comes first: it is the answer to "what should
	// I work on?" for the agent standing in it, and everything below is context.
	if len(plan.Yours) > 0 {
		mine := plan.Yours[0]
		fmt.Printf("\n  %s\n", paint("yours — this checkout claims it", "1;32", color))
		fmt.Printf("    %s  %-5s %s\n", mine.ID, mine.Progress(), kit.Show(mine.Title, 54))
		fmt.Printf("    %s %s\n", paint("→", "1;32", color), kit.Show(mine.Task, 68))
		fmt.Printf("       %s\n", mine.Doc)
		listItems(plan.Yours[1:])
	}

	if len(plan.Ready) > 0 {
		first := plan.Ready[0]
		// A checkout that already claims a feature must never be handed a different
		// one to start. Followed literally that is how two features end up sharing
		// one branch: the reader is standing in 0006's worktree and has just been
		// told to begin 0007 in it. What is free is still worth knowing; where to
		// begin it is not here.
		if plan.Claiming != "" {
			fmt.Printf("\n  %s\n", paint("free, but not in this checkout", "2", color))
			listItems(plan.Ready)
			fmt.Printf("\n  this checkout claims %s. Take one of those in a worktree of\n",
				plan.Claiming)
			fmt.Printf("  its own, never here:  kit claim %s\n", first.ID)
		} else {
			fmt.Printf("\n  %s\n", paint("start here", "1;32", color))
			fmt.Printf("    %s  %-5s %s\n", first.ID, first.Progress(), kit.Show(first.Title, 54))
			fmt.Printf("    %s %s\n", paint("→", "1;32", color), kit.Show(first.Task, 68))
			fmt.Printf("       %s\n", first.Doc)
			// The claim comes before the worktree, and kit claim prints the worktree
			// command once it has actually won. Printing it here instead would have
			// the agent building somewhere to work before knowing the work is theirs.
			fmt.Printf("       kit claim %s\n", first.ID)
			if rest := plan.Ready[1:]; len(rest) > 0 {
				fmt.Printf("\n  also free\n")
				listItems(rest)
			}
		}
	}

	if len(plan.Taken) > 0 {
		fmt.Printf("\n  %s\n", paint("taken", "2", color))
		for _, item := range plan.Taken {
			// When it last moved, because a count alone cannot tell an agent working
			// steadily from one that stopped an hour in. `stalled` only speaks at three
			// days, and by then the answer has cost two.
			moved := ""
			if !item.Moved.IsZero() {
				moved = ", " + ago(item.Moved)
			}
			fmt.Printf("    %s  %-5s %s — %s%s\n", item.ID, item.Progress(),
				kit.Show(item.Title, 34), item.ClaimedBy, moved)
		}
	}

	if len(plan.Blocked) > 0 {
		fmt.Printf("\n  %s\n", paint("blocked", "33", color))
		for _, item := range plan.Blocked {
			fmt.Printf("    %s  %s\n", item.ID, kit.Show(item.Title, 60))
			fmt.Printf("          waiting on: %s\n", kit.Show(item.WaitingOn, 56))
		}
	}

	// The far side of ship, below the board because none of it is claimable and above
	// the guidance because the guidance may send the reader here. It is the same command
	// everyone runs, so a coordinator has one place to look rather than a second tool.
	if len(plan.AwaitingReview) > 0 {
		fmt.Printf("\n  %s\n", paint("awaiting review", "36", color))
		for _, item := range plan.AwaitingReview {
			fmt.Printf("    %s  %s\n", item.ID, kit.Show(item.Title, 60))
			// Whose it is decides what the reader may do with it, so it goes on the
			// same line as the branch rather than being left to the guidance.
			marks := ""
			if item.NeedsHuman {
				marks += " · " + paint("needs a human", "33", color)
			}
			if item.Mine {
				marks += " · " + paint("you built this — not yours to merge", "33", color)
			}
			fmt.Printf("          %s · %s%s\n", item.Branch, waited(item.Since), marks)
		}
	}

	// Under the board, because it is a statement about everything above it. An empty
	// board and an unreachable remote produce the same page otherwise, which is how a
	// feature sat pushed and unmerged while `next` said nothing was planned.
	if plan.RemoteUnanswered {
		fmt.Printf("\n%s\n", paint(wrapAt("! the remote did not answer, so this board is "+
			"what the last fetch knew: claims may be stale, and anything awaiting review "+
			"is missing rather than absent. Try again, or run git fetch.", 74, "  "),
			"33", color))
	}

	if plan.Guidance != "" && len(plan.Yours) == 0 && len(plan.Ready) == 0 {
		fmt.Printf("\n%s\n", paint(wrapAt(plan.Guidance, 74, "  "), "33", color))
		// The command goes on its own line, unwrapped, so it can be copied. The
		// guidance says what to do; this is the thing to run.
		if plan.Empty() {
			fmt.Printf("    kit new \"<what the feature does>\"\n")
		}
	}
	fmt.Println()
}

// ago renders how long ago something happened, in the coarsest unit that is still
// informative: minutes while an agent is working, days once it has stopped.
func ago(when time.Time) string {
	switch d := time.Since(when); {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute") + " ago"
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour") + " ago"
	default:
		return plural(int(d.Hours()/24), "day") + " ago"
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// waited says how long a branch has been sitting there, in the coarsest unit that is
// still true. A day is the useful granularity: what the reader decides is whether this
// has been ignored, and nothing about an hour answers that.
func waited(since time.Time) string {
	if since.IsZero() {
		return "waiting"
	}
	switch days := kit.Days(since, time.Now()); {
	case days <= 0:
		return "waiting since today"
	case days == 1:
		return "waiting 1 day"
	default:
		return fmt.Sprintf("waiting %d days", days)
	}
}

// ---------------------------------------------------------------- audit

func runAudit(args []string) int {
	fs := newFlagSet("audit")
	asJSON := fs.Bool("json", false, "machine-readable output")
	strict := fs.Bool("strict", false, "exit 1 if anything HIGH is found")
	offline := fs.Bool("offline", false, "do not ask the remote which branches exist")
	paths := parseArgs(fs, args)

	all := survey(paths, *offline)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	protocol := currentProtocol()
	order := inParallel(all, func(s surveyed) audited {
		one := audited{Findings: []kit.Finding{}}
		switch {
		case s.Err != "":
			one.Error = s.Err
		case s.Repo.FeaturesDir != "":
			if findings := kit.Audit(s.Repo, today, protocol); findings != nil {
				one.Findings = findings
			}
			// After the audit, so it reports what actually went unanswered.
			one.RemoteUnanswered = s.Repo.RemoteUnanswered()
		}
		return one
	})
	// Keyed after the fan-out, not inside it: the map is shared, and two paths naming
	// one project have always reported under a single key.
	results := make(map[string]audited, len(all))
	for i, s := range all {
		results[s.Name] = order[i]
	}

	if *asJSON {
		if code := emit(results); code != 0 {
			return code
		}
	} else {
		printFindings(all, order, isTTY())
	}

	// A path the caller named could not be audited at all, which outranks whatever was
	// found in the ones that could: 2 is the more serious answer, and a script must not
	// read "one project is missing" as "there are findings".
	if reportUnreadable(all, "kit audit") {
		return 2
	}
	if *strict {
		for _, one := range results {
			for _, f := range one.Findings {
				if f.Severity == kit.High {
					return 1
				}
			}
		}
	}
	return 0
}

// audited is one path's audit in --json: what was found, or why nothing could be.
// `next` carries its failure inside the plan, which is one object per project and
// can say so itself; a findings list cannot, so this wraps it.
type audited struct {
	Findings []kit.Finding `json:"findings"`
	Error    string        `json:"error,omitempty"`

	// RemoteUnanswered says the remote was asked and did not answer, so this is a
	// weaker statement than it looks: shipped-not-pushed could not run at all, and the
	// claim checks read the last fetch. Without it a clean bill and an unreachable
	// remote print the same page.
	RemoteUnanswered bool `json:"remote_unanswered,omitempty"`
}

func printFindings(all []surveyed, results []audited, color bool) {
	actionable := 0
	tint := map[string]string{kit.High: "1;31", kit.Medium: "33", kit.Low: "2;37"}

	for i, s := range all {
		fmt.Print(paint("\n"+s.Name+"\n", "1", color))
		if s.Err != "" {
			fmt.Printf("  %s — nothing audited for this path\n\n", s.Err)
			continue
		}
		if s.Repo.FeaturesDir == "" {
			fmt.Printf("  not set up yet — run:  kit init %s\n\n", s.Repo.Root)
			continue
		}
		if results[i].RemoteUnanswered {
			fmt.Printf("  %s\n", paint("! the remote did not answer: shipped-not-pushed "+
				"did not run, and the claim\n    checks read the last fetch", "33", color))
		}
		findings := results[i].Findings
		if len(findings) == 0 {
			fmt.Printf("  docs and repo agree\n\n")
			continue
		}
		for _, severity := range kit.Severities {
			printed := false
			for _, f := range findings {
				if f.Severity != severity {
					continue
				}
				if !printed {
					fmt.Printf("\n  %s\n", paint(severity, tint[severity], color))
					printed = true
				}
				fmt.Printf("    %s  (%s)\n", f.Where, f.Code)
				fmt.Printf("      %s\n", f.Problem)
				fmt.Printf("      %s %s\n", paint("fix:", "2", color), f.Fix)
			}
		}
		for _, f := range findings {
			if f.Severity == kit.High || f.Severity == kit.Medium {
				actionable++
			}
		}
		fmt.Println()
	}

	if actionable > 0 {
		fmt.Printf("%d thing(s) to fix. HIGH first.\n\n", actionable)
	}
}

// ---------------------------------------------------------------- shared

// newFlagSet builds a subcommand's flags with help that describes the command,
// not just its options — an agent should not need the source to use it.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("kit "+name, flag.ExitOnError)
	fs.Usage = func() {
		// Nothing useful to do if printing help fails; the process is on its
		// way out either way.
		_, _ = fmt.Fprintln(fs.Output(), commandHelp[name])
		_, _ = fmt.Fprintln(fs.Output(), "\nOptions:")
		fs.PrintDefaults()
	}
	return fs
}

// parseArgs parses flags that appear anywhere, not only before the first path.
// `kit audit ~/a ~/b --json` is how people actually type it; Go's flag package
// stops at the first positional, so drive it in a loop.
func parseArgs(fs *flag.FlagSet, args []string) []string {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return positional
		}
		if fs.NArg() == 0 {
			return positional
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// surveyed is one path the board was asked about: the project it opened, or why it
// did not. Name is what the board calls it — the project's name, or the path as
// given when there is no project to name.
// fleetParallel caps how many projects are read at once. What a board read spends is
// one network round trip per project and almost nothing else — `ls-remote` to GitHub over
// SSH is 2.3s where the whole rest of an audit is 0.09s — so this bounds how many
// connections are opened to one forge, not CPU. 0034's `-parallel 4` caps a different
// resource: CPU-bound tests running beside whatever else the operator is doing.
const fleetParallel = 8

// inParallel reads every surveyed project concurrently, capped, and returns the answers
// in the order the paths were given. The projects share nothing — one Repo each, and
// every package-level table in internal/kit is read-only — so order is the only thing
// concurrency could take away, and 0026 made reporting positional precisely so a fleet
// board can be compared with the last one.
func inParallel[T any](all []surveyed, read func(surveyed) T) []T {
	out := make([]T, len(all))
	sem := make(chan struct{}, fleetParallel)
	var wg sync.WaitGroup
	for i, s := range all {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = read(s)
		}()
	}
	wg.Wait()
	return out
}

type surveyed struct {
	Repo *kit.Repo
	Name string
	Err  string
}

// survey opens every path it was given and never returns early. The surveyed
// projects are independent, so one of them being gone says nothing about the
// others — the same argument as remoteTimeout in internal/kit, where a board that
// stops because one remote is unreachable is worse than one that reports what it
// could reach and says what it could not. A path that cannot be read is a failure
// in place, in the order it was given, and the caller turns that into exit 2.
func survey(paths []string, offline bool) []surveyed {
	if len(paths) == 0 {
		paths = []string{kit.ProjectRoot(".")}
	}
	all := make([]surveyed, 0, len(paths))
	for _, given := range paths {
		root := expand(given)
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			all = append(all, surveyed{Name: given, Err: "not a directory"})
			continue
		}
		r, err := kit.Open(root)
		if err != nil {
			all = append(all, surveyed{Name: given, Err: err.Error()})
			continue
		}
		r.Offline = offline
		all = append(all, surveyed{Repo: r, Name: r.Name})
	}
	return all
}

// reportUnreadable says on stderr which paths could not be surveyed, and whether
// any could not. The board already carries the same failures on stdout so that a
// fleet reads in order; this is the copy a script sees, and it is where a single
// bad path has always reported itself.
func reportUnreadable(all []surveyed, who string) bool {
	failed := false
	for _, s := range all {
		if s.Err != "" {
			fmt.Fprintf(os.Stderr, "%s: %s: %s\n", who, s.Err, s.Name)
			failed = true
		}
	}
	return failed
}

func emit(payload any) int {
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit: %v\n", err)
		return 2
	}
	fmt.Println(string(out))
	return 0
}

func expand(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func paint(text, code string, color bool) string {
	if !color {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func isTTY() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb"
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// currentProtocol is the work-protocol section this binary would write, so `audit`
// can tell a project it is running an older one. Empty on a read failure, which
// simply skips the check rather than reporting a difference it could not verify.
func currentProtocol() string {
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		return ""
	}
	return string(raw)
}

// wrapAt breaks a sentence onto indented lines that fit a narrow terminal. The
// guidance is written once, in internal/kit, so that both output formats carry the
// same words; only its shape on screen belongs here.
func wrapAt(text string, width int, indent string) string {
	var lines []string
	line := indent
	for _, word := range strings.Fields(text) {
		if len(line)+len(word)+1 > width && line != indent {
			lines = append(lines, line)
			line = indent
		}
		if line != indent {
			line += " "
		}
		line += word
	}
	if line != indent {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
