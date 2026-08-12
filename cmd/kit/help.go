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
  kit block <id> "<question>"         park a decision only a human can make
  kit ship  <id>                      tick the last box: verify, stamp, freeze
  kit next  [PATH ...] [--json]       what should I work on?
  kit audit [PATH ...] [--json] [--strict]
                                      where do the docs disagree with the repo?
  kit help  [TOPIC]                   this, or a topic below
  kit version

How it works:
  Plans live in docs/features/, one markdown file per feature, named NNNN-name.md.
  The directory listing is the board and "status:" in the frontmatter is canonical.
  There is no database and no queue: the docs and the git branches are the state.

    Draft -> Accepted -> In progress -> Shipped   (exits: Dropped, Superseded)

  Draft is where a human settles the design. Once a doc is Accepted an agent can
  work it unattended, which is why "kit next" never offers a Draft.

The four rules:
  1. In progress features come before Accepted ones. Finishing beats starting.
  2. Within a feature, do the topmost unticked "## Tasks" box. List order is the
     order of work, so ordering needs no extra syntax.
  3. One agent per feature. A branch named after the doc is the claim on it.
  4. A feature named in an open docs/INBOX.md question is blocked until answered.

Blocked on something only a human can decide? Run "kit block <id> \"<question>\"",
which writes docs/INBOX.md in the format the tool parses, then move to another
feature. Never guess, and never idle.

Exit codes, so a script can tell these apart:
  0  it worked
  1  "kit audit --strict" found something HIGH. Only --strict ever exits 1
  2  the command could not run: unknown command, bad flag, missing path

Topics:
  kit help protocol      the rules agents follow, as written into AGENTS.md
  kit help template      the feature doc format, field by field
  kit help conventions   how the features directory works
  kit help checks        every audit finding and what it means
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
	"as-built":     "templates/TECH.md",
}

// commandHelp is the description and examples printed by `kit <cmd> --help`,
// above the flag list.
var commandHelp = map[string]string{
	"init": `kit init [PATH] [--stack STACK]

Set a project up, or fill in what an existing one is missing.

Creates the directory if it does not exist, makes it a git repo if it is not one,
and writes docs/features/{README,TEMPLATE,BACKLOG}.md, docs/INBOX.md, and AGENTS.md.
With --stack it also scaffolds the code layout and one CI workflow.

Your files are never overwritten or deleted, so it is safe to run on a populated
directory and safe to re-run. Re-running is how a project picks up protocol
changes: only the "## Work protocol" section of AGENTS.md is refreshed.

Deployment scaffolding is deliberately not generated. Add it when the project has
something to deploy.

  kit init ~/code/github.com/you/thing --stack go
  kit init .                     # fill in what is missing here`,

	"new": `kit new <title...>

Start a feature doc in the current project. Takes the next free four-digit ID,
fills in the title and today's date, and leaves status: Draft.

Fails rather than overwrites, so two agents drafting at once get an error instead
of one silently losing its work.

Then fill in Problem, Non-goals, Design, and Tasks. Anything unresolved goes in
## Open questions and the doc stays Draft; anything only a human can answer goes
in docs/INBOX.md. Set status: Accepted when it is settled.

  kit new "Shorten URLs"         # -> docs/features/0001-shorten-urls.md`,

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

Doing this by hand is where two audit findings come from — unticked-shipped and
shipped-no-date — so the command removes the mistake rather than reporting it
afterwards.

A Shipped doc is frozen history: durable facts belong in docs/TECH.md from then
on, and the doc itself must not be edited again.

  kit ship 0007
  kit ship 7`,

	"next": `kit next [PATH ...] [--json]

What should I work on? Reads the feature docs and git, and names one task.

Ranks In progress before Accepted, then by ID, and within a feature returns the
topmost unticked "## Tasks" box. A feature whose doc is named in an open INBOX
question is reported as blocked; one with a branch named after it is reported as
taken. Prints the git worktree command to start.

Reads only. With no PATH it uses the repo you are in, from any subdirectory.

  kit next                       # this project
  kit next ~/code/*              # every project at once
  kit next --json                # for agents`,

	"audit": `kit audit [PATH ...] [--json] [--strict]

Where do the docs disagree with the repo? Names the file, the problem, and the fix.

Findings are ranked HIGH (a doc says something untrue, or work is stuck and nobody
noticed), MEDIUM (incomplete or drifting), LOW (hygiene). See "kit help checks"
for every finding and what it means.

Reads only, and never edits a file. With no PATH it uses the repo you are in.

  kit audit                      # this project
  kit audit ~/code/*             # every project at once
  kit audit . --strict           # exit 1 on any HIGH, for CI`,
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
	fmt.Printf("Thresholds: stalled after %d days, inbox questions after %d.\n",
		kit.StalledDays, kit.InboxStaleDays)
}

func topicNames() []string {
	names := []string{"checks"}
	for name := range topics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
