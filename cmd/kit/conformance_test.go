package main

// The conformance suite drives the built binary the way a user does: build it once,
// run it against throwaway projects, and assert on stdout, stderr, exit codes, and
// the files it leaves behind. The other tests in this repo cover parsing primitives
// and doc creation; this covers the commands as they are actually invoked.
//
// This file is the harness, plus the smoke test that proves the harness itself
// works. The per-command suites live beside it.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	binDir    string // package-scoped temp dir, removed by TestMain
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

func TestMain(m *testing.M) { os.Exit(runSuite(m)) }

func runSuite(m *testing.M) int {
	dir, err := os.MkdirTemp("", "kit-conformance-")
	if err != nil {
		// The unit tests in this package do not need a binary, so let them run.
		// kitBinary reports this to whichever test actually needed one.
		buildErr = fmt.Errorf("temp dir for the built binary: %w", err)
		return m.Run()
	}
	binDir = dir
	// Nothing useful to do if the temp dir will not go away.
	defer func() { _ = os.RemoveAll(dir) }()
	return m.Run()
}

// kitBinary builds the CLI once for the whole package. Building per-test would
// dominate the suite's runtime; building in TestMain unconditionally would slow the
// unit tests that never invoke it.
func kitBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		if buildErr != nil {
			return
		}
		path := filepath.Join(binDir, "kit")
		if out, err := exec.Command("go", "build", "-o", path, ".").CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("go build: %w\n%s", err, out)
			return
		}
		binPath = path
	})
	if buildErr != nil {
		t.Fatalf("cannot run the conformance suite: %v", buildErr)
	}
	return binPath
}

// project is a throwaway directory to run the binary against.
type project struct {
	t    *testing.T
	Root string
}

func newProject(t *testing.T) *project {
	t.Helper()
	root := t.TempDir()
	// macOS hands out /var/folders/… which is a symlink to /private/var. The kit
	// reports resolved paths, so without this every path assertion fails for a
	// reason that has nothing to do with the kit.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return &project{t: t, Root: root}
}

// run invokes the binary inside the project and captures everything about it.
func (p *project) run(args ...string) *result {
	p.t.Helper()
	cmd := exec.Command(kitBinary(p.t), args...)
	cmd.Dir = p.Root
	// TERM=dumb keeps colour codes out of the assertions, and a git identity means
	// a helper that commits does not fail on a machine with none configured.
	cmd.Env = append(os.Environ(),
		"TERM=dumb",
		"GIT_AUTHOR_NAME=kit tests", "GIT_AUTHOR_EMAIL=kit@example.invalid",
		"GIT_COMMITTER_NAME=kit tests", "GIT_COMMITTER_EMAIL=kit@example.invalid",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			p.t.Fatalf("running %s: %v", "kit "+strings.Join(args, " "), err)
		}
		code = exit.ExitCode()
	}
	return &result{
		t: p.t, args: args,
		Stdout: stdout.String(), Stderr: stderr.String(), Code: code,
	}
}

// git runs a git command in the project, for setting up states `kit init` would
// otherwise create itself.
func (p *project) git(args ...string) {
	p.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = p.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		p.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// commitAll makes a commit, which branches need in order to exist: `git branch`
// reports nothing until there is something for a ref to point at, and a claim is a
// branch name.
func (p *project) commitAll(message string) {
	p.t.Helper()
	p.git("add", "-A")
	p.git("commit", "-q", "-m", message)
}

func (p *project) path(rel string) string { return filepath.Join(p.Root, rel) }

func (p *project) write(rel, content string) {
	p.t.Helper()
	full := p.path(rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		p.t.Fatalf("preparing %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		p.t.Fatalf("writing %s: %v", rel, err)
	}
}

func (p *project) read(rel string) string {
	p.t.Helper()
	raw, err := os.ReadFile(p.path(rel))
	if err != nil {
		p.t.Fatalf("reading %s: %v", rel, err)
	}
	return string(raw)
}

func (p *project) exists(rel string) bool {
	_, err := os.Stat(p.path(rel))
	return err == nil
}

// result is one invocation, and everything a test can assert about it.
type result struct {
	t              *testing.T
	args           []string
	Stdout, Stderr string
	Code           int
}

func (r *result) cmd() string { return "kit " + strings.Join(r.args, " ") }

// out is stdout and stderr together, for assertions that do not care which stream
// carried the text. Tests pinning the stdout/stderr split use outHas and errHas.
func (r *result) out() string { return r.Stdout + r.Stderr }

func (r *result) context() string {
	return fmt.Sprintf("\n--- stdout ---\n%s\n--- stderr ---\n%s", r.Stdout, r.Stderr)
}

// ok fails unless the command succeeded.
func (r *result) ok() *result {
	r.t.Helper()
	if r.Code != 0 {
		r.t.Fatalf("%s exited %d, want 0%s", r.cmd(), r.Code, r.context())
	}
	return r
}

// fails asserts an exact exit code. "It errored" is not a contract — a tool an agent
// scripts has to distinguish bad usage from a real finding.
func (r *result) fails(code int) *result {
	r.t.Helper()
	if r.Code != code {
		r.t.Fatalf("%s exited %d, want %d%s", r.cmd(), r.Code, code, r.context())
	}
	return r
}

func (r *result) has(want ...string) *result {
	r.t.Helper()
	for _, w := range want {
		if !strings.Contains(r.out(), w) {
			r.t.Errorf("%s never mentioned %q%s", r.cmd(), w, r.context())
		}
	}
	return r
}

func (r *result) lacks(unwanted ...string) *result {
	r.t.Helper()
	for _, u := range unwanted {
		if strings.Contains(r.out(), u) {
			r.t.Errorf("%s mentioned %q and should not have%s", r.cmd(), u, r.context())
		}
	}
	return r
}

func (r *result) outHas(want string) *result {
	r.t.Helper()
	if !strings.Contains(r.Stdout, want) {
		r.t.Errorf("%s: stdout lacks %q%s", r.cmd(), want, r.context())
	}
	return r
}

func (r *result) errHas(want string) *result {
	r.t.Helper()
	if !strings.Contains(r.Stderr, want) {
		r.t.Errorf("%s: stderr lacks %q%s", r.cmd(), want, r.context())
	}
	return r
}

// TestHarness exercises the harness itself: that the binary builds and runs, that
// exit codes and the two streams are captured separately, and that a project is a
// real directory the binary acts on. Everything else in the suite rests on this, so
// a failure here means the later suites are testing nothing.
func TestHarness(t *testing.T) {
	p := newProject(t)

	// Success goes to stdout, and the binary is the one we just built.
	p.run("version").ok().outHas("kit ")

	// Failure is a specific code with diagnostics on stderr — never stdout, which
	// has to stay pipe-clean.
	bad := p.run("nonesuch").fails(2)
	bad.errHas("unknown command")
	if strings.Contains(bad.Stdout, "unknown command") {
		t.Error("the error reached stdout; stdout is data and stderr is diagnostics")
	}

	// A fresh project is empty, and the binary sees it that way.
	p.run("next").ok().has("not set up yet")

	// File helpers reach the same directory the binary runs in.
	p.write("docs/INBOX.md", "# Inbox\n")
	if !p.exists("docs/INBOX.md") {
		t.Fatal("write did not land where exists looks")
	}
	if got := p.read("docs/INBOX.md"); got != "# Inbox\n" {
		t.Errorf("read returned %q, want the content write put there", got)
	}

	// git is available for setting up states init would otherwise create.
	p.git("init", "-q")
	if !p.exists(".git") {
		t.Error("the git helper did not produce a repository")
	}
}
