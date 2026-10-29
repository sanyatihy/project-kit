package kit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The forges `kit init` can scaffold CI for.
const (
	ForgeGitHub = "github"
	ForgeGitLab = "gitlab"
)

// Forges is every forge --forge accepts, for validation and help.
var Forges = []string{ForgeGitHub, ForgeGitLab}

// DetectForge reports which forge a project is on, or "" when nothing says.
//
// Every rung wants positive evidence. Absence never decides: a project with no CI file,
// no remote and an uninformative path is unknown, and `init` writes no CI file rather
// than guessing — a missing pipeline is visible, a pipeline for the wrong forge is not.
//
// The rungs are ordered by what they actually prove. A hostname cannot tell a self-hosted
// GitLab from a Gitea, so the project's own CI file outranks it.
func DetectForge(root string) string {
	// 1. The project's own answer.
	if info, err := os.Stat(filepath.Join(root, ".gitlab-ci.yml")); err == nil && !info.IsDir() {
		return ForgeGitLab
	}
	if info, err := os.Stat(filepath.Join(root, ".github", "workflows")); err == nil && info.IsDir() {
		return ForgeGitHub
	}
	// 2. Whatever the remote is.
	for _, remote := range gitLines(root, "remote") {
		if f := forgeFromHost(hostFromRemoteURL(gitOut(root, "remote", "get-url", remote))); f != "" {
			return f
		}
	}
	// 3. The directory convention `ModulePath` already trusts for go.mod. This is the
	//    only rung that answers on a fresh `git init`, which is exactly the state where
	//    the alternative is stopping to ask a human.
	return forgeFromHost(hostSegment(ModulePath(root)))
}

// forgeFromHost names the forge a hostname belongs to, or "" when it names none.
// Recognition is the whole point: an unrecognised host is not a default, it is unknown.
func forgeFromHost(host string) string {
	host = strings.ToLower(host)
	switch {
	case host == "":
		return ""
	case strings.Contains(host, "gitlab"):
		return ForgeGitLab
	case strings.Contains(host, "github"):
		return ForgeGitHub
	}
	return ""
}

// hostSegment is the leading path segment of a module path when it looks like a host.
// `gitlab.example.com/group/thing` -> `gitlab.example.com`; a bare directory name -> "".
func hostSegment(modulePath string) string {
	first, _, found := strings.Cut(modulePath, "/")
	if !found || !strings.Contains(first, ".") {
		return ""
	}
	return first
}

// hostFromRemoteURL pulls the host out of either git URL form: scp-like
// `git@host:group/thing.git`, or a real URL `https://user@host:443/group/thing.git`.
func hostFromRemoteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if scheme := strings.Index(raw, "://"); scheme >= 0 {
		rest := raw[scheme+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			rest = rest[:slash]
		}
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		if colon := strings.Index(rest, ":"); colon >= 0 {
			rest = rest[:colon]
		}
		return rest
	}
	// scp-like. The first colon ends the host, so a path containing one cannot confuse it.
	host, _, found := strings.Cut(raw, ":")
	if !found {
		return ""
	}
	if at := strings.LastIndex(host, "@"); at >= 0 {
		host = host[at+1:]
	}
	return host
}

// gitOut runs git and returns stdout, empty on any failure. Detection is best-effort by
// construction: a repo with no remote is a normal thing, not an error to report.
func gitOut(dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := gitCommand(ctx, dir, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return stdout.String()
}

func gitLines(dir string, args ...string) []string {
	var out []string
	for _, line := range strings.Split(gitOut(dir, args...), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
