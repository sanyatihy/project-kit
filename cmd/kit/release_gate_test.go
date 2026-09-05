package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func releaseFixture(t *testing.T) (*project, string) {
	t.Helper()
	a, _ := twoAgents(t)
	a.write("Makefile", readOrFail(t, "../../Makefile"))
	a.write("scripts/release.sh", readOrFail(t, "../../scripts/release.sh"))
	a.commitAll("release tooling")
	a.git("push", "origin", "main")
	shim := t.TempDir()
	writeExecutable(t, filepath.Join(shim, "make"), "#!/bin/sh\necho \"$1\" >> .git/gates-log\n[ \"${FAIL_GATE:-}\" != \"$1\" ] || exit 19\nif [ \"${MOVE_HEAD:-}\" = 1 ] && [ \"$1\" = check ]; then git commit --allow-empty -m moved >/dev/null || exit 18; fi\n")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(shim, "git"), "#!/bin/sh\nif [ \"${FAIL_STATUS:-}\" = 1 ] && [ \"$1\" = status ]; then exit 17; fi\nif [ \"${FAIL_PUSH:-}\" = 1 ] && [ \"$1\" = push ]; then exit 16; fi\nexec "+shellQuote(realGit)+" \"$@\"\n")
	return a, shim
}

func releaseRun(t *testing.T, p *project, shim string, extra ...string) (string, error) {
	t.Helper()
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(makePath, "release", "TAG=v1.0.0")
	cmd.Dir = p.Root
	cmd.Env = gitEnv(append([]string{"PATH=" + shim + string(os.PathListSeparator) + os.Getenv("PATH")}, extra...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestReleaseRunsEveryGateAndRetriesTheSameTag(t *testing.T) {
	t.Parallel()
	p, shim := releaseFixture(t)
	if out, err := releaseRun(t, p, shim, "FAIL_PUSH=1"); err == nil {
		t.Fatal("failed push passed:", out)
	}
	tag := p.gitOut(nil, "rev-parse", "refs/tags/v1.0.0")
	if got := p.read(".git/gates-log"); got != "check\nvuln\naudit\n" {
		t.Fatal("incomplete release gate:", got)
	}
	if out, err := releaseRun(t, p, shim); err != nil {
		t.Fatal("retry failed:", out)
	}
	if remote := p.gitOut(nil, "ls-remote", "origin", "refs/tags/v1.0.0"); !strings.HasPrefix(remote, tag) {
		t.Fatal("retry changed tag identity:", remote)
	}
	if out, err := releaseRun(t, p, shim); err != nil {
		t.Fatal("already published retry failed:", out)
	}
}

func TestReleaseRejectsFailedGatesUnknownStateAndMovingHEAD(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"FAIL_GATE=check", "FAIL_GATE=vuln", "FAIL_GATE=audit", "FAIL_STATUS=1", "MOVE_HEAD=1"} {
		t.Run(env, func(t *testing.T) {
			p, shim := releaseFixture(t)
			if out, err := releaseRun(t, p, shim, env); err == nil {
				t.Fatal("unsafe release passed:", out)
			}
			if p.resolves("refs/tags/v1.0.0") {
				t.Fatal("failed validation created a release tag")
			}
		})
	}
}

func TestReleaseNeverOverwritesAConflictingTag(t *testing.T) {
	t.Parallel()
	for _, remote := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "remote"}[remote], func(t *testing.T) {
			p, shim := releaseFixture(t)
			p.git("tag", "-a", "v1.0.0", "HEAD^", "-m", "another release")
			old := p.gitOut(nil, "rev-parse", "refs/tags/v1.0.0")
			if remote {
				p.git("push", "origin", "refs/tags/v1.0.0")
				p.git("tag", "-d", "v1.0.0")
			}
			if out, err := releaseRun(t, p, shim); err == nil {
				t.Fatal("conflicting tag was accepted:", out)
			}
			if remote {
				if got := p.gitOut(nil, "ls-remote", "origin", "refs/tags/v1.0.0"); !strings.HasPrefix(got, old) {
					t.Fatal("remote tag moved:", got)
				}
			} else if got := p.gitOut(nil, "rev-parse", "refs/tags/v1.0.0"); got != old {
				t.Fatal("local tag moved")
			}
		})
	}
}

func TestReleaseVerifiesPublicationAfterALostPushAcknowledgement(t *testing.T) {
	t.Parallel()
	p, shim := releaseFixture(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(shim, "git"), "#!/bin/sh\nif [ \"$1\" = push ]; then\n"+shellQuote(realGit)+" \"$@\" >/dev/null 2>&1 || exit $?\nexit 1\nfi\nexec "+shellQuote(realGit)+" \"$@\"\n")
	if out, err := releaseRun(t, p, shim); err != nil {
		t.Fatal("accepted publication was reported as lost:", out)
	}
	tag := p.gitOut(nil, "rev-parse", "refs/tags/v1.0.0")
	if got := p.gitOut(nil, "ls-remote", "origin", "refs/tags/v1.0.0"); !strings.HasPrefix(got, tag) {
		t.Fatal("publication was not verified:", got)
	}
}
