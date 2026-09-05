package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/sanyatihy/project-kit/internal/kit"
)

// subcommands is the dispatcher in main.go, read from the source. It used to be a
// hand-kept copy of that switch, so a command added to one and not the other was
// invisible: the list that was supposed to catch the drift drifted with it. `help`
// and `version` are dropped — neither takes a `--help` of its own.
func subcommands(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing main.go: %v", err)
	}
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "run" {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			clause, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				name := strings.Trim(lit.Value, `"`)
				if strings.HasPrefix(name, "-") || name == "help" || name == "version" {
					continue
				}
				names = append(names, name)
			}
			return true
		})
		return false
	})
	if len(names) == 0 {
		t.Fatal("found no commands in main.go's dispatcher")
	}
	return names
}

func TestEveryCommandHasHelp(t *testing.T) {
	t.Parallel()
	dispatched := subcommands(t)
	for _, name := range dispatched {
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
		if !slices.Contains(dispatched, name) {
			t.Errorf("commandHelp has %q, which main.go's dispatcher does not accept", name)
		}
	}
}

func TestOverviewListsEveryCommand(t *testing.T) {
	t.Parallel()
	for _, name := range append(subcommands(t), "help", "version") {
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

func TestProtocolNamesTheCommandsThatExist(t *testing.T) {
	t.Parallel()
	// A protocol that describes the manual path for something the tool does is a
	// protocol that teaches the mistake. `kit submit` exists because setting the
	// status and date by hand is where two audit findings came from.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	for _, want := range []string{"kit claim", "kit block", "kit submit", "kit audit --strict"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the protocol never mentions %q, so an agent following it does that step by hand", want)
		}
	}

	// The other half, and the one that rots: this text is written into every project,
	// and `kit init` gives the default one no Makefile at all. A rule naming a command
	// the reader does not have is a rule they cannot follow — the gate is named by the
	// project's own house rules, above the marker.
	for _, absent := range []string{"make check", "make install", "make release"} {
		if strings.Contains(string(raw), absent) {
			t.Errorf("the protocol names %q, which a project kit init scaffolds may not have",
				absent)
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

// helpTopicRe finds a topic as the overview advertises it. `[TOPIC]` in the usage
// line does not match, which is the point: only a named one is a promise.
var helpTopicRe = regexp.MustCompile(`kit help ([a-z][a-z-]*)`)

func TestEveryTopicTheOverviewNamesResolves(t *testing.T) {
	t.Parallel()
	// TestOverviewListsEveryTopic holds the other direction. This one catches a topic
	// the overview offers and the binary cannot print — the same class as a check code
	// advertised after the check was deleted, and it exits 2 in the reader's face.
	for _, m := range helpTopicRe.FindAllStringSubmatch(overview, -1) {
		if !slices.Contains(topicNames(), m[1]) {
			t.Errorf("the overview offers %q, which resolves to no topic", "kit help "+m[1])
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
		case !slices.Contains(kit.Severities, c.Severity):
			t.Errorf("check %q has severity %q, which is not a real level", c.Code, c.Severity)
		}
		seen[c.Code] = true
	}
	if len(kit.Checks) == 0 {
		t.Fatal("the check registry is empty")
	}
}
