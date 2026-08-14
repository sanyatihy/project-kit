package main

// Conformance: `kit init`. The guarantee that matters most is that it never
// destroys anything — it is run on populated repos, and repeatedly.

import (
	"strings"
	"testing"
)

func TestInitWritesTheDocsLayout(t *testing.T) {
	p := newProject(t)
	p.run("init").ok()

	for _, want := range []string{
		"docs/VISION.md",
		"docs/features/README.md",
		"docs/features/TEMPLATE.md",
		"docs/features/BACKLOG.md",
		"docs/INBOX.md",
		"AGENTS.md",
	} {
		if !p.exists(want) {
			t.Errorf("init did not write %s", want)
		}
	}
	// The protocol is what makes the docs actionable; a layout without it is inert.
	if !strings.Contains(p.read("AGENTS.md"), "## Work protocol") {
		t.Error("AGENTS.md has no work protocol section")
	}
}

func TestInitCreatesAGitRepoButLeavesAnExistingOneAlone(t *testing.T) {
	// The model depends on git: branches are claims and staleness reads the log.
	fresh := newProject(t)
	fresh.run("init").ok().has("git repository")
	if !fresh.exists(".git") {
		t.Error("init did not create a git repository")
	}

	already := newProject(t)
	already.git("init", "-q")
	already.run("init").ok().lacks("created    git repository")
}

func TestInitNeverOverwrites(t *testing.T) {
	p := newProject(t)
	p.write("docs/INBOX.md", "MINE — do not touch\n")
	// A project that already states its vision is the common case for adopting the
	// kit, and it is the one file here nobody could reconstruct.
	p.write("docs/VISION.md", "# Why we are building this\n")
	p.write("AGENTS.md", "# House rules\n\nKeep me.\n")

	p.run("init").ok()

	if got := p.read("docs/INBOX.md"); got != "MINE — do not touch\n" {
		t.Errorf("init overwrote docs/INBOX.md: %q", got)
	}
	if got := p.read("docs/VISION.md"); got != "# Why we are building this\n" {
		t.Errorf("init overwrote docs/VISION.md: %q", got)
	}
	if !strings.Contains(p.read("AGENTS.md"), "Keep me.") {
		t.Error("init destroyed existing AGENTS.md content")
	}
}

func TestInitRefreshesTheDocsItOwnsAndNothingElse(t *testing.T) {
	// The line this pins: a file the kit owns is one whose project copy nothing
	// reads, so replacing it takes nothing away. Everything a project fills in is
	// the only copy there is and must survive every re-run forever.
	p := newProject(t)
	p.run("init").ok()

	// A project scaffolded by an older kit: its reference doc predates a format change.
	p.write("docs/features/TEMPLATE.md", "# an older format\n")
	// And its own documents, which are the whole point of the distinction.
	p.write("docs/VISION.md", "MINE — what we are building\n")
	p.write("docs/INBOX.md", "- [ ] 2026-01-02 · 0001 · FYI · mine\n")
	p.write("docs/features/BACKLOG.md", "MINE — ideas\n")

	second := p.run("init").ok()

	if got := p.read("docs/features/TEMPLATE.md"); !strings.Contains(got, "needs:") {
		t.Error("init did not refresh the template it owns")
	}
	second.has("refreshed", "TEMPLATE.md")

	for _, mine := range []struct{ path, want string }{
		{"docs/VISION.md", "MINE — what we are building\n"},
		{"docs/INBOX.md", "- [ ] 2026-01-02 · 0001 · FYI · mine\n"},
		{"docs/features/BACKLOG.md", "MINE — ideas\n"},
	} {
		if got := p.read(mine.path); got != mine.want {
			t.Errorf("init touched %s, which the project owns: %q", mine.path, got)
		}
	}
}

func TestInitDoesNotClaimToRefreshWhatItDidNotChange(t *testing.T) {
	// A refresh reported on every run makes the word meaningless in the output
	// where it matters.
	p := newProject(t)
	p.run("init").ok()
	p.run("init").ok().lacks("refreshed  docs/features/TEMPLATE.md")
}

func TestInitIsIdempotent(t *testing.T) {
	// Re-running is the documented way to pick up protocol changes, so a second
	// run must report keeping things rather than creating them again.
	p := newProject(t)
	p.run("init").ok()
	second := p.run("init").ok()

	second.has("kept")
	if strings.Contains(second.Stdout, "created    docs/INBOX.md") {
		t.Error("a second init claimed to create docs/INBOX.md again")
	}
}

