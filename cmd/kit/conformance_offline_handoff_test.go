package main

import "testing"

func TestOfflineSubmissionDoesNotInventUnpushedWorkOrIntegration(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	a.run("claim", "0007").ok()
	w := a.claimWorktree("work/0007")
	w.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build it"))
	submitting(w, "0007").ok()
	w.git("push", "origin", "HEAD")
	for _, args := range [][]string{{"next", "--offline"}, {"next", "--offline", "--json"}} {
		w.run(args...).ok().has("unknown while offline").lacks("has local changes to submit", "is integrated", "is awaiting review")
	}
	if !w.plan("next", "--offline", "--json").Offline {
		t.Fatal("JSON did not label offline observations")
	}
	w.withGit("if [ \"$3\" = ls-remote ]; then exit 1; fi", "next").ok().has("unreadable").lacks("has local changes to submit")
}

func TestOfflineBoardCannotPromiseAFreeRemoteClaim(t *testing.T) {
	t.Parallel()
	a, _ := twoAgents(t)
	plan := a.plan("next", "--offline", "--json")
	if len(plan.Ready) != 0 || !plan.Offline {
		t.Fatalf("offline board advertised current ownership: %+v", plan)
	}
	a.run("next", "--offline").ok().has("unknown while offline").lacks("kit claim 0007")
}
