package kit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	StatusDraft      = "Draft"
	StatusAccepted   = "Accepted"
	StatusInProgress = "In progress"
	StatusShipped    = "Shipped"

	gitLogLimit = "500"
	gitTimeout  = 30 * time.Second
)

// AllStatuses is every status a feature doc may declare.
var AllStatuses = []string{StatusAccepted, StatusDraft, "Dropped", StatusInProgress,
	StatusShipped, "Superseded"}

var notAFeature = map[string]bool{
	"TEMPLATE.md": true, "README.md": true, "BACKLOG.md": true, "INBOX.md": true,
}

var featureDirs = []string{"docs/features", "features", "doc/features"}
var techDocs = []string{"docs/TECH.md", "TECH.md", "docs/ARCHITECTURE.md", "ARCHITECTURE.md"}
var inboxes = []string{"docs/INBOX.md", "INBOX.md"}
var protocols = []string{"AGENTS.md", "CLAUDE.md"}

var idPrefixRe = regexp.MustCompile(`^(\d+)`)

// Feature is one feature document.
type Feature struct {
	Path     string
	Rel      string
	ID       string
	Title    string
	Status   string
	Shipped  time.Time
	Check    time.Time
	Checked  time.Time
	Modified time.Time

	sections map[string]string
}

// Name is the ID and title, for display.
func (f *Feature) Name() string {
	id := f.ID
	if id == "" {
		id = "????"
	}
	return id + " " + f.Title
}

// Section returns a section's trimmed content, empty when absent.
func (f *Feature) Section(heading string) string { return f.sections[heading] }

// OpenTasks lists unticked task boxes in document order.
func (f *Feature) OpenTasks() []string { return Unticked(f.Section("Tasks")) }

// NextTask is the one to do now. List order is the order of work.
func (f *Feature) NextTask() string {
	if tasks := f.OpenTasks(); len(tasks) > 0 {
		return tasks[0]
	}
	return ""
}

// FilenamePrefix is the leading number of the filename, for the id cross-check.
func (f *Feature) FilenamePrefix() string {
	if m := idPrefixRe.FindStringSubmatch(filepath.Base(f.Path)); m != nil {
		return m[1]
	}
	return ""
}

// Repo is a project on disk. Layout is discovered, so there is no config to drift.
type Repo struct {
	Root     string
	Name     string
	Features []*Feature

	FeaturesDir string // "" when the project is not set up
	Tech        string
	Inbox       string
	Protocol    string

	commits []commit
	refs    []string
	loaded  bool
}

type commit struct {
	When    time.Time
	Message string
}

// Open reads a project.
func Open(root string) (*Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	r := &Repo{Root: abs, Name: filepath.Base(abs)}
	r.FeaturesDir = r.firstDir(featureDirs)
	r.Tech = r.firstFile(techDocs)
	r.Inbox = r.firstFile(inboxes)
	r.Protocol = r.firstFile(protocols)
	r.loadFeatures()
	return r, nil
}

