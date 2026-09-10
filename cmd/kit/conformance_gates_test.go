package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFormatGateFailsWhenFormatterFails(t *testing.T) {
	t.Parallel()
	for _, path := range []string{makefilePath, "../../templates/stacks/go/Makefile"} {
		t.Run(path, func(t *testing.T) {
			p := newProject(t)
			p.write("Makefile", readOrFail(t, path))
			shim := t.TempDir()
			writeExecutable(t, filepath.Join(shim, "gofmt"), "#!/bin/sh\necho formatter-failed >&2\nexit 7\n")
			cmd := exec.Command("make", "fmt-check")
			cmd.Dir, cmd.Env = p.Root, append(os.Environ(), "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("formatter failure passed: %s", out)
			}
		})
	}
}

func TestOldQuestionDoesNotGateIndependentWork(t *testing.T) {
	t.Parallel()
	p := setUp(t)
	feature(p, "0001", "blocked", "Blocked", "Draft", "- [ ] Build")
	feature(p, "0002", "independent", "Independent", "Draft", "- [ ] Build")
	askOn(p, "docs/features/0001-blocked.md", daysAgo(365), "Human decision")
	p.commitAll("plans and question")
	p.run("audit", "--strict").ok().has("question-stale")
	p.run("claim", "0001").fails(1)
	p.run("claim", "0002").ok()
}
