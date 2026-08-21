package main

// A refusal list is complete or it is not a list. `kit ship --help` named four of its
// eight and the protocol named the same four; `release`'s exit 2 list named six of ten;
// and `claim --help` said every refusal it makes is one the board already makes, which
// none of its seven exit 2 refusals are. Each omission reads as a promise: an agent that
// meets an unlisted refusal has been told the command could not have done that.
//
// Two lists, held together. `refusals` names every reason a command turns work away and
// the words its help has to carry; the second test counts the refusals in the source, so
// a new one cannot be added to the code without landing here, and from here it cannot be
// left out of the help.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// refusal is one reason a command turns work away: the words `kit <cmd> --help` has to
// carry for it, and what it is, for the failure message.
type refusal struct {
	says string
	why  string
}

var refusals = map[string]struct {
	// The functions that raise them. `pickRemote` is counted with its two callers
	// because its refusal is theirs: it reaches the reader as `kit claim` turning
	// them away, and nothing else surfaces it.
	fns  []string
	list []refusal
}{
	"ship": {[]string{"Ship"}, []refusal{
		{"box is ticked", "a box still unticked"},
		{"no ## Tasks boxes", "a doc with no boxes at all"},
		{"already Shipped", "a doc that is already Shipped"},
		{"Dropped", "a doc that was retired"},
		{"no status: line", "a doc with no status: to set"},
		{"needs: has not", "an unmet needs:"},
		{"is claimed by a branch", "a claim another checkout holds"},
		{"uncommitted or untracked files", "a dirty working tree"},
	}},
	"release": {[]string{"Release", "pickRemote"}, []refusal{
		{"no feature or chore", "an id nothing is named after"},
		{"nothing claims it", "a feature no branch claims"},
		{"two branches claim it", "a split claim"},
		{"this clone never created", "a claim another agent made"},
		{"cannot resolve", "a branch whose commits this checkout does not have"},
		{"not the remote's", "a local copy behind the branch the remote carries"},
		{"no default branch", "no trunk to read the branch's work against"},
		{"several remotes and no origin", "no way to tell which remote holds the claim"},
		{"the remote did not answer", "a remote that could not be asked"},
		{"--force was not given", "a branch carrying work"},
	}},
	"chore": {[]string{"ClaimChore", "pickRemote"}, []refusal{
		{"no title", "a chore with nothing to call it"},
		{"a title with no usable characters", "a title that slugs to nothing"},
		{"a title containing a four-digit number", "a branch that would claim a feature"},
		{"the project is not set up", "a project with no docs to record a chore in"},
		{"no commits yet", "a branch with nothing to point at"},
		{"this checkout already claims something", "a second unit of work in one checkout"},
		{"a chore under that name is already claimed", "work somebody else is already doing"},
		{"ahead of the default branch", "a claim carrying another feature's commits"},
		{"the branch is already on the remote", "a lost race"},
		{"the push said nothing about the ref", "a push that answered nothing"},
		{"several remotes and no origin", "no way to tell which remote decides"},
	}},
	"claim": {[]string{"ClaimFeature", "pickRemote"}, []refusal{
		{"no such feature", "an id nothing is named after"},
		{"two docs share the id", "one branch that would claim two features"},
		{"the id: disagrees with the filename", "a claim the board could not see"},
		{"no status:", "a doc that says nothing about its own state"},
		{"the doc is closed", "a Shipped or Dropped doc"},
		{"## Problem, ## Design or ## Tasks", "a doc nobody wrote"},
		{"open INBOX question", "a BLOCKING inbox line"},
		{"needs: has not shipped", "an unmet needs:"},
		{"no commits yet", "a branch with nothing to point at"},
		{"the plan not committed here", "a claim that would carry no plan"},
		{"not landed on the default branch", "a claim branching off the plan round"},
		{"ahead of the default branch", "a claim carrying another feature's commits"},
		{"this checkout already claims something", "a second unit of work in one checkout"},
		{"a branch already claims it", "a claim this checkout can already see"},
		{"another agent got there first", "a lost race"},
		{"the push said nothing about the ref", "a push that answered nothing"},
		{"several remotes and no origin", "no way to tell which remote decides"},
	}},
}

func TestEveryRefusalIsInTheCommandsHelp(t *testing.T) {
	t.Parallel()
	for cmd, spec := range refusals {
		for _, r := range spec.list {
			if !strings.Contains(commandHelp[cmd], r.says) {
				t.Errorf("kit %s refuses %s and its help never says %q — an agent that "+
					"meets it was told the command could not have done that",
					cmd, r.why, r.says)
			}
		}
	}

	// The protocol used to restate four of `ship`'s refusals as the whole list, which is
	// how a rule an agent reads every session came to describe a command that refuses for
	// twice as many reasons. One list, in the command that makes them.
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		t.Fatalf("reading the protocol: %v", err)
	}
	if !strings.Contains(string(raw), "kit ship --help") {
		t.Error("the protocol does not send the reader to `kit ship --help`, so its own " +
			"partial list of refusals is the only one they get")
	}
}

// refusalsIn finds the errors a function raises itself, as against the ones it passes
// on: a return whose value is built from `fmt.Errorf` or `&NotYours{}` is this command
// turning work away, while a bare `err` is something further down failing. The whole
// return is searched, because `claim` wraps the lost race in `unclaim` to take its
// branch back down before reporting it.
func refusalsIn(t *testing.T, fns []string) []string {
	t.Helper()
	const dir = "../../internal/kit"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	var where []string
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
			decl, ok := n.(*ast.FuncDecl)
			if !ok || !slices.Contains(fns, decl.Name.Name) {
				return true
			}
			ast.Inspect(decl.Body, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok || !raisesAnError(ret) {
					return true
				}
				where = append(where, fset.Position(ret.Pos()).String())
				return true
			})
			return false
		})
	}
	slices.Sort(where)
	return where
}

func raisesAnError(ret *ast.ReturnStmt) bool {
	raised := false
	ast.Inspect(ret, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.CallExpr:
			if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Errorf" {
				raised = true
			}
		case *ast.CompositeLit:
			if id, ok := e.Type.(*ast.Ident); ok && id.Name == "NotYours" {
				raised = true
			}
		}
		return !raised
	})
	return raised
}

func TestEveryRefusalInTheCodeIsListed(t *testing.T) {
	t.Parallel()
	for cmd, spec := range refusals {
		where := refusalsIn(t, spec.fns)
		if len(where) != len(spec.list) {
			t.Errorf("kit %s raises %d refusals and `refusals[%q]` lists %d.\n"+
				"Every one has to be in the command's help, so add it here with the words "+
				"the help carries:\n  %s",
				cmd, len(where), cmd, len(spec.list), strings.Join(where, "\n  "))
		}
	}
}
