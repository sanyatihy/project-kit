package main

// Conformance: `kit init`. The guarantee that matters most is that it never
// destroys anything — it is run on populated repos, and repeatedly.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestInitWritesTheDocsLayout(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.run("init").ok()

	for _, want := range []string{
		"docs/VISION.md",
		"docs/INBOX.md",
		"docs/TECH.md",
		"docs/features/BACKLOG.md",
		"AGENTS.md",
	} {
		if !p.exists(want) {
			t.Errorf("init did not write %s", want)
		}
	}
	// The layout is every file a project gets, and a conventions doc nothing reads is
	// not one: `kit new` writes a doc from the embedded template, so a placed copy is a
	// file the project maintains for the tool.
	if p.exists("docs/features/TEMPLATE.md") {
		t.Error("init wrote docs/features/TEMPLATE.md, which nothing reads")
	}
	// The protocol is what makes the docs actionable; a layout without it is inert.
	agents := p.read("AGENTS.md")
	if !strings.Contains(agents, "## Work protocol") {
		t.Error("AGENTS.md has no work protocol section")
	}
	// And it opens on the project rather than on a bare `##`, with the marker that says
	// where house rules survive a refresh. The file used to start mid-document, so the
	// promise that everything above the marker is kept reached nobody.
	if !strings.HasPrefix(agents, "# "+filepath.Base(p.Root)+"\n") {
		t.Errorf("AGENTS.md does not open with the project's name:\n%.80s", agents)
	}
	if !strings.Contains(agents, "House rules go here") {
		t.Errorf("AGENTS.md never says where a project's own rules go:\n%s", agents)
	}
}

func TestInitLinksClaudeMDToTheProtocol(t *testing.T) {
	t.Parallel()
	// The kit reads a protocol from either name and wrote only the first, so an agent
	// that reads CLAUDE.md found nothing until somebody typed the link by hand.
	p := newProject(t)
	p.run("init").ok().has("CLAUDE.md")

	target, err := os.Readlink(p.path("CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md is not a symlink: %v", err)
	}
	if target != "AGENTS.md" {
		t.Errorf("CLAUDE.md points at %q, want AGENTS.md", target)
	}
	// A relative target, so the link survives the directory being moved or cloned.
	if !strings.Contains(p.read("CLAUDE.md"), "## Work protocol") {
		t.Error("reading CLAUDE.md does not reach the protocol")
	}
}

func TestInitKeepsAClaudeFileTheProjectWrote(t *testing.T) {
	t.Parallel()
	// Nothing the project wrote is overwritten, and a repo carrying its own CLAUDE.md is
	// the common case this must not touch.
	p := newProject(t)
	p.write("CLAUDE.md", "# House rules\n\nOurs, not the kit's.\n")

	p.run("init").ok()

	if got := p.read("CLAUDE.md"); !strings.Contains(got, "Ours, not the kit's.") {
		t.Errorf("init overwrote the project's own CLAUDE.md:\n%s", got)
	}
}

func TestInitWritesAGitignoreThatGuessesAtNoLanguage(t *testing.T) {
	t.Parallel()
	// A refusal expressed as a file: the first `git add -A` an agent runs commits
	// whatever the editor and the OS left behind, and `kit claim` pushes HEAD, so it
	// reaches the branch the next agent works in.
	p := newProject(t)
	p.run("init").ok()

	got := p.read(".gitignore")
	for _, want := range []string{".DS_Store", "Thumbs.db", ".idea/", ".vscode/", "*.swp"} {
		if !strings.Contains(got, want) {
			t.Errorf("the default .gitignore does not ignore %s:\n%s", want, got)
		}
	}
	// What it ignores is what is true of every project, not what this one will be built
	// in. A language, a framework or a cloud here is a guess at a project one command old.
	for _, unwanted := range []string{"vendor/", "go.work", "node_modules", "__pycache__",
		"tfstate", "terraform", "ansible", "vault"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("the default .gitignore names %s, which is a guess at the stack:\n%s",
				unwanted, got)
		}
	}
}

