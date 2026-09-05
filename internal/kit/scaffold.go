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

// docLayout maps a template to its destination, relative to the project root.
//
// This is the whole document layout, and no other command creates a project file: one
// command, one answer to where a file came from. Every entry is a file the project fills
// in, which is why none is ever refreshed — its copy is the only copy there is.
// `docs/features/TEMPLATE.md` is not here: `kit new` writes a doc from the embedded
// template and nothing reads the placed copy, so it was a file a project maintained for
// the tool.
var docLayout = []struct{ src, dest string }{
	{"templates/VISION.md", "docs/VISION.md"},
	{"templates/INBOX.md", "docs/INBOX.md"},
	{"templates/TECH.md", "docs/TECH.md"},
	{"templates/BACKLOG.md", "docs/features/BACKLOG.md"},
}

// stack is one scaffold, and everything `Init` needs to know about it. `Init` knew two
// things about Go directly — that `main.go.tmpl` writes the entry point and that
// `internal/` wants a `.gitkeep` — and a second stack is what turns those from conditions
// in the code into data beside the templates.
type stack struct {
	// files are template-to-destination pairs, the destination filled from the project's
	// values.
	files [][2]string
	// entry is the template in files that writes an entry point, and entryGlobs the paths
	// that mean the project has one already. Scaffolding fills gaps and never writes a
	// second entry point over somebody's first. Empty entry means the stack has none.
	entry      string
	entryGlobs []string
	// keep is a directory the stack wants to exist even while it is empty, which git
	// otherwise drops. Empty means no such directory.
	keep string
}

// stackLayout is per-stack scaffolding. Deployment files are deliberately absent:
// they rot fastest and are not needed to write code. Add them when the project
// actually has something to deploy.
var stackLayout = map[string]stack{
	"go": {
		files: [][2]string{
			{"Makefile", "Makefile"},
			{"gomod", "go.mod"},
			{"golangci.yml", ".golangci.yml"},
			{"main.go.tmpl", "cmd/{{name}}/main.go"},
			{"README.md", "README.md"},
		},
		entry:      "main.go.tmpl",
		entryGlobs: []string{"main.go", "cmd/*/main.go"},
		keep:       "internal",
	},
	// The package directory is `{{pkg}}` and not `{{name}}`: a project called
	// `url-shortener` is not importable under that name.
	"python": {
		files: [][2]string{
			{"Makefile", "Makefile"},
			{"pyproject.toml", "pyproject.toml"},
			{"init.py", "src/{{pkg}}/__init__.py"},
			{"main.py", "src/{{pkg}}/__main__.py"},
			{"README.md", "README.md"},
		},
		entry:      "main.py",
		entryGlobs: []string{"main.py", "src/*/__main__.py"},
		keep:       "tests",
	},
}

// ignoreLayout is the .gitignore a stack writes instead of the default. Instead, not as
// well: one file to read beats two, at the price of repeating the editor and OS lines in
// every stack — the trade worth taking while there is one stack.
var ignoreLayout = map[string]string{
	"go":     "templates/stacks/go/gitignore",
	"python": "templates/stacks/python/gitignore",
}