func (r *Repo) firstDir(rels []string) string {
	for _, rel := range rels {
		p := filepath.Join(r.Root, rel)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func (r *Repo) firstFile(rels []string) string {
	for _, rel := range rels {
		p := filepath.Join(r.Root, rel)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

func (r *Repo) loadFeatures() {
	if r.FeaturesDir == "" {
		return
	}
	entries, err := os.ReadDir(r.FeaturesDir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || notAFeature[e.Name()] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if f := r.readFeature(filepath.Join(r.FeaturesDir, name)); f != nil {
			r.Features = append(r.Features, f)
		}
	}
}

func (r *Repo) readFeature(path string) *Feature {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil // an unreadable file is a filesystem problem, not a doc problem
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	fields, body := ParseFrontmatter(Normalise(string(raw)))
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		rel = path
	}
	f := &Feature{
		Path:     path,
		Rel:      filepath.ToSlash(rel),
		ID:       fields["id"],
		Title:    fields["title"],
		Status:   fields["status"],
		Shipped:  ParseDate(fields["shipped"]),
		Check:    ParseDate(fields["check"]),
		Checked:  ParseDate(fields["checked"]),
		Modified: info.ModTime(),
		sections: ParseSections(body),
	}
	if f.Title == "" {
		f.Title = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return f
}

// Rel makes a path relative to the project root, for display.
func (r *Repo) Rel(path string) string {
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func (r *Repo) git(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.Root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (r *Repo) loadGit() {
	if r.loaded {
		return
	}
	r.loaded = true

	raw := r.git("log", "-n"+gitLogLimit, "--format=%cI\x1f%B\x1e")
	for _, chunk := range strings.Split(raw, "\x1e") {
		when, message, found := strings.Cut(strings.TrimSpace(chunk), "\x1f")
		if !found {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, when)
		if err != nil {
			continue
		}
		r.commits = append(r.commits, commit{When: parsed, Message: message})
	}

	seen := map[string]bool{}
	for _, args := range [][]string{
		{"branch", "--format=%(refname:short)"},
		{"branch", "-r", "--format=%(refname:short)"},
	} {
		for _, line := range strings.Split(r.git(args...), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				seen[line] = true
			}
		}
	}
	for _, line := range strings.Split(r.git("worktree", "list", "--porcelain"), "\n") {
		if strings.HasPrefix(line, "branch ") {
			parts := strings.Split(strings.TrimSpace(line), "/")
			seen[parts[len(parts)-1]] = true
		}
	}
	for name := range seen {
		r.refs = append(r.refs, name)
	}
	sort.Strings(r.refs)
}

// branchName drops a remote prefix, so origin/0009-thing and 0009-thing are
// recognised as one branch rather than two claims on the same feature.
func branchName(ref string) string {
	if i := strings.Index(ref, "/"); i >= 0 && (strings.HasPrefix(ref, "origin/") ||
		strings.HasPrefix(ref, "remotes/")) {
		return ref[i+1:]
	}
	return ref
}

func isTrunk(name string) bool {
	return name == "main" || name == "master" || name == "HEAD"
}

// Claims lists the distinct branches claiming a feature. More than one means two
// agents believe they own it.
func (r *Repo) Claims(id string) []string {
	r.loadGit()
	seen := map[string]bool{}
	var out []string
	for _, ref := range r.refs {
		name := branchName(ref)
		if isTrunk(name) || seen[name] || !Mentions(name, id) {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ClaimFor returns the branch claiming a feature, or "". The default branch
// never counts as a claim.
func (r *Repo) ClaimFor(id string) string {
	if claims := r.Claims(id); len(claims) > 0 {
		return claims[0]
	}
	return ""
}

// OrphanClaims lists branches named like a claim — a leading four-digit ID —
// whose feature doc does not exist.
//
// Only branches that look like claims are considered. Ordinary working branches
// cite no ID and must never be reported, or the check becomes noise.
func (r *Repo) OrphanClaims() []string {
	r.loadGit()
	known := map[string]bool{}
	for _, f := range r.Features {
		if f.ID != "" {
			known[f.ID] = true
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, ref := range r.refs {
		name := branchName(ref)
		if isTrunk(name) || seen[name] {
			continue
		}
		m := idPrefixRe.FindStringSubmatch(name)
		if m == nil || known[m[1]] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// LastMentioned is the newest commit citing this ID, or the zero time.
func (r *Repo) LastMentioned(id string) time.Time {
	r.loadGit()
	for _, c := range r.commits { // git log is newest-first
		if Mentions(c.Message, id) {
			return c.When
		}
	}
	return time.Time{}
}

// LastTouched is the last commit date for a path, falling back to mtime when
// untracked. A check that silently does nothing on an untracked file is worse
// than no check at all.
func (r *Repo) LastTouched(path string) time.Time {
	if out := strings.TrimSpace(r.git("log", "-1", "--format=%cI", "--", path)); out != "" {
		if parsed, err := time.Parse(time.RFC3339, out); err == nil {
			return parsed
		}
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// OpenQuestion is one unticked inbox line.
//
// Feature is the ID from the line's second field, empty when the line does not
// follow the documented format. Blocking decisions use it in preference to the
// whole line, so a question that merely mentions another feature in passing does
// not stop work on it.
type OpenQuestion struct {
	When    time.Time
	Text    string
	Feature string
	FYI     bool
}

// inboxSep separates the fields of an inbox line:
//
//   - [ ] YYYY-MM-DD · <feature-id> · BLOCKING|FYI · <question>
const inboxSep = "·"

// parseInboxLine pulls the feature ID and the BLOCKING/FYI marker out of a line
// in the documented format, reporting whether it was in that format at all.
//
// The question text is deliberately not validated: it is prose and may contain
// anything, including the separator.
func parseInboxLine(text string) (id string, fyi, ok bool) {
	fields := strings.SplitN(text, inboxSep, 4)
	if len(fields) < 3 {
		return "", false, false
	}
	if ParseDate(strings.TrimSpace(fields[0])).IsZero() {
		return "", false, false
	}
	marker := strings.TrimSpace(fields[2])
	switch {
	case strings.EqualFold(marker, "FYI"):
		fyi = true
	case strings.EqualFold(marker, "BLOCKING"):
		fyi = false
	default:
		return "", false, false
	}
	if id = strings.TrimSpace(fields[1]); id == "" {
		return "", false, false
	}
	return id, fyi, true
}

// OpenQuestions reads the inbox, skipping fenced examples.
func (r *Repo) OpenQuestions() []OpenQuestion {
	if r.Inbox == "" {
		return nil
	}
	raw, err := os.ReadFile(r.Inbox)
	if err != nil {
		return nil
	}
	var out []OpenQuestion
	for _, text := range Unticked(Normalise(string(raw))) {
		if text == "" {
			continue
		}
		q := OpenQuestion{When: ParseDate(text), Text: text}
		if id, fyi, ok := parseInboxLine(text); ok {
			q.Feature, q.FYI = id, fyi
		} else {
			q.FYI = isFYI(text)
		}
		out = append(out, q)
	}
	return out
}

// BlockersFor lists open questions that block this feature.
//
// A line marked FYI is a note, not a blocker — it is recorded so the human sees
// it, but work continues. Anything else blocks, because an unmarked question is
// more likely to be a real one than a courtesy note.
//
// A line in the documented format is matched on its feature-ID field alone.
// Anything else falls back to scanning the whole line, which over-blocks: a
// question that names the ID only in passing will stop work. That is the safe
// direction to be wrong in — over-blocking is visible and annoying, while missing
// a blocker sends an agent into work a human has not cleared. `inbox-unparsed`
// reports the lines that land here so they can be fixed rather than tolerated.
func (r *Repo) BlockersFor(id string) []string {
	var out []string
	for _, q := range r.OpenQuestions() {
		if q.FYI {
			continue
		}
		matched := Mentions(q.Text, id)
		if q.Feature != "" {
			matched = Mentions(q.Feature, id)
		}
		if matched {
			out = append(out, q.Text)
		}
	}
	return out
}

func isFYI(text string) bool {
	for _, field := range strings.Split(text, "·") {
		if strings.EqualFold(strings.TrimSpace(field), "FYI") {
			return true
		}
	}
	return false
}

// ActiveFeatures lists features open for work: In progress before Accepted,
// then by ID. That ordering is the whole WIP policy.
func (r *Repo) ActiveFeatures() []*Feature {
	rank := map[string]int{StatusInProgress: 0, StatusAccepted: 1}
	var out []*Feature
	for _, f := range r.Features {
		if _, ok := rank[f.Status]; ok {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rank[out[i].Status] != rank[out[j].Status] {
			return rank[out[i].Status] < rank[out[j].Status]
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ProjectRoot finds the repo root containing start, so commands work from any
// subdirectory. Falls back to start when there is no git.
func ProjectRoot(start string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", start, "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	return start
}

// HasHistory reports whether the repo has any commits, distinguishing a project
// that is under way from one that was only just created.
func (r *Repo) HasHistory() bool {
	r.loadGit()
	return len(r.commits) > 0
}
