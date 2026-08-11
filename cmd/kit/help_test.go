package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// subcommands must stay in step with the dispatcher in main.go.
var subcommands = []string{"init", "new", "next", "audit"}

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
