package main

import (
	"strings"
	"testing"
)

func TestUpgradePreservesLegacyWorkHistoryAndCustomizedFiles(t *testing.T) {
	t.Parallel()
	a, b := twoAgents(t)
	frozen := "docs/features/0001-history.md"
	a.write(frozen, doc("0001", "History", "Shipped", "shipped: 2026-01-01", "- [x] Delivered"))
	a.write("Makefile", "# Custom project gate\ncheck:\n\t@echo custom\n")
	a.write(".github/workflows/custom.yml", "name: preserve-my-pipeline\n")
	a.write("app.txt", "existing application\n")
	a.write("AGENTS.md", "# House rules\n\nKeep `## Work protocol` references and this rule.\n\n## Work protocol\n\nOld rules.\n")
	a.commitAll("legacy project with custom setup")
	a.git("push", "origin", "main")
	a.git("checkout", "-b", "0007-widget")
	a.write("in-flight.txt", "work awaiting review\n")
	a.commitAll("legacy implementation")
	a.git("push", "-u", "origin", "HEAD")
	before := map[string]string{}
	for _, path := range []string{frozen, "Makefile", ".github/workflows/custom.yml", "app.txt", "in-flight.txt", "docs/features/0007-widget.md"} {
		before[path] = a.read(path)
	}
	a.run("init", "--stack", "go").ok()
	for path, want := range before {
		if got := a.read(path); got != want {
			t.Errorf("upgrade changed %s", path)
		}
	}
	if !strings.HasPrefix(a.read("AGENTS.md"), "# House rules\n\nKeep `## Work protocol` references and this rule.\n\n") {
		t.Fatal("upgrade lost house rules")
	}
	a.commitAll("upgrade project guidance")
	a.run("audit", "--strict").ok()
	b.git("pull", "--ff-only", "origin", "main")
	b.run("claim", "0007").fails(1)
	a.write("docs/features/0007-widget.md", doc("0007", "Widget", "Draft", "", "- [x] Build it"))
	a.commitAll("finish legacy work")
	a.run("ship", "0007").ok()
	a.run("reset", "0007", "review feedback").ok()
	if a.read(frozen) != before[frozen] {
		t.Fatal("review migration rewrote integrated legacy history")
	}
}
