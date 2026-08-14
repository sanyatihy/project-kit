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

	res, err := kit.Init(templates, root, *stack)
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

	// Singular here is what produced a plan with one feature covering a whole
	// project: the tool asked for a feature and the agent wrote a feature. Bootstrap
	// means the whole set, and accepting them is not the agent's to do.
	fmt.Printf("\nnext:\n")
	fmt.Printf("  write docs/VISION.md — what this is for, and what it is not\n")
	fmt.Printf("  then decompose it: one kit new per feature the vision describes,\n")
	fmt.Printf("  each one a pull request a human can review in one sitting\n")
	fmt.Printf("  leave them Draft — accepting them is the human's call\n")
	// Said here because this is where a bootstrap run reads what to do next, and the
	// omission is what let `kit claim` win a branch carrying no plan: a claim pushes
	// HEAD, so a plan that is only in the working tree does not travel with it.
	fmt.Printf("  then land all of it on the default branch — a pull request, if that\n")
	fmt.Printf("  branch is protected. A claim pushes HEAD, so nothing uncommitted\n")
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
	res, err := kit.Accept(repo, id)
	if err != nil {
		// Exit 2 throughout: unlike `claim`, none of these is another agent holding the
		// feature, so "take something else" would be the wrong instruction.
		fmt.Fprintf(os.Stderr, "kit accept: %v\n", err)
		return 2
	}

	fmt.Printf("\n  accepted  %s  (was Draft)\n", res.Doc)
	fmt.Printf("\nnext:\n")
	// Where the edit lands is the whole reason this prints anything. Accepting inside the
	// draft's own pull request is one review and one merge per feature; accepting after it
	// merges is a second pull request for a one-line change, which is what the protocol
	// left undefined and a human had to guess at every time.
	fmt.Printf("  commit this on %s so it lands in the same pull request as the draft\n",
		res.Branch)
	fmt.Printf("  the branch is deleted on merge — its name is the claim name\n")
	fmt.Printf("  kit next\n\n")
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
	fmt.Printf("  set status: In progress in the doc, and work there — not here\n\n")
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

	repos, code := openAll(paths, "kit next", *offline)
	if code != 0 {
		return code
	}

	// surveyed keeps the staleness check next to the plan it belongs to, so both output
	// formats carry the same fact and cannot drift.
	surveyed := func(r *kit.Repo) kit.Plan {
		plan := kit.Survey(r)
		plan.ProtocolStale = kit.ProtocolIsStale(r, currentProtocol())
		return plan
	}

	if *asJSON {
		payload := map[string]kit.Plan{}
		for _, r := range repos {
			payload[r.Name] = surveyed(r)
		}
		return emit(payload)
	}

	color := isTTY()
	for _, r := range repos {
		fmt.Print(paint("\n"+r.Name+"\n", "1", color))
		if r.FeaturesDir == "" {
			fmt.Printf("  not set up yet — run:  kit init %s\n\n", r.Root)
			continue
		}
		printPlan(r, surveyed(r), color)
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
			fmt.Printf("    %s  %-5s %s — %s\n", item.ID, item.Progress(), kit.Show(item.Title, 38), item.ClaimedBy)
		}
	}

	if len(plan.Blocked) > 0 {
		fmt.Printf("\n  %s\n", paint("blocked", "33", color))
		for _, item := range plan.Blocked {
			fmt.Printf("    %s  %s\n", item.ID, kit.Show(item.Title, 60))
			fmt.Printf("          waiting on: %s\n", kit.Show(item.WaitingOn, 56))
		}
	}

	// The drafts listing is data the JSON payload does not carry, so it stays here.
	// The conclusion under it does not: that comes from plan.Guidance, which --json
	// carries too, so the two formats cannot end up saying different things.
	if plan.Empty() {
		if drafts := kit.Drafts(r); len(drafts) > 0 {
			fmt.Printf("\n  nothing open, but there are drafts:\n")
			for _, f := range drafts {
				fmt.Printf("    %s  %s\n", f.ID, kit.Show(f.Title, 60))
				fmt.Printf("          %s\n", f.Rel)
			}
		}
	}
	if plan.Guidance != "" && len(plan.Yours) == 0 && len(plan.Ready) == 0 {
		fmt.Printf("\n%s\n", paint(wrapAt(plan.Guidance, 74, "  "), "33", color))
		// The command goes on its own line, unwrapped, so it can be copied. The
		// guidance says what to do; this is the thing to run.
		if plan.Empty() && len(kit.Drafts(r)) == 0 {
			fmt.Printf("    kit new \"<what the feature does>\"\n")
		}
	}
	fmt.Println()
}

// ---------------------------------------------------------------- audit

func runAudit(args []string) int {
	fs := newFlagSet("audit")
	asJSON := fs.Bool("json", false, "machine-readable output")
	strict := fs.Bool("strict", false, "exit 1 if anything HIGH is found")
	offline := fs.Bool("offline", false, "do not ask the remote which branches exist")
	paths := parseArgs(fs, args)

	repos, code := openAll(paths, "kit audit", *offline)
	if code != 0 {
		return code
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	results := make(map[string][]kit.Finding, len(repos))
	order := make([][]kit.Finding, 0, len(repos))
	for _, r := range repos {
		var findings []kit.Finding
		if r.FeaturesDir != "" {
			findings = kit.Audit(r, today, currentProtocol())
		}
		if findings == nil {
			findings = []kit.Finding{}
		}
		results[r.Name] = findings
		order = append(order, findings)
	}

	if *asJSON {
		if code := emit(results); code != 0 {
			return code
		}
	} else {
		printFindings(repos, order, isTTY())
	}

	if *strict {
		for _, findings := range results {
			for _, f := range findings {
				if f.Severity == kit.High {
					return 1
				}
			}
		}
	}
	return 0
}

func printFindings(repos []*kit.Repo, all [][]kit.Finding, color bool) {
	actionable := 0
	tint := map[string]string{kit.High: "1;31", kit.Medium: "33", kit.Low: "2;37"}

	for i, r := range repos {
		fmt.Print(paint("\n"+r.Name+"\n", "1", color))
		if r.FeaturesDir == "" {
			fmt.Printf("  not set up yet — run:  kit init %s\n\n", r.Root)
			continue
		}
		findings := all[i]
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

func openAll(paths []string, who string, offline bool) ([]*kit.Repo, int) {
	if len(paths) == 0 {
		paths = []string{kit.ProjectRoot(".")}
	}
	repos := make([]*kit.Repo, 0, len(paths))
	for _, given := range paths {
		root := expand(given)
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s: not a directory: %s\n", who, given)
			return nil, 2
		}
		r, err := kit.Open(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", who, err)
			return nil, 2
		}
		r.Offline = offline
		repos = append(repos, r)
	}
	return repos, 0
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
