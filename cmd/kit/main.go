// Command kit bootstraps projects and tells you what to work on.
//
//	kit init  [PATH] [--stack go]   set a project up, or fill in what is missing
//	kit next                        what should I work on?
//	kit audit                       where do the docs disagree with the repo?
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
	"slices"
	"strings"
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
	case "claim":
		return runClaim(os.Args[2:])
	case "release":
		return runRelease(os.Args[2:])
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
	fs := newFlagSet("init", args)
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
	if *stack != "" && !slices.Contains(stacks, *stack) {
		fmt.Fprintf(os.Stderr, "kit init: unknown stack %q (have: %s)\n",
			*stack, strings.Join(stacks, ", "))
		return 2
	}
	if *forge != "" && !slices.Contains(kit.Forges, *forge) {
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

	// The ids the docs already hold, reserved once. A doc written before `kit new`
	// reserved anything holds a number nothing recorded, so deleting it would hand that
	// number out again — and re-running init is the one moment every existing project
	// passes through. Best effort: a remote that cannot be reached is a warning and not
	// a reason to fail a scaffold that has already written the files.
	if repo, err := kit.Open(root); err == nil {
		switch n, err := kit.BackfillIDs(repo); {
		case err != nil:
			fmt.Fprintf(os.Stderr, "kit init: warning — could not reserve the ids "+
				"already in use: %v\n", err)
		case n > 0:
			fmt.Printf("  reserved   %s already in use\n", plural(n, "id"))
		}
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
	// means the whole set, and which of them go ahead is not the agent's to decide.
	fmt.Printf("\nnext:\n")
	fmt.Printf("  write docs/VISION.md — what this is for, and what it is not\n")
	fmt.Printf("  then decompose it: one kit new per feature the vision describes,\n")
	fmt.Printf("  each one a change a human can review in one sitting\n")
	fmt.Printf("  leave them Draft — kit ship and kit drop write a status, nothing else\n")
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
		if isHelpFlag(a) {
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
	drafted, err := kit.NewFeature(templates, repo, title, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit new: %v\n", err)
		return 2
	}
	fmt.Printf("\n  created   %s\n", repo.Rel(drafted.Path))
	if drafted.Remote != "" {
		fmt.Printf("  reserved  %s on %s — no other agent can be handed it\n",
			drafted.ID, drafted.Remote)
	} else {
		// Said out loud rather than degraded quietly. The reservation is the whole of
		// what stops two agents being handed one number, and a project with no remote
		// has nothing to make it against — which is the state the old behaviour was in
		// everywhere, and the reason it stayed invisible for 57 features.
		fmt.Printf("\n  no remote, so nothing reserved %s — another agent drafting off\n",
			drafted.ID)
		fmt.Printf("  this trunk right now can be handed the same id\n")
	}
	fmt.Printf("\n  fill in Problem, Non-goals, Design, and Tasks.\n")
	fmt.Printf("  leave status: Draft while anything is unresolved — put those in\n")
	fmt.Printf("  ## Open questions, or docs/INBOX.md if only a human can answer.\n")
	// The gate this described is gone: `Accepted` was removed after 82 docs reached it
	// zero times, and `next` offers a written Draft the moment its round lands.
	fmt.Printf("  kit next offers it as soon as the plan is written, and kit ship\n")
	fmt.Printf("  and kit drop are the only things that write a status.\n\n")
	return 0
}

// ---------------------------------------------------------------- drop

func runDrop(args []string) int {
	fs := newFlagSet("drop", args)
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
	why := strings.TrimSpace(strings.Join(rest[1:], " "))
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		// No doc holds the number, which used to end here. A reservation may hold it, and
		// returning that to circulation is the disposal nothing else has: an id handed out
		// for a draft that was then cut could be reached by no command at all.
		return releaseReservation(repo, rest[0], why)
	}
	res, err := kit.Drop(repo, id, why, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit drop: %v\n", err)
		return 2
	}

	fmt.Printf("\n  dropped  %s  (was %s)\n", res.Doc, res.Was)
	if res.Commit != "" {
		fmt.Printf("  commit   %s — the status is committed, not left in the tree\n", res.Commit)
	}
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

// releaseReservation is `kit drop` on an id no doc holds: the reservation goes back, and
// the reason goes nowhere durable, because there is no doc to write a ## Log entry in.
func releaseReservation(repo *kit.Repo, typed, why string) int {
	released, err := kit.ReleaseID(repo, typed, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit drop: %v\n", err)
		return 2
	}
	fmt.Printf("\n  released %s on %s", released.ID, released.Remote)
	if released.Title != "" {
		fmt.Printf("  (was reserved for %q)", released.Title)
	}
	fmt.Printf("\n  the number is free — the next kit new may be handed it\n")
	if why == "" {
		fmt.Printf("\n  no reason recorded, and there is no doc to record one in —\n")
		fmt.Printf("  say why in the pull request that carries this\n")
	}
	fmt.Printf("\nnext:\n  kit next\n\n")
	return 0
}

// ---------------------------------------------------------------- claim

func runClaim(args []string) int {
	fs := newFlagSet("claim", args)
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
	if claimed.Excluded != "" {
		fmt.Printf("  wrote    %s/ to %s — this clone only, so git and grep skip it\n",
			kit.WorktreesDir, claimed.Excluded)
	}
	// The claim is already won, so this cannot fail the command — but an operator who
	// is not told will `git add -A` in the trunk and commit a second copy of the source.
	if claimed.ExcludeErr != nil {
		fmt.Fprintf(os.Stderr, "  warning  could not exclude %s/: %v\n"+
			"           add the line yourself before creating the worktree\n",
			kit.WorktreesDir, claimed.ExcludeErr)
	}
	fmt.Printf("\nnext:\n")
	fmt.Printf("  git worktree add %s %s\n", repo.WorktreePath(claimed.Branch), claimed.Branch)
	// No status edit to instruct: the branch just pushed is what says this is started,
	// and asking for a second copy of that in the doc is what the model forbade all along.
	fmt.Printf("  work there, not here — the branch is the claim, so nothing else to set\n\n")
	return 0
}

// ---------------------------------------------------------------- release

func runRelease(args []string) int {
	fs := newFlagSet("release", args)
	force := fs.Bool("force", false, "delete the branch even though it carries work")
	rest := parseArgs(fs, args)
	if len(rest) != 1 {
		fmt.Fprintf(os.Stderr, "kit release: need one feature id\n\n%s\n", commandHelp["release"])
		return 2
	}
	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit release: %v\n", err)
		return 2
	}
	id, err := kit.ResolveID(repo, rest[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit release: %v\n", err)
		return 2
	}

	res, err := kit.Release(repo, id, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit release: %v\n", err)
		return 2
	}

	fmt.Printf("\n  released  %s\n", res.Branch)
	if res.Remote == "" {
		fmt.Printf("  the claim was local only, so there was nothing on a remote to delete\n")
	} else {
		fmt.Printf("  deleted   %s/%s — the feature is free for anyone to claim\n",
			res.Remote, res.Branch)
	}
	// A local branch left behind still reads as a claim in this checkout, so saying it is
	// gone when it is not would hand the next agent a feature the board here calls taken.
	if res.Kept != "" {
		fmt.Printf("\n  the local branch is still here: %s\n", res.Kept)
		fmt.Printf("  leave that checkout, then: git branch -D %s\n", res.Branch)
	}
	fmt.Println()
	return 0
}

// ---------------------------------------------------------------- block

func runBlock(args []string) int {
	fs := newFlagSet("block", args)
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
	// A question blocks by id, so a doc whose frontmatter says two things about itself
	// blocks nothing anyone can act on. Same refusal `ship` and `drop` make.
	if _, err := repo.WriteTarget(id); err != nil {
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
	fs := newFlagSet("ship", args)
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
	if res.Commit != "" {
		fmt.Printf("  commit   %s — the stamp is committed, not left in the tree\n", res.Commit)
	}
	fmt.Printf("\nnext:\n")
	if res.TechCreated {
		fmt.Printf("  created %s — the as-built doc. See: kit help as-built\n", res.TechDoc)
	}
	if res.TechDoc != "" {
		fmt.Printf("  move durable facts into %s, one line each — this doc is frozen history now\n", res.TechDoc)
	}
	// The three destinations, printed where the routing is actually decided: the protocol
	// states them at the start of a session and this is the end of one.
	fmt.Printf("  leave in ## Log only what changed the plan — the rest of the work belongs in the PR\n")
	// Worded to lead with what to keep, because the risk is an agent reading "compact" as
	// "delete": the doc is the only record of why the code is like this.
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
	fs := newFlagSet("next", args)
	asJSON := fs.Bool("json", false, "machine-readable output")
	offline := fs.Bool("offline", false, "do not ask the remote which branches exist")
	if rest := parseArgs(fs, args); len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "kit next: takes no arguments — it reads the project you "+
			"are in\nfor several projects: for d in ~/code/*; do (cd $d && kit next); done\n")
		return 2
	}

	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit next: %v\n", err)
		return 2
	}
	repo.Offline = *offline

	// Both formats carry the same facts, decided once, because an agent reading --json
	// and a person reading the board must never be told different things.
	notSetUp := ""
	if repo.FeaturesDir == "" {
		notSetUp = "not set up yet — run: kit init " + repo.Root
	}
	plan := kit.NewPlan()
	plan.Error = notSetUp
	if notSetUp == "" {
		plan = kit.Survey(repo)
		drift := kit.ProtocolState(repo, currentProtocol())
		plan.ProtocolStale = drift != kit.ProtocolCurrent
		plan.ProtocolAhead = drift == kit.ProtocolAhead
	}

	// Surveyed nothing, so the empty board is not a claim that there is nothing to do.
	// Exit 2 — could not run — the same answer `audit` gives the same directory.
	code := 0
	if notSetUp != "" {
		code = 2
	}
	if *asJSON {
		if failed := emit(plan); failed != 0 {
			return failed
		}
		return code
	}
	color := isTTY()
	fmt.Print(paint("\n"+repo.Name+"\n", "1", color))
	if notSetUp != "" {
		fmt.Printf("  not set up yet — run:  kit init %s\n\n", repo.Root)
		return code
	}
	printPlan(plan, color)
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

func printPlan(plan kit.Plan, color bool) {
	// Above the board, because an agent that has read the plan and started working has
	// already walked past anything printed after it. Never changes the exit code: the plan
	// is still right, it is the rules around it that drifted.
	// Never changes the exit code: the plan is still right, it is the rules around it
	// that drifted. Which way they drifted decides what to say — `kit init .` writes the
	// binary's protocol over the project's, so on the ahead side it destroys the newer one.
	switch {
	case plan.ProtocolAhead:
		fmt.Println(paint("  ! this project writes the protocol and the kit on $PATH is "+
			"older — run make install; kit init . here would write the older one back",
			"33", color))
	case plan.ProtocolStale:
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
			if item.Moved != nil {
				moved = ", " + ago(*item.Moved)
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
		// guidance says what to do; this is the thing to run. Not while something is
		// awaiting review: there the guidance says merging is the work, and a second
		// command under it is the page giving two answers.
		if plan.Empty() && len(plan.AwaitingReview) == 0 {
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
	fs := newFlagSet("audit", args)
	asJSON := fs.Bool("json", false, "machine-readable output")
	strict := fs.Bool("strict", false, "exit 1 if anything HIGH is found")
	offline := fs.Bool("offline", false, "do not ask the remote which branches exist")
	if rest := parseArgs(fs, args); len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "kit audit: takes no arguments — it reads the project you "+
			"are in\nfor several projects: for d in ~/code/*; do (cd $d && kit audit); done\n")
		return 2
	}

	repo, err := kit.Open(kit.ProjectRoot("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit audit: %v\n", err)
		return 2
	}
	repo.Offline = *offline

	result := audited{Findings: []kit.Finding{}}
	if repo.FeaturesDir == "" {
		result.Error = "not set up yet — run: kit init " + repo.Root
	} else {
		if findings := kit.Audit(repo, today(), currentProtocol()); findings != nil {
			result.Findings = findings
		}
		// After the audit, so it reports what actually went unanswered.
		result.RemoteUnanswered = repo.RemoteUnanswered()
		// --offline drops the same checks an unreachable remote does, and used to drop
		// them without a word: a HIGH check simply did not run and --strict exited 0.
		if *offline {
			result.Skipped = kit.CheckCodes(kit.OfflineChecks)
		}
	}

	if *asJSON {
		if code := emit(result); code != 0 {
			return code
		}
	} else {
		printFindings(repo, result, isTTY())
	}

	// A directory with no docs/features was not audited: nothing was read, so "no
	// findings" is not a statement about it. Exit 2 — could not run — before --strict is
	// consulted, so a CI gate cannot go green on a repo that has no plan at all.
	if result.Error != "" {
		return 2
	}
	if *strict {
		for _, f := range result.Findings {
			if f.Severity == kit.High {
				return 1
			}
		}
	}
	return 0
}

// today is the date every dated check is measured against, read once and passed in.
// A clock inside a check leaves a test unable to catch a UTC-midnight off-by-one.
func today() time.Time { return time.Now().UTC().Truncate(24 * time.Hour) }

// audited is the audit in --json: what was found, or why nothing could be. A bare
// list has nowhere to say why it is empty, so the findings are wrapped.
type audited struct {
	Findings []kit.Finding `json:"findings"`
	Error    string        `json:"error,omitempty"`

	// RemoteUnanswered says the remote was asked and did not answer, so this is a
	// weaker statement than it looks: shipped-not-pushed could not run at all, and the
	// claim checks read the last fetch. Without it a clean bill and an unreachable
	// remote print the same page.
	RemoteUnanswered bool `json:"remote_unanswered,omitempty"`

	// Skipped names the checks --offline asked for and did not run. Same weakening as
	// above, chosen rather than suffered, and it has to be said for the same reason.
	Skipped []string `json:"skipped,omitempty"`
}

func printFindings(repo *kit.Repo, result audited, color bool) {
	tint := map[string]string{kit.High: "1;31", kit.Medium: "33", kit.Low: "2;37"}

	fmt.Print(paint("\n"+repo.Name+"\n", "1", color))
	if result.Error != "" {
		fmt.Printf("  not set up yet — run:  kit init %s\n\n", repo.Root)
		return
	}
	skipped := strings.Join(kit.CheckCodes(kit.OfflineChecks), ", ")
	if result.RemoteUnanswered {
		fmt.Printf("  %s\n", paint("! the remote did not answer: "+skipped+
			"\n    did not run, and the claim checks read the last fetch", "33", color))
	}
	if len(result.Skipped) > 0 {
		fmt.Printf("  %s\n", paint("! --offline: "+strings.Join(result.Skipped, ", ")+
			"\n    did not run, and the claim checks read the last fetch", "33", color))
	}
	if len(result.Findings) == 0 {
		fmt.Printf("  docs and repo agree\n\n")
		return
	}
	for _, severity := range kit.Severities {
		printed := false
		for _, f := range result.Findings {
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
	fmt.Println()

	// Counted by severity, and said only when there is something to say. The line used
	// to read "N thing(s) to fix. HIGH first." whether or not any finding was HIGH.
	high, medium := 0, 0
	for _, f := range result.Findings {
		switch f.Severity {
		case kit.High:
			high++
		case kit.Medium:
			medium++
		}
	}
	switch {
	case high > 0:
		fmt.Printf("%s, %s. Fix the HIGH first.\n\n", count(high, "HIGH finding"),
			count(medium, "MEDIUM"))
	case medium > 0:
		fmt.Printf("%s to fix.\n\n", count(medium, "MEDIUM finding"))
	}
}

// count writes "1 thing" or "3 things", because "1 thing(s)" is what the summary line
// said for every count including one.
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// ---------------------------------------------------------------- shared

// newFlagSet builds a subcommand's flags with help that describes the command,
// not just its options — an agent should not need the source to use it.
//
// The args decide where that help goes. Asked for, it is the output: `kit next -h`
// wrote to stderr and exited 0, so redirecting it produced an empty file while
// `kit new --help` produced the text. Printed after a bad flag it is a diagnostic and
// stays with the error, because flag writes both to one stream and stdout has to stay
// clean on a failure.
func newFlagSet(name string, args []string) *flag.FlagSet {
	out := os.Stderr
	if slices.ContainsFunc(args, isHelpFlag) {
		out = os.Stdout
	}
	fs := flag.NewFlagSet("kit "+name, flag.ExitOnError)
	fs.SetOutput(out)
	fs.Usage = func() {
		// Nothing useful to do if printing help fails; the process is on its
		// way out either way.
		_, _ = fmt.Fprintln(out, commandHelp[name])
		_, _ = fmt.Fprintln(out, "\nOptions:")
		fs.PrintDefaults()
	}
	return fs
}

// isHelpFlag names the three spellings Go's flag package answers to.
func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "-help"
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

func emit(payload any) int {
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit: %v\n", err)
		return 2
	}
	fmt.Println(string(out))
	return 0
}

// expand resolves a leading ~, which a shell would have done for an unquoted path but
// does not for one typed inside quotes. `init` is the one command still taking a path.
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

// currentProtocol is the work-protocol section this binary would write, so `audit`
// can tell a project it is running an older one. Empty on a read failure, which
// simply skips the check rather than reporting a difference it could not verify.
func currentProtocol() string {
	raw, err := fs.ReadFile(templates, kit.ProtocolTemplate)
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
