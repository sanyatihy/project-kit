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

// ProtocolTemplate is where the protocol is written, both inside the embedded assets and
// inside the one project that carries them. Named once so `audit` and the command that
// reads the embedded copy cannot look at two different files.
const ProtocolTemplate = "templates/AGENTS-section.md"

// stackLayout is per-stack scaffolding. Deployment files are deliberately absent:
// they rot fastest and are not needed to write code. Add them when the project
// actually has something to deploy.
var stackLayout = map[string][][2]string{
	"go": {
		{"Makefile", "Makefile"},
		{"gitignore", ".gitignore"},
		{"gomod", "go.mod"},
		{"main.go.tmpl", "cmd/{{name}}/main.go"},
		{"README.md", "README.md"},
	},
}

// ciLayout is the one part of a stack that depends on where the project is hosted.
// Nothing is written for a forge that is not in here, including the unknown one — a
// missing pipeline is visible and a pipeline for the wrong forge is not.
var ciLayout = map[string]map[string][2]string{
	"go": {
		ForgeGitHub: {"check.yml", ".github/workflows/check.yml"},
		ForgeGitLab: {"gitlab-ci.yml", ".gitlab-ci.yml"},
	},
}

var goVersionRe = regexp.MustCompile(`go(\d+\.\d+(?:\.\d+)?)`)

// Result records what an Init did, for reporting.
type Result struct {
	Created   []string
	Refreshed []string
	Kept      []string

	// Forge is the forge the CI file was written for, empty when nothing said which.
	// CINotWritten records that a stack with a pipeline got none for that reason, so
	// the caller can say so rather than leaving the omission silent.
	Forge        string
	CINotWritten bool
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
// A project carries only what it fills in: the documents `kit init` writes are
// `AGENTS.md` and, with a stack, the code layout. `docs/VISION.md` is the operator's to
// write and a stub would answer none of it; `docs/INBOX.md` and `docs/TECH.md` are made
// by the command that first needs them.
//
// Your files are never overwritten. The kit-owned protocol section inside
// AGENTS.md is refreshed, so projects pick up changes without losing their own
// rules.
// forge names where the project is hosted, deciding which CI file a stack gets. Empty
// means detect it; detection returning nothing means no CI file is written at all.
func Init(templates fs.FS, root, stack, forge string) (*Result, error) {
	res := &Result{}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, err
		}
		res.Created = append(res.Created, root+"/")
	}

	// Before anything is scaffolded, so the first rung reads the project's own CI file
	// rather than one this run just wrote.
	if forge == "" {
		forge = DetectForge(root)
	}
	res.Forge = forge

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
				root, values, res); err != nil {
				return nil, err
			}
		}
		if ci, ok := ciLayout[stack][forge]; ok {
			src := path.Join("templates/stacks", stack, ci[0])
			if err := place(templates, src, filepath.Join(root, ci[1]),
				root, values, res); err != nil {
				return nil, err
			}
		} else if len(ciLayout[stack]) > 0 {
			res.CINotWritten = true
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

	// The board is a directory listing, so `next`, `audit` and `new` all exit 2 until it
	// exists — and git drops an empty directory, so a clone of a project that has planned
	// nothing yet would arrive without one. Same rule as the stack's: a .gitkeep only
	// earns its place where git would otherwise drop the directory.
	featuresDir := FindFeaturesDir(root)
	if dir := filepath.Join(root, featuresDir); isEmptyDir(dir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
			return nil, err
		}
		res.Created = append(res.Created, featuresDir+"/.gitkeep")
	}

	if err := writeProtocol(templates, root, res); err != nil {
		return nil, err
	}
	return res, nil
}

// place writes one template, and keeps an existing destination. Nothing outside the
// protocol section of AGENTS.md is the kit's to rewrite: every file left here is one
// the project fills in, so its copy is the only copy there is.
func place(templates fs.FS, src, dest, root string, values map[string]string, res *Result) error {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		rel = dest
	}
	rel = filepath.ToSlash(rel)
	if _, err := os.Stat(dest); err == nil {
		res.Kept = append(res.Kept, rel)
		return nil
	}
	raw, err := fs.ReadFile(templates, src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dest, []byte(fill(string(raw), values)), 0o644); err != nil {
		return err
	}
	res.Created = append(res.Created, rel)
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
		// A title and the marker note, not the bare protocol. The file opened on an `##`
		// heading with no `#` above it and nothing saying the section is regenerated, so
		// the promise that house rules survive above the marker reached nobody, and the
		// first thing every agent read had no project name on it.
		header := "# " + filepath.Base(root) + "\n\n" +
			"House rules go here, above the marker comment below.\n\n" +
			"<!-- The section below is written by `kit init` from the kit's own template.\n" +
			"     Edits inside it are lost on the next run; put durable house rules above\n" +
			"     this comment. -->\n\n"
		if err := os.WriteFile(dest, []byte(header+protocol), 0o644); err != nil {
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

// ProtocolDrift is how a project's work-protocol section stands against the one the
// running binary would write.
type ProtocolDrift int

const (
	// ProtocolCurrent is no difference, or nothing that could be compared.
	ProtocolCurrent ProtocolDrift = iota
	// ProtocolBehind is the project following rules an older kit scaffolded.
	ProtocolBehind
	// ProtocolAhead is the project holding the newer protocol and the binary an older
	// copy of it.
	ProtocolAhead
)

// ProtocolState reports which side of a protocol difference is behind.
//
// Shared by `audit`, which reports it, and `next`, which warns about it. One binary on
// `$PATH` serves every project and only `kit init` rewrites `AGENTS.md`, so a project runs
// whatever rules it was scaffolded with and nothing inside it can tell — which is why the
// command every session starts with has to say so, not only the one nobody is told to run.
//
// Direction is reported because the repair is destructive in one direction: `kit init .`
// replaces the section with the binary's copy, so offering it to the side holding the
// newer text overwrites the newer protocol with the older one. It did that here, silently,
// mid-edit. The one project that can be ahead is the one carrying the template the binary
// embeds — everywhere else the section is kit-owned, written by `kit init`, and the file
// says in a comment that edits inside it are lost.
//
// `protocol` is passed in because internal/kit cannot embed a parent directory.
func ProtocolState(r *Repo, protocol string) ProtocolDrift {
	if r.Protocol == "" || protocol == "" {
		return ProtocolCurrent
	}
	raw, err := os.ReadFile(r.Protocol)
	if err != nil {
		return ProtocolCurrent
	}
	current := Normalise(string(raw))
	updated, ok := ReplaceSection(current, ProtocolMarker, protocol)
	if !ok || updated == current {
		return ProtocolCurrent
	}
	source, err := os.ReadFile(filepath.Join(r.Root, ProtocolTemplate))
	if err == nil && strings.TrimSpace(Normalise(string(source))) !=
		strings.TrimSpace(Normalise(protocol)) {
		return ProtocolAhead
	}
	return ProtocolBehind
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
