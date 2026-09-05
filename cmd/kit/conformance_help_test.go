package main

// An outside review of this tool, run against `kit help` and the subcommands
// without reading the repository, reached four conclusions that were wrong. Every
// one was a fair reading of what the binary printed: the facts existed in the code
// and nowhere a user could see them.
//
// The kit promises that an agent never needs this repository to use it. These are
// that promise, made checkable.

import (
	"strings"
	"testing"
)

func TestEveryTopicIsReachableFromTheOverview(t *testing.T) {
	t.Parallel()
	// A topic nobody is told about may as well not be embedded — which is exactly
	// how the INBOX format stayed invisible while the tool enforced it.
	p := newProject(t)
	overviewText := p.run("help").ok().Stdout
	for _, topic := range topicNames() {
		if !strings.Contains(overviewText, "kit help "+topic) {
			t.Errorf("topic %q is not advertised in the overview", topic)
		}
	}
}

func TestHelpAskedForIsOutputAndNotADiagnostic(t *testing.T) {
	t.Parallel()
	// `kit next -h > next.txt` wrote an empty file: flag prints usage to the stream it
	// prints a bad-flag error to, and that is stderr. `kit new --help` wrote to stdout,
	// because it answers the flag itself — so the binary disagreed with itself about
	// what asking for help is.
	p := newProject(t)
	for _, cmd := range append(subcommands(t), "help") {
		for _, flag := range []string{"-h", "--help"} {
			r := p.run(cmd, flag).ok()
			if strings.TrimSpace(r.Stdout) == "" {
				t.Errorf("kit %s %s wrote nothing to stdout%s", cmd, flag, r.context())
			}
			if strings.TrimSpace(r.Stderr) != "" {
				t.Errorf("kit %s %s wrote help to stderr, so redirecting it loses the "+
					"text and keeps the noise%s", cmd, flag, r.context())
			}
		}
	}
}

func TestABadFlagKeepsItsDiagnosticsOffStdout(t *testing.T) {
	t.Parallel()
	// The other half of routing help by what was asked for: usage printed after an
	// error is part of the error, and stdout stays clean on a failure.
	r := newProject(t).run("audit", "--nonesuch").fails(2)
	if strings.TrimSpace(r.Stdout) != "" {
		t.Errorf("a rejected flag put text on stdout%s", r.context())
	}
	r.errHas("nonesuch", "kit audit")
}
