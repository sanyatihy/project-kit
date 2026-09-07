package main

// Conformance: the shape of `--json`. It is the format the kit documents for agents, so
// it is a contract and not a rendering — and nothing pinned it. Every group marshalled as
// `null` when empty while `findings` marshalled as `[]`, every unclaimed feature carried
// `moved: 0001-01-01T00:00:00Z`, and neither command was ever run with `--json` against a
// directory that is not set up, which is the one path that reports through `error`.
//
// Keys, not prose: what is pinned here is which fields exist and what shape they take.

import (
	"encoding/json"
	"strings"
	"testing"
)

// payload parses one `--json` run into its raw keys, whatever the exit code: the shape
// has to hold on the paths that report a failure too.
func (p *project) payload(want int, args ...string) map[string]json.RawMessage {
	p.t.Helper()
	r := p.run(args...).fails(want)
	var out map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.Stdout), &out); err != nil {
		p.t.Fatalf("%s emitted invalid JSON: %v\n%s", r.cmd(), err, r.Stdout)
	}
	return out
}

// isArray reports whether a key carries a JSON array rather than `null` or nothing.
func isArray(t *testing.T, payload map[string]json.RawMessage, key string) {
	t.Helper()
	raw, ok := payload[key]
	if !ok {
		t.Errorf("%q is absent; a consumer cannot tell an empty board from an old binary", key)
		return
	}
	if !strings.HasPrefix(strings.TrimSpace(string(raw)), "[") {
		t.Errorf("%q is %s, and an empty list is `[]` — `null` is a third shape every "+
			"consumer has to special case", key, raw)
	}
}

func TestTheBoardsGroupsAreAlwaysLists(t *testing.T) {
	t.Parallel()
	// A board with nothing in any group, which is where `null` used to come from.
	p := setUp(t)
	got := p.payload(0, "next", "--json")
	for _, key := range []string{"yours", "ready", "taken", "blocked", "parked"} {
		isArray(t, got, key)
	}
	if _, ok := got["guidance"]; !ok {
		t.Error("`guidance` is absent, so --json carries a board and no instruction")
	}
}

func TestNoBoardEntryClaimsToHaveMovedInYearOne(t *testing.T) {
	t.Parallel()
	// `moved` is a time.Time, and `omitempty` does nothing for a struct. Every feature
	// nobody has claimed reported a last-moved date of 0001-01-01, which an agent
	// reading the format for staleness has no way to tell from a real one.
	p := setUp(t)
	feature(p, "0001", "thing", "A thing", "Draft", "- [ ] Do it")
	p.commitAll("a feature")

	r := p.run("next", "--json").ok()
	if strings.Contains(r.Stdout, "0001-01-01T00:00:00Z") {
		t.Errorf("--json carries a zero time, which reads as a real date%s", r.context())
	}
	if strings.Contains(r.Stdout, `"moved"`) {
		t.Errorf("an unclaimed feature carries a moved time%s", r.context())
	}
}

func TestJSONReportsADirectoryThatWasNeverSurveyed(t *testing.T) {
	t.Parallel()
	// Exit 2 in both commands, and the reason in the payload rather than only on
	// stderr: an agent reading --json gets a board or a findings list either way, and
	// without `error` an empty one reads as "nothing to do" and "nothing wrong".
	p := newProject(t)

	board := p.payload(2, "next", "--json")
	for _, key := range []string{"yours", "ready", "taken", "blocked", "parked"} {
		isArray(t, board, key)
	}
	if !strings.Contains(string(board["error"]), "not set up yet") {
		t.Errorf("next --json says %s, which does not say the project was never read",
			board["error"])
	}

	found := p.payload(2, "audit", "--json")
	isArray(t, found, "findings")
	if !strings.Contains(string(found["error"]), "not set up yet") {
		t.Errorf("audit --json says %s, which does not say the project was never read",
			found["error"])
	}
}

func TestACleanAuditReportsAnEmptyListAndNoError(t *testing.T) {
	t.Parallel()
	// The other half: a project that was read and had nothing wrong. `error` has to be
	// absent, or every consumer reads a clean bill as a failure to run.
	got := cleanProject(t).payload(0, "audit", "--json")
	isArray(t, got, "findings")
	if raw, ok := got["error"]; ok {
		t.Errorf("a clean audit carries error: %s", raw)
	}
}
