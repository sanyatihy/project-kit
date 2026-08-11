package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// subcommands must stay in step with the dispatcher in main.go.
var subcommands = []string{"init", "new", "block", "ship", "next", "audit"}

func TestEveryCommandHasHelp(t *testing.T) {
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
	if !strings.Contains(overview, "Exit codes") {
		t.Error("the overview never mentions exit codes")
	}
	for _, code := range []string{"0", "1", "2"} {
		if !strings.Contains(overview, "\n  "+code+"  ") {
			t.Errorf("exit code %s is not documented in the overview", code)
		}
	}
}

func TestOverviewListsEveryTopic(t *testing.T) {
	for _, topic := range topicNames() {
		if !strings.Contains(overview, "kit help "+topic) {
			t.Errorf("topic %q is not advertised in the overview", topic)
		}
	}
}

func TestEveryTopicResolvesInTheBinary(t *testing.T) {
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
}

// In kit, `- [ ]` means "a PR-sized unit of work, do the topmost one" (protocol rule
// 3). Guidance that uses the same syntax for "a property to verify" would be read as
// a task list by an agent following the protocol, and it would work through it.
func TestNoTopicLooksLikeATaskList(t *testing.T) {
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

func TestCheckRegistryIsSaneAndComplete(t *testing.T) {
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
