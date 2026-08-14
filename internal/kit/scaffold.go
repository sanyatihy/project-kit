package kit

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ProtocolMarker delimits the kit-owned section inside a project's AGENTS.md.
const ProtocolMarker = "## Work protocol"

// docLayout maps a template to its destination, relative to the project root.
//
// `owned` marks the files the kit refreshes on every run rather than writing once.
// The test is not who wrote it first but whether the project would notice its copy
// being replaced: nothing reads either of these — `kit new` writes from the embedded
// template, and the conventions doc is read by nobody — so an edit to them has no
// effect, and replacing them costs nothing while stopping them describing a format
// the tool no longer produces. Anything a project fills in is the opposite: its copy
// is the only copy, and it is never touched.
var docLayout = []struct {
	src, dest string
	owned     bool
}{
	{"templates/VISION.md", "docs/VISION.md", false},
	{"templates/features-README.md", "docs/features/README.md", true},
	{"templates/features-TEMPLATE.md", "docs/features/TEMPLATE.md", true},
	{"templates/BACKLOG.md", "docs/features/BACKLOG.md", false},
	{"templates/INBOX.md", "docs/INBOX.md", false},
}

// stackLayout is per-stack scaffolding. Deployment files are deliberately absent:
// they rot fastest and are not needed to write code. Add them when the project
// actually has something to deploy.
var stackLayout = map[string][][2]string{
	"go": {
		{"Makefile", "Makefile"},
		{"gitignore", ".gitignore"},
		{"gomod", "go.mod"},
		{"main.go.tmpl", "cmd/{{name}}/main.go"},
		{"check.yml", ".github/workflows/check.yml"},
		{"README.md", "README.md"},
	},
}

var goVersionRe = regexp.MustCompile(`go(\d+\.\d+(?:\.\d+)?)`)

// Result records what an Init did, for reporting.
type Result struct {
	Created   []string
	Refreshed []string
	Kept      []string
}

