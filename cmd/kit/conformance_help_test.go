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

func TestHelpAnswersWhatAnOutsideReviewGotWrong(t *testing.T) {
	t.Parallel()
	p := newProject(t)

	// Everything the binary can tell someone, in one place.
	all := p.run("help").ok().Stdout
	for _, topic := range []string{"protocol", "template", "checks",
		"inbox", "vision", "as-built", "go-checklist"} {
		all += p.run("help", topic).ok().Stdout
	}
	// Both streams: the flag package prints a command's help on stderr, so reading
	// stdout alone left every "kit <cmd> --help" out of what this checks.
	for _, cmd := range subcommands(t) {
		all += p.run(cmd, "--help").out()
	}

	for _, tc := range []struct {
		concluded string // what the review wrongly concluded
		needs     []string
	}{{
		concluded: "docs/INBOX.md has no defined format",
		needs:     []string{"BLOCKING", "FYI", "· <feature-id> ·"},
	}, {
		concluded: "kit next cannot see claims made in another clone",
		needs:     []string{"remote branches"},
	}, {
		concluded: "features have no defined order",
		needs:     []string{"Lowest ID first"},
	}, {
		concluded: "nothing exits non-zero when audit finds something HIGH",
		needs:     []string{"--strict"},
	}, {
		// The protocol is what an agent reads before it does anything, so the three
		// facts that make concurrency safe have to be in it and not only in the code.
		concluded: "a hand-made branch claims a feature as well as the tool does",
		needs:     []string{"kit claim"},
	}, {
		concluded: "a claimed checkout is a fine place to start the next feature",
		needs:     []string{"one feature per checkout."},
	}, {
		// The one class of work the kit coordinated with nothing. The protocol sent it
		// to a branch nothing pushed and nothing recorded, so two agents given the same
		// bug both started it and neither found out.
		concluded: "work that is not a feature is coordinated by nothing",
		needs:     []string{"kit chore", "records one line"},
	}, {
		concluded: "sequential IDs are handed out safely, so they cannot collide",
		needs:     []string{"Where two agents can still collide", "duplicate-id"},
	}, {
		// The recipe this used to look for is now a command, so the fact it carried
		// has to be in what the command says about itself.
		concluded: "deleting the local branch releases a claim",
		needs:     []string{"kit release", "because that ref is the claim"},
	}} {
		t.Run(tc.concluded, func(t *testing.T) {
			for _, need := range tc.needs {
				if !strings.Contains(all, need) {
					t.Errorf("the binary never says %q, so %q is still a fair reading of it",
						need, tc.concluded)
				}
			}
		})
	}
}

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
