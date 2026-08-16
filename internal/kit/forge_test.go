package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostFromRemoteURL(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"https://gitlab.example.com/infra/thing.git", "gitlab.example.com"},
		{"https://user@gitlab.example.com:8443/infra/thing.git", "gitlab.example.com"},
		{"ssh://git@gitlab.example.com:22/infra/thing.git", "gitlab.example.com"},
		{"git@github.com:owner/thing.git", "github.com"},
		{"github.com:owner/thing.git", "github.com"},
		{"", ""},
		{"not-a-url", ""},
	}
	for _, c := range cases {
		if got := hostFromRemoteURL(c.raw); got != c.want {
			t.Errorf("hostFromRemoteURL(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestForgeFromHostNeedsRecognition(t *testing.T) {
	cases := []struct{ host, want string }{
		{"github.com", ForgeGitHub},
		{"gitlab.com", ForgeGitLab},
		{"gitlab.example.com", ForgeGitLab},
		{"GitLab.Example.Com", ForgeGitLab},
		{"github.enterprise.example", ForgeGitHub},
		// An unrecognised host is unknown, never a default. This is the line that stops
		// rung 3 writing a pipeline for a forge nobody named.
		{"git.internal.corp", ""},
		{"bitbucket.org", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := forgeFromHost(c.host); got != c.want {
			t.Errorf("forgeFromHost(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

func TestHostSegment(t *testing.T) {
	cases := []struct{ modulePath, want string }{
		{"gitlab.example.com/infra/thing", "gitlab.example.com"},
		{"github.com/owner/thing", "github.com"},
		{"thing", ""},          // ModulePath's bare-directory fallback
		{"internal/thing", ""}, // a leading segment with no dot is not a host
	}
	for _, c := range cases {
		if got := hostSegment(c.modulePath); got != c.want {
			t.Errorf("hostSegment(%q) = %q, want %q", c.modulePath, got, c.want)
		}
	}
}

// TestDetectForgeRungOneOutranksThePath checks the ordering that matters: a project
// carrying its own CI file is believed over anything a directory name suggests, because
// a hostname cannot tell a self-hosted GitLab from a Gitea and the file can.
func TestDetectForgeRungOneOutranksThePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "github.com", "owner", "thing")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitlab-ci.yml"), []byte("stages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectForge(root); got != ForgeGitLab {
		t.Errorf("DetectForge = %q, want %q — the project's own CI file outranks its path", got, ForgeGitLab)
	}
}

func TestDetectForgeReadsTheDirectoryConvention(t *testing.T) {
	root := filepath.Join(t.TempDir(), "gitlab.example.com", "infra", "thing")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := DetectForge(root); got != ForgeGitLab {
		t.Errorf("DetectForge = %q, want %q", got, ForgeGitLab)
	}
}

func TestDetectForgeAdmitsItDoesNotKnow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "thing")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := DetectForge(root); got != "" {
		t.Errorf("DetectForge = %q, want \"\" — nothing here names a forge", got)
	}
}