func TestInitStackGoReplacesTheDefaultGitignore(t *testing.T) {
	t.Parallel()
	// A stack's copy is the whole file, not an addition to the default: one file to
	// read. So it carries Go's lines and repeats the editor and OS ones.
	p := newProject(t)
	p.run("init", "--stack", "go").ok()

	got := p.read(".gitignore")
	for _, want := range []string{"vendor/", "go.work", "*.test", ".DS_Store", ".idea/"} {
		if !strings.Contains(got, want) {
			t.Errorf("the go .gitignore does not ignore %s:\n%s", want, got)
		}
	}
	// Infrastructure this stack scaffolds none of. Deployment is deliberately not
	// generated, so ignoring its output was fifteen lines about files that never appear.
	for _, unwanted := range []string{"terraform", "tfstate", "tfvars", "vault", "ansible"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("the go .gitignore still names %s, which the stack scaffolds none of:\n%s",
				unwanted, got)
		}
	}
	// And it is written from the stack once, not written twice and reported both ways.
	if n := strings.Count(p.run("init", "--stack", "go").ok().Stdout, ".gitignore"); n != 1 {
		t.Errorf("a re-run reported .gitignore %d times, want 1", n)
	}
}

func TestInitCreatesAGitRepoButLeavesAnExistingOneAlone(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// Every file the layout places, held against a project that already wrote it. This
	// is the guarantee `kit init` is run on populated repositories for: it is worth more
	// than any line the tool might add.
	mine := map[string]string{
		// A project that already states its vision is the common case for adopting the
		// kit, and it is the one file here nobody could reconstruct.
		"docs/VISION.md":           "# Why we are building this\n",
		"docs/INBOX.md":            "MINE — do not touch\n",
		"docs/TECH.md":             "# As built\n\nOne process, one database.\n",
		"docs/features/BACKLOG.md": "MINE — ideas\n",
		".gitignore":               "build/\n",
	}
	p := newProject(t)
	for path, want := range mine {
		p.write(path, want)
	}
	p.write("AGENTS.md", "# House rules\n\nKeep me.\n")

	p.run("init").ok()

	for path, want := range mine {
		if got := p.read(path); got != want {
			t.Errorf("init overwrote %s: %q", path, got)
		}
	}
	if !strings.Contains(p.read("AGENTS.md"), "Keep me.") {
		t.Error("init destroyed existing AGENTS.md content")
	}
}

func TestInitRewritesNothingItPlacedOnceTheProjectHasFilledItIn(t *testing.T) {
	t.Parallel()
	// Re-running is the documented way to pick up a protocol change, so the run that
	// does it lands on a project that has been worked in. Nothing outside the protocol
	// section of AGENTS.md is the kit's to rewrite: every file the layout places is one
	// the project fills in, and its copy is the only copy there is.
	p := newProject(t)
	p.run("init").ok()

	mine := map[string]string{
		"docs/VISION.md":           "MINE — what we are building\n",
		"docs/INBOX.md":            "- [ ] 2026-01-02 · 0001 · FYI · mine\n",
		"docs/TECH.md":             "MINE — one process, one database\n",
		"docs/features/BACKLOG.md": "MINE — ideas\n",
		".gitignore":               "build/\n",
	}
	for path, want := range mine {
		p.write(path, want)
	}

	second := p.run("init").ok()

	for path, want := range mine {
		if got := p.read(path); got != want {
			t.Errorf("a second init touched %s, which the project owns: %q", path, got)
		}
	}
	// A refresh reported on every run makes the word meaningless in the output where it
	// matters, and the only thing that can honestly be refreshed is the protocol.
	second.lacks("refreshed")
}

func TestInitIsIdempotent(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	if !strings.Contains(got, "take a feature with `kit claim <id>`") {
		t.Error("the refreshed section does not contain the current protocol")
	}
}