// Stacks lists the scaffolds available in the embedded templates.
func Stacks(templates fs.FS) []string {
	entries, err := fs.ReadDir(templates, "templates/stacks")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Init sets a project up, or fills in what an existing one is missing.
//
// Your files are never overwritten. The kit-owned protocol section inside
// AGENTS.md is refreshed, so projects pick up changes without losing their own
// rules.
func Init(templates fs.FS, root, stack string) (*Result, error) {
	res := &Result{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, err
		}
		res.Created = append(res.Created, root+"/")
	}

	// The model depends on git: branches are claims, and staleness reads the log.
	if !isGitRepo(root) {
		if err := run(root, "git", "init", "-q"); err == nil {
			res.Created = append(res.Created, "git repository")
		} else {
			fmt.Fprintln(os.Stderr, "kit init: warning — could not create a git repository;"+
				" claims and staleness will not work")
		}
	}

	if stack != "" {
		values := map[string]string{
			"name":      filepath.Base(root),
			"module":    ModulePath(root),
			"goversion": goVersion(),
		}
		// A project that already has an entry point must not get a second one
		// named after its directory. Scaffolding fills gaps; it never duplicates.
		entryExists := hasMainPackage(root)
		for _, pair := range stackLayout[stack] {
			if pair[0] == "main.go.tmpl" && entryExists {
				res.Kept = append(res.Kept, "cmd/ (already has an entry point)")
				continue
			}
			src := path.Join("templates/stacks", stack, pair[0])
			if err := place(templates, src, filepath.Join(root, fill(pair[1], values)),
				root, values, false, res); err != nil {
				return nil, err
			}
		}
		// .gitkeep only earns its place in a directory git would otherwise drop.
		if internal := filepath.Join(root, "internal"); isEmptyDir(internal) {
			keep := filepath.Join(internal, ".gitkeep")
			if err := os.MkdirAll(internal, 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(keep, nil, 0o644); err != nil {
				return nil, err
			}
			res.Created = append(res.Created, "internal/.gitkeep")
		}
	}

	featuresDir := FindFeaturesDir(root)
	for _, doc := range docLayout {
		dest := filepath.Join(root, strings.Replace(doc.dest, "docs/features", featuresDir, 1))
		if err := place(templates, doc.src, dest, root, nil, doc.owned, res); err != nil {
			return nil, err
		}
	}

	if err := writeProtocol(templates, root, res); err != nil {
		return nil, err
	}
	return res, nil
}

// place writes one template. An existing destination is kept, unless the kit owns
// the file outright, in which case it is refreshed so it cannot fall behind the tool.
func place(templates fs.FS, src, dest, root string, values map[string]string, owned bool, res *Result) error {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		rel = dest
	}
	rel = filepath.ToSlash(rel)
	existed := false
	if _, err := os.Stat(dest); err == nil {
		if !owned {
			res.Kept = append(res.Kept, rel)
			return nil
		}
		existed = true
	}
	raw, err := fs.ReadFile(templates, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	want := fill(string(raw), values)
	if existed {
		// A refresh that changes nothing is not a refresh. Reporting one on every
		// run would make the word meaningless in the output where it matters.
		if current, err := os.ReadFile(dest); err == nil && Normalise(string(current)) == Normalise(want) {
			res.Kept = append(res.Kept, rel)
			return nil
		}
	}
	if err := os.WriteFile(dest, []byte(want), 0o644); err != nil {
		return err
	}
	if existed {
		res.Refreshed = append(res.Refreshed, rel)
	} else {
		res.Created = append(res.Created, rel)
	}
	return nil
}

// writeProtocol creates AGENTS.md, appends the protocol to an existing one, or
// refreshes a stale protocol in place.
func writeProtocol(templates fs.FS, root string, res *Result) error {
	raw, err := fs.ReadFile(templates, "templates/AGENTS-section.md")
	if err != nil {
		return err
	}
	protocol := string(raw)
	dest := filepath.Join(root, "AGENTS.md")

	existing, err := os.ReadFile(dest)
	if os.IsNotExist(err) {
		if err := os.WriteFile(dest, []byte(protocol), 0o644); err != nil {
			return err
		}
		res.Created = append(res.Created, "AGENTS.md")
		return nil
	} else if err != nil {
		return err
	}

	current := Normalise(string(existing))
	if !strings.Contains(current, ProtocolMarker) {
		if err := os.WriteFile(dest, []byte(current+"\n"+protocol), 0o644); err != nil {
			return err
		}
		res.Created = append(res.Created, "AGENTS.md (work protocol appended)")
		return nil
	}

	updated, ok := ReplaceSection(current, ProtocolMarker, protocol)
	if !ok || updated == current {
		res.Kept = append(res.Kept, "AGENTS.md")
		return nil
	}
	if err := os.WriteFile(dest, []byte(updated), 0o644); err != nil {
		return err
	}
	res.Refreshed = append(res.Refreshed, "AGENTS.md (work protocol)")
	return nil
}

// ReplaceSection swaps one `## Heading` section, leaving every other section
// untouched. The protocol is kit-owned content inside a user-owned file, so it
// is refreshed rather than left to rot — but only that section.
func ReplaceSection(text, marker, replacement string) (string, bool) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == marker {
			start = i
			break
		}
	}
	if start == -1 {
		return text, false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	out := append([]string{}, lines[:start]...)
	out = append(out, strings.Split(strings.TrimSpace(replacement), "\n")...)
	out = append(out, "")
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n"), true
}

// hasMainPackage reports whether the project already has an entry point, at the
// root or under cmd/.
func hasMainPackage(root string) bool {
	if _, err := os.Stat(filepath.Join(root, "main.go")); err == nil {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(root, "cmd", "*", "main.go"))
	return len(matches) > 0
}

// isEmptyDir reports whether a path is absent or an empty directory — the only
// case where a .gitkeep does anything.
func isEmptyDir(path string) bool {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return true
	}
	return err == nil && len(entries) == 0
}

// FindFeaturesDir respects a layout the project already uses, so re-running
// never forks it.
func FindFeaturesDir(root string) string {
	for _, rel := range featureDirs {
		if info, err := os.Stat(filepath.Join(root, rel)); err == nil && info.IsDir() {
			return rel
		}
	}
	return "docs/features"
}

// ModulePath infers a Go module path from where the project sits on disk:
// ~/code/github.com/owner/thing -> github.com/owner/thing. Falls back to the
// bare directory name.
func ModulePath(root string) string {
	parts := strings.Split(filepath.ToSlash(root), "/")
	for i, part := range parts {
		if strings.Contains(part, ".") && i+2 <= len(parts)-1 {
			return strings.Join(parts[i:], "/")
		}
	}
	return filepath.Base(root)
}

func goVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "go", "version").Output()
	if err == nil {
		if m := goVersionRe.FindStringSubmatch(string(out)); m != nil {
			return m[1]
		}
	}
	return "1.21"
}

func fill(text string, values map[string]string) string {
	for key, value := range values {
		text = strings.ReplaceAll(text, "{{"+key+"}}", value)
	}
	return text
}

func isGitRepo(root string) bool {
	return run(root, "git", "rev-parse", "--git-dir") == nil
}

func run(dir, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, append([]string{"-C", dir}, args...)...)
	return cmd.Run()
}
