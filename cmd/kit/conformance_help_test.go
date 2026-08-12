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
	p := newProject(t)

	// Everything the binary can tell someone, in one place.
	all := p.run("help").ok().Stdout
	for _, topic := range []string{"protocol", "template", "conventions", "checks",
		"inbox", "as-built", "go-checklist", "deployment"} {
		all += p.run("help", topic).ok().Stdout
	}
	for _, cmd := range subcommands {
		all += p.run(cmd, "--help").Stdout
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
		concluded: "features of equal status have no defined order",
		needs:     []string{"lowest ID first"},
	}, {
		concluded: "nothing exits non-zero when audit finds something HIGH",
		needs:     []string{"--strict"},
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