// ciLayout is the one part of a stack that depends on where the project is hosted.
// Nothing is written for a forge that is not in here, including the unknown one — a
// missing pipeline is visible and a pipeline for the wrong forge is not.
var ciLayout = map[string]map[string][2]string{
	"go": {
		ForgeGitHub: {"check.yml", ".github/workflows/check.yml"},
		ForgeGitLab: {"gitlab-ci.yml", ".gitlab-ci.yml"},
	},
	"python": {
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
			"pkg":       PackageName(filepath.Base(root)),
			"module":    ModulePath(root),
			"goversion": goVersion(),
		}
		// A project that already has an entry point must not get a second one
		// named after its directory. Scaffolding fills gaps; it never duplicates.
		sc := stackLayout[stack]
		entryExists := sc.entry != "" && hasEntryPoint(root, sc.entryGlobs)
		for _, pair := range sc.files {
			if pair[0] == sc.entry && entryExists {
				res.Kept = append(res.Kept, entryDir(pair[1])+" (already has an entry point)")
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
		// .gitkeep only earns its place in a directory git would otherwise drop, and
		// only where the stack asked for one.
		if dir := filepath.Join(root, sc.keep); sc.keep != "" && isEmptyDir(dir) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
				return nil, err
			}
			res.Created = append(res.Created, sc.keep+"/.gitkeep")
		}
	}

	// A .gitignore is a refusal expressed as a file. Without one the first `git add -A`
	// an agent runs commits whatever the editor and the operating system left behind —
	// and `kit claim` pushes HEAD, so it reaches the branch the next agent works in. The
	// default ignores what is true of every project and never guesses at the language.
	if err := place(templates, ignoreFor(stack), filepath.Join(root, ".gitignore"), root,
		map[string]string{"name": filepath.Base(root)}, res); err != nil {
		return nil, err
	}

	featuresDir := FindFeaturesDir(root)
	for _, doc := range docLayout {
		dest := filepath.Join(root, strings.Replace(doc.dest, "docs/features", featuresDir, 1))
		if err := place(templates, doc.src, dest, root, nil, res); err != nil {
			return nil, err
		}
	}

	if err := writeProtocol(templates, root, res); err != nil {
		return nil, err
	}
	linkProtocolAlias(root, res)
	return res, nil
}

// linkProtocolAlias points CLAUDE.md at AGENTS.md, because the kit reads a protocol from
// either name and wrote only the first: an agent that reads the second found nothing
// unless somebody typed the link by hand, which two projects did.
//
// A symlink and not a copy — one git object, so the two names cannot drift apart. The cost
// is Windows, where a checkout with core.symlinks off holds the string "AGENTS.md" as text
// and the create itself needs a privilege; that is a warning rather than an error, because
// the protocol is in AGENTS.md either way. Lstat and not Stat, so a link whose target is
// missing counts as somebody's file and is kept.
func linkProtocolAlias(root string, res *Result) {
	alias := filepath.Join(root, "CLAUDE.md")
	if _, err := os.Lstat(alias); err == nil {
		res.Kept = append(res.Kept, "CLAUDE.md")
		return
	}
	if err := os.Symlink("AGENTS.md", alias); err != nil {
		fmt.Fprintf(os.Stderr, "kit init: warning — could not link CLAUDE.md to AGENTS.md: %v\n"+
			"an agent reading that name finds no protocol; AGENTS.md carries it\n", err)
		return
	}
	res.Created = append(res.Created, "CLAUDE.md -> AGENTS.md")
}

// ignoreFor names the .gitignore template a stack gets, and the default for no stack.
func ignoreFor(stack string) string {
	if src, ok := ignoreLayout[stack]; ok {
		return src
	}
	return "templates/gitignore"
}

// place writes one template, and keeps an existing destination. Nothing outside the
// protocol section of AGENTS.md is the kit's to rewrite: every file it places is one the
// project fills in, so its copy is the only copy there is.
func place(templates fs.FS, src, dest, root string, values map[string]string, res *Result) error {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		rel = dest
	}
	rel = filepath.ToSlash(rel)
	if _, err := os.Lstat(dest); err == nil {
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
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		res.Kept = append(res.Kept, rel)
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString(fill(string(raw), values))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
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

// hasEntryPoint reports whether the project already has one, by the globs its stack
// declares. Nothing here names a language.
func hasEntryPoint(root string, globs []string) bool {
	for _, glob := range globs {
		matches, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(glob)))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// entryDir is the directory an entry point is reported under when the project already has
// one: the first segment of the destination, so the message names `cmd/` or `src/` without
// the placeholder the rest of the path carries.
func entryDir(dest string) string {
	if i := strings.Index(dest, "/"); i > 0 {
		return dest[:i+1]
	}
	return dest
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

// PackageName is the project name as an importable identifier: a directory called
// `url-shortener` holds a package called `url_shortener`, because the hyphen a repository
// name carries is not one an import statement can.
func PackageName(name string) string {
	pkg := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		}
		return '_'
	}, name)
	// An identifier cannot start with a digit, and a leading underscore would read as
	// private to every tool that looks at it.
	if pkg == "" || (pkg[0] >= '0' && pkg[0] <= '9') {
		pkg = "pkg_" + pkg
	}
	return pkg
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
