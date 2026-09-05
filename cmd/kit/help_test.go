package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

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

// ruleNumberRe is a cross-reference to a numbered rule, which is the thing that rots:
// inserting a rule above one renumbers it and no prose moves with it.
var ruleNumberRe = regexp.MustCompile(`(?i)\brule \d+`)

func TestNothingUserFacingCitesARuleByNumber(t *testing.T) {
	t.Parallel()
	// Two cross-references pointed at the wrong rules for months, because 0007 and
	// 0009 inserted rules above them and numbers in prose do not move. Naming the
	// rule removes the class rather than the two instances.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	if m := ruleNumberRe.FindString(string(raw)); m != "" {
		t.Errorf("the protocol cites %q — a rule inserted above it will silently "+
			"misdirect the reader; cite the rule by its bolded name instead", m)
	}

	// The same rot reaches the user through what the commands print, and `kit ship`
	// refused a foreign claim with "rule 8" long after the numbering had moved. Only
	// string literals are read: a comment citing a rule misdirects nobody but the
	// next reader of the source, and `_test.go` files are not user-facing at all.
	for _, dir := range []string{".", "../../internal/kit"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}
		fset := token.NewFileSet()
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", name, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				if m := ruleNumberRe.FindString(lit.Value); m != "" {
					t.Errorf("%s prints %q — cite the rule by name, or name the "+
						"command that enforces it", fset.Position(lit.Pos()), m)
				}
				return true
			})
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
	for _, want := range []string{"kit claim", "kit block", "kit ship", "kit audit --strict"} {
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

// ruleRe is a numbered protocol rule as the list writes one.
var ruleRe = regexp.MustCompile(`(?m)^(\d+)\. \*\*`)

func TestTheProtocolsRulesAreNumberedWithoutAGap(t *testing.T) {
	t.Parallel()
	// Cutting a rule means renumbering every rule below it by hand, and this round did
	// it off by one: the list ran 10, 12, 13. Nothing cites a rule by number, so a gap
	// misdirects nobody — it just tells every agent reading the protocol that a rule was
	// removed and the removal was not finished.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	for i, m := range ruleRe.FindAllStringSubmatch(string(raw), -1) {
		if want := strconv.Itoa(i + 1); m[1] != want {
			t.Fatalf("the protocol's rule %d is numbered %s — the list has a gap or a "+
				"repeat, so a rule looks deleted and half-deleted", i+1, m[1])
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

// identRe is a hyphenated lower-case token: the shape every check code has, and the
// shape a few ordinary words in the help text share.
var identRe = regexp.MustCompile(`[a-z][a-z0-9]*(?:-[a-z0-9]+)+`)

// proseWords are the hyphenated English words the help text uses. Anything else of that
// shape has to name something the binary has, because `kit ship --help` went on
// advertising `doc-bloat` for a release after the check was deleted and nothing in the
// build compared what the help says with what `audit` can report.
var proseWords = map[string]bool{
	"four-digit": true, "kit-owned": true, "mid-flight": true,
	"re-run": true, "ticked-over-total": true,
}

func TestEveryCheckCodeInHelpIsRegistered(t *testing.T) {
	t.Parallel()
	known := map[string]bool{}
	for _, c := range kit.Checks {
		known[c.Code] = true
	}
	for _, name := range topicNames() {
		known[name] = true
	}

	// String literals only, for TestNothingUserFacingCitesARuleByNumber's reason: a
	// comment naming a dead check misleads the next reader of the source, not a user.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "help.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing help.go: %v", err)
	}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		for _, at := range identRe.FindAllStringIndex(lit.Value, -1) {
			word := lit.Value[at[0]:at[1]]
			if known[word] || proseWords[word] || insideALongerToken(lit.Value, at) {
				continue
			}
			t.Errorf("%s prints %q, which is not a check in the registry.\n"+
				"Delete it, or add the check. If it is an ordinary hyphenated word, "+
				"add it to proseWords.", fset.Position(lit.Pos()), word)
		}
		return true
	})
}

// insideALongerToken reports whether a match is only the tail of something bigger: a
// filename (`0001-shorten-urls.md`), a path (`.gitlab-ci.yml`), or a capitalised word
// the pattern cannot start at (`Non-goals`, `Re-running`).
func insideALongerToken(s string, at []int) bool {
	edge := func(i int) bool {
		if i < 0 || i >= len(s) {
			return false
		}
		c := rune(s[i])
		return strings.ContainsRune("./-", c) || unicode.IsLetter(c) || unicode.IsDigit(c)
	}
	return edge(at[0]-1) || edge(at[1])
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
	"template": true,
	"inbox":    true,
}

// In kit, `- [ ]` means "an implementing task commit, do the topmost one". Guidance that
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
					"an agent working the topmost unticked box would try to work through these",
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
