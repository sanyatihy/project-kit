package main

// A green `make check` has to mean a green CI, and the two lists drift because
// nothing compares them: the gate is a Makefile target and the job is a second copy
// of it in YAML. `main` was red for a lint the documented gate never ran.
//
// The Makefile is the one home for the list. `CI_STEPS` names it; the job may add
// setup steps, but every command it runs has to be on the list.

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	makefilePath = "../../Makefile"
	workflowPath = "../../.github/workflows/check.yml"
)

var (
	ciStepsRe = regexp.MustCompile(`(?m)^CI_STEPS\s*:?=\s*(.*)$`)
	// A target line: a name at the start of a line, then its prerequisites. Recipes
	// are tab-indented and variables have a space before `:=`, so neither matches.
	targetRe  = regexp.MustCompile(`(?m)^([a-zA-Z][a-zA-Z0-9_-]*):(?:[ \t]+(.*))?$`)
	runStepRe = regexp.MustCompile(`(?m)^\s*-\s+run:\s+(.*)$`)
)

func readOrFail(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(raw)
}

// makeTargets maps every target in the Makefile to its prerequisites.
func makeTargets(src string) map[string][]string {
	targets := map[string][]string{}
	for _, m := range targetRe.FindAllStringSubmatch(src, -1) {
		targets[m[1]] = strings.Fields(m[2])
	}
	return targets
}

// reaches records every target `make name` would run, name included.
func reaches(targets map[string][]string, name string, seen map[string]bool) {
	if seen[name] {
		return
	}
	seen[name] = true
	for _, dep := range targets[name] {
		reaches(targets, dep, seen)
	}
}

func TestCIRunsExactlyTheMakefilesList(t *testing.T) {
	t.Parallel()
	makefile := readOrFail(t, makefilePath)
	declared := ciStepsRe.FindStringSubmatch(makefile)
	if declared == nil {
		t.Fatalf("%s declares no CI_STEPS, so nothing states what CI runs", makefilePath)
	}
	want := strings.Fields(declared[1])

	targets := makeTargets(makefile)
	for _, step := range want {
		if _, ok := targets[step]; !ok {
			t.Errorf("CI_STEPS names %q, which is not a target in %s", step, makefilePath)
		}
	}

	var got []string
	for _, m := range runStepRe.FindAllStringSubmatch(readOrFail(t, workflowPath), -1) {
		cmd := strings.TrimSpace(m[1])
		if !strings.HasPrefix(cmd, "make ") {
			t.Errorf("%s runs %q, which no developer can run by name.\n"+
				"Give it a Makefile target and put that target in CI_STEPS.", workflowPath, cmd)
			continue
		}
		got = append(got, strings.TrimSpace(strings.TrimPrefix(cmd, "make ")))
	}
	if !slices.Equal(got, want) {
		t.Errorf("the job runs %v and CI_STEPS says %v — one of them is not the gate", got, want)
	}
}

// The split is the point: `check` is everything a developer can run offline, and what
// is left needs the network. A network step that crept into the gate would make the
// documented pre-submit command fail on a plane.
func TestTheGateIsCIMinusTheNetwork(t *testing.T) {
	t.Parallel()
	makefile := readOrFail(t, makefilePath)
	workflow := readOrFail(t, workflowPath)

	gate := map[string]bool{}
	reaches(makeTargets(makefile), "check", gate)
	if !gate["lint"] {
		t.Error("`make check` no longer reaches `lint`, so the documented gate lints nothing")
	}
	for _, step := range strings.Fields(ciStepsRe.FindStringSubmatch(makefile)[1]) {
		if step != "check" && gate[step] {
			t.Errorf("`make check` reaches %q, which needs the network", step)
		}
	}

	// CI provides the required linter before running the gate.
	if !strings.Contains(workflow, "golangci/golangci-lint-action@") {
		t.Errorf("%s installs no linter, so `make check` there cannot run", workflowPath)
	}
	if strings.Contains(workflow, "version: latest") {
		t.Errorf("%s takes whatever golangci-lint released today, which turns the trunk "+
			"red with no commit behind it", workflowPath)
	}
}

func TestGoScaffoldProvidesTheLinterItsGateRequires(t *testing.T) {
	t.Parallel()
	github := readOrFail(t, "../../templates/stacks/go/check.yml")
	gitlab := readOrFail(t, "../../templates/stacks/go/gitlab-ci.yml")
	for _, want := range []string{"version: v2.12.2", "install-only: true", "timeout-minutes: 15"} {
		if !strings.Contains(github, want) {
			t.Errorf("Go GitHub gate lacks %q", want)
		}
	}
	for _, want := range []string{"golangci-lint@v2.12.2", "timeout: 15m"} {
		if !strings.Contains(gitlab, want) {
			t.Errorf("Go GitLab gate lacks %q", want)
		}
	}
	if !strings.Contains(readOrFail(t, workflowPath), "fetch-depth: 0") {
		t.Error("the kit audit needs trunk history to protect frozen documents")
	}
}
