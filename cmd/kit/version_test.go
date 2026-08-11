package main

import "testing"

// The case that matters is `go install`: the Makefile's ldflags are not applied, so
// without the build-info fallback a tagged release reports "dev" and there is no way
// to tell which version is on your PATH.
func TestResolveVersionPrefersTheStampButFallsBackToBuildInfo(t *testing.T) {
	for _, tc := range []struct {
		name              string
		stamped, embedded string
		want              string
		why               string
	}{{
		name:    "go install of a tag",
		stamped: "dev", embedded: "v0.1.0", want: "v0.1.0",
		why: "a released binary must report its tag, not the compiled-in default",
	}, {
		name:    "go install @latest off an untagged branch",
		stamped: "dev", embedded: "v0.0.0-20260811123327-244cf617e28d",
		want: "v0.0.0-20260811123327-244cf617e28d",
		why:  "a pseudo-version still identifies the commit, which beats reporting nothing",
	}, {
		name:    "make build stamps ldflags",
		stamped: "v0.1.0-3-gabc1234-dirty", embedded: "v0.1.0", want: "v0.1.0-3-gabc1234-dirty",
		why: "git describe carries the commit and the dirty marker; the module version cannot",
	}, {
		name:    "plain go build",
		stamped: "dev", embedded: "(devel)", want: "dev",
		why: "(devel) is Go's placeholder and is noise to print at users",
	}, {
		name:    "no build info at all",
		stamped: "dev", embedded: "", want: "dev",
		why: "ReadBuildInfo can fail; the command must still print something",
	}, {
		name:    "empty stamp is not a version",
		stamped: "", embedded: "v0.1.0", want: "v0.1.0",
		why: "an ldflags typo yielding an empty string should not blank out the output",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveVersion(tc.stamped, tc.embedded); got != tc.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q — %s",
					tc.stamped, tc.embedded, got, tc.want, tc.why)
			}
		})
	}
}

// version() must never print an empty string: `kit version` is how an agent checks
// what it is running, and "kit " with nothing after it is worse than a wrong answer.
func TestVersionAlwaysReportsSomething(t *testing.T) {
	if got := version(); got == "" {
		t.Error("version() returned empty, so `kit version` would print a bare \"kit \"")
	}
}