func TestInitStackGoScaffoldsTheCodeLayout(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	// --forge because a temp directory names no forge, and the CI file is the one part
	// of a stack that depends on where the project is hosted.
	p.run("init", "--stack", "go", "--forge", "github").ok()

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

func TestInitStackPythonScaffoldsTheCodeLayout(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.run("init", "widget", "--stack", "python").ok()

	for _, want := range []string{
		"widget/Makefile", "widget/.gitignore", "widget/pyproject.toml", "widget/README.md",
		"widget/src/widget/__init__.py", "widget/src/widget/__main__.py", "widget/tests/.gitkeep",
	} {
		if !p.exists(want) {
			t.Errorf("--stack python did not write %s", want)
		}
	}

	// The keep directory belongs to the stack that asked for it. Go's is internal/, and a
	// Python project getting one was the assumption `Init` used to carry.
	if p.exists("widget/internal/.gitkeep") {
		t.Error("--stack python wrote internal/.gitkeep, which is the go stack's")
	}

	pyproject := p.read("widget/pyproject.toml")
	for _, want := range []string{"requires-python", "strict = true", "-W error"} {
		if !strings.Contains(pyproject, want) {
			t.Errorf("pyproject.toml does not carry %q:\n%s", want, pyproject)
		}
	}
	if make := p.read("widget/Makefile"); !strings.Contains(make, "check: tools fmt-check lint type test") {
		t.Errorf("make check is not the same gate:\n%s", make)
	}

	for _, unwanted := range []string{"widget/terraform", "widget/Dockerfile", "widget/compose.yaml"} {
		if p.exists(unwanted) {
			t.Errorf("--stack python scaffolded %s, which is deliberately not generated", unwanted)
		}
	}
}

// A directory name is not an import name: `url-shortener` holds `url_shortener`, and a
// package directory named after the repository would not import at all.
func TestInitStackPythonNamesAnImportablePackage(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.run("init", "url-shortener", "--stack", "python").ok()

	if !p.exists("url-shortener/src/url_shortener/__main__.py") {
		t.Error("the package directory is not an importable name")
	}
	if got := p.read("url-shortener/pyproject.toml"); !strings.Contains(got, "src/url_shortener") {
		t.Errorf("pyproject.toml does not point at the package directory:\n%s", got)
	}
}

func TestInitStackPythonDoesNotAddASecondEntryPoint(t *testing.T) {
	t.Parallel()
	// The same guarantee the go stack has, read through the globs the stack declares
	// rather than through a filename in Init.
	p := newProject(t)
	p.write("src/thing/__main__.py", "raise SystemExit(0)\n")
	p.run("init", "--stack", "python").ok().has("already has an entry point")
	if p.read("src/thing/__main__.py") == "" {
		t.Error("the existing entry point was emptied")
	}
}

func TestInitStackPythonWritesOnePipelinePerForge(t *testing.T) {
	t.Parallel()
	gh := newProject(t)
	gh.run("init", "--stack", "python", "--forge", "github").ok()
	if !gh.exists(".github/workflows/check.yml") || gh.exists(".gitlab-ci.yml") {
		t.Error("--forge github did not write exactly the GitHub workflow")
	}
	if got := gh.read(".github/workflows/check.yml"); !strings.Contains(got, "make check") {
		t.Errorf("the workflow does not run the project's own gate:\n%s", got)
	}

	gl := newProject(t)
	gl.run("init", "--stack", "python", "--forge", "gitlab").ok()
	if !gl.exists(".gitlab-ci.yml") || gl.exists(".github/workflows/check.yml") {
		t.Error("--forge gitlab did not write exactly the GitLab pipeline")
	}

	// A missing pipeline is visible and one for the wrong forge is not, so nothing is
	// written when nothing says which forge this is — and the run says so.
	none := newProject(t)
	r := none.run("init", "--stack", "python").ok()
	if none.exists(".github/workflows/check.yml") || none.exists(".gitlab-ci.yml") {
		t.Error("guessed a forge when nothing named one")
	}
	r.has("no CI file")
}

func TestInitStackGoInfersTheModulePathFromDisk(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.write("github.com/acme/widget/.keep", "")
	p.run("init", "github.com/acme/widget", "--stack", "go").ok()

	gomod := p.read("github.com/acme/widget/go.mod")
	if !strings.Contains(gomod, "github.com/acme/widget") {
		t.Errorf("module path was not inferred from the directory layout:\n%s", gomod)
	}
}

func TestInitStackGoDoesNotAddASecondEntryPoint(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	p := newProject(t)
	r := p.run("init", "--stack", "brainfuck").fails(2)
	r.errHas("unknown stack")
	// The message has to say what is available, or the user is left guessing.
	r.errHas("go")
}

// The forge decides one file, and getting it wrong is silent: a GitHub workflow in a
// GitLab project never runs and nothing reports it. These four cases are the ladder.

func TestInitBelievesTheProjectsOwnCIFile(t *testing.T) {
	t.Parallel()
	// Rung 1. A hostname cannot tell a self-hosted GitLab from a Gitea; the CI file the
	// project already carries can, and it outranks everything below it.
	p := newProject(t)
	p.write(".gitlab-ci.yml", "stages: [check]\n")
	p.run("init", "--stack", "go").ok()

	if p.exists(".github/workflows/check.yml") {
		t.Error("wrote a GitHub workflow into a project carrying .gitlab-ci.yml")
	}
	// And it kept theirs: place() never overwrites a file the project already has.
	if got := p.read(".gitlab-ci.yml"); !strings.Contains(got, "stages: [check]") {
		t.Errorf("the project's own pipeline was replaced:\n%s", got)
	}
}

func TestInitReadsTheForgeFromTheRemote(t *testing.T) {
	t.Parallel()
	// Rung 2.
	p := newProject(t)
	p.git("init", "-q")
	p.git("remote", "add", "origin", "https://github.com/owner/thing.git")
	p.run("init", "--stack", "go").ok()

	if !p.exists(".github/workflows/check.yml") {
		t.Error("a github.com remote did not produce a workflow")
	}
}

func TestInitWritesNoCIFileWhenNothingNamesAForge(t *testing.T) {
	t.Parallel()
	// Rung 4, and the reason the whole ladder exists: a missing pipeline is visible,
	// and a pipeline for the wrong forge is not. The closing line has to give an agent
	// both ways out, because the forge is a discoverable fact and not a human's call.
	p := newProject(t)
	r := p.run("init", "--stack", "go").ok()

	if p.exists(".github/workflows/check.yml") || p.exists(".gitlab-ci.yml") {
		t.Error("guessed a forge when nothing named one")
	}
	r.has("no CI file")
	r.has("git remote add")
	r.has("--forge")
}

func TestInitForgeOverridesDetection(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	p.git("init", "-q")
	p.git("remote", "add", "origin", "https://github.com/owner/thing.git")
	p.run("init", "--stack", "go", "--forge", "gitlab").ok()

	if !p.exists(".gitlab-ci.yml") {
		t.Error("--forge did not override the remote")
	}
	if p.exists(".github/workflows/check.yml") {
		t.Error("--forge gitlab still wrote a GitHub workflow")
	}
}

func TestInitRejectsAnUnknownForge(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	r := p.run("init", "--forge", "bitbucket").fails(2)
	r.errHas("unknown forge")
	r.errHas("gitlab")
}

func TestInitReservesTheIDsTheDocsAlreadyHold(t *testing.T) {
	t.Parallel()
	// A doc written before `kit new` reserved anything holds a number nothing recorded,
	// so deleting it would hand that number out a second time. Re-running init is the
	// one moment every project that predates the reservations passes through, which is
	// why nothing reports an unreserved id as a finding.
	p, _ := twoAgents(t)
	if got := p.reservedIDs(); len(got) != 0 {
		t.Fatalf("the fixture already reserves %v, so this proves nothing", got)
	}

	p.run("init").ok().has("reserved   1 id")

	if got := p.reservedIDs(); !slices.Contains(got, "0007") {
		t.Fatalf("the remote holds %v, none of which reserves the 0007 already in use", got)
	}
	// And the number is now out of circulation, however the doc leaves.
	if err := os.Remove(p.path("docs/features/0007-widget.md")); err != nil {
		t.Fatalf("deleting the doc: %v", err)
	}
	p.run("new", "Widget again").ok()
	if p.exists("docs/features/0007-widget-again.md") {
		t.Error("0007 was handed out again after its doc was deleted")
	}
}

func TestInitReservesNothingTwiceAndNothingWithoutARemote(t *testing.T) {
	t.Parallel()
	// Idempotent, like the rest of init: the second run has nothing left to reserve and
	// says nothing. And a project with no remote has nowhere to reserve anything, which
	// must not turn a scaffold into a failure.
	p, _ := twoAgents(t)
	p.run("init").ok().has("reserved")
	p.run("init").ok().lacks("reserved")

	local := setUp(t)
	local.run("new", "Alone").ok()
	local.run("init").ok().lacks("reserved")
}
