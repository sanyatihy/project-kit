package main

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// subcommands must stay in step with the dispatcher in main.go.
var subcommands = []string{"init", "new", "accept", "claim", "block", "ship", "drop", "next", "audit"}

func TestEveryCommandHasHelp(t *testing.T) {
	t.Parallel()
	for _, name := range subcommands {
		text, ok := commandHelp[name]
		if !ok {
			t.Errorf("no commandHelp for %q — `kit %s --help` would print bare flags", name, name)
			continue
		}
		if !strings.HasPrefix(text, "kit "+name) {
			t.Errorf("commandHelp[%q] should open with its usage line, got %.40q", name, text)
		}
	}
	for name := range commandHelp {
		if !contains(subcommands, name) {
			t.Errorf("commandHelp has %q, which is not a subcommand", name)
		}
	}
}

func TestOverviewListsEveryCommand(t *testing.T) {
	t.Parallel()
	for _, name := range append(subcommands, "help", "version") {
		if !strings.Contains(overview, "kit "+name) {
			t.Errorf("overview never mentions %q; an agent reading only the binary would miss it", name)
		}
	}
}

// The exit codes are a contract that TestExitCodesAreDistinct enforces on the
// binary. The binary also has to say what they mean, or the only statement of the
// contract is a test the user cannot read.
func TestOverviewDocumentsTheExitCodes(t *testing.T) {
	t.Parallel()
	if !strings.Contains(overview, "Exit codes") {
		t.Error("the overview never mentions exit codes")
	}
	for _, code := range []string{"0", "1", "2"} {
		if !strings.Contains(overview, "\n  "+code+"  ") {
			t.Errorf("exit code %s is not documented in the overview", code)
		}
	}
}

func TestProtocolCitesRulesByNameNotNumber(t *testing.T) {
	t.Parallel()
	// Two cross-references pointed at the wrong rules for months, because 0007 and
	// 0009 inserted rules above them and numbers in prose do not move. Naming the
	// rule removes the class rather than the two instances.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	if m := regexp.MustCompile(`(?i)\brule \d+`).FindString(string(raw)); m != "" {
		t.Errorf("the protocol cites %q — a rule inserted above it will silently "+
			"misdirect the reader; cite the rule by its bolded name instead", m)
	}
}

func TestNoCommandTellsAnAgentToAcceptADraft(t *testing.T) {
	t.Parallel()
	// Accepting is the human's gate, and a tool that instructs an agent past it
	// makes the gate advisory. 0014 removed this from `kit next` and from the
	// protocol text, and it survived in `init` and `new` because the test read the
	// protocol rather than what the commands print — which is where an agent meets
	// the instruction.
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading the command surface: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, "fmt.Printf") && !strings.Contains(line, "fmt.Fprintf") {
			continue
		}
		for _, banned := range []string{"set status: Accepted", "set it to Accepted", "mark it Accepted"} {
			if strings.Contains(line, banned) {
				t.Errorf("a command prints %q — that is the human's call:\n  %s",
					banned, strings.TrimSpace(line))
			}
		}
	}
}

func TestProtocolNamesTheCommandsThatExist(t *testing.T) {
	t.Parallel()
	// A protocol that describes the manual path for something the tool does is a
	// protocol that teaches the mistake. `kit ship` exists because setting the
	// status and date by hand is where two audit findings came from.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	for _, want := range []string{"kit claim", "kit block", "kit ship", "make check", "kit audit --strict"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the protocol never mentions %q, so an agent following it does that step by hand", want)
		}
	}
}

func TestOverviewListsEveryTopic(t *testing.T) {
	t.Parallel()
	for _, topic := range topicNames() {
		if !strings.Contains(overview, "kit help "+topic) {
			t.Errorf("topic %q is not advertised in the overview", topic)
		}
	}
}

func TestEveryTopicResolvesInTheBinary(t *testing.T) {
	t.Parallel()
	for topic, path := range topics {
		raw, err := fs.ReadFile(templates, path)
		if err != nil {
			t.Errorf("topic %q -> %q is not embedded: %v", topic, path, err)
			continue
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			t.Errorf("topic %q resolves to an empty document", topic)
		}
	}
}

// showsTaskSyntax are the topics whose job is to demonstrate the feature-doc format,
// so a `- [ ]` in them is the subject matter rather than a mistake.
var showsTaskSyntax = map[string]bool{
	"template":    true,
	"conventions": true,
	"inbox":       true,
}

// In kit, `- [ ]` means "a PR-sized unit of work, do the topmost one". Guidance that
// uses the same syntax for "a property to verify" would be read as a task list by an
// agent following the protocol, and it would work through it.
func TestNoTopicLooksLikeATaskList(t *testing.T) {
	t.Parallel()
	for topic, path := range topics {
		if showsTaskSyntax[topic] {
			continue
		}
		raw, err := fs.ReadFile(templates, path)
		if err != nil {
			continue // TestEveryTopicResolvesInTheBinary reports this
		}
		for i, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") {
				t.Errorf("%s:%d in topic %q is a task box: %q\n"+
					"an agent following rule 3 would try to work through these",
					path, i+1, topic, trimmed)
			}
		}
	}
}

// `kit help checks` is what an agent reads to know when a finding will appear, and
// the two thresholds were restated in prose in three places. Retuning them left the
// prose behind, so the numbers now come from the constants themselves.
func TestThresholdProseCarriesTheRealNumbers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code string
		days int
	}{
		{"stalled", kit.StalledDays},
		{"inbox-stale", kit.InboxStaleDays},
		{"fyi-stale", kit.FYIStaleDays},
	} {
		want := fmt.Sprintf("%d day", tc.days)
		found := false
		for _, c := range kit.Checks {
			if c.Code == tc.code {
				found = true
				if !strings.Contains(c.Means, want) {
					t.Errorf("check %q says %q, which does not carry its threshold of %s",
						tc.code, c.Means, want)
				}
			}
		}
		if !found {
			t.Errorf("no check named %q", tc.code)
		}
	}
}

func TestCheckRegistryIsSaneAndComplete(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, c := range kit.Checks {
		switch {
		case c.Code == "":
			t.Error("a check has no code")
		case seen[c.Code]:
			t.Errorf("duplicate check code %q", c.Code)
		case c.Means == "":
			t.Errorf("check %q has no explanation, so `kit help checks` would be useless", c.Code)
		case !contains(kit.Severities, c.Severity):
			t.Errorf("check %q has severity %q, which is not a real level", c.Code, c.Severity)
		}
		seen[c.Code] = true
	}
	if len(kit.Checks) == 0 {
		t.Fatal("the check registry is empty")
	}
}
