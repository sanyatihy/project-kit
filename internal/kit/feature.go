package kit

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The feature document: its fields, its sections, and the branch name it implies.

// AllStatuses is every status a feature doc may declare.
//
// Three, and no more. There is no "In progress" because the claim branch says work has
// started, and no "Accepted" because in 82 docs across five projects nothing ever reached
// it: every feature went from Draft to Shipped. A doc carrying a status this does not
// list is reported as `bad-status`, which is the migration for both.
var AllStatuses = []string{StatusDraft, StatusDropped, StatusShipped}

// terminal reports whether a status is an end state, after which a branch bearing the
// feature's name is a merged leftover rather than somebody's work in progress.
func terminal(status string) bool {
	return status == StatusShipped || status == StatusDropped
}

var notAFeature = map[string]bool{
	"TEMPLATE.md": true, "README.md": true, "BACKLOG.md": true, "INBOX.md": true,
}

var idPrefixRe = regexp.MustCompile(`^(\d+)`)

// Feature is one feature document.
type Feature struct {
	Path    string
	Rel     string
	ID      string
	Title   string
	Status  string
	Needs   []string // feature ids that must ship before this can start
	Shipped time.Time

	// shippedText is the frontmatter line as written. Shipped alone cannot tell a doc
	// with no date from one whose date nothing could parse, and `shipped-no-date`
	// reported the second as the first — on a doc that plainly had one.
	shippedText string

	// Unreadable marks a doc the kit could not open. Everything above it then comes from
	// the filename, and nothing below it exists — so `audit` reports the file and asks no
	// other question of it, because the answers would all be about an empty struct.
	Unreadable bool

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

// PlannedSections are what an agent needs in order to work a feature at all. `Log` is
// not among them: it is written as the work happens, so it is checked at Shipped only.
var PlannedSections = []string{"Problem", "Design", "Tasks"}

// Unwritten lists the sections an agent would need and cannot read, in document order.
//
// Shared by `audit`, which reports it, and `claim`, which refuses over it. Once a Draft
// can be claimed, the emptiness test is the only thing standing between a doc straight
// out of `kit new` and an agent sent to build it — so both have to mean the same thing
// by "nobody wrote the plan". The template's guidance lives in comments, which every
// parser here strips, so an empty section was genuinely never written.
func (f *Feature) Unwritten() []string {
	var out []string
	for _, heading := range PlannedSections {
		if f.Section(heading) == "" {
			out = append(out, heading)
		}
	}
	return out
}

// OpenTasks lists unticked task boxes in document order.
func (f *Feature) OpenTasks() []string { return Unticked(f.Section("Tasks")) }

// TaskCount is the open and total box counts. A feature with no boxes at all has
// nothing for an agent to do and needs different advice from one whose boxes are
// all ticked.
func (f *Feature) TaskCount() (open, total int) { return CountTasks(f.Section("Tasks")) }

// NextTask is the one to do now. List order is the order of work.
func (f *Feature) NextTask() string {
	if tasks := f.OpenTasks(); len(tasks) > 0 {
		return tasks[0]
	}
	return ""
}

// Branch is the branch name that claims this feature: the doc's filename without
// its extension. The doc names the branch and the branch cites the doc, so either
// one gives you the other and there is no mapping to keep in sync.
func (f *Feature) Branch() string {
	return strings.TrimSuffix(filepath.Base(f.Path), ".md")
}

// FilenamePrefix is the leading number of the filename, for the id cross-check.
func (f *Feature) FilenamePrefix() string {
	if m := idPrefixRe.FindStringSubmatch(filepath.Base(f.Path)); m != nil {
		return m[1]
	}
	return ""
}