func TestInitAppendsTheProtocolToAnExistingAgentsFile(t *testing.T) {
	p := newProject(t)
	p.write("AGENTS.md", "# House rules\n\nNever deploy on Friday.\n")

	p.run("init").ok().has("appended")

	got := p.read("AGENTS.md")
	if !strings.Contains(got, "Never deploy on Friday.") {
		t.Error("appending the protocol lost the project's own rules")
	}
	if !strings.Contains(got, "## Work protocol") {
		t.Error("the protocol was not appended")
	}
}

func TestInitRefreshesOnlyTheProtocolSection(t *testing.T) {
	// The protocol is kit-owned content inside a user-owned file. Everything
	// outside that one section belongs to the project and must survive.
	p := newProject(t)
	p.write("AGENTS.md", `# House rules

Above the section.

## Work protocol

STALE TEXT THAT MUST GO

## House conventions

Below the section.
`)

	p.run("init").ok().has("refreshed")

	got := p.read("AGENTS.md")
	if strings.Contains(got, "STALE TEXT THAT MUST GO") {
		t.Error("the stale protocol survived the refresh")
	}
	for _, keep := range []string{"Above the section.", "## House conventions", "Below the section."} {
		if !strings.Contains(got, keep) {
			t.Errorf("refreshing the protocol destroyed %q, which is not kit-owned", keep)
		}
	}
	if !strings.Contains(got, "One agent per feature") {
		t.Error("the refreshed section does not contain the current protocol")
	}
}

func TestInitRespectsAnExistingFeaturesLayout(t *testing.T) {
	// A project already using features/ must not end up with both that and
	// docs/features/ — re-running would otherwise fork the board in two.
	p := newProject(t)
	p.write("features/0001-existing.md", "---\nid: 0001\n---\n")

	p.run("init").ok()

	if p.exists("docs/features/README.md") {
		t.Error("init forked the board: the project uses features/, not docs/features/")
	}
	if !p.exists("features/README.md") {
		t.Error("init did not fill in the layout the project already uses")
	}
}

func TestInitStackGoScaffoldsTheCodeLayout(t *testing.T) {
	p := newProject(t)
	p.run("init", "--stack", "go").ok()

	for _, want := range []string{
		"Makefile", ".gitignore", "go.mod", "README.md",
		".github/workflows/check.yml", "internal/.gitkeep",
	} {
		if !p.exists(want) {
			t.Errorf("--stack go did not write %s", want)
		}
	}

	gomod := p.read("go.mod")
	if !strings.Contains(gomod, "module ") {
		t.Errorf("go.mod has no module line:\n%s", gomod)
	}
	if !strings.Contains(gomod, "go 1.") {
		t.Errorf("go.mod pins no Go version:\n%s", gomod)
	}

	// Deployment is deliberately not scaffolded: it rots fastest and is not needed
	// to write code.
	for _, unwanted := range []string{"terraform", "ansible", "Dockerfile", "compose.yaml"} {
		if p.exists(unwanted) {
			t.Errorf("--stack go scaffolded %s, which is deliberately not generated", unwanted)
		}
	}
}

func TestInitStackGoInfersTheModulePathFromDisk(t *testing.T) {
	p := newProject(t)
	p.write("github.com/acme/widget/.keep", "")
	p.run("init", "github.com/acme/widget", "--stack", "go").ok()

	gomod := p.read("github.com/acme/widget/go.mod")
	if !strings.Contains(gomod, "github.com/acme/widget") {
		t.Errorf("module path was not inferred from the directory layout:\n%s", gomod)
	}
}

func TestInitStackGoDoesNotAddASecondEntryPoint(t *testing.T) {
	// Scaffolding fills gaps; it never duplicates. A project with an entry point
	// must not gain a second one named after its directory.
	atRoot := newProject(t)
	atRoot.write("main.go", "package main\n\nfunc main() {}\n")
	atRoot.run("init", "--stack", "go").ok().has("already has an entry point")

	underCmd := newProject(t)
	underCmd.write("cmd/mytool/main.go", "package main\n\nfunc main() {}\n")
	underCmd.run("init", "--stack", "go").ok().has("already has an entry point")
	if underCmd.read("cmd/mytool/main.go") == "" {
		t.Error("the existing entry point was emptied")
	}
}

func TestInitRejectsAnUnknownStack(t *testing.T) {
	p := newProject(t)
	r := p.run("init", "--stack", "brainfuck").fails(2)
	r.errHas("unknown stack")
	// The message has to say what is available, or the user is left guessing.
	r.errHas("go")
}
